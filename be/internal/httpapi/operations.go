package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"laci-backend/internal/domain"
	"laci-backend/internal/storage"
)

const backupLockKey int64 = 0x4c414349424b5550

var errBackupInProgress = errors.New("backup already in progress")

func (a *API) verifyRecaptcha(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if readJSON(r, &body) != nil || body.Token == "" {
		writeJSON(w, 200, map[string]any{"valid": false})
		return
	}
	if a.cfg.RecaptchaSecret == "" && a.cfg.AppEnv != "production" {
		writeJSON(w, 200, map[string]any{"valid": true})
		return
	}
	form := url.Values{"secret": {a.cfg.RecaptchaSecret}, "response": {body.Token}, "remoteip": {clientIP(r)}}
	request, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, "https://www.google.com/recaptcha/api/siteverify", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := outboundClient.Do(request)
	if err != nil {
		writeJSON(w, 200, map[string]any{"valid": false})
		return
	}
	defer response.Body.Close()
	var result struct {
		Success bool    `json:"success"`
		Score   float64 `json:"score"`
	}
	_ = json.NewDecoder(response.Body).Decode(&result)
	writeJSON(w, 200, map[string]any{"valid": result.Success && (result.Score == 0 || result.Score >= 0.5)})
}

func (a *API) download(w http.ResponseWriter, r *http.Request) {
	kind, id := chi.URLParam(r, "kind"), chi.URLParam(r, "id")
	token := r.URL.Query().Get("token")
	authorized := token != "" && a.validDownloadToken(token, kind, id)
	table := ""
	folderName := "lampiran"
	switch kind {
	case "arsip":
		table = "ArsipSurat"
		folderName = "arsip-surat"
	case "berkas-pimpinan":
		table = "BerkasPimpinan"
		folderName = "berkas-pimpinan"
	case "berkas-sp":
		table = "BerkasSP"
		folderName = "berkas-sp"
	case "pengajuan":
		table = "PengajuanBerkas"
		folderName = "pengajuan"
	default:
		http.NotFound(w, r)
		return
	}
	userID, userRole := "", ""
	if !authorized {
		user, _, authErr := a.currentUser(r)
		if authErr != nil {
			if !errors.Is(authErr, errMissingSession) && !errors.Is(authErr, pgx.ErrNoRows) {
				writeError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Layanan autentikasi tidak tersedia")
				return
			}
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token download tidak valid")
			return
		}
		if !user.IsActive {
			writeError(w, http.StatusForbidden, "ACCOUNT_INACTIVE", "Akun belum diaktifkan")
			return
		}
		userID, userRole = user.ID, user.Role
	}
	var owner, key string
	if err := a.db.QueryRow(r.Context(), `SELECT "userId",COALESCE("file",'') FROM "`+table+`" WHERE "id"=$1`, id).Scan(&owner, &key); err != nil {
		writeLookupError(w, err, "NOT_FOUND", "File tidak ditemukan")
		return
	}
	if key == "" {
		http.NotFound(w, r)
		return
	}
	if !authorized && (owner == userID || userRole == "SEKRETARIS_CABANG") {
		authorized = true
	}
	if !authorized {
		writeError(w, 401, "UNAUTHORIZED", "Token download tidak valid")
		return
	}
	if strings.HasPrefix(key, "/storage") {
		writeError(w, 410, "LEGACY_FILE", "File lama lokal tidak tersedia di cloud")
		return
	}
	encrypted, err := a.r2.Get(r.Context(), key)
	if err != nil {
		writeError(w, 502, "STORAGE_FAILED", "Gagal mengambil file")
		return
	}
	plain, err := a.crypto.DecryptFile(encrypted)
	if err != nil {
		writeError(w, 500, "DECRYPTION_FAILED", "Gagal membaca file")
		return
	}
	ext := filepath.Ext(strings.TrimSuffix(filepath.Base(key), ".enc"))
	if ext == "" {
		ext = "." + originalExt(key)
	}
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", contentType)
	disposition := "attachment"
	if r.URL.Query().Get("preview") == "true" {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s-%s%s"`, disposition, folderName, id, ext))
	w.Header().Set("Content-Length", itoa(len(plain)))
	_, _ = w.Write(plain)
}

func (a *API) validDownloadToken(token, kind, id string) bool {
	tokenID, ok := a.crypto.VerifyDownloadToken(token)
	return ok && (tokenID == kind+":"+id || tokenID == id)
}

func (a *API) downloadAlias(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("kind", kind)
		routeContext.URLParams.Add("id", chi.URLParam(r, "id"))
		a.download(w, r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeContext)))
	}
}

func (a *API) userImage(w http.ResponseWriter, r *http.Request) {
	user, _, err := a.currentUser(r)
	if err != nil && !errors.Is(err, errMissingSession) && !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Layanan autentikasi tidak tersedia")
		return
	}
	if err != nil || !user.IsActive {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Belum terautentikasi")
		return
	}
	id := chi.URLParam(r, "id")
	if id != user.ID && user.Role != "SEKRETARIS_CABANG" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Gambar pengguna tidak ditemukan")
		return
	}
	var key string
	if err := a.db.QueryRow(r.Context(), `SELECT COALESCE("image",'') FROM "User" WHERE "id"=$1`, id).Scan(&key); err != nil {
		writeLookupError(w, err, "NOT_FOUND", "Gambar pengguna tidak ditemukan")
		return
	}
	if key == "" {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
		http.Redirect(w, r, key, http.StatusTemporaryRedirect)
		return
	}
	encrypted, err := a.r2.Get(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	plain, err := a.crypto.DecryptFile(encrypted)
	if err != nil {
		writeError(w, 500, "DECRYPTION_FAILED", "Gagal membaca gambar")
		return
	}
	w.Header().Set("Content-Type", mimeForExtension(originalExt(key)))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = w.Write(plain)
}

func mimeForExtension(ext string) string {
	ext = strings.TrimPrefix(strings.ToLower(ext), ".")
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	case "pdf":
		return "application/pdf"
	case "doc":
		return "application/msword"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "ppt":
		return "application/vnd.ms-powerpoint"
	case "pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	}
	return "application/octet-stream"
}
func originalExt(key string) string {
	name := strings.TrimSuffix(filepath.Base(key), ".enc")
	parts := strings.Split(name, "-")
	if len(parts) > 2 {
		return parts[len(parts)-1]
	}
	if ext := filepath.Ext(name); ext != "" {
		return strings.TrimPrefix(ext, ".")
	}
	return "bin"
}

func (a *API) emailStats(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	var all, today, sent, failed, pending, verification, success, pUser, pAdmin, pStatus int
	err := a.db.QueryRow(r.Context(), `SELECT COUNT(*),COUNT(*) FILTER(WHERE "createdAt">=CURRENT_DATE),COUNT(*) FILTER(WHERE "status"='SENT'),COUNT(*) FILTER(WHERE "status"='FAILED'),COUNT(*) FILTER(WHERE "status"='PENDING'),COUNT(*) FILTER(WHERE "type"='VERIFICATION'),COUNT(*) FILTER(WHERE "type"='VERIFIED_SUCCESS'),COUNT(*) FILTER(WHERE "type"='PENGAJUAN_USER'),COUNT(*) FILTER(WHERE "type"='PENGAJUAN_ADMIN'),COUNT(*) FILTER(WHERE "type"='PENGAJUAN_STATUS') FROM "LogEmail"`).Scan(&all, &today, &sent, &failed, &pending, &verification, &success, &pUser, &pAdmin, &pStatus)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil statistik email")
		return
	}
	writeJSON(w, 200, map[string]int{"totalAll": all, "totalToday": today, "totalSent": sent, "totalFailed": failed, "totalPending": pending, "verification": verification, "verifiedSuccess": success, "pengajuanUser": pUser, "pengajuanAdmin": pAdmin, "pengajuanStatus": pStatus})
}

func (a *API) listEmailLogs(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	page, limit := parsePage(r)
	filters := []string{}
	args := []any{}
	add := func(condition string, value any) {
		args = append(args, value)
		filters = append(filters, fmt.Sprintf(condition, len(args)))
	}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		args = append(args, q)
		filters = append(filters, fmt.Sprintf(`(l."to" ILIKE '%%' || $%d || '%%' OR l."subject" ILIKE '%%' || $%d || '%%')`, len(args), len(args)))
	}
	if status := r.URL.Query().Get("status"); status != "" && status != "ALL" {
		if status != "SENT" && status != "FAILED" && status != "PENDING" {
			writeError(w, http.StatusBadRequest, "INVALID_FILTER", "Status email tidak valid")
			return
		}
		add(`l."status"::text=$%d`, status)
	}
	if kind := r.URL.Query().Get("type"); kind != "" && kind != "ALL" {
		if kind != "VERIFICATION" && kind != "VERIFIED_SUCCESS" && kind != "PENGAJUAN_USER" && kind != "PENGAJUAN_ADMIN" && kind != "PENGAJUAN_STATUS" {
			writeError(w, http.StatusBadRequest, "INVALID_FILTER", "Jenis email tidak valid")
			return
		}
		add(`l."type"::text=$%d`, kind)
	}
	if raw := r.URL.Query().Get("dateFrom"); raw != "" {
		date, err := time.Parse("2006-01-02", raw)
		if err != nil {
			writeError(w, 400, "INVALID_DATE", "Tanggal awal tidak valid")
			return
		}
		add(`l."createdAt">=$%d`, date)
	}
	if raw := r.URL.Query().Get("dateTo"); raw != "" {
		date, err := time.Parse("2006-01-02", raw)
		if err != nil {
			writeError(w, 400, "INVALID_DATE", "Tanggal akhir tidak valid")
			return
		}
		add(`l."createdAt"<$%d`, date.AddDate(0, 0, 1))
	}
	where := ""
	if len(filters) > 0 {
		where = " WHERE " + strings.Join(filters, " AND ")
	}
	var total int
	if err := a.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM "LogEmail" l`+where, args...).Scan(&total); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menghitung log email")
		return
	}
	if total == 0 || page > totalPages(total, limit) {
		writeJSON(w, 200, map[string]any{"data": []any{}, "total": total, "totalPages": totalPages(total, limit), "currentPage": page})
		return
	}
	sortColumn := `l."createdAt"`
	switch r.URL.Query().Get("sortKey") {
	case "to":
		sortColumn = `LOWER(l."to")`
	case "subject":
		sortColumn = `LOWER(l."subject")`
	case "type":
		sortColumn = `LOWER(l."type"::text)`
	case "status":
		sortColumn = `LOWER(l."status"::text)`
	}
	direction := "DESC"
	if strings.EqualFold(r.URL.Query().Get("sortDir"), "asc") {
		direction = "ASC"
	}
	query := `SELECT l."id",l."to",l."subject",l."type"::text,l."status"::text,l."errorMessage",l."retryCount",l."metadata",l."createdAt",l."updatedAt",COALESCE(u."emailVerified",false) AS "isVerified" FROM "LogEmail" l LEFT JOIN "User" u ON LOWER(u."email")=LOWER(l."to")` + where + ` ORDER BY ` + sortColumn + ` ` + direction + `,l."id" ` + direction + fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, limit, (page-1)*limit)
	rows, err := a.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil log email")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil log email")
		return
	}
	writeJSON(w, 200, map[string]any{"data": data, "total": total, "totalPages": totalPages(total, limit), "currentPage": page})
}

func (a *API) retryEmail(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	id := chi.URLParam(r, "id")
	release, locked, err := a.acquireEmailRetryLock(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Layanan email tidak tersedia")
		return
	}
	if !locked {
		writeError(w, http.StatusConflict, "RETRY_IN_PROGRESS", "Pengiriman ulang email sedang berjalan")
		return
	}
	defer release()
	var to, subject, kind, status string
	var metadata *string
	if err := a.db.QueryRow(r.Context(), `SELECT "to","subject","type"::text,"status"::text,"metadata" FROM "LogEmail" WHERE "id"=$1`, id).Scan(&to, &subject, &kind, &status, &metadata); err != nil {
		writeLookupError(w, err, "NOT_FOUND", "Log tidak ditemukan")
		return
	}
	if status == "SENT" {
		writeError(w, 400, "ALREADY_SENT", "Email sudah terkirim")
		return
	}
	var meta map[string]string
	if metadata != nil && json.Unmarshal([]byte(*metadata), &meta) != nil {
		writeError(w, 500, "DATA_UNREADABLE", "Metadata email tidak dapat dibaca")
		return
	}
	var body string
	switch kind {
	case "VERIFICATION":
		tx, err := a.db.Begin(r.Context())
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menyiapkan OTP")
			return
		}
		defer tx.Rollback(r.Context())
		var name string
		err = tx.QueryRow(r.Context(), `SELECT "name" FROM "User" WHERE LOWER("email")=LOWER($1) AND "emailVerified"=false FOR UPDATE`, to).Scan(&name)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, 409, "VERIFICATION_UNAVAILABLE", "Akun tidak ditemukan atau sudah terverifikasi")
			} else {
				writeError(w, 503, "DATABASE_UNAVAILABLE", "Layanan verifikasi tidak tersedia")
			}
			return
		}
		otp, err := randomDigits(6)
		if err != nil {
			writeError(w, 500, "OTP_FAILED", "Gagal membuat OTP")
			return
		}
		if _, err := tx.Exec(r.Context(), `DELETE FROM "Verification" WHERE "identifier"=$1`, strings.ToLower(to)); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menyiapkan OTP")
			return
		}
		if _, err := tx.Exec(r.Context(), `INSERT INTO "Verification"("id","identifier","value","expiresAt","createdAt","updatedAt") VALUES($1,$2,$3,NOW()+INTERVAL '5 minutes',NOW(),NOW())`, newID(), strings.ToLower(to), a.otpDigest(strings.ToLower(to), otp)); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menyiapkan OTP")
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menyiapkan OTP")
			return
		}
		body = verificationEmailHTML(name, otp)
	case "VERIFIED_SUCCESS":
		var name string
		if err := a.db.QueryRow(r.Context(), `SELECT "name" FROM "User" WHERE LOWER("email")=LOWER($1) AND "emailVerified"=true`, to).Scan(&name); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, 409, "USER_NOT_VERIFIED", "Akun tidak ditemukan atau belum terverifikasi")
			} else {
				writeError(w, 503, "DATABASE_UNAVAILABLE", "Layanan pengguna tidak tersedia")
			}
			return
		}
		body = verifiedSuccessEmailHTML(name)
	case "PENGAJUAN_USER", "PENGAJUAN_ADMIN", "PENGAJUAN_STATUS":
		var err error
		body, err = a.pengajuanRetryBody(r.Context(), kind, meta)
		if err != nil {
			writeError(w, 409, "PENGAJUAN_UNAVAILABLE", "Pengajuan tidak tersedia atau status telah berubah")
			return
		}
	default:
		writeError(w, 422, "RETRY_UNSUPPORTED", "Jenis email tidak didukung untuk pengiriman ulang")
		return
	}
	if _, err := a.db.Exec(r.Context(), `UPDATE "LogEmail" SET "status"='PENDING',"retryCount"="retryCount"+1,"errorMessage"=NULL,"updatedAt"=NOW() WHERE "id"=$1`, id); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal memperbarui status email")
		return
	}
	cleanHeader := strings.NewReplacer("\r", " ", "\n", " ")
	message := []byte("From: " + cleanHeader.Replace(a.cfg.MailFromName) + " <" + a.cfg.MailFromAddress + ">\r\nTo: " + cleanHeader.Replace(to) + "\r\nSubject: " + cleanHeader.Replace(subject) + "\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n" + body)
	err = sendSMTP(r.Context(), a.cfg.SMTPHost, a.cfg.SMTPPort, a.cfg.SMTPUser, a.cfg.SMTPPass, a.cfg.MailFromAddress, to, message)
	if err != nil {
		if _, updateErr := a.db.Exec(r.Context(), `UPDATE "LogEmail" SET "status"='FAILED',"errorMessage"=$2,"updatedAt"=NOW() WHERE "id"=$1`, id, err.Error()); updateErr != nil {
			slog.Error("email retry status update failed", "error", updateErr)
		}
		writeError(w, 502, "EMAIL_FAILED", "Gagal mengirim ulang email")
		return
	}
	if _, err := a.db.Exec(r.Context(), `UPDATE "LogEmail" SET "status"='SENT',"updatedAt"=NOW() WHERE "id"=$1`, id); err != nil {
		slog.Error("email sent but status update failed", "error", err)
		writeError(w, 500, "DATABASE_ERROR", "Email terkirim tetapi status log gagal diperbarui")
		return
	}
	a.notify(r.Context(), map[string]any{"model": "LogEmail", "type": "mutation", "id": id})
	writeJSON(w, 200, map[string]any{"success": true, "message": "Email berhasil dikirim ulang"})
}

func (a *API) acquireEmailRetryLock(ctx context.Context, id string) (func(), bool, error) {
	keyBytes := sha256.Sum256([]byte("laci-email-retry:" + id))
	key := int64(binary.BigEndian.Uint64(keyBytes[:8]))
	conn, err := a.db.Acquire(ctx)
	if err != nil {
		return nil, false, err
	}
	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, key).Scan(&locked); err != nil {
		conn.Release()
		return nil, false, err
	}
	if !locked {
		conn.Release()
		return nil, false, nil
	}
	return func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var unlocked bool
		if err := conn.QueryRow(unlockCtx, `SELECT pg_advisory_unlock($1)`, key).Scan(&unlocked); err != nil || !unlocked {
			slog.Error("email retry lock release failed", "id", id, "error", err)
			_ = conn.Conn().Close(unlockCtx)
		}
		conn.Release()
	}, true, nil
}

func (a *API) pengajuanRetryBody(ctx context.Context, kind string, metadata map[string]string) (string, error) {
	id := metadata["id"]
	if id == "" {
		return "", errors.New("missing pengajuan id")
	}
	var encryptedNumber, name, status string
	var createdAt time.Time
	var encryptedReason *string
	err := a.db.QueryRow(ctx, `SELECT b."noSurat",u."name",b."status"::text,b."alasanPenolakan",b."createdAt" FROM "PengajuanBerkas" b JOIN "User" u ON u."id"=b."userId" WHERE b."id"=$1`, id).Scan(&encryptedNumber, &name, &status, &encryptedReason, &createdAt)
	if err != nil {
		return "", err
	}
	number, err := a.crypto.DecryptText(encryptedNumber)
	if err != nil {
		return "", err
	}
	data := pengajuanEmailData{UserName: name, PACName: name, Number: number, Date: createdAt.Format("02/01/2006"), DetailURL: a.cfg.FrontendURL + "/dashboard/pengajuan-berkas/" + id, Status: status}
	switch kind {
	case "PENGAJUAN_USER":
		if status != "PENDING" {
			return "", errors.New("pengajuan status changed")
		}
		return pengajuanCreatedUserEmailHTML(data), nil
	case "PENGAJUAN_ADMIN":
		if status != "PENDING" {
			return "", errors.New("pengajuan status changed")
		}
		return pengajuanCreatedAdminEmailHTML(data), nil
	case "PENGAJUAN_STATUS":
		if metadata["status"] != status {
			return "", errors.New("pengajuan status changed")
		}
		if encryptedReason != nil {
			data.Reason, err = a.crypto.DecryptText(*encryptedReason)
			if err != nil {
				return "", err
			}
		}
		return pengajuanStatusEmailHTML(data), nil
	}
	return "", pgx.ErrNoRows
}

func (a *API) listBackups(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	objects, err := a.r2.List(r.Context(), "backups/")
	if err != nil {
		writeError(w, 502, "STORAGE_FAILED", "Gagal mengambil daftar backup")
		return
	}
	sort.Slice(objects, func(i, j int) bool {
		if objects[i].LastModified.Equal(objects[j].LastModified) {
			return objects[i].Key > objects[j].Key
		}
		return objects[i].LastModified.After(objects[j].LastModified)
	})
	items := make([]map[string]any, 0, len(objects))
	for _, object := range objects {
		if !validBackupKey(object.Key) {
			continue
		}
		items = append(items, map[string]any{"key": object.Key, "filename": strings.TrimPrefix(object.Key, "backups/"), "size": object.Size, "lastModified": object.LastModified})
	}
	writeJSON(w, 200, map[string]any{"data": items})
}
func (a *API) createBackup(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	result, err := a.executeBackup(r.Context())
	if err != nil {
		if errors.Is(err, errBackupInProgress) {
			writeError(w, http.StatusConflict, "BACKUP_IN_PROGRESS", "Backup sedang berjalan")
			return
		}
		writeError(w, 500, "BACKUP_FAILED", "Gagal membuat backup database")
		return
	}
	if err := a.createLog(r.Context(), r, authFrom(r).User, "CREATE", "USER", "Berhasil melakukan backup database manual: "+result, ""); err != nil {
		slog.Warn("backup activity logging failed", "error", err)
	}
	writeJSON(w, 201, map[string]any{"success": "Database berhasil dibackup!", "filename": result})
}
func (a *API) cronBackup(w http.ResponseWriter, r *http.Request) {
	if a.cfg.CronSecret == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+a.cfg.CronSecret)) != 1 {
		writeError(w, 401, "UNAUTHORIZED", "Unauthorized")
		return
	}
	name, err := a.executeBackup(r.Context())
	if err != nil {
		if errors.Is(err, errBackupInProgress) {
			writeError(w, http.StatusConflict, "BACKUP_IN_PROGRESS", "Backup sedang berjalan")
			return
		}
		writeError(w, 500, "BACKUP_FAILED", "Gagal membuat backup database")
		return
	}
	var adminID string
	if err := a.db.QueryRow(r.Context(), `SELECT "id" FROM "User" WHERE "role"='SEKRETARIS_CABANG' AND "isActive"=true ORDER BY "createdAt" LIMIT 1`).Scan(&adminID); err == nil {
		if err := a.createLog(r.Context(), r, domain.User{ID: adminID}, "CREATE", "USER", "Berhasil melakukan backup database otomatis (Cron): "+name, ""); err != nil {
			slog.Warn("cron backup activity logging failed", "error", err)
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		slog.Warn("cron backup activity logging failed", "error", err)
	}
	writeJSON(w, 200, map[string]any{"success": true, "filename": name})
}
func (a *API) executeBackup(ctx context.Context) (string, error) {
	conn, err := a.db.Acquire(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Release()
	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, backupLockKey).Scan(&locked); err != nil {
		return "", err
	}
	if !locked {
		return "", errBackupInProgress
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var unlocked bool
		if err := conn.QueryRow(unlockCtx, `SELECT pg_advisory_unlock($1)`, backupLockKey).Scan(&unlocked); err != nil || !unlocked {
			slog.Error("backup lock release failed", "error", err)
			_ = conn.Conn().Close(unlockCtx)
		}
	}()
	tmp, err := os.CreateTemp("", "laci-backup-*.dump")
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(path)
	databaseURL, password, err := pgDumpConnection(a.cfg.DatabaseURL)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "pg_dump", "--dbname="+databaseURL, "--format=custom", "--no-owner", "--no-privileges", "--file="+path)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "PGPASSWORD=" + password}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pg_dump gagal: %s", strings.TrimSpace(stderr.String()))
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	encrypted, err := os.CreateTemp("", "laci-backup-*.dump.enc")
	if err != nil {
		return "", err
	}
	defer os.Remove(encrypted.Name())
	defer encrypted.Close()
	if err := a.crypto.EncryptBackup(ctx, encrypted, file); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	info, err := encrypted.Stat()
	if err != nil {
		return "", err
	}
	if _, err := encrypted.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	existing, err := a.r2.List(ctx, "backups/")
	if err != nil {
		return "", err
	}
	name := "laci-" + time.Now().UTC().Format("20060102-150405") + "-" + newID() + ".dump.enc"
	if err := a.r2.PutReader(ctx, "backups/"+name, "application/octet-stream", encrypted, info.Size()); err != nil {
		return "", err
	}
	for _, key := range oldestBackupKeys(existing) {
		if err := a.r2.Delete(ctx, key); err != nil {
			slog.Error("backup retention failed", "key", key, "error", err)
		}
	}
	return name, nil
}

func oldestBackupKeys(objects []storage.Object) []string {
	valid := make([]storage.Object, 0, len(objects))
	for _, object := range objects {
		if validBackupKey(object.Key) {
			valid = append(valid, object)
		}
	}
	count := len(valid) - 9
	if count <= 0 {
		return nil
	}
	sort.Slice(valid, func(i, j int) bool {
		if valid[i].LastModified.Equal(valid[j].LastModified) {
			return valid[i].Key < valid[j].Key
		}
		return valid[i].LastModified.Before(valid[j].LastModified)
	})
	keys := make([]string, 0, count)
	for _, object := range valid[:count] {
		keys = append(keys, object.Key)
	}
	return keys
}

func pgDumpConnection(databaseURL string) (string, string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return "", "", errors.New("backup requires a PostgreSQL connection URL")
	}
	password := ""
	if parsed.User != nil {
		username := parsed.User.Username()
		password, _ = parsed.User.Password()
		parsed.User = url.User(username)
	}
	query := parsed.Query()
	if value := query.Get("password"); value != "" {
		password = value
	}
	query.Del("password")
	parsed.RawQuery = query.Encode()
	return parsed.String(), password, nil
}
func (a *API) deleteBackup(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	key := r.URL.Query().Get("key")
	if !validBackupKey(key) {
		writeError(w, 400, "INVALID_KEY", "Key backup tidak valid")
		return
	}
	if err := a.r2.Delete(r.Context(), key); err != nil {
		writeError(w, 502, "STORAGE_FAILED", "Gagal menghapus backup")
		return
	}
	if err := a.createLog(r.Context(), r, authFrom(r).User, "DELETE", "USER", "Berhasil menghapus backup database: "+strings.TrimPrefix(key, "backups/"), ""); err != nil {
		slog.Warn("backup deletion activity logging failed", "error", err)
	}
	writeJSON(w, 200, map[string]any{"success": "Backup database berhasil dihapus!"})
}
func (a *API) backupDownloadURL(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	key := r.URL.Query().Get("key")
	if !validBackupKey(key) {
		writeError(w, 400, "INVALID_KEY", "Key backup tidak valid")
		return
	}
	if strings.HasSuffix(key, ".dump.enc") {
		writeJSON(w, 200, map[string]any{"url": "/api/backups/file?key=" + url.QueryEscape(key)})
		return
	}
	signed, err := a.r2.SignedURL(r.Context(), key, 10*time.Minute)
	if err != nil {
		writeError(w, 502, "STORAGE_FAILED", "Gagal membuat tautan download")
		return
	}
	writeJSON(w, 200, map[string]any{"url": signed})
}

func (a *API) downloadEncryptedBackup(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	key := r.URL.Query().Get("key")
	if !validBackupKey(key) || !strings.HasSuffix(key, ".dump.enc") {
		writeError(w, 400, "INVALID_KEY", "Key backup tidak valid")
		return
	}
	object, err := a.r2.Open(r.Context(), key)
	if err != nil {
		writeError(w, 502, "STORAGE_FAILED", "Gagal mengambil backup")
		return
	}
	defer object.Close()
	plain, err := os.CreateTemp("", "laci-backup-download-*.dump")
	if err != nil {
		writeError(w, 500, "BACKUP_FAILED", "Gagal menyiapkan unduhan backup")
		return
	}
	defer os.Remove(plain.Name())
	defer plain.Close()
	if err := a.crypto.DecryptBackup(r.Context(), plain, object); err != nil {
		writeError(w, 502, "BACKUP_INVALID", "Backup tidak dapat didekripsi")
		return
	}
	if _, err := plain.Seek(0, io.SeekStart); err != nil {
		writeError(w, 500, "BACKUP_FAILED", "Gagal menyiapkan unduhan backup")
		return
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": strings.TrimSuffix(strings.TrimPrefix(key, "backups/"), ".enc")}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeContent(w, r, "", time.Time{}, plain)
}

func validBackupKey(key string) bool {
	name := strings.TrimPrefix(key, "backups/")
	return strings.HasPrefix(key, "backups/") && name != "" && !strings.Contains(name, "..") && !strings.ContainsAny(name, "/\\\"") && strings.IndexFunc(name, unicode.IsControl) == -1
}

func sendSMTP(ctx context.Context, host string, port int, user, password, from, to string, message []byte) error {
	if host == "" || from == "" {
		return fmt.Errorf("SMTP configuration missing")
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))
	connection, err := (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	defer connection.Close()
	stop := context.AfterFunc(ctx, func() { _ = connection.Close() })
	defer stop()
	deadline := time.Now().Add(15 * time.Second)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err := connection.SetDeadline(deadline); err != nil {
		return err
	}
	if port == 465 {
		secure := tls.Client(connection, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
		if err := secure.HandshakeContext(ctx); err != nil {
			return err
		}
		connection = secure
	}
	client, err := smtp.NewClient(connection, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
				return err
			}
		} else if !isLoopbackSMTPHost(host) {
			return errors.New("SMTP server does not support STARTTLS")
		}
	}
	if user != "" {
		if err := client.Auth(smtp.PlainAuth("", user, password, host)); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func isLoopbackSMTPHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

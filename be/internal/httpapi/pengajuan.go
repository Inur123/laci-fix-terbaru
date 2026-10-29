package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"laci-backend/internal/domain"
)

const (
	pengajuanNumberMaxLength       = 255
	pengajuanPurposeMaxLength      = 1000
	pengajuanDescriptionMaxLength  = 5000
	pengajuanRejectReasonMaxLength = 2000
)

func (a *API) activePACUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(r.Context(), `SELECT "id","name" FROM "User" WHERE "role"='SEKRETARIS_PAC' AND "isActive"=true AND "emailVerified"=true ORDER BY "name"`)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil pengguna PAC")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca pengguna PAC")
		return
	}
	writeJSON(w, 200, map[string]any{"data": data})
}

func (a *API) listPengajuan(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	mode := "pac"
	if auth.User.Role == "SEKRETARIS_CABANG" {
		mode = "cabang"
	}
	a.listPengajuanMode(w, r, mode)
}
func (a *API) listReferencePengajuan(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_PAC") {
		return
	}
	a.listPengajuanMode(w, r, "reference")
}

func (a *API) listPengajuanMode(w http.ResponseWriter, r *http.Request, mode string) {
	auth := authFrom(r)
	page, limit := parsePage(r)
	periodID := ""
	var err error
	switch mode {
	case "pac":
		periodID, err = a.targetPeriode(r.Context(), auth.User.ID, viewPeriode(r))
	case "cabang":
		periodID, err = a.targetPeriode(r.Context(), auth.User.ID, viewPeriode(r))
	case "reference":
		err = a.db.QueryRow(r.Context(), `SELECT p."id" FROM "Periode" p JOIN "User" u ON u."id"=p."userId" WHERE p."isActive"=true AND u."role"='SEKRETARIS_CABANG' LIMIT 1`).Scan(&periodID)
	}
	if err != nil {
		if periodLookupFailed(w, err) {
			return
		}
		writeJSON(w, 200, map[string]any{"data": []any{}, "total": 0, "totalPages": 0})
		return
	}
	where := `b."periodeIdPac"=$1 AND b."userId"=$2`
	args := []any{periodID, auth.User.ID}
	if mode == "cabang" || mode == "reference" {
		where = `b."periodeId"=$1`
		args = []any{periodID}
	}
	pac := r.URL.Query().Get("pac")
	if pac == "" {
		pac = r.URL.Query().Get("userId")
	}
	if (mode == "cabang" || mode == "reference") && pac != "" && pac != "ALL" {
		where += ` AND b."userId"=$2`
		args = append(args, pac)
	}
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	status := r.URL.Query().Get("status")
	receiver := r.URL.Query().Get("penerima")
	if status != "" && status != "ALL" && !oneOf(status, "PENDING", "DITERIMA", "DITOLAK") {
		writeError(w, 400, "INVALID_FILTER", "Filter status tidak valid")
		return
	}
	if receiver != "" && receiver != "ALL" && !oneOf(receiver, "IPNU", "IPPNU", "BERSAMA", "CBP_KPP") {
		writeError(w, 400, "INVALID_FILTER", "Filter penerima tidak valid")
		return
	}
	if status != "" && status != "ALL" {
		args = append(args, status)
		where += ` AND b."status"=$` + strconv.Itoa(len(args)) + `::"StatusPengajuan"`
	}
	if receiver != "" && receiver != "ALL" {
		args = append(args, receiver)
		where += ` AND b."penerima"=$` + strconv.Itoa(len(args)) + `::"PenerimaSurat"`
	}
	const columns = `SELECT b."id",b."userId",b."periodeId",b."periodeIdPac",b."noSurat",b."penerima"::text,b."tanggal",b."keperluan",b."deskripsi",b."file",b."status"::text,b."alasanPenolakan",b."createdAt",b."updatedAt",u."name" AS "userName",u."email" AS "userEmail",pp."nama" AS "periodePacNama",pc."nama" AS "periodeCabangNama" FROM "PengajuanBerkas" b JOIN "User" u ON u."id"=b."userId" JOIN "Periode" pp ON pp."id"=b."periodeIdPac" JOIN "Periode" pc ON pc."id"=b."periodeId" WHERE `
	sortKey := r.URL.Query().Get("sortKey")
	sortColumn, sqlSort := pengajuanSortColumn(sortKey)
	if q == "" && sqlSort {
		var total int
		if err := a.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM "PengajuanBerkas" b WHERE `+where, args...).Scan(&total); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menghitung pengajuan")
			return
		}
		if total == 0 || page > totalPages(total, limit) {
			writeJSON(w, 200, map[string]any{"data": []any{}, "total": total, "totalPages": totalPages(total, limit)})
			return
		}
		dir := "DESC"
		if r.URL.Query().Get("sortDir") == "asc" {
			dir = "ASC"
		}
		pageArgs := append(append([]any{}, args...), limit, (page-1)*limit)
		rows, err := a.db.Query(r.Context(), columns+where+` ORDER BY `+sortColumn+` `+dir+`, b."id" `+dir+` LIMIT $`+strconv.Itoa(len(args)+1)+` OFFSET $`+strconv.Itoa(len(args)+2), pageArgs...)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil pengajuan")
			return
		}
		data, err := scanMaps(rows)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil pengajuan")
			return
		}
		for _, item := range data {
			if err := enrichPengajuanListItem(a, item); err != nil {
				writeError(w, 500, "DATA_UNREADABLE", "Data pengajuan tidak dapat dibaca")
				return
			}
		}
		writeJSON(w, 200, map[string]any{"data": data, "total": total, "totalPages": totalPages(total, limit)})
		return
	}
	rows, err := a.db.Query(r.Context(), columns+where+` ORDER BY b."createdAt" DESC, b."id" DESC`, args...)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil pengajuan")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil pengajuan")
		return
	}
	filtered := []map[string]any{}
	for _, item := range data {
		if err := enrichPengajuanListItem(a, item); err != nil {
			writeError(w, 500, "DATA_UNREADABLE", "Data pengajuan tidak dapat dibaca")
			return
		}
		if q != "" && !mapContains(item, q, "noSurat", "keperluan", "userName") {
			continue
		}
		filtered = append(filtered, item)
	}
	if sortKey == "pengaju" {
		sortKey = "userName"
	}
	sortMaps(filtered, sortKey, r.URL.Query().Get("sortDir"), "createdAt")
	writeJSON(w, 200, paginateMaps(filtered, page, limit))
}

func pengajuanSortColumn(key string) (string, bool) {
	switch key {
	case "":
		return `b."createdAt"`, true
	case "tanggal":
		return `b."tanggal"`, true
	case "createdAt":
		return `b."createdAt"`, true
	case "updatedAt":
		return `b."updatedAt"`, true
	case "penerima":
		return `b."penerima"::text`, true
	case "status":
		return `b."status"::text`, true
	default:
		return "", false
	}
}

func enrichPengajuanListItem(a *API, item map[string]any) error {
	if err := decryptFieldStrict(a.crypto, item, "noSurat", "keperluan", "deskripsi", "alasanPenolakan"); err != nil {
		return err
	}
	item["user"] = map[string]any{"name": item["userName"], "email": item["userEmail"]}
	item["periodePac"] = map[string]any{"id": item["periodeIdPac"], "nama": item["periodePacNama"]}
	item["periodeCabang"] = map[string]any{"id": item["periodeId"], "nama": item["periodeCabangNama"]}
	return nil
}

func (a *API) getPengajuan(w http.ResponseWriter, r *http.Request) { a.getPengajuanMode(w, r, false) }
func (a *API) getReferencePengajuan(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_PAC") {
		return
	}
	a.getPengajuanMode(w, r, true)
}
func (a *API) getPengajuanMode(w http.ResponseWriter, r *http.Request, reference bool) {
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	where := `b."id"=$1 AND (b."userId"=$2 OR $3::text='SEKRETARIS_CABANG')`
	args := []any{id, auth.User.ID, auth.User.Role}
	if reference {
		where = `b."id"=$1 AND b."periodeId"=(SELECT p."id" FROM "Periode" p JOIN "User" u ON u."id"=p."userId" WHERE p."isActive"=true AND u."role"='SEKRETARIS_CABANG' LIMIT 1)`
		args = []any{id}
	}
	rows, err := a.db.Query(r.Context(), `SELECT b."id",b."userId",b."periodeId",b."periodeIdPac",b."noSurat",b."penerima"::text,b."tanggal",b."keperluan",b."deskripsi",b."file",b."status"::text,b."alasanPenolakan",b."createdAt",b."updatedAt",u."name" AS "userName",u."email" AS "userEmail",pp."nama" AS "periodePacNama",pc."nama" AS "periodeCabangNama" FROM "PengajuanBerkas" b JOIN "User" u ON u."id"=b."userId" JOIN "Periode" pp ON pp."id"=b."periodeIdPac" JOIN "Periode" pc ON pc."id"=b."periodeId" WHERE `+where, args...)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil pengajuan")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca pengajuan")
		return
	}
	if len(data) == 0 {
		writeError(w, 404, "NOT_FOUND", "Pengajuan tidak ditemukan")
		return
	}
	item := data[0]
	if err := enrichPengajuanListItem(a, item); err != nil {
		writeError(w, 500, "DATA_UNREADABLE", "Data pengajuan tidak dapat dibaca")
		return
	}
	writeJSON(w, 200, map[string]any{"data": item})
}

func (a *API) createPengajuan(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_PAC") {
		return
	}
	a.mutatePengajuan(w, r, true)
}
func (a *API) updatePengajuan(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_PAC") {
		return
	}
	a.mutatePengajuan(w, r, false)
}
func (a *API) mutatePengajuan(w http.ResponseWriter, r *http.Request, creating bool) {
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		writeError(w, 400, "INVALID_FORM", "Form pengajuan tidak valid")
		return
	}
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	pacPeriod := ""
	cabangPeriod := ""
	oldFile := ""
	if creating {
		var err error
		pacPeriod, err = a.targetPeriode(r.Context(), auth.User.ID, "")
		if err != nil {
			if periodLookupFailed(w, err) {
				return
			}
			writeError(w, 400, "NO_ACTIVE_PERIODE", "Tidak ada periode aktif. Silakan aktifkan periode terlebih dahulu.")
			return
		}
		err = a.db.QueryRow(r.Context(), `SELECT p."id" FROM "Periode" p JOIN "User" u ON u."id"=p."userId" WHERE p."isActive"=true AND u."role"='SEKRETARIS_CABANG' LIMIT 1`).Scan(&cabangPeriod)
		if err != nil {
			if periodLookupFailed(w, err) {
				return
			}
			writeError(w, 400, "NO_CABANG_PERIODE", "Tidak ada periode aktif Cabang. Hubungi Sekretaris Cabang.")
			return
		}
		id = newID()
	} else {
		var status string
		if err := a.db.QueryRow(r.Context(), `SELECT "periodeIdPac","periodeId",COALESCE("file",''),"status"::text FROM "PengajuanBerkas" WHERE "id"=$1 AND "userId"=$2`, id, auth.User.ID).Scan(&pacPeriod, &cabangPeriod, &oldFile, &status); err != nil {
			writeLookupError(w, err, "NOT_FOUND", "Pengajuan tidak ditemukan")
			return
		}
		if status != "PENDING" {
			writeError(w, 400, "NOT_PENDING", "Hanya pengajuan dengan status PENDING yang dapat diedit")
			return
		}
	}
	number := strings.TrimSpace(r.FormValue("noSurat"))
	receiver := strings.TrimSpace(r.FormValue("penerima"))
	purpose := strings.TrimSpace(r.FormValue("keperluan"))
	description := strings.TrimSpace(r.FormValue("deskripsi"))
	date, err := parseDate(r.FormValue("tanggal"))
	if number == "" || purpose == "" || err != nil || !oneOf(receiver, "IPNU", "IPPNU", "BERSAMA", "CBP_KPP") {
		writeError(w, 400, "VALIDATION_ERROR", "Nomor surat, penerima, tanggal, dan keperluan wajib diisi")
		return
	}
	if utf8.RuneCountInString(number) > pengajuanNumberMaxLength {
		writeError(w, 400, "VALIDATION_ERROR", "Nomor surat maksimal 255 karakter")
		return
	}
	if utf8.RuneCountInString(purpose) > pengajuanPurposeMaxLength {
		writeError(w, 400, "VALIDATION_ERROR", "Keperluan maksimal 1000 karakter")
		return
	}
	if utf8.RuneCountInString(description) > pengajuanDescriptionMaxLength {
		writeError(w, 400, "VALIDATION_ERROR", "Deskripsi maksimal 5000 karakter")
		return
	}
	fileKey, err := a.handleOptionalUpload(r, "file", "pengajuan-berkas", 2<<20, oldFile)
	if err != nil {
		writeUploadFailure(w, err)
		return
	}
	if creating && fileKey == "" {
		writeError(w, 400, "FILE_REQUIRED", "File harus diunggah")
		return
	}
	newFile := fileKey != "" && fileKey != oldFile
	saved := false
	defer func() {
		if newFile && !saved {
			a.deleteStorageObject(fileKey)
		}
	}()
	encrypted, err := a.encryptStrings(number, purpose, description)
	if err != nil {
		writeEncryptionError(w)
		return
	}
	encNumber, encPurpose, encDesc := encrypted[0], encrypted[1], nullable(encrypted[2])
	var result pgconn.CommandTag
	if creating {
		result, err = a.db.Exec(r.Context(), `INSERT INTO "PengajuanBerkas"("id","userId","periodeId","periodeIdPac","noSurat","penerima","tanggal","keperluan","deskripsi","file","status","createdAt","updatedAt") VALUES($1,$2,$3,$4,$5,$6::"PenerimaSurat",$7,$8,$9,$10,'PENDING',NOW(),NOW())`, id, auth.User.ID, cabangPeriod, pacPeriod, encNumber, receiver, date, encPurpose, encDesc, fileKey)
	} else {
		result, err = a.db.Exec(r.Context(), `UPDATE "PengajuanBerkas" SET "noSurat"=$3,"penerima"=$4::"PenerimaSurat","tanggal"=$5,"keperluan"=$6,"deskripsi"=$7,"file"=$8,"updatedAt"=NOW() WHERE "id"=$1 AND "userId"=$2 AND "status"='PENDING'`, id, auth.User.ID, encNumber, receiver, date, encPurpose, encDesc, fileKey)
	}
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menyimpan pengajuan")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, 409, "NOT_PENDING", "Hanya pengajuan dengan status PENDING yang dapat diedit")
		return
	}
	saved = true
	if fileKey != oldFile && oldFile != "" && !strings.HasPrefix(oldFile, "/storage") {
		a.deleteStorageObject(oldFile)
	}
	action, message := "UPDATE", "Pengajuan berhasil diperbarui!"
	if creating {
		action = "CREATE"
		message = "Pengajuan berhasil dibuat!"
	}
	_ = a.createLog(r.Context(), r, auth.User, action, "PENGAJUAN_BERKAS", action+" pengajuan PAC: "+number, id)
	a.notify(r.Context(), map[string]any{"type": "pengajuan.changed", "id": id, "userId": auth.User.ID})
	if creating {
		a.sendBackgroundMail(r.Context(), func(ctx context.Context) {
			a.sendPengajuanCreated(ctx, auth.User, number, id)
		})
	}
	writeJSON(w, map[bool]int{true: 201, false: 200}[creating], map[string]any{"success": message, "data": map[string]string{"id": id}})
}

func (a *API) deletePengajuan(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	var owner, file, number string
	if err := a.db.QueryRow(r.Context(), `SELECT "userId","file","noSurat" FROM "PengajuanBerkas" WHERE "id"=$1`, id).Scan(&owner, &file, &number); err != nil {
		writeLookupError(w, err, "NOT_FOUND", "Pengajuan tidak ditemukan")
		return
	}
	if owner != auth.User.ID && auth.User.Role != "SEKRETARIS_CABANG" {
		writeError(w, 403, "FORBIDDEN", "Akses ditolak")
		return
	}
	if _, err := a.db.Exec(r.Context(), `DELETE FROM "PengajuanBerkas" WHERE "id"=$1`, id); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menghapus pengajuan")
		return
	}
	if file != "" && !strings.HasPrefix(file, "/storage") {
		a.deleteStorageObject(file)
	}
	plain, _ := a.crypto.DecryptText(number)
	_ = a.createLog(r.Context(), r, auth.User, "DELETE", "PENGAJUAN_BERKAS", "Menghapus pengajuan PAC: "+plain, id)
	a.notify(r.Context(), map[string]any{"type": "pengajuan.deleted", "id": id, "userId": auth.User.ID})
	writeJSON(w, 200, map[string]any{"success": "Pengajuan berhasil dihapus!"})
}

func (a *API) updatePengajuanStatus(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	var body struct {
		Status string `json:"status"`
		Alasan string `json:"alasanPenolakan"`
	}
	if readJSON(r, &body) != nil || !oneOf(body.Status, "DITERIMA", "DITOLAK") {
		writeError(w, 400, "INVALID_STATUS", "Status tidak valid")
		return
	}
	reasonText := strings.TrimSpace(body.Alasan)
	if body.Status == "DITOLAK" && reasonText == "" {
		writeError(w, 400, "REASON_REQUIRED", "Alasan penolakan harus diisi")
		return
	}
	if utf8.RuneCountInString(reasonText) > pengajuanRejectReasonMaxLength {
		writeError(w, 400, "VALIDATION_ERROR", "Alasan penolakan maksimal 2000 karakter")
		return
	}
	id := chi.URLParam(r, "id")
	var number, email, name, ownerID, currentStatus string
	if err := a.db.QueryRow(r.Context(), `SELECT b."noSurat",u."email",u."name",b."userId",b."status"::text FROM "PengajuanBerkas" b JOIN "User" u ON u."id"=b."userId" WHERE b."id"=$1`, id).Scan(&number, &email, &name, &ownerID, &currentStatus); err != nil {
		writeLookupError(w, err, "NOT_FOUND", "Pengajuan tidak ditemukan")
		return
	}
	if currentStatus != "PENDING" {
		writeError(w, http.StatusConflict, "NOT_PENDING", "Pengajuan sudah diproses")
		return
	}
	var reason any = nil
	if body.Status == "DITOLAK" {
		v, err := a.crypto.EncryptText(reasonText)
		if err != nil {
			writeEncryptionError(w)
			return
		}
		reason = v
	}
	result, err := a.db.Exec(r.Context(), `UPDATE "PengajuanBerkas" SET "status"=$2::"StatusPengajuan","alasanPenolakan"=$3,"updatedAt"=NOW() WHERE "id"=$1 AND "status"='PENDING'`, id, body.Status, reason)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengubah status pengajuan")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "NOT_PENDING", "Pengajuan sudah diproses")
		return
	}
	auth := authFrom(r)
	plain, _ := a.crypto.DecryptText(number)
	action, label := "REJECT", "ditolak"
	if body.Status == "DITERIMA" {
		action = "APPROVE"
		label = "diterima"
	}
	logLabel := "Menolak"
	if body.Status == "DITERIMA" {
		logLabel = "Menerima"
	}
	_ = a.createLog(r.Context(), r, auth.User, action, "PENGAJUAN_BERKAS", logLabel+" pengajuan PAC: "+plain, id)
	a.notify(r.Context(), map[string]any{"type": "pengajuan.status", "id": id, "userId": ownerID, "status": body.Status})
	a.sendBackgroundMail(r.Context(), func(ctx context.Context) {
		a.sendPengajuanStatus(ctx, email, name, plain, id, body.Status, reasonText)
	})
	writeJSON(w, 200, map[string]any{"success": "Pengajuan berhasil " + label + "!"})
}

func (a *API) pengajuanStats(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	periodID := ""
	column := "periodeIdPac"
	var err error
	if auth.User.Role == "SEKRETARIS_CABANG" {
		periodID, err = a.targetPeriode(r.Context(), auth.User.ID, viewPeriode(r))
		column = "periodeId"
	} else {
		periodID, err = a.targetPeriode(r.Context(), auth.User.ID, "")
	}
	if err != nil {
		if periodLookupFailed(w, err) {
			return
		}
		writeJSON(w, 200, emptyPengajuanStats())
		return
	}
	where := `"` + column + `"=$1`
	args := []any{periodID}
	if auth.User.Role != "SEKRETARIS_CABANG" {
		where += ` AND "userId"=$2`
		args = append(args, auth.User.ID)
	} else if uid := r.URL.Query().Get("userId"); uid != "" && uid != "ALL" {
		where += ` AND "userId"=$2`
		args = append(args, uid)
	}
	a.writePengajuanStats(w, r, where, args...)
}
func (a *API) referencePengajuanStats(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_PAC") {
		return
	}
	var periodID string
	if err := a.db.QueryRow(r.Context(), `SELECT p."id" FROM "Periode" p JOIN "User" u ON u."id"=p."userId" WHERE p."isActive"=true AND u."role"='SEKRETARIS_CABANG' LIMIT 1`).Scan(&periodID); err != nil {
		if periodLookupFailed(w, err) {
			return
		}
		writeJSON(w, 200, emptyPengajuanStats())
		return
	}
	a.writePengajuanStats(w, r, `"periodeId"=$1`, periodID)
}
func (a *API) writePengajuanStats(w http.ResponseWriter, r *http.Request, where string, args ...any) {
	var total, ipnu, ippnu, bersama, cbp, pending, accepted, rejected int
	err := a.db.QueryRow(r.Context(), `SELECT COUNT(*),COUNT(*) FILTER(WHERE "penerima"='IPNU'),COUNT(*) FILTER(WHERE "penerima"='IPPNU'),COUNT(*) FILTER(WHERE "penerima"='BERSAMA'),COUNT(*) FILTER(WHERE "penerima"='CBP_KPP'),COUNT(*) FILTER(WHERE "status"='PENDING'),COUNT(*) FILTER(WHERE "status"='DITERIMA'),COUNT(*) FILTER(WHERE "status"='DITOLAK') FROM "PengajuanBerkas" WHERE `+where, args...).Scan(&total, &ipnu, &ippnu, &bersama, &cbp, &pending, &accepted, &rejected)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil statistik pengajuan")
		return
	}
	writeJSON(w, 200, map[string]int{"total": total, "ipnu": ipnu, "ippnu": ippnu, "bersama": bersama, "cbpKpp": cbp, "pending": pending, "diterima": accepted, "ditolak": rejected})
}
func emptyPengajuanStats() map[string]int {
	return map[string]int{"total": 0, "ipnu": 0, "ippnu": 0, "bersama": 0, "cbpKpp": 0, "pending": 0, "diterima": 0, "ditolak": 0}
}

func (a *API) pengajuanDownloadToken(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	var owner string
	if err := a.db.QueryRow(r.Context(), `SELECT "userId" FROM "PengajuanBerkas" WHERE "id"=$1`, id).Scan(&owner); err != nil {
		writeLookupError(w, err, "NOT_FOUND", "Pengajuan tidak ditemukan")
		return
	}
	if owner != auth.User.ID && auth.User.Role != "SEKRETARIS_CABANG" {
		writeError(w, 403, "FORBIDDEN", "Akses ditolak")
		return
	}
	token, err := a.crypto.DownloadToken("pengajuan:"+id, 5*time.Minute)
	if err != nil {
		writeError(w, 500, "TOKEN_FAILED", "Gagal membuat token")
		return
	}
	writeJSON(w, 200, map[string]any{"token": token})
}

func (a *API) sendPengajuanCreated(ctx context.Context, user domain.User, number, id string) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	detail := a.cfg.FrontendURL + "/dashboard/pengajuan-berkas/" + id
	data := pengajuanEmailData{UserName: user.Name, PACName: user.Name, Number: number, Date: time.Now().Format("02/01/2006"), DetailURL: detail}
	_ = a.sendMail(ctx, user.Email, "Pengajuan Berhasil: "+user.Name+" - "+number, pengajuanCreatedUserEmailHTML(data), "PENGAJUAN_USER", map[string]any{"id": id})
	if a.cfg.AdminNotificationEmail != "" {
		_ = a.sendMail(ctx, a.cfg.AdminNotificationEmail, "[NOTIFIKASI] Pengajuan Baru: "+user.Name+" - "+number, pengajuanCreatedAdminEmailHTML(data), "PENGAJUAN_ADMIN", map[string]any{"id": id})
	}
}
func (a *API) sendPengajuanStatus(ctx context.Context, email, name, number, id, status, reason string) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	detail := a.cfg.FrontendURL + "/dashboard/pengajuan-berkas/" + id
	data := pengajuanEmailData{UserName: name, PACName: name, Number: number, DetailURL: detail, Status: status, Reason: reason}
	_ = a.sendMail(ctx, email, "Update Pengajuan: "+status+" - "+name+" ("+number+")", pengajuanStatusEmailHTML(data), "PENGAJUAN_STATUS", map[string]any{"id": id, "status": status})
}

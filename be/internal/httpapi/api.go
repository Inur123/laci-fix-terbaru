package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"laci-backend/internal/config"
	"laci-backend/internal/domain"
	"laci-backend/internal/security"
	"laci-backend/internal/storage"
)

type API struct {
	cfg         config.Config
	db          *pgxpool.Pool
	crypto      *security.Crypto
	r2          objectStore
	hub         *eventHub
	limits      *rateLimiter
	notifyQueue chan string
	instanceID  string
	streamDone  <-chan struct{}
	mail        *backgroundMailExecutor
}

type objectStore interface {
	Put(context.Context, string, string, []byte) error
	PutReader(context.Context, string, string, io.Reader, int64) error
	Get(context.Context, string) ([]byte, error)
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
	List(context.Context, string) ([]storage.Object, error)
	SignedURL(context.Context, string, time.Duration) (string, error)
}

type contextKey string

const userContextKey contextKey = "user"

var errMissingSession = errors.New("missing session")

var outboundClient = &http.Client{Timeout: 15 * time.Second}

func New(cfg config.Config, db *pgxpool.Pool, crypto *security.Crypto, r2 *storage.R2) *API {
	return &API{cfg: cfg, db: db, crypto: crypto, r2: r2, hub: newEventHub(), limits: newRateLimiter(), notifyQueue: make(chan string, 256), instanceID: newID(), mail: newBackgroundMailExecutor()}
}

func (a *API) SetStreamShutdown(done <-chan struct{}) { a.streamDone = done }

func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(a.securityHeaders)
	r.Use(a.cors)
	r.Use(a.verifyOrigin)
	r.Use(a.accessLog)
	r.Use(a.limitRequestBody)
	r.Use(middleware.Compress(5))

	r.Get("/healthz", a.health)
	r.Route("/api", func(r chi.Router) {
		r.Get("/public/stats", a.publicStats)
		r.Get("/public/readiness", a.publicReadiness)
		r.Get("/docs", a.openAPISpec)
		r.With(a.rateLimit(5, time.Minute)).Get("/public/data", a.publicData)
		r.With(a.rateLimit(30, time.Minute)).Get("/public/phbi", a.publicPHBI)
		r.With(a.rateLimit(30, time.Minute)).Post("/recaptcha/verify", a.verifyRecaptcha)
		r.Get("/presensi/{id}/public", a.publicPresensi)
		r.With(a.rateLimit(20, time.Minute, "presensi-absensi")).Post("/presensi/{id}/absensi", a.submitPresensi)
		r.Get("/participants/{id}", a.participantDetail)
		r.Get("/realtime", a.realtime)
		r.Get("/downloads/{kind}/{id}", a.download)
		r.Get("/arsip/download/{id}", a.downloadAlias("arsip"))
		r.Get("/berkas-pimpinan/download/{id}", a.downloadAlias("berkas-pimpinan"))
		r.Get("/berkas-sp/download/{id}", a.downloadAlias("berkas-sp"))
		r.Get("/pengajuan-berkas/download/{id}", a.downloadAlias("pengajuan"))
		r.Get("/manajemen-user/{id}/image", a.userImage)

		r.Route("/auth", func(r chi.Router) {
			r.With(a.rateLimit(10, time.Minute)).Post("/login", a.login)
			r.With(a.rateLimit(5, time.Hour)).Post("/register", a.register)
			r.Post("/logout", a.logout)
			r.Get("/session", a.session)
			r.With(a.rateLimit(5, 10*time.Minute)).Post("/otp/send", a.sendOTP)
			r.With(a.rateLimit(10, 10*time.Minute)).Post("/otp/verify", a.verifyOTP)
			r.Get("/verification-status", a.verificationStatus)
			r.Get("/google", a.googleStart)
			r.Get("/callback/google", a.googleCallback)
			r.Get("/google/callback", a.googleCallback)
		})

		r.Group(func(r chi.Router) {
			r.Use(a.authenticate)
			r.Get("/me", a.me)
			r.Patch("/me", a.updateMe)

			r.Get("/periodes", a.listPeriodes)
			r.Post("/periodes", a.createPeriode)
			r.Get("/periodes/{id}", a.getPeriode)
			r.Patch("/periodes/{id}", a.updatePeriode)
			r.Post("/periodes/{id}/activate", a.activatePeriode)
			r.Delete("/periodes/{id}", a.deletePeriode)

			r.Get("/arsip", a.listArsip)
			r.Post("/arsip", a.createArsip)
			r.Get("/arsip/stats", a.arsipStats)
			r.Get("/arsip/{id}", a.getArsip)
			r.Patch("/arsip/{id}", a.updateArsip)
			r.Delete("/arsip/{id}", a.deleteArsip)
			r.Post("/arsip/{id}/download-token", a.arsipDownloadToken)

			r.Get("/berkas-pimpinan", a.listBerkasPimpinan)
			r.Post("/berkas-pimpinan", a.createBerkasPimpinan)
			r.Get("/berkas-pimpinan/stats", a.berkasPimpinanStats)
			r.Get("/berkas-pimpinan/{id}", a.getBerkasPimpinan)
			r.Patch("/berkas-pimpinan/{id}", a.updateBerkasPimpinan)
			r.Delete("/berkas-pimpinan/{id}", a.deleteBerkasPimpinan)
			r.Post("/berkas-pimpinan/{id}/download-token", a.berkasPimpinanDownloadToken)

			r.Get("/berkas-sp", a.listBerkasSP)
			r.Post("/berkas-sp", a.createBerkasSP)
			r.Get("/berkas-sp/stats", a.berkasSPStats)
			r.Get("/berkas-sp/{id}", a.getBerkasSP)
			r.Patch("/berkas-sp/{id}", a.updateBerkasSP)
			r.Delete("/berkas-sp/{id}", a.deleteBerkasSP)
			r.Post("/berkas-sp/{id}/download-token", a.berkasSPDownloadToken)

			r.Get("/anggota", a.listAnggota)
			r.Post("/anggota", a.createAnggota)
			r.Get("/anggota/stats", a.anggotaStats)
			r.Get("/anggota/active-users", a.activeUsers)
			r.Post("/anggota/copy", a.copyAnggota)
			r.Get("/anggota/{id}", a.getAnggota)
			r.Patch("/anggota/{id}", a.updateAnggota)
			r.Delete("/anggota/{id}", a.deleteAnggota)
			r.Get("/anggota/{id}/image", a.anggotaImage)

			r.Get("/agenda-kegiatan", a.listAgenda)
			r.Post("/agenda-kegiatan", a.createAgenda)
			r.Get("/agenda-kegiatan/stats", a.agendaStats)
			r.Get("/agenda-kegiatan/{id}", a.getAgenda)
			r.Patch("/agenda-kegiatan/{id}", a.updateAgenda)
			r.Delete("/agenda-kegiatan/{id}", a.deleteAgenda)

			r.Get("/presensi", a.listPresensi)
			r.Post("/presensi", a.createPresensi)
			r.Get("/presensi/{id}", a.getPresensi)
			r.Patch("/presensi/{id}", a.updatePresensi)
			r.Patch("/presensi/{id}/status", a.updatePresensiStatus)
			r.Delete("/presensi/{id}", a.deletePresensi)

			r.Get("/pengajuan", a.listPengajuan)
			r.Post("/pengajuan", a.createPengajuan)
			r.Get("/pengajuan/stats", a.pengajuanStats)
			r.Get("/pengajuan/active-pac", a.activePACUsers)
			r.Get("/pengajuan/reference", a.listReferencePengajuan)
			r.Get("/pengajuan/reference/stats", a.referencePengajuanStats)
			r.Get("/pengajuan/reference/{id}", a.getReferencePengajuan)
			r.Get("/pengajuan/{id}", a.getPengajuan)
			r.Patch("/pengajuan/{id}", a.updatePengajuan)
			r.Patch("/pengajuan/{id}/status", a.updatePengajuanStatus)
			r.Delete("/pengajuan/{id}", a.deletePengajuan)
			r.Post("/pengajuan/{id}/download-token", a.pengajuanDownloadToken)

			r.Get("/dashboard/stats", a.dashboardStats)
			r.Get("/logs", a.listLogs)
			r.Get("/logs/stats", a.logStats)
			r.Get("/logs/monitoring", a.logMonitoring)
			r.Get("/logs/{id}", a.getLog)
			r.Post("/logs", a.createClientLog)

			r.Get("/users", a.listUsers)
			r.Get("/users/stats", a.userStats)
			r.Get("/users/{id}", a.getUser)
			r.Patch("/users/{id}/status", a.toggleUserStatus)
			r.Post("/users/{id}/reset-password", a.resetUserPassword)
			r.Delete("/users/{id}", a.deleteUser)

			r.Get("/email-logs", a.listEmailLogs)
			r.Get("/email-logs/stats", a.emailStats)
			r.Post("/email-logs/{id}/retry", a.retryEmail)

			r.Get("/backups", a.listBackups)
			r.Post("/backups", a.createBackup)
			r.Get("/backups/download", a.backupDownloadURL)
			r.Get("/backups/file", a.downloadEncryptedBackup)
			r.Delete("/backups", a.deleteBackup)
		})

		r.Get("/cron/backup", a.cronBackup)
	})
	return r
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.db.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Database tidak tersedia")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (a *API) openAPISpec(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.dataOpenAPISpec())
}

func (a *API) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Debug("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

func (a *API) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == a.cfg.FrontendURL {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-API-Key, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		user, session, err := a.currentUser(r)
		if err != nil {
			if !errors.Is(err, errMissingSession) && !errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Layanan autentikasi tidak tersedia")
				return
			}
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Belum terautentikasi")
			return
		}
		if !user.IsActive {
			writeError(w, http.StatusForbidden, "ACCOUNT_INACTIVE", "Akun belum diaktifkan")
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, authContext{User: user, Session: session})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type authContext struct {
	User    domain.User
	Session domain.Session
}

func authFrom(r *http.Request) authContext {
	return r.Context().Value(userContextKey).(authContext)
}

func (a *API) currentUser(r *http.Request) (domain.User, domain.Session, error) {
	token := ""
	if cookie, err := r.Cookie(a.cfg.SessionCookie); err == nil {
		token = cookie.Value
	}
	if token == "" {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		}
	}
	if !strings.HasPrefix(token, "g1_") {
		return domain.User{}, domain.Session{}, errMissingSession
	}

	var user domain.User
	var session domain.Session
	err := a.db.QueryRow(r.Context(), `
		SELECT s."id", s."userId", s."token", s."expiresAt", s."createdAt",
		       u."id", u."name", u."email", u."emailVerified", u."image", u."role"::text,
		       u."isActive", u."periodeAktifId", u."createdAt", u."updatedAt", u."lastLogoutAt"
		FROM "Session" s JOIN "User" u ON u."id"=s."userId"
		WHERE s."token" IN ($1,$2) AND s."expiresAt">NOW()`, token, a.sessionDigest(token)).Scan(
		&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt,
		&user.ID, &user.Name, &user.Email, &user.EmailVerified, &user.Image, &user.Role,
		&user.IsActive, &user.PeriodeAktifID, &user.CreatedAt, &user.UpdatedAt, &user.LastLogoutAt,
	)
	if err != nil {
		return domain.User{}, domain.Session{}, err
	}
	session.Token = token
	return user, session, nil
}

func (a *API) targetPeriode(ctx context.Context, userID, viewID string) (string, error) {
	if viewID != "" {
		var exists bool
		if err := a.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM "Periode" WHERE "id"=$1 AND "userId"=$2)`, viewID, userID).Scan(&exists); err != nil {
			return "", err
		}
		if exists {
			return viewID, nil
		}
	}
	var id string
	err := a.db.QueryRow(ctx, `SELECT "id" FROM "Periode" WHERE "userId"=$1 AND "isActive"=true LIMIT 1`, userID).Scan(&id)
	return id, err
}

func periodLookupFailed(w http.ResponseWriter, err error) bool {
	if err == nil || errors.Is(err, pgx.ErrNoRows) {
		return false
	}
	writeError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Gagal memeriksa periode")
	return true
}

func viewPeriode(r *http.Request) string {
	if v := r.URL.Query().Get("periodeId"); v != "" {
		return v
	}
	if cookie, err := r.Cookie("view_periode_id"); err == nil {
		return cookie.Value
	}
	return ""
}

func parsePage(r *http.Request) (page, limit int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if page > 1_000_000 {
		page = 1_000_000
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 500 {
		limit = 500
	}
	return
}

func readJSON(r *http.Request, dst any) error {
	value, err := readLimited(r.Body, 2<<20)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("hanya satu objek JSON yang diizinkan")
	}
	return nil
}

func readUpload(file multipart.File, max int64) ([]byte, error) {
	defer file.Close()
	return readLimited(file, max)
}

func (a *API) deleteStorageObject(key string) {
	if key == "" || strings.HasPrefix(key, "/storage") {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.r2.Delete(ctx, key); err != nil {
		slog.Warn("storage cleanup failed", "key", key, "error", err)
	}
}

func readLimited(reader io.Reader, max int64) ([]byte, error) {
	limited := io.LimitReader(reader, max+1)
	value, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(value)) > max {
		return nil, fmt.Errorf("ukuran file melebihi batas")
	}
	return value, nil
}

func validateUpload(filename string, value []byte, imagesOnly bool) (string, error) {
	if len(value) == 0 {
		return "", errors.New("file kosong")
	}
	ext := strings.ToLower(filepath.Ext(filename))
	imageTypes := map[string]string{".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".webp": "image/webp"}
	detected := strings.Split(http.DetectContentType(value), ";")[0]
	if expected, ok := imageTypes[ext]; ok {
		if detected != expected {
			return "", errors.New("isi file tidak sesuai dengan format gambar")
		}
		return expected, nil
	}
	if imagesOnly {
		return "", errors.New("format gambar harus JPG, PNG, atau WebP")
	}
	switch ext {
	case ".pdf":
		if detected != "application/pdf" {
			return "", errors.New("isi file bukan PDF yang valid")
		}
		return "application/pdf", nil
	case ".doc", ".ppt":
		if detected != "application/octet-stream" && detected != "application/x-ole-storage" && detected != "application/msword" && detected != "application/vnd.ms-powerpoint" {
			return "", errors.New("isi file Office tidak valid")
		}
		return mimeForExtension(ext), nil
	case ".docx", ".pptx":
		if detected != "application/zip" && detected != "application/octet-stream" {
			return "", errors.New("isi file Office tidak valid")
		}
		return mimeForExtension(ext), nil
	default:
		return "", errors.New("format file tidak diizinkan")
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, domain.ErrorBody{Success: false, Code: code, Message: message})
}

func writeLookupError(w http.ResponseWriter, err error, code, message string) {
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, code, message)
		return
	}
	writeError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Layanan data tidak tersedia")
}

func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func (a *API) encryptStrings(values ...string) ([]string, error) {
	encrypted := make([]string, len(values))
	for i, value := range values {
		ciphertext, err := a.crypto.EncryptText(value)
		if err != nil {
			return nil, err
		}
		encrypted[i] = ciphertext
	}
	return encrypted, nil
}

func writeEncryptionError(w http.ResponseWriter) {
	writeError(w, http.StatusInternalServerError, "ENCRYPTION_FAILED", "Gagal memproses data")
}

func totalPages(total, limit int) int {
	if total <= 0 || limit <= 0 {
		return 0
	}
	return 1 + (total-1)/limit
}

func rowMap(rows pgx.Rows) (map[string]any, error) {
	values, err := rows.Values()
	if err != nil {
		return nil, err
	}
	fields := rows.FieldDescriptions()
	result := make(map[string]any, len(values))
	for i, value := range values {
		name := string(fields[i].Name)
		result[name] = normalizeAuditTimestamp(name, fields[i].DataTypeOID, value)
	}
	return result, nil
}

// Audit timestamps are stored as Jakarta wall time in timestamp columns.
// Date fields are left untouched because they represent calendar dates.
func normalizeAuditTimestamp(name string, oid uint32, value any) any {
	if oid != pgtype.TimestampOID || (name != "createdAt" && name != "updatedAt") {
		return value
	}
	stamp, ok := value.(time.Time)
	if !ok {
		return value
	}
	wib := time.FixedZone("WIB", 7*60*60)
	return time.Date(stamp.Year(), stamp.Month(), stamp.Day(), stamp.Hour(), stamp.Minute(), stamp.Second(), stamp.Nanosecond(), wib).UTC()
}

func scanMaps(rows pgx.Rows) ([]map[string]any, error) {
	defer rows.Close()
	result := []map[string]any{}
	for rows.Next() {
		item, err := rowMap(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func decryptFieldStrict(c *security.Crypto, item map[string]any, keys ...string) error {
	for _, key := range keys {
		value, ok := item[key].(string)
		if !ok || value == "" {
			continue
		}
		plain, err := c.DecryptText(value)
		if err != nil {
			return fmt.Errorf("decrypt %s: %w", key, err)
		}
		item[key] = plain
	}
	return nil
}

func parseDate(value string) (time.Time, error) {
	formats := []string{time.RFC3339, "2006-01-02", "2006-01-02T15:04", "2006-01-02 15:04:05"}
	for _, format := range formats {
		if parsed, err := time.Parse(format, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date: %s", value)
}

func requireRole(w http.ResponseWriter, r *http.Request, role string) bool {
	if authFrom(r).User.Role != role {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Akses ditolak")
		return false
	}
	return true
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type eventHub struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

func newEventHub() *eventHub           { return &eventHub{clients: map[chan string]struct{}{}} }
func (h *eventHub) add(ch chan string) { h.mu.Lock(); h.clients[ch] = struct{}{}; h.mu.Unlock() }
func (h *eventHub) remove(ch chan string) {
	h.mu.Lock()
	delete(h.clients, ch)
	close(ch)
	h.mu.Unlock()
}
func (h *eventHub) publish(value string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- value:
		default:
		}
	}
}

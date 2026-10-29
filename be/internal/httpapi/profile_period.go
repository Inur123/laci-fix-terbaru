package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"laci-backend/internal/security"
)

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": authFrom(r).User})
}

func (a *API) updateMe(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_FORM", "Form profil tidak valid")
		return
	}
	name := strings.Join(strings.Fields(r.FormValue("name")), " ")
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")
	if name == "" {
		name = auth.User.Name
	}
	if email == "" {
		email = auth.User.Email
	}
	address, err := mail.ParseAddress(email)
	if err != nil || len(address.Address) > 255 {
		writeError(w, http.StatusBadRequest, "INVALID_EMAIL", "Format email tidak valid")
		return
	}
	email = strings.ToLower(address.Address)
	if len([]rune(name)) < 2 || len([]rune(name)) > 100 || !validName.MatchString(name) {
		writeError(w, http.StatusBadRequest, "INVALID_NAME", "Nama tidak valid")
		return
	}
	if password != "" && (len(password) < 6 || len(password) > 128) {
		writeError(w, http.StatusBadRequest, "INVALID_PASSWORD", "Password minimal 6 karakter")
		return
	}
	emailChanged := email != auth.User.Email
	if emailChanged {
		var taken bool
		if err := a.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM "User" WHERE LOWER("email")=$1 AND "id"<>$2)`, email, auth.User.ID).Scan(&taken); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal memeriksa email")
			return
		}
		if taken {
			writeError(w, http.StatusConflict, "EMAIL_TAKEN", "Email sudah digunakan oleh pengguna lain")
			return
		}
	}

	var image any = auth.User.Image
	newImageKey := ""
	defer func() {
		if newImageKey != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = a.r2.Delete(ctx, newImageKey)
		}
	}()
	file, header, err := r.FormFile("image")
	if err == nil {
		if header.Size > 2<<20 {
			file.Close()
			writeError(w, 400, "FILE_TOO_LARGE", "Ukuran gambar maksimal 2MB")
			return
		}
		plain, err := readUpload(file, 2<<20)
		if err != nil {
			writeError(w, 400, "INVALID_FILE", "Gambar tidak dapat dibaca")
			return
		}
		contentType, err := validateUpload(header.Filename, plain, true)
		if err != nil {
			writeError(w, 400, "INVALID_FILE", err.Error())
			return
		}
		encrypted, err := a.crypto.EncryptFile(plain)
		if err != nil {
			writeError(w, 500, "ENCRYPTION_FAILED", "Gagal memproses gambar")
			return
		}
		key := "profile/" + security.EncryptedFilename(header.Filename)
		if err := a.r2.Put(r.Context(), key, contentType, encrypted); err != nil {
			writeError(w, 502, "STORAGE_FAILED", "Gagal mengunggah gambar")
			return
		}
		newImageKey = key
		image = key
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal memperbarui profil")
		return
	}
	defer tx.Rollback(r.Context())
	_, err = tx.Exec(r.Context(), `UPDATE "User" SET "name"=$2,"email"=$3,"image"=$4,"emailVerified"=CASE WHEN "email"<>$3 THEN false ELSE "emailVerified" END,"updatedAt"=NOW() WHERE "id"=$1`, auth.User.ID, name, email, image)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal memperbarui profil")
		return
	}
	if password != "" {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), 10)
		if hashErr != nil {
			writeError(w, 500, "HASH_FAILED", "Gagal memproses password")
			return
		}
		_, err = tx.Exec(r.Context(), `UPDATE "Account" SET "password"=$2,"updatedAt"=NOW() WHERE "userId"=$1 AND "providerId"='credential'`, auth.User.ID, string(hash))
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal memperbarui password")
			return
		}
	}
	if emailChanged {
		if _, err := tx.Exec(r.Context(), `DELETE FROM "Account" WHERE "userId"=$1 AND "providerId"<>'credential'`, auth.User.ID); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal melepaskan akun Google lama")
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal memperbarui profil")
		return
	}
	uploadedImage := newImageKey != ""
	newImageKey = ""
	if uploadedImage && auth.User.Image != nil && *auth.User.Image != "" && !strings.HasPrefix(*auth.User.Image, "http") && !strings.HasPrefix(*auth.User.Image, "data:") {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = a.r2.Delete(ctx, *auth.User.Image)
		cancel()
	}
	_ = a.createLog(r.Context(), r, auth.User, "UPDATE", "USER", "Mengupdate profil", auth.User.ID)
	message := "Profil berhasil diperbarui!"
	if emailChanged {
		message = "Profil diperbarui. Silakan verifikasi alamat email baru."
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": message})
}

func (a *API) listPeriodes(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	rows, err := a.db.Query(r.Context(), `SELECT "id","nama","userId","isActive","createdAt","updatedAt" FROM "Periode" WHERE "userId"=$1 ORDER BY "createdAt" DESC, "id" DESC`, auth.User.ID)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil periode")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil periode")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (a *API) getPeriode(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	var id, name, userID string
	var active bool
	var created, updated time.Time
	err := a.db.QueryRow(r.Context(), `SELECT "id","nama","userId","isActive","createdAt","updatedAt" FROM "Periode" WHERE "id"=$1 AND "userId"=$2`, chi.URLParam(r, "id"), auth.User.ID).Scan(&id, &name, &userID, &active, &created, &updated)
	if err != nil {
		writeLookupError(w, err, "NOT_FOUND", "Periode tidak ditemukan")
		return
	}
	writeJSON(w, 200, map[string]any{"data": map[string]any{"id": id, "nama": name, "userId": userID, "isActive": active, "createdAt": created, "updatedAt": updated}})
}

func (a *API) createPeriode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Nama string `json:"nama"`
	}
	if readJSON(r, &body) != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Data periode tidak valid")
		return
	}
	name, valid := normalizePeriodeName(body.Nama)
	if !valid {
		writeError(w, http.StatusBadRequest, "INVALID_NAME", "Nama periode wajib diisi dan maksimal 100 karakter")
		return
	}
	auth := authFrom(r)
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membuat periode")
		return
	}
	defer tx.Rollback(r.Context())
	var ownerID string
	if err := tx.QueryRow(r.Context(), `SELECT "id" FROM "User" WHERE "id"=$1 FOR UPDATE`, auth.User.ID).Scan(&ownerID); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal memeriksa pengguna")
		return
	}
	var count int
	if err := tx.QueryRow(r.Context(), `SELECT COUNT(*) FROM "Periode" WHERE "userId"=$1`, auth.User.ID).Scan(&count); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal memeriksa periode")
		return
	}
	id := newID()
	active := count == 0
	_, err = tx.Exec(r.Context(), `INSERT INTO "Periode"("id","nama","userId","isActive","createdAt","updatedAt") VALUES($1,$2,$3,$4,NOW(),NOW())`, id, name, auth.User.ID, active)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, 409, "DUPLICATE_PERIODE", "Nama periode ini sudah ada!")
			return
		}
		writeError(w, 500, "DATABASE_ERROR", "Gagal membuat periode")
		return
	}
	if active {
		if _, err := tx.Exec(r.Context(), `UPDATE "User" SET "periodeAktifId"=$2,"updatedAt"=NOW() WHERE "id"=$1`, auth.User.ID, id); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengaktifkan periode baru")
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membuat periode")
		return
	}
	_ = a.createLog(r.Context(), r, auth.User, "CREATE", "PERIODE", "Membuat periode baru: "+name, id)
	a.notify(r.Context(), map[string]any{"type": "periode.created", "userId": auth.User.ID, "id": id})
	writeJSON(w, http.StatusCreated, map[string]any{"success": "Periode berhasil dibuat!", "data": map[string]any{"id": id, "nama": name, "userId": auth.User.ID, "isActive": active}})
}

func (a *API) updatePeriode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Nama string `json:"nama"`
	}
	if readJSON(r, &body) != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Data periode tidak valid")
		return
	}
	name, valid := normalizePeriodeName(body.Nama)
	if !valid {
		writeError(w, http.StatusBadRequest, "INVALID_NAME", "Nama periode wajib diisi dan maksimal 100 karakter")
		return
	}
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	result, err := a.db.Exec(r.Context(), `UPDATE "Periode" SET "nama"=$3,"updatedAt"=NOW() WHERE "id"=$1 AND "userId"=$2`, id, auth.User.ID, name)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, 409, "DUPLICATE_PERIODE", "Nama periode ini sudah ada!")
			return
		}
		writeError(w, 500, "DATABASE_ERROR", "Gagal memperbarui periode")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, 404, "NOT_FOUND", "Periode tidak ditemukan")
		return
	}
	_ = a.createLog(r.Context(), r, auth.User, "UPDATE", "PERIODE", "Mengupdate periode: "+name, id)
	a.notify(r.Context(), map[string]any{"type": "periode.updated", "userId": auth.User.ID, "id": id})
	writeJSON(w, http.StatusOK, map[string]any{"success": "Periode berhasil diperbarui!"})
}

func (a *API) activatePeriode(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengaktifkan periode")
		return
	}
	defer tx.Rollback(r.Context())
	var ownerID string
	if err := tx.QueryRow(r.Context(), `SELECT "id" FROM "User" WHERE "id"=$1 FOR UPDATE`, auth.User.ID).Scan(&ownerID); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal memeriksa pengguna")
		return
	}
	var name string
	if err = tx.QueryRow(r.Context(), `SELECT "nama" FROM "Periode" WHERE "id"=$1 AND "userId"=$2`, id, auth.User.ID).Scan(&name); err != nil {
		writeLookupError(w, err, "NOT_FOUND", "Periode tidak ditemukan")
		return
	}
	if _, err := tx.Exec(r.Context(), `UPDATE "Periode" SET "isActive"=false,"updatedAt"=NOW() WHERE "userId"=$1`, auth.User.ID); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menonaktifkan periode lama")
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE "Periode" SET "isActive"=true,"updatedAt"=NOW() WHERE "id"=$1`, id)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengaktifkan periode")
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE "User" SET "periodeAktifId"=$2,"updatedAt"=NOW() WHERE "id"=$1`, auth.User.ID, id)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengaktifkan periode")
		return
	}
	if tx.Commit(r.Context()) != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengaktifkan periode")
		return
	}
	_ = a.createLog(r.Context(), r, auth.User, "UPDATE", "PERIODE", "Mengaktifkan periode: "+name, id)
	a.notify(r.Context(), map[string]any{"type": "periode.updated", "userId": auth.User.ID, "id": id})
	writeJSON(w, 200, map[string]any{"success": "Periode berhasil diaktifkan!"})
}

func (a *API) deletePeriode(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menghapus periode")
		return
	}
	defer tx.Rollback(r.Context())
	var ownerID string
	if err := tx.QueryRow(r.Context(), `SELECT "id" FROM "User" WHERE "id"=$1 FOR UPDATE`, auth.User.ID).Scan(&ownerID); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal memeriksa pengguna")
		return
	}
	var name string
	err = tx.QueryRow(r.Context(), `DELETE FROM "Periode" WHERE "id"=$1 AND "userId"=$2 AND "isActive"=false RETURNING "nama"`, id, auth.User.ID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		var active bool
		err = tx.QueryRow(r.Context(), `SELECT "isActive" FROM "Periode" WHERE "id"=$1 AND "userId"=$2`, id, auth.User.ID).Scan(&active)
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, 404, "NOT_FOUND", "Periode tidak ditemukan")
		} else if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal memeriksa periode")
		} else if active {
			writeError(w, 400, "ACTIVE_PERIODE", "Periode aktif tidak dapat dihapus")
		}
		return
	}
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menghapus periode")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menghapus periode")
		return
	}
	_ = a.createLog(r.Context(), r, auth.User, "DELETE", "PERIODE", "Menghapus periode: "+name, id)
	a.notify(r.Context(), map[string]any{"type": "periode.deleted", "userId": auth.User.ID, "id": id})
	writeJSON(w, http.StatusOK, map[string]any{"success": "Periode berhasil dihapus!"})
}

func normalizePeriodeName(value string) (string, bool) {
	name := strings.Join(strings.Fields(value), " ")
	return name, name != "" && len([]rune(name)) <= 100
}

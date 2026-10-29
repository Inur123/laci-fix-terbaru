package httpapi

import (
	"net/http"
	"net/mail"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"laci-backend/internal/security"
)

var phonePattern = regexp.MustCompile(`^[0-9]{10,15}$`)

func (a *API) listPresensi(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	page, limit := parsePage(r)
	periodeID, err := a.targetPeriode(r.Context(), auth.User.ID, viewPeriode(r))
	if err != nil {
		if periodLookupFailed(w, err) {
			return
		}
		writeJSON(w, 200, map[string]any{"data": []any{}, "total": 0, "totalPages": 0})
		return
	}
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	status := r.URL.Query().Get("status")
	key := r.URL.Query().Get("sortKey")
	const base = `SELECT p."id",p."userId",p."periodeId",p."namaKegiatan",p."tempat",p."penyelenggara",p."tanggal",p."jamMulai",p."jamSelesai",p."isActive",p."isForcedOpen",p."forcedOpenAt",p."createdAt",p."updatedAt",COUNT(d."id") AS "participantCount" FROM "Presensi" p LEFT JOIN "PresensiData" d ON d."presensiId"=p."id" WHERE p."userId"=$1 AND p."periodeId"=$2 GROUP BY p."id"`
	if q == "" && (status == "" || status == "ALL") && (key == "" || key == "createdAt" || key == "tanggal") {
		var total int
		if err := a.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM "Presensi" WHERE "userId"=$1 AND "periodeId"=$2`, auth.User.ID, periodeID).Scan(&total); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menghitung presensi")
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
		sortColumn := `p."createdAt"`
		if key == "tanggal" {
			sortColumn = `p."tanggal"`
		}
		rows, err := a.db.Query(r.Context(), base+` ORDER BY `+sortColumn+` `+dir+`, p."id" `+dir+` LIMIT $3 OFFSET $4`, auth.User.ID, periodeID, limit, (page-1)*limit)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil presensi")
			return
		}
		data, err := scanMaps(rows)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil presensi")
			return
		}
		for _, item := range data {
			enrichPresensiListItem(item)
		}
		writeJSON(w, 200, map[string]any{"data": data, "total": total, "totalPages": totalPages(total, limit)})
		return
	}
	rows, err := a.db.Query(r.Context(), base+` ORDER BY p."createdAt" DESC, p."id" DESC`, auth.User.ID, periodeID)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil presensi")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil presensi")
		return
	}
	filtered := []map[string]any{}
	for _, item := range data {
		open := enrichPresensiListItem(item)
		if q != "" && !mapContains(item, q, "namaKegiatan", "tempat", "penyelenggara") {
			continue
		}
		if status == "OPEN" && !open {
			continue
		}
		if status == "CLOSED" && open {
			continue
		}
		filtered = append(filtered, item)
	}
	if key == "isActive" {
		asc := r.URL.Query().Get("sortDir") == "asc"
		sort.SliceStable(filtered, func(i, j int) bool {
			ai, _ := filtered[i]["open"].(bool)
			aj, _ := filtered[j]["open"].(bool)
			if asc {
				return !ai && aj
			}
			return ai && !aj
		})
	} else {
		sortMaps(filtered, key, r.URL.Query().Get("sortDir"), "createdAt")
	}
	writeJSON(w, 200, paginateMaps(filtered, page, limit))
}

func enrichPresensiListItem(item map[string]any) bool {
	open := presensiOpen(item)
	item["open"] = open
	item["_count"] = map[string]any{"dataPresensi": item["participantCount"]}
	return open
}

func (a *API) publicPresensi(w http.ResponseWriter, r *http.Request) { a.presensiDetail(w, r, false) }
func (a *API) getPresensi(w http.ResponseWriter, r *http.Request)    { a.presensiDetail(w, r, true) }
func (a *API) presensiDetail(w http.ResponseWriter, r *http.Request, includePrivate bool) {
	id := chi.URLParam(r, "id")
	rows, err := a.db.Query(r.Context(), `SELECT "id","userId","periodeId","namaKegiatan","tempat","penyelenggara","tanggal","jamMulai","jamSelesai","isActive","isForcedOpen","forcedOpenAt","createdAt","updatedAt" FROM "Presensi" WHERE "id"=$1`, id)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil presensi")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca presensi")
		return
	}
	if len(data) == 0 {
		writeError(w, 404, "NOT_FOUND", "Sesi presensi tidak ditemukan")
		return
	}
	item := data[0]
	if includePrivate {
		auth := authFrom(r)
		owner, _ := item["userId"].(string)
		if owner != auth.User.ID && auth.User.Role != "SEKRETARIS_CABANG" {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Akses ditolak")
			return
		}
	}
	item["open"] = presensiOpen(item)
	participants := []map[string]any{}
	if includePrivate {
		pRows, err := a.db.Query(r.Context(), `SELECT "id","presensiId","namaLengkap","email","noHp","emailHash","noHpHash","organisasi","tingkat","jabatan","instansi","createdAt" FROM "PresensiData" WHERE "presensiId"=$1 ORDER BY "createdAt" DESC, "id" DESC`, id)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil peserta")
			return
		}
		participants, err = scanMaps(pRows)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca peserta")
			return
		}
		for _, participant := range participants {
			if err := decryptFieldStrict(a.crypto, participant, "namaLengkap", "email", "noHp"); err != nil {
				writeError(w, 500, "DATA_UNREADABLE", "Data peserta tidak dapat dibaca")
				return
			}
		}
	}
	item["dataPresensi"] = participants
	writeJSON(w, 200, map[string]any{"data": item})
}

func (a *API) createPresensi(w http.ResponseWriter, r *http.Request) { a.mutatePresensi(w, r, true) }
func (a *API) updatePresensi(w http.ResponseWriter, r *http.Request) { a.mutatePresensi(w, r, false) }
func (a *API) mutatePresensi(w http.ResponseWriter, r *http.Request, creating bool) {
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeError(w, 400, "INVALID_FORM", "Form presensi tidak valid")
		return
	}
	name := strings.TrimSpace(r.FormValue("namaKegiatan"))
	place := strings.TrimSpace(r.FormValue("tempat"))
	organizer := strings.TrimSpace(r.FormValue("penyelenggara"))
	date, err := parseDate(r.FormValue("tanggal"))
	start := r.FormValue("jamMulai")
	end := r.FormValue("jamSelesai")
	if name == "" || place == "" || organizer == "" || err != nil || start == "" || end == "" {
		writeError(w, 400, "VALIDATION_ERROR", "Semua field harus diisi")
		return
	}
	if !validPresensiTimeRange(start, end) {
		writeError(w, 400, "INVALID_TIME_RANGE", "Jam selesai harus setelah jam mulai")
		return
	}
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	if creating {
		period, e := a.targetPeriode(r.Context(), auth.User.ID, "")
		if e != nil {
			if periodLookupFailed(w, e) {
				return
			}
			writeError(w, 400, "NO_ACTIVE_PERIODE", "Anda belum memiliki periode aktif")
			return
		}
		id = newID()
		_, err = a.db.Exec(r.Context(), `INSERT INTO "Presensi"("id","userId","periodeId","namaKegiatan","tempat","penyelenggara","tanggal","jamMulai","jamSelesai","isActive","isForcedOpen","createdAt","updatedAt") VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,true,false,NOW(),NOW())`, id, auth.User.ID, period, name, place, organizer, date, start, end)
	} else {
		result, e := a.db.Exec(r.Context(), `UPDATE "Presensi" SET "namaKegiatan"=$3,"tempat"=$4,"penyelenggara"=$5,"tanggal"=$6,"jamMulai"=$7,"jamSelesai"=$8,"updatedAt"=NOW() WHERE "id"=$1 AND "userId"=$2`, id, auth.User.ID, name, place, organizer, date, start, end)
		err = e
		if e == nil && result.RowsAffected() == 0 {
			writeError(w, 404, "NOT_FOUND", "Presensi tidak ditemukan")
			return
		}
	}
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menyimpan kegiatan presensi")
		return
	}
	action, message := "UPDATE", "Kegiatan presensi berhasil diperbarui!"
	if creating {
		action = "CREATE"
		message = "Kegiatan presensi berhasil dibuat!"
	}
	_ = a.createLog(r.Context(), r, auth.User, action, "PRESENSI", action+" kegiatan presensi: "+name, id)
	a.notify(r.Context(), map[string]any{"type": "log", "module": "PRESENSI", "action": action, "entityId": id})
	writeJSON(w, map[bool]int{true: 201, false: 200}[creating], map[string]any{"success": message, "data": map[string]string{"id": id}})
}

func (a *API) submitPresensi(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeError(w, 400, "INVALID_FORM", "Form presensi tidak valid")
		return
	}
	id := chi.URLParam(r, "id")
	name := strings.TrimSpace(r.FormValue("namaLengkap"))
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	phone := strings.ReplaceAll(strings.TrimSpace(r.FormValue("noHp")), " ", "")
	organization := strings.TrimSpace(r.FormValue("organisasi"))
	if len([]rune(name)) < 3 || len([]rune(name)) > 100 {
		writeError(w, 400, "INVALID_NAME", "Nama minimal 3 karakter")
		return
	}
	address, emailErr := mail.ParseAddress(email)
	if emailErr != nil || address.Address != email || len(email) > 255 {
		writeError(w, 400, "INVALID_EMAIL", "Format email tidak valid")
		return
	}
	if !phonePattern.MatchString(phone) {
		writeError(w, 400, "INVALID_PHONE", "Nomor HP harus 10-15 digit angka")
		return
	}
	if organization == "" {
		writeError(w, 400, "INVALID_ORGANIZATION", "Organisasi wajib diisi")
		return
	}
	encName, err := a.crypto.EncryptText(name)
	if err != nil {
		writeError(w, 500, "ENCRYPTION_FAILED", "Gagal memproses data presensi")
		return
	}
	encEmail, err := a.crypto.EncryptText(email)
	if err != nil {
		writeError(w, 500, "ENCRYPTION_FAILED", "Gagal memproses data presensi")
		return
	}
	encPhone, err := a.crypto.EncryptText(phone)
	if err != nil {
		writeError(w, 500, "ENCRYPTION_FAILED", "Gagal memproses data presensi")
		return
	}
	participantID := newID()
	accessToken, err := a.crypto.DownloadToken(participantID, 30*time.Minute)
	if err != nil {
		writeError(w, 500, "TOKEN_FAILED", "Gagal membuat akses hasil presensi")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal memulai presensi")
		return
	}
	defer tx.Rollback(r.Context())
	rows, err := tx.Query(r.Context(), `SELECT "id","tanggal","jamMulai","jamSelesai","isActive","isForcedOpen","forcedOpenAt" FROM "Presensi" WHERE "id"=$1 FOR SHARE`, id)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil sesi presensi")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca sesi presensi")
		return
	}
	if len(data) == 0 {
		writeError(w, 404, "NOT_FOUND", "Sesi presensi tidak ditemukan")
		return
	}
	if !presensiOpen(data[0]) {
		writeError(w, 400, "PRESENSI_CLOSED", "Presensi sudah ditutup")
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO "PresensiData"("id","presensiId","namaLengkap","email","noHp","emailHash","noHpHash","organisasi","tingkat","jabatan","instansi","createdAt") VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW())`, participantID, id, encName, encEmail, encPhone, security.Hash(email), security.Hash(phone), organization, nullable(r.FormValue("tingkat")), nullable(r.FormValue("jabatan")), nullable(r.FormValue("instansi")))
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, 409, "ALREADY_ATTENDED", "Mohon maaf, email atau nomor HP ini sudah absen di kegiatan ini")
			return
		}
		writeError(w, 500, "DATABASE_ERROR", "Gagal menyimpan data presensi (Masalah Server)")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menyimpan data presensi")
		return
	}
	a.notify(r.Context(), map[string]any{"type": "mutation", "model": "PresensiData", "action": "create", "presensiId": id})
	writeJSON(w, 201, map[string]any{"success": "Berhasil melakukan presensi!", "participantId": participantID, "accessToken": accessToken})
}

func (a *API) deletePresensi(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	var name string
	if err := a.db.QueryRow(r.Context(), `DELETE FROM "Presensi" WHERE "id"=$1 AND "userId"=$2 RETURNING "namaKegiatan"`, id, auth.User.ID).Scan(&name); err != nil {
		writeLookupError(w, err, "NOT_FOUND", "Presensi tidak ditemukan")
		return
	}
	_ = a.createLog(r.Context(), r, auth.User, "DELETE", "PRESENSI", "Menghapus kegiatan presensi: "+name, id)
	a.notify(r.Context(), map[string]any{"type": "log", "module": "PRESENSI", "action": "DELETE", "entityId": id})
	writeJSON(w, 200, map[string]any{"success": "Kegiatan presensi berhasil dihapus!"})
}

func (a *API) updatePresensiStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Mode string `json:"mode"`
	}
	if readJSON(r, &body) != nil || !oneOf(body.Mode, "AUTO", "MANUAL_CLOSE") {
		writeError(w, 400, "INVALID_MODE", "Mode status tidak valid")
		return
	}
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	active := body.Mode == "AUTO"
	label := "ditutup Manual"
	if active {
		label = "diatur Otomatis"
	}
	var name string
	err := a.db.QueryRow(r.Context(), `UPDATE "Presensi" SET "isActive"=$3,"isForcedOpen"=false,"forcedOpenAt"=NULL,"updatedAt"=NOW() WHERE "id"=$1 AND "userId"=$2 RETURNING "namaKegiatan"`, id, auth.User.ID, active).Scan(&name)
	if err != nil {
		writeLookupError(w, err, "NOT_FOUND", "Presensi tidak ditemukan")
		return
	}
	_ = a.createLog(r.Context(), r, auth.User, "UPDATE", "PRESENSI", "Mengubah status presensi "+label+": "+name, id)
	a.notify(r.Context(), map[string]any{"type": "log", "module": "PRESENSI", "action": "UPDATE", "entityId": id})
	writeJSON(w, 200, map[string]any{"success": "Status presensi berhasil " + label + "!", "data": map[string]any{"id": id, "isActive": active}})
}

func (a *API) participantDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := a.db.Query(r.Context(), `SELECT d."id",d."presensiId",d."namaLengkap",d."email",d."noHp",d."organisasi",d."tingkat",d."jabatan",d."instansi",d."createdAt",p."userId" AS "ownerUserId" FROM "PresensiData" d JOIN "Presensi" p ON p."id"=d."presensiId" WHERE d."id"=$1`, id)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil peserta")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca peserta")
		return
	}
	if len(data) == 0 {
		writeError(w, 404, "NOT_FOUND", "Peserta tidak ditemukan")
		return
	}
	authorized := false
	if tokenID, valid := a.crypto.VerifyDownloadToken(r.URL.Query().Get("token")); valid && tokenID == id {
		authorized = true
	}
	if !authorized {
		if user, _, sessionErr := a.currentUser(r); sessionErr == nil && user.IsActive {
			ownerID, _ := data[0]["ownerUserId"].(string)
			authorized = user.Role == "SEKRETARIS_CABANG" || user.ID == ownerID
		}
	}
	if !authorized {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Akses data peserta tidak diizinkan")
		return
	}
	delete(data[0], "ownerUserId")
	if err := decryptFieldStrict(a.crypto, data[0], "namaLengkap", "email", "noHp"); err != nil {
		writeError(w, 500, "DATA_UNREADABLE", "Data peserta tidak dapat dibaca")
		return
	}
	writeJSON(w, 200, map[string]any{"data": data[0]})
}

func presensiOpen(item map[string]any) bool {
	active, _ := item["isActive"].(bool)
	if !active {
		return false
	}
	date, ok := item["tanggal"].(time.Time)
	if !ok {
		return false
	}
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.Local
	}
	now := time.Now().In(loc)
	event := date.In(loc)
	if now.Format("2006-01-02") != event.Format("2006-01-02") {
		return false
	}
	start, ok1 := minutes(item["jamMulai"])
	end, ok2 := minutes(item["jamSelesai"])
	if !ok1 || !ok2 {
		return false
	}
	current := now.Hour()*60 + now.Minute()
	return current >= start && current <= end
}
func minutes(value any) (int, bool) {
	s, ok := value.(string)
	if !ok {
		return 0, false
	}
	parsed, err := time.Parse("15:04", s)
	if err != nil {
		return 0, false
	}
	return parsed.Hour()*60 + parsed.Minute(), true
}

func validPresensiTimeRange(start, end string) bool {
	startMinutes, startOK := minutes(start)
	endMinutes, endOK := minutes(end)
	return startOK && endOK && endMinutes > startMinutes
}

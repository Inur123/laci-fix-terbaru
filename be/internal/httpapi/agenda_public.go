package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
)

var agendaColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (a *API) listAgenda(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
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
	sortKey := r.URL.Query().Get("sortKey")
	const base = `SELECT "id","userId","periodeId","judul","deskripsi","lokasi","warna","tanggalMulai","tanggalSelesai","createdAt","updatedAt" FROM "AgendaKegiatan" WHERE "userId"=$1 AND "periodeId"=$2`
	if q == "" && (status == "" || status == "ALL") && (sortKey == "" || sortKey == "createdAt" || sortKey == "tanggalMulai") {
		var total int
		if err := a.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM "AgendaKegiatan" WHERE "userId"=$1 AND "periodeId"=$2`, auth.User.ID, periodeID).Scan(&total); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menghitung agenda")
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
		sortColumn := `"createdAt"`
		if sortKey == "tanggalMulai" {
			sortColumn = `"tanggalMulai"`
		}
		rows, err := a.db.Query(r.Context(), base+` ORDER BY `+sortColumn+` `+dir+`, "id" `+dir+` LIMIT $3 OFFSET $4`, auth.User.ID, periodeID, limit, (page-1)*limit)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil agenda kegiatan")
			return
		}
		data, err := scanMaps(rows)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil agenda kegiatan")
			return
		}
		for _, item := range data {
			if err := enrichAgendaListItem(a, item); err != nil {
				writeError(w, 500, "DATA_UNREADABLE", "Data agenda tidak dapat dibaca")
				return
			}
		}
		writeJSON(w, 200, map[string]any{"data": data, "total": total, "totalPages": totalPages(total, limit)})
		return
	}
	rows, err := a.db.Query(r.Context(), base+` ORDER BY "createdAt" DESC, "id" DESC`, auth.User.ID, periodeID)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil agenda kegiatan")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil agenda kegiatan")
		return
	}
	filtered := []map[string]any{}
	for _, item := range data {
		if err := enrichAgendaListItem(a, item); err != nil {
			writeError(w, 500, "DATA_UNREADABLE", "Data agenda tidak dapat dibaca")
			return
		}
		if q != "" && !mapContains(item, q, "judul", "deskripsi", "lokasi") {
			continue
		}
		if status != "" && status != "ALL" && item["status"] != status {
			continue
		}
		filtered = append(filtered, item)
	}
	sortMaps(filtered, sortKey, r.URL.Query().Get("sortDir"), "createdAt")
	writeJSON(w, 200, paginateMaps(filtered, page, limit))
}

func enrichAgendaListItem(a *API, item map[string]any) error {
	if err := decryptFieldStrict(a.crypto, item, "judul", "deskripsi", "lokasi"); err != nil {
		return err
	}
	item["status"] = agendaStatus(item["tanggalMulai"].(time.Time), asTimePtr(item["tanggalSelesai"]))
	return nil
}

func (a *API) getAgenda(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	auth := authFrom(r)
	rows, err := a.db.Query(r.Context(), `SELECT "id","userId","periodeId","judul","deskripsi","lokasi","warna","tanggalMulai","tanggalSelesai","createdAt","updatedAt" FROM "AgendaKegiatan" WHERE "id"=$1 AND "userId"=$2`, chi.URLParam(r, "id"), auth.User.ID)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil kegiatan")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca kegiatan")
		return
	}
	if len(data) == 0 {
		writeError(w, 404, "NOT_FOUND", "Kegiatan tidak ditemukan")
		return
	}
	if err := decryptFieldStrict(a.crypto, data[0], "judul", "deskripsi", "lokasi"); err != nil {
		writeError(w, 500, "DATA_UNREADABLE", "Data agenda tidak dapat dibaca")
		return
	}
	writeJSON(w, 200, map[string]any{"data": data[0]})
}

func (a *API) createAgenda(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	a.mutateAgenda(w, r, true)
}
func (a *API) updateAgenda(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	a.mutateAgenda(w, r, false)
}

func (a *API) mutateAgenda(w http.ResponseWriter, r *http.Request, creating bool) {
	var body struct {
		Judul          string `json:"judul"`
		Deskripsi      string `json:"deskripsi"`
		Lokasi         string `json:"lokasi"`
		Warna          string `json:"warna"`
		TanggalMulai   string `json:"tanggalMulai"`
		TanggalSelesai string `json:"tanggalSelesai"`
	}
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") || strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			writeError(w, 400, "INVALID_REQUEST", "Data kegiatan tidak valid")
			return
		}
		body.Judul = r.FormValue("judul")
		body.Deskripsi = r.FormValue("deskripsi")
		body.Lokasi = r.FormValue("lokasi")
		body.Warna = r.FormValue("warna")
		body.TanggalMulai = r.FormValue("tanggalMulai")
		body.TanggalSelesai = r.FormValue("tanggalSelesai")
	} else if readJSON(r, &body) != nil {
		writeError(w, 400, "INVALID_REQUEST", "Data kegiatan tidak valid")
		return
	}
	body.Judul = strings.TrimSpace(body.Judul)
	body.Deskripsi = strings.TrimSpace(body.Deskripsi)
	body.Lokasi = strings.TrimSpace(body.Lokasi)
	body.Warna = strings.TrimSpace(body.Warna)
	body.TanggalMulai = strings.TrimSpace(body.TanggalMulai)
	body.TanggalSelesai = strings.TrimSpace(body.TanggalSelesai)
	if body.Judul == "" || body.Warna == "" || body.TanggalMulai == "" {
		writeError(w, 400, "VALIDATION_ERROR", "Judul, warna, dan tanggal mulai wajib diisi")
		return
	}
	if utf8.RuneCountInString(body.Judul) > 255 || utf8.RuneCountInString(body.Lokasi) > 500 || utf8.RuneCountInString(body.Deskripsi) > 5000 {
		writeError(w, 400, "VALIDATION_ERROR", "Data kegiatan melebihi batas karakter")
		return
	}
	if !agendaColorPattern.MatchString(body.Warna) {
		writeError(w, 400, "INVALID_COLOR", "Warna kegiatan tidak valid")
		return
	}
	start, err := parseDate(body.TanggalMulai)
	if err != nil {
		writeError(w, 400, "INVALID_DATE", "Tanggal mulai tidak valid")
		return
	}
	var end any = nil
	if body.TanggalSelesai != "" {
		parsed, e := parseDate(body.TanggalSelesai)
		if e != nil {
			writeError(w, 400, "INVALID_DATE", "Tanggal selesai tidak valid")
			return
		}
		if parsed.Before(start) {
			writeError(w, 400, "INVALID_DATE_RANGE", "Tanggal selesai tidak boleh lebih awal dari tanggal mulai")
			return
		}
		end = parsed
	}
	encrypted, err := a.encryptStrings(body.Judul, body.Deskripsi, body.Lokasi)
	if err != nil {
		writeEncryptionError(w)
		return
	}
	encTitle, encDesc, encLoc := encrypted[0], nullable(encrypted[1]), nullable(encrypted[2])
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	if creating {
		periodeID, e := a.targetPeriode(r.Context(), auth.User.ID, "")
		if e != nil {
			if periodLookupFailed(w, e) {
				return
			}
			writeError(w, 400, "NO_ACTIVE_PERIODE", "Tidak ada periode aktif")
			return
		}
		id = newID()
		_, err = a.db.Exec(r.Context(), `INSERT INTO "AgendaKegiatan"("id","userId","periodeId","judul","deskripsi","lokasi","warna","tanggalMulai","tanggalSelesai","createdAt","updatedAt") VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())`, id, auth.User.ID, periodeID, encTitle, encDesc, encLoc, body.Warna, start, end)
	} else {
		result, e := a.db.Exec(r.Context(), `UPDATE "AgendaKegiatan" SET "judul"=$3,"deskripsi"=$4,"lokasi"=$5,"warna"=$6,"tanggalMulai"=$7,"tanggalSelesai"=$8,"updatedAt"=NOW() WHERE "id"=$1 AND "userId"=$2`, id, auth.User.ID, encTitle, encDesc, encLoc, body.Warna, start, end)
		err = e
		if e == nil && result.RowsAffected() == 0 {
			writeError(w, 404, "NOT_FOUND", "Kegiatan tidak ditemukan")
			return
		}
	}
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menyimpan kegiatan")
		return
	}
	action, message := "UPDATE", "Kegiatan berhasil diperbarui!"
	if creating {
		action = "CREATE"
		message = "Kegiatan berhasil ditambahkan!"
	}
	_ = a.createLog(r.Context(), r, auth.User, action, "AGENDA_KEGIATAN", action+" kegiatan: "+body.Judul, id)
	a.notify(r.Context(), map[string]any{"type": "agenda.changed", "id": id, "userId": auth.User.ID})
	writeJSON(w, map[bool]int{true: 201, false: 200}[creating], map[string]any{"success": message, "data": map[string]string{"id": id}})
}

func (a *API) deleteAgenda(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	result, err := a.db.Exec(r.Context(), `DELETE FROM "AgendaKegiatan" WHERE "id"=$1 AND "userId"=$2`, id, auth.User.ID)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menghapus kegiatan")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, 404, "NOT_FOUND", "Kegiatan tidak ditemukan")
		return
	}
	_ = a.createLog(r.Context(), r, auth.User, "DELETE", "AGENDA_KEGIATAN", "Menghapus kegiatan ID: "+id, id)
	a.notify(r.Context(), map[string]any{"type": "agenda.deleted", "id": id, "userId": auth.User.ID})
	writeJSON(w, 200, map[string]any{"success": "Kegiatan berhasil dihapus."})
}

func (a *API) agendaStats(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	auth := authFrom(r)
	periodeID, err := a.targetPeriode(r.Context(), auth.User.ID, viewPeriode(r))
	if err != nil {
		if periodLookupFailed(w, err) {
			return
		}
		writeJSON(w, 200, map[string]int{"total": 0, "mendatang": 0, "selesai": 0})
		return
	}
	var total, upcoming, done int
	err = a.db.QueryRow(r.Context(), `SELECT COUNT(*),COUNT(*) FILTER (WHERE "tanggalMulai">CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),COUNT(*) FILTER (WHERE ("tanggalSelesai" IS NOT NULL AND "tanggalSelesai"<CURRENT_TIMESTAMP AT TIME ZONE 'UTC') OR ("tanggalSelesai" IS NULL AND (("tanggalMulai" AT TIME ZONE 'UTC') AT TIME ZONE 'Asia/Jakarta')::date<(CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Jakarta')::date)) FROM "AgendaKegiatan" WHERE "userId"=$1 AND "periodeId"=$2`, auth.User.ID, periodeID).Scan(&total, &upcoming, &done)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil statistik")
		return
	}
	writeJSON(w, 200, map[string]int{"total": total, "mendatang": upcoming, "selesai": done})
}

func agendaStatus(start time.Time, end *time.Time) string {
	now := time.Now()
	if now.Before(start) {
		return "MENDATANG"
	}
	if end != nil {
		if !now.After(*end) {
			return "BERLANGSUNG"
		}
		return "SELESAI"
	}
	endOfDay := time.Date(start.Year(), start.Month(), start.Day(), 23, 59, 59, 999999999, start.Location())
	if !now.After(endOfDay) {
		return "BERLANGSUNG"
	}
	return "SELESAI"
}
func asTimePtr(v any) *time.Time {
	if v == nil {
		return nil
	}
	if t, ok := v.(time.Time); ok {
		return &t
	}
	return nil
}

func (a *API) publicStats(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(r.Context(), `SELECT u."name",COUNT(a."id") FROM "User" u LEFT JOIN "Anggota" a ON a."userId"=u."id" WHERE u."role"='SEKRETARIS_PAC' AND u."isActive"=true GROUP BY u."id",u."name" ORDER BY u."name"`)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil statistik publik")
		return
	}
	distribution := []map[string]any{}
	totalAnggota := 0
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			rows.Close()
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca statistik publik")
			return
		}
		distribution = append(distribution, map[string]any{"name": name, "count": count})
		totalAnggota += count
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca statistik publik")
		return
	}
	var surat, sp, kegiatan int
	if err := a.db.QueryRow(r.Context(), `SELECT (SELECT COUNT(*) FROM "ArsipSurat"),(SELECT COUNT(*) FROM "BerkasSP"),(SELECT COUNT(*) FROM "AgendaKegiatan")`).Scan(&surat, &sp, &kegiatan); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menghitung statistik publik")
		return
	}
	eventRows, err := a.db.Query(r.Context(), `SELECT "judul","tanggalMulai","lokasi","warna" FROM "AgendaKegiatan" WHERE "tanggalMulai">=NOW() ORDER BY "tanggalMulai" ASC LIMIT 5`)
	events := []map[string]any{}
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil kegiatan publik")
		return
	}
	for eventRows.Next() {
		var title, warna string
		var location *string
		var date time.Time
		if err := eventRows.Scan(&title, &date, &location, &warna); err != nil {
			eventRows.Close()
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca kegiatan publik")
			return
		}
		plainTitle, err := a.crypto.DecryptText(title)
		if err != nil {
			eventRows.Close()
			writeError(w, 500, "DATA_UNREADABLE", "Data kegiatan tidak dapat dibaca")
			return
		}
		loc := "Lokasi tidak ditentukan"
		if location != nil {
			v, err := a.crypto.DecryptText(*location)
			if err != nil {
				eventRows.Close()
				writeError(w, 500, "DATA_UNREADABLE", "Data kegiatan tidak dapat dibaca")
				return
			}
			if v != "" {
				loc = v
			}
		}
		events = append(events, map[string]any{"judul": plainTitle, "tanggal": date, "lokasi": loc, "warna": warna})
	}
	err = eventRows.Err()
	eventRows.Close()
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca kegiatan publik")
		return
	}
	var activeName *string
	if err := a.db.QueryRow(r.Context(), `SELECT (SELECT p."nama" FROM "Periode" p JOIN "User" u ON u."id"=p."userId" WHERE u."role"='SEKRETARIS_CABANG' AND p."isActive"=true LIMIT 1)`).Scan(&activeName); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca periode aktif")
		return
	}
	anggotaCount, suratCount := totalAnggota, 0
	if activeName != nil {
		if err := a.db.QueryRow(r.Context(), `SELECT (SELECT COUNT(*) FROM "Anggota" a JOIN "Periode" p ON p."id"=a."periodeId" WHERE p."nama"=$1),(SELECT COUNT(*) FROM "PengajuanBerkas" pb JOIN "Periode" p ON p."id"=pb."periodeId" WHERE p."nama"=$1)`, *activeName).Scan(&anggotaCount, &suratCount); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menghitung data periode")
			return
		}
	}
	writeJSON(w, 200, map[string]any{"success": true, "timestamp": time.Now(), "anggotaCount": anggotaCount, "suratCount": suratCount, "summary": map[string]int{"totalAnggota": totalAnggota, "totalSurat": surat, "totalSertifikatSP": sp, "totalKegiatan": kegiatan}, "distribution": distribution, "events": events})
}

func (a *API) publicReadiness(w http.ResponseWriter, r *http.Request) {
	var ready bool
	err := a.db.QueryRow(r.Context(), `SELECT EXISTS(
		SELECT 1 FROM "User" u JOIN "Periode" p ON p."userId"=u."id"
		WHERE u."role"='SEKRETARIS_CABANG' AND u."isActive"=true AND p."isActive"=true
	)`).Scan(&ready)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal memeriksa kesiapan sistem")
		return
	}
	writeJSON(w, 200, map[string]bool{"ready": ready})
}

func (a *API) publicPHBI(w http.ResponseWriter, r *http.Request) {
	year := r.URL.Query().Get("year")
	if year == "" {
		year = itoa(time.Now().Year())
	}
	yearNumber, err := strconv.Atoi(year)
	if len(year) != 4 || err != nil || yearNumber < 1900 || yearNumber > 2100 {
		writeError(w, 400, "INVALID_YEAR", "Tahun tidak valid")
		return
	}
	endpoint := "https://api-hari-libur.vercel.app/api?year=" + url.QueryEscape(year)
	request, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
	response, err := outboundClient.Do(request)
	if err != nil {
		writeError(w, 502, "UPSTREAM_ERROR", "Gagal mengambil data kalender nasional")
		return
	}
	defer response.Body.Close()
	var payload any
	if response.StatusCode >= 300 || json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload) != nil {
		writeError(w, 502, "UPSTREAM_ERROR", "Gagal mengambil data kalender nasional")
		return
	}
	holidays := payload
	if data, ok := payload.(map[string]any); ok {
		if nested, exists := data["data"]; exists {
			holidays = nested
		} else if nested, exists := data["holidays"]; exists {
			holidays = nested
		}
	}
	writeJSON(w, 200, map[string]any{"success": true, "year": year, "source": "https://github.com/andifahruddinakas/api-hari-libur", "holidays": holidays})
}

func (a *API) publicData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if a.cfg.APIKey == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-API-Key")), []byte(a.cfg.APIKey)) != 1 {
		writeError(w, 401, "INVALID_API_KEY", "Invalid API Key")
		return
	}
	tables := []struct {
		name, query string
		encrypted   []string
	}{
		{"arsipSurat", `SELECT a.*,a."jenisSurat"::text AS "jenisText",a."organisasi"::text AS "orgText",u."name" AS "uploader",p."nama" AS "periode" FROM "ArsipSurat" a JOIN "User" u ON u."id"=a."userId" JOIN "Periode" p ON p."id"=a."periodeId" ORDER BY a."createdAt" DESC, a."id" DESC`, []string{"noSurat", "pengirimPenerima", "perihal", "deskripsi"}},
		{"anggota", `SELECT a.*,a."jenisKelamin"::text AS "genderText",u."name" AS "uploader",p."nama" AS "periode" FROM "Anggota" a JOIN "User" u ON u."id"=a."userId" JOIN "Periode" p ON p."id"=a."periodeId" ORDER BY a."createdAt" DESC, a."id" DESC`, anggotaEncryptedFields},
		{"pengajuanBerkas", `SELECT b.*,b."penerima"::text AS "penerimaText",b."status"::text AS "statusText",u."name" AS "pacName",p."nama" AS "periodePac" FROM "PengajuanBerkas" b JOIN "User" u ON u."id"=b."userId" JOIN "Periode" p ON p."id"=b."periodeIdPac" ORDER BY b."createdAt" DESC, b."id" DESC`, []string{"noSurat", "keperluan", "deskripsi", "alasanPenolakan"}},
		{"berkasPimpinan", `SELECT b.*,u."name" AS "uploader",p."nama" AS "periode" FROM "BerkasPimpinan" b JOIN "User" u ON u."id"=b."userId" JOIN "Periode" p ON p."id"=b."periodeId" ORDER BY b."createdAt" DESC, b."id" DESC`, []string{"nama", "catatan"}},
		{"berkasSP", `SELECT b.*,b."organisasi"::text AS "orgText",u."name" AS "uploader",p."nama" AS "periode" FROM "BerkasSP" b JOIN "User" u ON u."id"=b."userId" JOIN "Periode" p ON p."id"=b."periodeId" ORDER BY b."createdAt" DESC, b."id" DESC`, []string{"nama", "catatan"}},
		{"agendaKegiatan", `SELECT a.*,u."name" AS "uploader" FROM "AgendaKegiatan" a JOIN "User" u ON u."id"=a."userId" ORDER BY a."createdAt" DESC, a."id" DESC`, []string{"judul", "deskripsi", "lokasi"}},
	}
	result := map[string]any{"success": true, "timestamp": time.Now()}
	for _, table := range tables {
		rows, err := a.db.Query(r.Context(), table.query)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil data publik")
			return
		}
		items, err := scanMaps(rows)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil data publik")
			return
		}
		for _, item := range items {
			if err := decryptFieldStrict(a.crypto, item, table.encrypted...); err != nil {
				writeError(w, 500, "DATA_UNREADABLE", "Data publik tidak dapat dibaca")
				return
			}
			if v, ok := item["jenisText"]; ok {
				item["jenisSurat"] = v
			}
			if v, ok := item["orgText"]; ok {
				item["organisasi"] = v
			}
			if v, ok := item["genderText"]; ok {
				item["jenisKelamin"] = v
			}
			if v, ok := item["penerimaText"]; ok {
				item["penerima"] = v
			}
			if v, ok := item["statusText"]; ok {
				item["status"] = v
			}
		}
		result[table.name] = items
	}
	periodRows, err := a.db.Query(r.Context(), `SELECT "id","nama","userId","isActive","createdAt","updatedAt" FROM "Periode" ORDER BY "createdAt" DESC`)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil periode")
		return
	}
	periods, err := scanMaps(periodRows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca periode")
		return
	}
	userRows, err := a.db.Query(r.Context(), `SELECT "id","name","email","role"::text,"isActive","emailVerified","createdAt" FROM "User" ORDER BY "createdAt" DESC`)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil pengguna")
		return
	}
	users, err := scanMaps(userRows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca pengguna")
		return
	}
	result["periode"] = periods
	result["users"] = users
	writeJSON(w, 200, result)
}

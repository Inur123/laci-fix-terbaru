package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	page, limit := parsePage(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := r.URL.Query().Get("status")
	verified := r.URL.Query().Get("emailStatus")
	if status != "" && status != "ALL" && status != "ACTIVE" && status != "INACTIVE" || verified != "" && verified != "ALL" && verified != "VERIFIED" && verified != "UNVERIFIED" {
		writeError(w, 400, "INVALID_FILTER", "Filter pengguna tidak valid")
		return
	}
	where := ` FROM "User" WHERE "role"='SEKRETARIS_PAC' AND ($1::text='' OR "name" ILIKE '%' || $1 || '%' OR "email" ILIKE '%' || $1 || '%') AND ($2::text='' OR $2='ALL' OR "isActive"=($2='ACTIVE')) AND ($3::text='' OR $3='ALL' OR "emailVerified"=($3='VERIFIED'))`
	var total int
	if err := a.db.QueryRow(r.Context(), `SELECT COUNT(*)`+where, q, status, verified).Scan(&total); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menghitung pengguna")
		return
	}
	if total == 0 || page > totalPages(total, limit) {
		writeJSON(w, 200, map[string]any{"data": []any{}, "total": total, "totalPages": totalPages(total, limit)})
		return
	}
	column := map[string]string{"name": `LOWER("name")`, "email": `LOWER("email")`, "role": `LOWER("role"::text)`, "isActive": `"isActive"`, "emailVerified": `"emailVerified"`, "createdAt": `"createdAt"`, "updatedAt": `"updatedAt"`}[r.URL.Query().Get("sortKey")]
	if column == "" {
		column = `"createdAt"`
	}
	direction := "DESC"
	if strings.EqualFold(r.URL.Query().Get("sortDir"), "asc") {
		direction = "ASC"
	}
	rows, err := a.db.Query(r.Context(), `SELECT "id","name","email","emailVerified","image","role"::text,"isActive","periodeAktifId","createdAt","updatedAt","lastLogoutAt"`+where+` ORDER BY `+column+` `+direction+`,"id" `+direction+` LIMIT $4 OFFSET $5`, q, status, verified, limit, (page-1)*limit)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil pengguna")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil pengguna")
		return
	}
	writeJSON(w, 200, map[string]any{"data": data, "total": total, "totalPages": totalPages(total, limit)})
}

func (a *API) userStats(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	var total, active, inactive, verified, unverified int
	err := a.db.QueryRow(r.Context(), `SELECT COUNT(*),COUNT(*) FILTER(WHERE "isActive"),COUNT(*) FILTER(WHERE NOT "isActive"),COUNT(*) FILTER(WHERE "emailVerified"),COUNT(*) FILTER(WHERE NOT "emailVerified") FROM "User" WHERE "role"='SEKRETARIS_PAC'`).Scan(&total, &active, &inactive, &verified, &unverified)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil statistik pengguna")
		return
	}
	writeJSON(w, 200, map[string]int{"total": total, "aktif": active, "nonaktif": inactive, "terverifikasi": verified, "belumVerifikasi": unverified})
}

func (a *API) getUser(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	if auth.User.Role != "SEKRETARIS_CABANG" && auth.User.ID != id {
		writeError(w, 403, "FORBIDDEN", "Akses ditolak")
		return
	}
	rows, err := a.db.Query(r.Context(), `SELECT u."id",u."name",u."email",u."emailVerified",u."image",u."role"::text,u."isActive",u."periodeAktifId",u."createdAt",u."updatedAt",u."lastLogoutAt",p."id" AS "activePeriodId",p."nama" AS "periodeAktif" FROM "User" u LEFT JOIN "Periode" p ON p."userId"=u."id" AND p."isActive"=true WHERE u."id"=$1 LIMIT 1`, id)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil pengguna")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca pengguna")
		return
	}
	if len(data) == 0 {
		writeError(w, 404, "NOT_FOUND", "User tidak ditemukan")
		return
	}
	item := data[0]
	periodID, toPeriod := item["activePeriodId"].(string)
	if !toPeriod {
		periodID = ""
	}
	stats := map[string]int{"totalArsip": 0, "totalBerkasPimpinan": 0, "totalPengajuan": 0, "totalAnggota": 0, "totalLog": 0, "totalBerkasSP": 0}
	if periodID != "" {
		var totalArsip, totalBerkasPimpinan, totalPengajuan, totalAnggota, totalLog, totalBerkasSP int
		if err := a.db.QueryRow(r.Context(), `SELECT (SELECT COUNT(*) FROM "ArsipSurat" WHERE "userId"=$1 AND "periodeId"=$2),(SELECT COUNT(*) FROM "BerkasPimpinan" WHERE "userId"=$1 AND "periodeId"=$2),(SELECT COUNT(*) FROM "PengajuanBerkas" WHERE "userId"=$1 AND "periodeIdPac"=$2),(SELECT COUNT(*) FROM "Anggota" WHERE "userId"=$1 AND "periodeId"=$2),(SELECT COUNT(*) FROM "LogActivity" WHERE "userId"=$1 AND "periodeId"=$2),(SELECT COUNT(*) FROM "BerkasSP" WHERE "userId"=$1 AND "periodeId"=$2)`, id, periodID).Scan(&totalArsip, &totalBerkasPimpinan, &totalPengajuan, &totalAnggota, &totalLog, &totalBerkasSP); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil statistik pengguna")
			return
		}
		stats["totalArsip"] = totalArsip
		stats["totalBerkasPimpinan"] = totalBerkasPimpinan
		stats["totalPengajuan"] = totalPengajuan
		stats["totalAnggota"] = totalAnggota
		stats["totalLog"] = totalLog
		stats["totalBerkasSP"] = totalBerkasSP
	}
	for k, v := range stats {
		item[k] = v
	}
	kaderCounts := map[string]int{"Makesta": 0, "Lakmud": 0, "Latin": 0, "Latpel": 0, "Lakut": 0, "Diklatama": 0, "Diklatmad": 0}
	educationCounts := map[string]int{"SD": 0, "MI": 0, "SMP": 0, "MTs": 0, "SMA": 0, "SMK": 0, "MAN": 0, "KULIAH": 0}
	if periodID != "" {
		rows, err := a.db.Query(r.Context(), `SELECT k."namaPerkaderan" FROM "Perkaderan" k JOIN "Anggota" a ON a."id"=k."anggotaId" WHERE a."userId"=$1 AND a."periodeId"=$2`, id, periodID)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil perkaderan pengguna")
			return
		}
		for rows.Next() {
			var encrypted string
			if err := rows.Scan(&encrypted); err != nil {
				rows.Close()
				writeError(w, 500, "DATABASE_ERROR", "Gagal membaca perkaderan pengguna")
				return
			}
			plain, err := a.crypto.DecryptText(encrypted)
			if err != nil {
				rows.Close()
				writeError(w, 500, "DATA_UNREADABLE", "Data perkaderan tidak dapat dibaca")
				return
			}
			if key := titleKader(plain); key != "" {
				if _, ok := kaderCounts[key]; ok {
					kaderCounts[key]++
				}
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca perkaderan pengguna")
			return
		}
		eRows, err := a.db.Query(r.Context(), `SELECT COALESCE("jenjangPendidikan",''),COUNT(*) FROM "Anggota" WHERE "userId"=$1 AND "periodeId"=$2 GROUP BY "jenjangPendidikan"`, id, periodID)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil pendidikan pengguna")
			return
		}
		for eRows.Next() {
			var key string
			var count int
			if err := eRows.Scan(&key, &count); err != nil {
				eRows.Close()
				writeError(w, 500, "DATABASE_ERROR", "Gagal membaca pendidikan pengguna")
				return
			}
			if _, ok := educationCounts[key]; ok {
				educationCounts[key] = count
			}
		}
		err = eRows.Err()
		eRows.Close()
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca pendidikan pengguna")
			return
		}
	}
	item["perkaderanCounts"] = kaderCounts
	item["pendidikanCounts"] = educationCounts
	if item["periodeAktif"] == nil {
		item["periodeAktif"] = "Tidak ada periode aktif"
	}
	writeJSON(w, 200, map[string]any{"data": item})
}

func (a *API) toggleUserStatus(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	var name string
	var active bool
	err := a.db.QueryRow(r.Context(), `UPDATE "User" SET "isActive"=NOT "isActive","updatedAt"=NOW() WHERE "id"=$1 AND "role"='SEKRETARIS_PAC' RETURNING "name","isActive"`, id).Scan(&name, &active)
	if err != nil {
		writeLookupError(w, err, "NOT_FOUND", "User tidak ditemukan")
		return
	}
	_ = a.createLog(r.Context(), r, auth.User, "UPDATE", "USER", map[bool]string{true: "Mengaktifkan", false: "Menonaktifkan"}[active]+" akun user: "+name, id)
	a.notify(r.Context(), map[string]any{"type": "mutation", "model": "User", "id": id})
	writeJSON(w, 200, map[string]any{"success": "Status akun " + name + " berhasil diubah!", "isActive": active})
}
func (a *API) resetUserPassword(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	id := chi.URLParam(r, "id")
	temporaryPassword, err := secureTemporaryPassword()
	if err != nil {
		writeError(w, 500, "PASSWORD_FAILED", "Gagal membuat password sementara")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(temporaryPassword), 10)
	if err != nil {
		writeError(w, 500, "PASSWORD_FAILED", "Gagal memproses password sementara")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mereset password")
		return
	}
	defer tx.Rollback(r.Context())
	result, err := tx.Exec(r.Context(), `UPDATE "Account" SET "password"=$2,"updatedAt"=NOW() WHERE "userId"=$1 AND "providerId"='credential'`, id, string(hash))
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mereset password")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, 404, "NOT_FOUND", "Akun credential tidak ditemukan")
		return
	}
	if _, err := tx.Exec(r.Context(), `DELETE FROM "Session" WHERE "userId"=$1`, id); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengakhiri sesi lama")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mereset password")
		return
	}
	auth := authFrom(r)
	_ = a.createLog(r.Context(), r, auth.User, "UPDATE", "USER", "Mereset password user: "+id, id)
	a.notify(r.Context(), map[string]any{"type": "mutation", "model": "User", "id": id})
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"success": "Password berhasil direset. Password sementara telah disalin.", "temporaryPassword": temporaryPassword})
}

func secureTemporaryPassword() (string, error) {
	value := make([]byte, 18)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
func (a *API) deleteUser(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	if id == auth.User.ID {
		writeError(w, 400, "SELF_DELETE", "Tidak dapat menghapus akun sendiri")
		return
	}
	result, err := a.db.Exec(r.Context(), `DELETE FROM "User" WHERE "id"=$1 AND "role"='SEKRETARIS_PAC'`, id)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menghapus pengguna")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, 404, "NOT_FOUND", "User tidak ditemukan")
		return
	}
	_ = a.createLog(r.Context(), r, auth.User, "DELETE", "USER", "Menghapus user: "+id, id)
	a.notify(r.Context(), map[string]any{"type": "mutation", "model": "User", "id": id})
	writeJSON(w, 200, map[string]any{"success": "User berhasil dihapus secara permanen!"})
}

func (a *API) dashboardStats(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	periodID, err := a.targetPeriode(r.Context(), auth.User.ID, viewPeriode(r))
	if err != nil {
		if periodLookupFailed(w, err) {
			return
		}
		periodID = ""
	}
	personal := map[string]any{"anggota": 0, "surat": 0, "berkasPimpinan": 0, "berkasSP": 0, "pengajuan": 0, "userCount": 0, "periode": 0, "kegiatan": 0, "presensi": 0, "globalAnggota": 0, "globalArsip": 0, "globalPimpinan": 0, "globalPengajuan": 0, "trend": []map[string]any{}}
	var anggota, surat, pimpinan, sp, pengajuan, userCount, periode, kegiatan, presensi int
	if periodID != "" {
		pengColumn := "periodeIdPac"
		if auth.User.Role == "SEKRETARIS_CABANG" {
			pengColumn = "periodeId"
		}
		pengWhere := `"` + pengColumn + `"=$2`
		if auth.User.Role != "SEKRETARIS_CABANG" {
			pengWhere += ` AND "userId"=$1`
		}
		query := `SELECT (SELECT COUNT(*) FROM "Anggota" WHERE "userId"=$1 AND "periodeId"=$2),(SELECT COUNT(*) FROM "ArsipSurat" WHERE "userId"=$1 AND "periodeId"=$2),(SELECT COUNT(*) FROM "BerkasPimpinan" WHERE "userId"=$1 AND "periodeId"=$2),(SELECT COUNT(*) FROM "BerkasSP" WHERE "userId"=$1 AND "periodeId"=$2),(SELECT COUNT(*) FROM "PengajuanBerkas" WHERE ` + pengWhere + `),(SELECT COUNT(*) FROM "User" WHERE "role"='SEKRETARIS_PAC' AND "isActive"=true AND "emailVerified"=true),(SELECT COUNT(*) FROM "Periode" WHERE "userId"=$1),(SELECT COUNT(*) FROM "AgendaKegiatan" WHERE "userId"=$1 AND "periodeId"=$2),(SELECT COUNT(*) FROM "Presensi" WHERE "userId"=$1 AND "periodeId"=$2)`
		if err := a.db.QueryRow(r.Context(), query, auth.User.ID, periodID).Scan(&anggota, &surat, &pimpinan, &sp, &pengajuan, &userCount, &periode, &kegiatan, &presensi); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil statistik dashboard")
			return
		}
	}
	personal["anggota"] = anggota
	personal["surat"] = surat
	personal["berkasPimpinan"] = pimpinan
	personal["berkasSP"] = sp
	personal["pengajuan"] = pengajuan
	personal["userCount"] = userCount
	personal["periode"] = periode
	personal["kegiatan"] = kegiatan
	personal["presensi"] = presensi
	var globalAnggota, globalArsip, globalPimpinan, globalPengajuan int
	if err := a.db.QueryRow(r.Context(), `SELECT (SELECT COUNT(*) FROM "Anggota" a JOIN "Periode" p ON p."id"=a."periodeId" WHERE p."isActive"=true),(SELECT COUNT(*) FROM "ArsipSurat" a JOIN "Periode" p ON p."id"=a."periodeId" WHERE p."isActive"=true),(SELECT COUNT(*) FROM "BerkasPimpinan" b JOIN "Periode" p ON p."id"=b."periodeId" WHERE p."isActive"=true),(SELECT COUNT(*) FROM "PengajuanBerkas" b JOIN "Periode" pp ON pp."id"=b."periodeIdPac" JOIN "Periode" pc ON pc."id"=b."periodeId" WHERE b."status"='DITERIMA' AND (pp."isActive"=true OR pc."isActive"=true))`).Scan(&globalAnggota, &globalArsip, &globalPimpinan, &globalPengajuan); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil statistik global")
		return
	}
	personal["globalAnggota"] = globalAnggota
	personal["globalArsip"] = globalArsip
	personal["globalPimpinan"] = globalPimpinan
	personal["globalPengajuan"] = globalPengajuan
	trend := make([]map[string]any, 6)
	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(loc)
	for i := 0; i < 6; i++ {
		date := now.AddDate(0, i-5, 0)
		trend[i] = map[string]any{"name": indonesianMonthShort(date.Month()), "value": 0}
	}
	if periodID != "" {
		rows, err := a.db.Query(r.Context(), `SELECT date_trunc('month',"createdAt") AS month,COUNT(*) FROM "ArsipSurat" WHERE "userId"=$1 AND "periodeId"=$2 AND "createdAt">=date_trunc('month',NOW())-INTERVAL '5 months' GROUP BY month`, auth.User.ID, periodID)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil tren arsip")
			return
		}
		for rows.Next() {
			var month time.Time
			var count int
			if err := rows.Scan(&month, &count); err != nil {
				rows.Close()
				writeError(w, 500, "DATABASE_ERROR", "Gagal membaca tren arsip")
				return
			}
			for _, entry := range trend {
				if entry["name"] == indonesianMonthShort(month.Month()) {
					entry["value"] = count
				}
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca tren arsip")
			return
		}
	}
	personal["trend"] = trend
	role := "PAC"
	var monitoring any = nil
	if auth.User.Role == "SEKRETARIS_CABANG" {
		role = "CABANG"
		leaderRows, err := a.db.Query(r.Context(), `SELECT * FROM (SELECT u."id",u."name",u."image",(SELECT COUNT(*) FROM "Anggota" a JOIN "Periode" p ON p."id"=a."periodeId" WHERE a."userId"=u."id" AND p."isActive"=true) AS anggotas,(SELECT COUNT(*) FROM "ArsipSurat" a JOIN "Periode" p ON p."id"=a."periodeId" WHERE a."userId"=u."id" AND p."isActive"=true) AS arsip,(SELECT COUNT(*) FROM "PengajuanBerkas" b WHERE b."userId"=u."id" AND b."status"='DITERIMA') AS pengajuan FROM "User" u WHERE u."role"='SEKRETARIS_PAC' AND u."isActive"=true) ranked ORDER BY anggotas+arsip+pengajuan DESC LIMIT 50`)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil leaderboard")
			return
		}
		leaders, err := scanMaps(leaderRows)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca leaderboard")
			return
		}
		for _, u := range leaders {
			ang := asInt(u["anggotas"])
			ars := asInt(u["arsip"])
			peng := asInt(u["pengajuan"])
			u["stats"] = map[string]int{"anggotas": ang, "arsipSurats": ars, "pengajuanBerkass": peng}
			u["score"] = ang + ars + peng
		}
		sortMaps(leaders, "score", "desc", "score")
		var pending int
		if err := a.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM "User" WHERE "role"='SEKRETARIS_PAC' AND "isActive"=false`).Scan(&pending); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menghitung verifikasi pending")
			return
		}
		perkaderan := map[string]int{"Makesta": 0, "Lakmud": 0, "Latin": 0, "Latpel": 0, "Lakut": 0, "Diklatama": 0, "Diklatmad": 0}
		kaderRows, err := a.db.Query(r.Context(), `SELECT k."namaPerkaderan" FROM "Perkaderan" k JOIN "Anggota" a ON a."id"=k."anggotaId" JOIN "Periode" p ON p."id"=a."periodeId" WHERE p."isActive"=true`)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil perkaderan")
			return
		}
		for kaderRows.Next() {
			var encrypted string
			if err := kaderRows.Scan(&encrypted); err != nil {
				kaderRows.Close()
				writeError(w, 500, "DATABASE_ERROR", "Gagal membaca perkaderan")
				return
			}
			plain, err := a.crypto.DecryptText(encrypted)
			if err != nil {
				kaderRows.Close()
				writeError(w, 500, "DATA_UNREADABLE", "Data perkaderan tidak dapat dibaca")
				return
			}
			if label := titleKader(plain); label != "" {
				perkaderan[label]++
			}
		}
		err = kaderRows.Err()
		kaderRows.Close()
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca perkaderan")
			return
		}
		pendidikan := map[string]int{"SD": 0, "MI": 0, "SMP": 0, "MTs": 0, "SMA": 0, "SMK": 0, "MAN": 0, "KULIAH": 0}
		pendidikanRows, err := a.db.Query(r.Context(), `SELECT d."jenjang",COUNT(*) FROM "Pendidikan" d JOIN "Anggota" a ON a."id"=d."anggotaId" JOIN "Periode" p ON p."id"=a."periodeId" WHERE p."isActive"=true GROUP BY d."jenjang"`)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil pendidikan")
			return
		}
		for pendidikanRows.Next() {
			var label string
			var count int
			if err := pendidikanRows.Scan(&label, &count); err != nil {
				pendidikanRows.Close()
				writeError(w, 500, "DATABASE_ERROR", "Gagal membaca pendidikan")
				return
			}
			if _, exists := pendidikan[label]; exists {
				pendidikan[label] = count
			}
		}
		err = pendidikanRows.Err()
		pendidikanRows.Close()
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca pendidikan")
			return
		}
		monitoring = map[string]any{"leaderboard": leaders, "global": map[string]any{"totalAnggota": globalAnggota, "totalSurat": globalArsip + globalPimpinan + globalPengajuan, "totalPAC": userCount, "verifikasiPending": pending, "perkaderan": perkaderan, "pendidikan": pendidikan}}
	}
	writeJSON(w, 200, map[string]any{"role": role, "emailVerified": auth.User.EmailVerified, "personal": personal, "monitoring": monitoring})
}

func titleKader(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "MAKESTA":
		return "Makesta"
	case "LAKMUD":
		return "Lakmud"
	case "LATIN":
		return "Latin"
	case "LATPEL":
		return "Latpel"
	case "LAKUT":
		return "Lakut"
	case "DIKLATAMA":
		return "Diklatama"
	case "DIKLATMAD":
		return "Diklatmad"
	}
	return ""
}
func indonesianMonthShort(month time.Month) string {
	return []string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}[int(month)-1]
}
func asInt(value any) int {
	switch v := value.(type) {
	case int64:
		return int(v)
	case int32:
		return int(v)
	case int:
		return v
	}
	return 0
}

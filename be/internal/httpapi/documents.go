package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"laci-backend/internal/security"
)

func (a *API) listArsip(w http.ResponseWriter, r *http.Request) {
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
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	org := r.URL.Query().Get("organisasi")
	jenis := r.URL.Query().Get("jenis")
	if org != "" && org != "ALL" && !oneOf(org, "IPNU", "IPPNU", "BERSAMA", "CBP_KPP") {
		writeError(w, 400, "INVALID_FILTER", "Filter organisasi tidak valid")
		return
	}
	if jenis != "" && jenis != "ALL" && !oneOf(jenis, "MASUK", "KELUAR") {
		writeError(w, 400, "INVALID_FILTER", "Filter jenis surat tidak valid")
		return
	}
	where := ` WHERE a."userId"=$1 AND a."periodeId"=$2`
	args := []any{auth.User.ID, periodeID}
	if org != "" && org != "ALL" {
		args = append(args, org)
		where += ` AND a."organisasi"=$` + strconv.Itoa(len(args)) + `::"Organisasi"`
	}
	if jenis != "" && jenis != "ALL" {
		args = append(args, jenis)
		where += ` AND a."jenisSurat"=$` + strconv.Itoa(len(args)) + `::"JenisSurat"`
	}
	const columns = `SELECT a."id",a."userId",a."periodeId",a."noSurat",a."jenisSurat"::text,a."tanggal",a."pengirimPenerima",a."deskripsi",a."file",a."perihal",a."createdAt",a."updatedAt",a."organisasi"::text,u."name" AS "uploader",p."nama" AS "periode" FROM "ArsipSurat" a JOIN "User" u ON u."id"=a."userId" JOIN "Periode" p ON p."id"=a."periodeId"`
	sortKey := r.URL.Query().Get("sortKey")
	sortColumn, sqlSort := arsipSortColumn(sortKey)
	if query == "" && sqlSort {
		var total int
		if err := a.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM "ArsipSurat" a`+where, args...).Scan(&total); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menghitung arsip surat")
			return
		}
		if total == 0 || page > totalPages(total, limit) {
			writeJSON(w, 200, map[string]any{"data": []any{}, "total": total, "totalPages": totalPages(total, limit)})
			return
		}
		dir := "DESC"
		nulls := "NULLS LAST"
		if r.URL.Query().Get("sortDir") == "asc" {
			dir = "ASC"
			nulls = "NULLS FIRST"
		}
		pageArgs := append(append([]any{}, args...), limit, (page-1)*limit)
		rows, err := a.db.Query(r.Context(), columns+where+` ORDER BY `+sortColumn+` `+dir+` `+nulls+`, a."id" `+dir+` LIMIT $`+strconv.Itoa(len(args)+1)+` OFFSET $`+strconv.Itoa(len(args)+2), pageArgs...)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil arsip surat")
			return
		}
		data, err := scanMaps(rows)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil arsip surat")
			return
		}
		for _, item := range data {
			if err := decryptFieldStrict(a.crypto, item, "noSurat", "pengirimPenerima", "deskripsi", "perihal"); err != nil {
				writeError(w, 500, "DATA_UNREADABLE", "Data arsip tidak dapat dibaca")
				return
			}
			normalizeOptionalFile(item)
		}
		writeJSON(w, 200, map[string]any{"data": data, "total": total, "totalPages": totalPages(total, limit)})
		return
	}
	rows, err := a.db.Query(r.Context(), columns+where+` ORDER BY a."createdAt" DESC, a."id" DESC`, args...)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil arsip surat")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil arsip surat")
		return
	}
	filtered := make([]map[string]any, 0, len(data))
	for _, item := range data {
		if err := decryptFieldStrict(a.crypto, item, "noSurat", "pengirimPenerima", "deskripsi", "perihal"); err != nil {
			writeError(w, 500, "DATA_UNREADABLE", "Data arsip tidak dapat dibaca")
			return
		}
		normalizeOptionalFile(item)
		if query != "" && !mapContains(item, query, "noSurat", "pengirimPenerima", "deskripsi", "perihal") {
			continue
		}
		filtered = append(filtered, item)
	}
	sortMaps(filtered, sortKey, r.URL.Query().Get("sortDir"), "createdAt")
	writeJSON(w, 200, paginateMaps(filtered, page, limit))
}

func arsipSortColumn(key string) (string, bool) {
	switch key {
	case "", "createdAt":
		return `a."createdAt"`, true
	case "tanggal":
		return `a."tanggal"`, true
	case "updatedAt":
		return `a."updatedAt"`, true
	case "organisasi":
		return `a."organisasi"::text`, true
	case "jenisSurat":
		return `a."jenisSurat"::text`, true
	default:
		return "", false
	}
}

func (a *API) arsipStats(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	periodeID, err := a.targetPeriode(r.Context(), auth.User.ID, viewPeriode(r))
	if err != nil {
		if periodLookupFailed(w, err) {
			return
		}
		writeJSON(w, 200, map[string]int{"total": 0, "masuk": 0, "keluar": 0, "ipnu": 0, "ippnu": 0, "bersama": 0, "cbpkpp": 0})
		return
	}
	var total, masuk, keluar, ipnu, ippnu, bersama, cbpkpp int
	err = a.db.QueryRow(r.Context(), `SELECT COUNT(*),COUNT(*) FILTER(WHERE "jenisSurat"='MASUK'),COUNT(*) FILTER(WHERE "jenisSurat"='KELUAR'),COUNT(*) FILTER(WHERE "organisasi"='IPNU'),COUNT(*) FILTER(WHERE "organisasi"='IPPNU'),COUNT(*) FILTER(WHERE "organisasi"='BERSAMA'),COUNT(*) FILTER(WHERE "organisasi"='CBP_KPP') FROM "ArsipSurat" WHERE "userId"=$1 AND "periodeId"=$2`, auth.User.ID, periodeID).Scan(&total, &masuk, &keluar, &ipnu, &ippnu, &bersama, &cbpkpp)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil statistik arsip")
		return
	}
	writeJSON(w, 200, map[string]int{"total": total, "masuk": masuk, "keluar": keluar, "ipnu": ipnu, "ippnu": ippnu, "bersama": bersama, "cbpkpp": cbpkpp})
}

func (a *API) getArsip(w http.ResponseWriter, r *http.Request) {
	a.getEncryptedDocument(w, r, "ArsipSurat", []string{"noSurat", "pengirimPenerima", "deskripsi", "perihal"})
}

func (a *API) createArsip(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		writeError(w, 400, "INVALID_FORM", "Form arsip tidak valid")
		return
	}
	periodeID, err := a.targetPeriode(r.Context(), auth.User.ID, "")
	if err != nil {
		if periodLookupFailed(w, err) {
			return
		}
		writeError(w, 400, "NO_ACTIVE_PERIODE", "Tidak ada periode aktif. Silakan aktifkan periode terlebih dahulu.")
		return
	}
	noSurat := strings.TrimSpace(r.FormValue("noSurat"))
	jenis := r.FormValue("jenisSurat")
	tanggal, derr := parseDate(r.FormValue("tanggal"))
	pp := strings.TrimSpace(r.FormValue("pengirimPenerima"))
	perihal := strings.TrimSpace(r.FormValue("perihal"))
	org := r.FormValue("organisasi")
	desc := strings.TrimSpace(r.FormValue("deskripsi"))
	if noSurat == "" || pp == "" || perihal == "" || derr != nil || !oneOf(jenis, "MASUK", "KELUAR") || !oneOf(org, "IPNU", "IPPNU", "BERSAMA", "CBP_KPP") {
		writeError(w, 400, "VALIDATION_ERROR", "Field wajib belum lengkap")
		return
	}
	fileKey, err := a.handleOptionalUpload(r, "file", "arsip", 2<<20, "")
	if err != nil {
		writeUploadFailure(w, err)
		return
	}
	encrypted, err := a.encryptStrings(noSurat, pp, perihal, desc)
	if err != nil {
		if fileKey != "" {
			a.deleteStorageObject(fileKey)
		}
		writeEncryptionError(w)
		return
	}
	encNo, encPP, encPerihal, encDesc := encrypted[0], encrypted[1], encrypted[2], nullable(encrypted[3])
	id := newID()
	_, err = a.db.Exec(r.Context(), `INSERT INTO "ArsipSurat"("id","userId","periodeId","noSurat","jenisSurat","tanggal","pengirimPenerima","deskripsi","file","perihal","organisasi","createdAt","updatedAt") VALUES($1,$2,$3,$4,$5::"JenisSurat",$6,$7,$8,$9,$10,$11::"Organisasi",NOW(),NOW())`, id, auth.User.ID, periodeID, encNo, jenis, tanggal, encPP, encDesc, nullable(fileKey), encPerihal, org)
	if err != nil {
		if fileKey != "" {
			a.deleteStorageObject(fileKey)
		}
		writeError(w, 500, "DATABASE_ERROR", "Gagal menyimpan arsip surat")
		return
	}
	_ = a.createLog(r.Context(), r, auth.User, "CREATE", "ARSIP_SURAT", "Membuat arsip surat: "+noSurat, id)
	a.notify(r.Context(), map[string]any{"type": "arsip.created", "id": id, "userId": auth.User.ID})
	writeJSON(w, 201, map[string]any{"success": "Arsip surat berhasil dibuat!", "data": map[string]any{"id": id}})
}

func (a *API) updateArsip(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		writeError(w, 400, "INVALID_FORM", "Form arsip tidak valid")
		return
	}
	var oldFile string
	if err := a.db.QueryRow(r.Context(), `SELECT COALESCE("file",'') FROM "ArsipSurat" WHERE "id"=$1 AND "userId"=$2`, id, auth.User.ID).Scan(&oldFile); err != nil {
		writeLookupError(w, err, "NOT_FOUND", "Arsip surat tidak ditemukan")
		return
	}
	noSurat := strings.TrimSpace(r.FormValue("noSurat"))
	jenis := r.FormValue("jenisSurat")
	tanggal, derr := parseDate(r.FormValue("tanggal"))
	pp := strings.TrimSpace(r.FormValue("pengirimPenerima"))
	perihal := strings.TrimSpace(r.FormValue("perihal"))
	org := r.FormValue("organisasi")
	desc := strings.TrimSpace(r.FormValue("deskripsi"))
	if noSurat == "" || pp == "" || perihal == "" || derr != nil || !oneOf(jenis, "MASUK", "KELUAR") || !oneOf(org, "IPNU", "IPPNU", "BERSAMA", "CBP_KPP") {
		writeError(w, 400, "VALIDATION_ERROR", "Field wajib belum lengkap")
		return
	}
	fileKey, err := a.handleOptionalUpload(r, "file", "arsip", 2<<20, oldFile)
	if err != nil {
		writeUploadFailure(w, err)
		return
	}
	encrypted, err := a.encryptStrings(noSurat, pp, perihal, desc)
	if err != nil {
		if fileKey != oldFile {
			a.deleteStorageObject(fileKey)
		}
		writeEncryptionError(w)
		return
	}
	encNo, encPP, encPerihal, encDesc := encrypted[0], encrypted[1], encrypted[2], nullable(encrypted[3])
	result, err := a.db.Exec(r.Context(), `UPDATE "ArsipSurat" SET "noSurat"=$3,"jenisSurat"=$4::"JenisSurat","tanggal"=$5,"pengirimPenerima"=$6,"deskripsi"=$7,"file"=$8,"perihal"=$9,"organisasi"=$10::"Organisasi","updatedAt"=NOW() WHERE "id"=$1 AND "userId"=$2`, id, auth.User.ID, encNo, jenis, tanggal, encPP, encDesc, nullable(fileKey), encPerihal, org)
	if err != nil {
		if fileKey != oldFile {
			a.deleteStorageObject(fileKey)
		}
		writeError(w, 500, "DATABASE_ERROR", "Gagal memperbarui arsip surat")
		return
	}
	if result.RowsAffected() == 0 {
		if fileKey != oldFile {
			a.deleteStorageObject(fileKey)
		}
		writeError(w, 404, "NOT_FOUND", "Arsip surat tidak ditemukan")
		return
	}
	if fileKey != oldFile && oldFile != "" && !strings.HasPrefix(oldFile, "/storage") {
		a.deleteStorageObject(oldFile)
	}
	_ = a.createLog(r.Context(), r, auth.User, "UPDATE", "ARSIP_SURAT", "Mengupdate arsip surat: "+noSurat, id)
	a.notify(r.Context(), map[string]any{"type": "arsip.updated", "id": id, "userId": auth.User.ID})
	writeJSON(w, 200, map[string]any{"success": "Arsip surat berhasil diperbarui!"})
}

func (a *API) deleteArsip(w http.ResponseWriter, r *http.Request) {
	a.deleteDocument(w, r, "ArsipSurat", "ARSIP_SURAT", "Arsip surat berhasil dihapus!")
}
func (a *API) arsipDownloadToken(w http.ResponseWriter, r *http.Request) {
	a.documentDownloadToken(w, r, "ArsipSurat", "arsip")
}

func (a *API) listBerkasPimpinan(w http.ResponseWriter, r *http.Request) {
	a.listSimpleDocument(w, r, "BerkasPimpinan", []string{"nama", "catatan"}, "")
}
func (a *API) getBerkasPimpinan(w http.ResponseWriter, r *http.Request) {
	a.getEncryptedDocument(w, r, "BerkasPimpinan", []string{"nama", "catatan"})
}
func (a *API) berkasPimpinanStats(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	periodeID, err := a.targetPeriode(r.Context(), auth.User.ID, viewPeriode(r))
	if err != nil {
		if periodLookupFailed(w, err) {
			return
		}
		writeJSON(w, 200, map[string]int{"total": 0, "bulanIni": 0})
		return
	}
	var total, month int
	err = a.db.QueryRow(r.Context(), `SELECT COUNT(*),COUNT(*) FILTER(WHERE date_trunc('month',"tanggal")=date_trunc('month',CURRENT_DATE)) FROM "BerkasPimpinan" WHERE "userId"=$1 AND "periodeId"=$2`, auth.User.ID, periodeID).Scan(&total, &month)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil statistik")
		return
	}
	writeJSON(w, 200, map[string]int{"total": total, "bulanIni": month})
}
func (a *API) createBerkasPimpinan(w http.ResponseWriter, r *http.Request) {
	a.mutateSimpleDocument(w, r, "BerkasPimpinan", "berkas-pimpinan", "BERKAS_PIMPINAN", 5<<20, true)
}
func (a *API) updateBerkasPimpinan(w http.ResponseWriter, r *http.Request) {
	a.mutateSimpleDocument(w, r, "BerkasPimpinan", "berkas-pimpinan", "BERKAS_PIMPINAN", 5<<20, false)
}
func (a *API) deleteBerkasPimpinan(w http.ResponseWriter, r *http.Request) {
	a.deleteDocument(w, r, "BerkasPimpinan", "BERKAS_PIMPINAN", "Berkas berhasil dihapus!")
}
func (a *API) berkasPimpinanDownloadToken(w http.ResponseWriter, r *http.Request) {
	a.documentDownloadToken(w, r, "BerkasPimpinan", "berkas-pimpinan")
}
func (a *API) listBerkasSP(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	a.listSimpleDocument(w, r, "BerkasSP", []string{"nama", "catatan"}, r.URL.Query().Get("organisasi"))
}
func (a *API) getBerkasSP(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	a.getEncryptedDocument(w, r, "BerkasSP", []string{"nama", "catatan"})
}
func (a *API) berkasSPStats(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	auth := authFrom(r)
	periodeID, err := a.targetPeriode(r.Context(), auth.User.ID, viewPeriode(r))
	if err != nil {
		if periodLookupFailed(w, err) {
			return
		}
		writeJSON(w, 200, map[string]int{"total": 0, "ipnu": 0, "ippnu": 0})
		return
	}
	var total, ipnu, ippnu int
	err = a.db.QueryRow(r.Context(), `SELECT COUNT(*),COUNT(*) FILTER(WHERE "organisasi"='IPNU'),COUNT(*) FILTER(WHERE "organisasi"='IPPNU') FROM "BerkasSP" WHERE "userId"=$1 AND "periodeId"=$2`, auth.User.ID, periodeID).Scan(&total, &ipnu, &ippnu)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil statistik")
		return
	}
	writeJSON(w, 200, map[string]int{"total": total, "ipnu": ipnu, "ippnu": ippnu})
}
func (a *API) createBerkasSP(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	a.mutateSimpleDocument(w, r, "BerkasSP", "berkas-sp", "BERKAS_SP", 2<<20, true)
}
func (a *API) updateBerkasSP(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	a.mutateSimpleDocument(w, r, "BerkasSP", "berkas-sp", "BERKAS_SP", 2<<20, false)
}
func (a *API) deleteBerkasSP(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	a.deleteDocument(w, r, "BerkasSP", "BERKAS_SP", "Berkas SP berhasil dihapus!")
}
func (a *API) berkasSPDownloadToken(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	a.documentDownloadToken(w, r, "BerkasSP", "berkas-sp")
}
func (a *API) listSimpleDocument(w http.ResponseWriter, r *http.Request, table string, encrypted []string, org string) {
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
	where := ` WHERE b."userId"=$1 AND b."periodeId"=$2`
	args := []any{auth.User.ID, periodeID}
	var columns, countFrom string
	switch table {
	case "BerkasPimpinan":
		columns = `SELECT b.*,p."nama" AS "periodeNama" FROM "BerkasPimpinan" b JOIN "Periode" p ON p."id"=b."periodeId"`
		countFrom = ` FROM "BerkasPimpinan" b`
	case "BerkasSP":
		columns = `SELECT b.*,b."organisasi"::text AS "organisasiText",p."nama" AS "periodeNama" FROM "BerkasSP" b JOIN "Periode" p ON p."id"=b."periodeId"`
		countFrom = ` FROM "BerkasSP" b`
		if org != "" && org != "ALL" {
			if !oneOf(org, "IPNU", "IPPNU") {
				writeError(w, 400, "INVALID_FILTER", "Filter organisasi tidak valid")
				return
			}
			args = append(args, org)
			where += ` AND b."organisasi"=$3::"Organisasi"`
		}
	default:
		writeError(w, 500, "INVALID_DOCUMENT_TYPE", "Jenis dokumen tidak dikenal")
		return
	}
	sortKey := r.URL.Query().Get("sortKey")
	sortColumn, sqlSort := simpleDocumentSortColumn(table, sortKey)
	if q == "" && sqlSort {
		var total int
		if err := a.db.QueryRow(r.Context(), `SELECT COUNT(*)`+countFrom+where, args...).Scan(&total); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menghitung data")
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
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil data")
			return
		}
		data, err := scanMaps(rows)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil data")
			return
		}
		for _, item := range data {
			if err := enrichSimpleDocument(a, item, encrypted); err != nil {
				writeError(w, 500, "DATA_UNREADABLE", "Data dokumen tidak dapat dibaca")
				return
			}
		}
		writeJSON(w, 200, map[string]any{"data": data, "total": total, "totalPages": totalPages(total, limit)})
		return
	}
	rows, err := a.db.Query(r.Context(), columns+where+` ORDER BY b."createdAt" DESC, b."id" DESC`, args...)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil data")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil data")
		return
	}
	filtered := []map[string]any{}
	for _, item := range data {
		if err := enrichSimpleDocument(a, item, encrypted); err != nil {
			writeError(w, 500, "DATA_UNREADABLE", "Data dokumen tidak dapat dibaca")
			return
		}
		if q != "" && !mapContains(item, q, encrypted...) {
			continue
		}
		filtered = append(filtered, item)
	}
	sortMaps(filtered, r.URL.Query().Get("sortKey"), r.URL.Query().Get("sortDir"), "createdAt")
	writeJSON(w, 200, paginateMaps(filtered, page, limit))
}

func simpleDocumentSortColumn(table, key string) (string, bool) {
	switch table {
	case "BerkasPimpinan":
		switch key {
		case "", "createdAt":
			return `b."createdAt"`, true
		case "tanggal":
			return `b."tanggal"`, true
		}
	case "BerkasSP":
		switch key {
		case "", "createdAt":
			return `b."createdAt"`, true
		case "tanggalMulai":
			return `b."tanggalMulai"`, true
		case "tanggalBerakhir", "status":
			return `b."tanggalBerakhir"`, true
		case "organisasi":
			return `b."organisasi"::text`, true
		}
	}
	return "", false
}

func enrichSimpleDocument(a *API, item map[string]any, encrypted []string) error {
	if v, ok := item["organisasiText"]; ok {
		item["organisasi"] = v
		delete(item, "organisasiText")
	}
	if err := decryptFieldStrict(a.crypto, item, encrypted...); err != nil {
		return err
	}
	normalizeOptionalFile(item)
	return nil
}

func normalizeOptionalFile(item map[string]any) {
	if value, ok := item["file"].(string); ok && strings.TrimSpace(value) == "" {
		item["file"] = nil
	}
}

func (a *API) getEncryptedDocument(w http.ResponseWriter, r *http.Request, table string, encrypted []string) {
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	var query string
	switch table {
	case "ArsipSurat":
		query = `SELECT a."id",a."userId",a."periodeId",a."noSurat",a."jenisSurat"::text,a."tanggal",a."pengirimPenerima",a."deskripsi",a."file",a."perihal",a."createdAt",a."updatedAt",a."organisasi"::text,p."nama" AS "periodeNama" FROM "ArsipSurat" a JOIN "Periode" p ON p."id"=a."periodeId" WHERE a."id"=$1 AND a."userId"=$2`
	case "BerkasPimpinan":
		query = `SELECT b.*,p."nama" AS "periodeNama" FROM "BerkasPimpinan" b JOIN "Periode" p ON p."id"=b."periodeId" WHERE b."id"=$1 AND b."userId"=$2`
	case "BerkasSP":
		query = `SELECT b.*,b."organisasi"::text AS "organisasiText",p."nama" AS "periodeNama" FROM "BerkasSP" b JOIN "Periode" p ON p."id"=b."periodeId" WHERE b."id"=$1 AND b."userId"=$2`
	}
	rows, err := a.db.Query(r.Context(), query, id, auth.User.ID)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil data")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca data")
		return
	}
	if len(data) == 0 {
		writeError(w, 404, "NOT_FOUND", "Data tidak ditemukan")
		return
	}
	item := data[0]
	if v, ok := item["organisasiText"]; ok {
		item["organisasi"] = v
		delete(item, "organisasiText")
	}
	if err := decryptFieldStrict(a.crypto, item, encrypted...); err != nil {
		writeError(w, 500, "DATA_UNREADABLE", "Data dokumen tidak dapat dibaca")
		return
	}
	normalizeOptionalFile(item)
	if name, ok := item["periodeNama"]; ok {
		item["periode"] = map[string]any{"nama": name}
	}
	writeJSON(w, 200, map[string]any{"data": item})
}

func (a *API) mutateSimpleDocument(w http.ResponseWriter, r *http.Request, table, prefix, module string, maxFile int64, creating bool) {
	auth := authFrom(r)
	if err := r.ParseMultipartForm(maxFile + (2 << 20)); err != nil {
		writeError(w, 400, "INVALID_FORM", "Form tidak valid")
		return
	}
	id := chi.URLParam(r, "id")
	periodeID := ""
	oldFile := ""
	if creating {
		var err error
		periodeID, err = a.targetPeriode(r.Context(), auth.User.ID, "")
		if err != nil {
			if periodLookupFailed(w, err) {
				return
			}
			writeError(w, 400, "NO_ACTIVE_PERIODE", "Tidak ada periode aktif. Silakan aktifkan periode terlebih dahulu.")
			return
		}
		id = newID()
	} else {
		var err error
		query := `SELECT "periodeId",COALESCE("file",'') FROM "` + table + `" WHERE "id"=$1 AND "userId"=$2`
		err = a.db.QueryRow(r.Context(), query, id, auth.User.ID).Scan(&periodeID, &oldFile)
		if err != nil {
			writeLookupError(w, err, "NOT_FOUND", "Data tidak ditemukan")
			return
		}
	}
	name := strings.TrimSpace(r.FormValue("nama"))
	catatan := strings.TrimSpace(r.FormValue("catatan"))
	if name == "" {
		writeError(w, 400, "VALIDATION_ERROR", "Nama harus diisi")
		return
	}
	var tanggal, mulai, akhir time.Time
	org := r.FormValue("organisasi")
	if table == "BerkasPimpinan" {
		var err error
		tanggal, err = parseDate(r.FormValue("tanggal"))
		if err != nil {
			writeError(w, 400, "INVALID_DATE", "Tanggal harus diisi")
			return
		}
	} else {
		var startErr, endErr error
		mulai, startErr = parseDate(r.FormValue("tanggalMulai"))
		akhir, endErr = parseDate(r.FormValue("tanggalBerakhir"))
		if startErr != nil || endErr != nil || !oneOf(org, "IPNU", "IPPNU") {
			writeError(w, 400, "VALIDATION_ERROR", "Organisasi dan tanggal harus diisi")
			return
		}
		if akhir.Before(mulai) {
			writeError(w, 400, "INVALID_DATE_RANGE", "Tanggal berakhir tidak boleh sebelum tanggal mulai")
			return
		}
	}
	encrypted, err := a.encryptStrings(name, catatan)
	if err != nil {
		writeEncryptionError(w)
		return
	}
	encName, encCatatan := encrypted[0], nullable(encrypted[1])
	fileKey, err := a.handleOptionalUpload(r, "file", prefix, maxFile, oldFile)
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
	var result pgconn.CommandTag
	if table == "BerkasPimpinan" {
		if creating {
			result, err = a.db.Exec(r.Context(), `INSERT INTO "BerkasPimpinan"("id","userId","periodeId","nama","tanggal","catatan","file","createdAt","updatedAt") VALUES($1,$2,$3,$4,$5,$6,$7,NOW(),NOW())`, id, auth.User.ID, periodeID, encName, tanggal, encCatatan, nullable(fileKey))
		} else {
			result, err = a.db.Exec(r.Context(), `UPDATE "BerkasPimpinan" SET "nama"=$3,"tanggal"=$4,"catatan"=$5,"file"=$6,"updatedAt"=NOW() WHERE "id"=$1 AND "userId"=$2`, id, auth.User.ID, encName, tanggal, encCatatan, nullable(fileKey))
		}
	} else {
		if creating {
			result, err = a.db.Exec(r.Context(), `INSERT INTO "BerkasSP"("id","userId","periodeId","nama","tanggalMulai","tanggalBerakhir","catatan","file","organisasi","createdAt","updatedAt") VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::"Organisasi",NOW(),NOW())`, id, auth.User.ID, periodeID, encName, mulai, akhir, encCatatan, nullable(fileKey), org)
		} else {
			result, err = a.db.Exec(r.Context(), `UPDATE "BerkasSP" SET "nama"=$3,"tanggalMulai"=$4,"tanggalBerakhir"=$5,"catatan"=$6,"file"=$7,"organisasi"=$8::"Organisasi","updatedAt"=NOW() WHERE "id"=$1 AND "userId"=$2`, id, auth.User.ID, encName, mulai, akhir, encCatatan, nullable(fileKey), org)
		}
	}
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menyimpan data")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, 404, "NOT_FOUND", "Data tidak ditemukan")
		return
	}
	saved = true
	if fileKey != oldFile && oldFile != "" && !strings.HasPrefix(oldFile, "/storage") {
		a.deleteStorageObject(oldFile)
	}
	action := "UPDATE"
	message := "Data berhasil diperbarui!"
	if creating {
		action = "CREATE"
		message = "Data berhasil dibuat!"
	}
	_ = a.createLog(r.Context(), r, auth.User, action, module, action+" "+name, id)
	a.notify(r.Context(), map[string]any{"type": strings.ToLower(module) + ".changed", "id": id, "userId": auth.User.ID})
	writeJSON(w, map[bool]int{true: 201, false: 200}[creating], map[string]any{"success": message, "data": map[string]string{"id": id}})
}

func (a *API) handleOptionalUpload(r *http.Request, field, prefix string, max int64, old string) (string, error) {
	file, header, err := r.FormFile(field)
	if err == http.ErrMissingFile {
		return old, nil
	}
	if err != nil {
		return "", err
	}
	if header.Filename == "" && header.Size == 0 {
		file.Close()
		return old, nil
	}
	if header.Size > max {
		file.Close()
		return "", &userError{"Ukuran file melebihi batas"}
	}
	plain, err := readUpload(file, max)
	if err != nil {
		return "", err
	}
	contentType, err := validateUpload(header.Filename, plain, false)
	if err != nil {
		return "", err
	}
	encrypted, err := a.crypto.EncryptFile(plain)
	if err != nil {
		return "", err
	}
	key := prefix + "/" + security.EncryptedFilename(header.Filename)
	if err = a.r2.Put(r.Context(), key, contentType, encrypted); err != nil {
		slog.Warn("storage upload failed", "error", err)
		return "", &storageUploadError{err}
	}
	return key, nil
}

type storageUploadError struct{ error }

func writeUploadFailure(w http.ResponseWriter, err error) {
	if _, failed := err.(*storageUploadError); failed {
		writeError(w, 502, "STORAGE_FAILED", "Layanan penyimpanan tidak tersedia")
		return
	}
	writeError(w, 400, "UPLOAD_FAILED", err.Error())
}

func (a *API) deleteDocument(w http.ResponseWriter, r *http.Request, table, module, message string) {
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	var file string
	query := `SELECT COALESCE("file",'') FROM "` + table + `" WHERE "id"=$1 AND "userId"=$2`
	if err := a.db.QueryRow(r.Context(), query, id, auth.User.ID).Scan(&file); err != nil {
		writeLookupError(w, err, "NOT_FOUND", "Data tidak ditemukan")
		return
	}
	result, err := a.db.Exec(r.Context(), `DELETE FROM "`+table+`" WHERE "id"=$1 AND "userId"=$2`, id, auth.User.ID)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menghapus data")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, 404, "NOT_FOUND", "Data tidak ditemukan")
		return
	}
	a.deleteStorageObject(file)
	_ = a.createLog(r.Context(), r, auth.User, "DELETE", module, "Menghapus data "+id, id)
	a.notify(r.Context(), map[string]any{"type": strings.ToLower(module) + ".deleted", "id": id, "userId": auth.User.ID})
	writeJSON(w, 200, map[string]any{"success": message})
}

func (a *API) documentDownloadToken(w http.ResponseWriter, r *http.Request, table, kind string) {
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	var exists bool
	if err := a.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM "`+table+`" WHERE "id"=$1 AND "userId"=$2)`, id, auth.User.ID).Scan(&exists); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal memeriksa file")
		return
	}
	if !exists {
		writeError(w, 404, "NOT_FOUND", "Data tidak ditemukan")
		return
	}
	token, err := a.crypto.DownloadToken(kind+":"+id, 5*time.Minute)
	if err != nil {
		writeError(w, 500, "TOKEN_FAILED", "Gagal membuat token")
		return
	}
	writeJSON(w, 200, map[string]any{"token": token})
}

func paginateMaps(data []map[string]any, page, limit int) map[string]any {
	total := len(data)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	start := total
	if page-1 <= total/limit {
		start = (page - 1) * limit
	}
	end := total
	if total-start > limit {
		end = start + limit
	}
	return map[string]any{"data": data[start:end], "total": total, "totalPages": totalPages(total, limit)}
}
func mapContains(item map[string]any, q string, keys ...string) bool {
	for _, key := range keys {
		if v, ok := item[key].(string); ok && strings.Contains(strings.ToLower(v), q) {
			return true
		}
	}
	return false
}
func sortMaps(data []map[string]any, key, dir, fallback string) {
	if key == "" {
		key = fallback
	}
	desc := dir != "asc"
	sort.SliceStable(data, func(i, j int) bool {
		comparison := compareValues(data[i][key], data[j][key])
		if desc {
			return comparison > 0
		}
		return comparison < 0
	})
}

func compareValues(a, b any) int {
	if av, ok := a.(time.Time); ok {
		bv, valid := b.(time.Time)
		if !valid || av.Equal(bv) {
			return 0
		}
		if av.Before(bv) {
			return -1
		}
		return 1
	}
	if av, ok := numericValue(a); ok {
		bv, valid := numericValue(b)
		if !valid || av == bv {
			return 0
		}
		if av < bv {
			return -1
		}
		return 1
	}
	return strings.Compare(normalizeComparableString(toString(a)), normalizeComparableString(toString(b)))
}

func normalizeComparableString(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, value)
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func numericValue(value any) (float64, bool) {
	switch number := value.(type) {
	case int:
		return float64(number), true
	case int32:
		return float64(number), true
	case int64:
		return float64(number), true
	case float32:
		return float64(number), true
	case float64:
		return number, true
	default:
		return 0, false
	}
}
func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}
func oneOf(value string, values ...string) bool {
	for _, v := range values {
		if value == v {
			return true
		}
	}
	return false
}
func itoa(value int) string { return strconv.Itoa(value) }

type userError struct{ message string }

func (e *userError) Error() string { return e.message }

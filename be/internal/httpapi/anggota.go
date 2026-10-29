package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"laci-backend/internal/security"
)

var anggotaEncryptedFields = []string{"namaLengkap", "nik", "nia", "tempatLahir", "alamatLengkap", "noHp", "hobi", "jabatan", "noRfid", "pekerjaan", "namaInstansiPendidikan"}
var errAnggotaRelationUnreadable = errors.New("anggota relation unreadable")
var anggotaNIKPattern = regexp.MustCompile(`^[0-9]{16}$`)
var anggotaPhonePattern = regexp.MustCompile(`^\+?[0-9]{8,15}$`)

func (a *API) listAnggota(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	page, limit := parsePage(r)
	targetID, err := a.targetPeriode(r.Context(), auth.User.ID, viewPeriode(r))
	if err != nil {
		if periodLookupFailed(w, err) {
			return
		}
		writeJSON(w, 200, map[string]any{"data": []any{}, "total": 0, "totalPages": 0})
		return
	}
	var periodName string
	if err := a.db.QueryRow(r.Context(), `SELECT "nama" FROM "Periode" WHERE "id"=$1`, targetID).Scan(&periodName); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca periode")
		return
	}
	selectedUser := r.URL.Query().Get("userId")
	explicitPeriod := r.URL.Query().Get("periodeId")
	if explicitPeriod != "" {
		if err := a.db.QueryRow(r.Context(), `SELECT "nama" FROM "Periode" WHERE "id"=$1 AND "userId"=$2`, explicitPeriod, auth.User.ID).Scan(&periodName); err != nil {
			writeLookupError(w, err, "NOT_FOUND", "Periode tidak ditemukan")
			return
		}
	}
	query := `SELECT a."id",a."userId",a."periodeId",a."nik",a."nia",a."email",a."foto",a."namaLengkap",a."tempatLahir",a."tanggalLahir",a."jenisKelamin"::text,a."alamatLengkap",a."noHp",a."hobi",a."jabatan",a."noRfid",a."pekerjaan",a."jenjangPendidikan",a."namaInstansiPendidikan",a."createdAt",a."updatedAt",u."name" AS "uploader",p."nama" AS "periodeNama" FROM "Anggota" a JOIN "User" u ON u."id"=a."userId" JOIN "Periode" p ON p."id"=a."periodeId" WHERE `
	args := []any{}
	if explicitPeriod != "" {
		query += `a."userId"=$1 AND a."periodeId"=$2`
		args = []any{auth.User.ID, explicitPeriod}
	} else if auth.User.Role != "SEKRETARIS_CABANG" {
		pid := targetID
		query += `a."userId"=$1 AND a."periodeId"=$2`
		args = []any{auth.User.ID, pid}
	} else if selectedUser != "" && selectedUser != "ALL" {
		if selectedUser == auth.User.ID {
			query += `a."userId"=$1 AND a."periodeId"=$2`
			args = []any{selectedUser, targetID}
		} else {
			query += `a."userId"=$1 AND p."nama"=$2`
			args = []any{selectedUser, periodName}
		}
	} else {
		query += `p."nama"=$1`
		args = []any{periodName}
	}
	query += ` ORDER BY a."createdAt" DESC, a."id" DESC`
	rows, err := a.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil data anggota")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil data anggota")
		return
	}
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	filtered := []map[string]any{}
	for _, item := range data {
		if err := decryptFieldStrict(a.crypto, item, anggotaEncryptedFields...); err != nil {
			writeError(w, 500, "DATA_UNREADABLE", "Data anggota tidak dapat dibaca")
			return
		}
		item["user"] = map[string]any{"id": item["userId"], "name": item["uploader"]}
		item["periode"] = map[string]any{"id": item["periodeId"], "nama": item["periodeNama"]}
		if q != "" && !mapContains(item, q, "namaLengkap", "nik", "nia", "email", "noHp", "jabatan") {
			continue
		}
		filtered = append(filtered, item)
	}
	sortKey := r.URL.Query().Get("sortKey")
	switch sortKey {
	case "dibuatOleh":
		sortKey = "uploader"
	case "periode":
		sortKey = "periodeNama"
	}
	sortMaps(filtered, sortKey, r.URL.Query().Get("sortDir"), "createdAt")
	result := paginateMaps(filtered, page, limit)
	if r.URL.Query().Get("includeRelations") == "true" {
		if err := a.attachAnggotaRelations(r.Context(), result["data"].([]map[string]any)); err != nil {
			writeAnggotaRelationError(w, err)
			return
		}
	}
	writeJSON(w, 200, result)
}

func (a *API) getAnggota(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	rows, err := a.db.Query(r.Context(), `SELECT "id","userId","periodeId","nik","nia","email","foto","namaLengkap","tempatLahir","tanggalLahir","jenisKelamin"::text,"alamatLengkap","noHp","hobi","jabatan","noRfid","pekerjaan","jenjangPendidikan","namaInstansiPendidikan","createdAt","updatedAt" FROM "Anggota" WHERE "id"=$1 AND ("userId"=$2 OR $3='SEKRETARIS_CABANG')`, id, auth.User.ID, auth.User.Role)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil anggota")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca anggota")
		return
	}
	if len(data) == 0 {
		writeError(w, 404, "NOT_FOUND", "Data anggota tidak ditemukan")
		return
	}
	item := data[0]
	if err := decryptFieldStrict(a.crypto, item, anggotaEncryptedFields...); err != nil {
		writeError(w, 500, "DATA_UNREADABLE", "Data anggota tidak dapat dibaca")
		return
	}
	if err := a.attachAnggotaRelations(r.Context(), data); err != nil {
		writeAnggotaRelationError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"data": item})
}

func writeAnggotaRelationError(w http.ResponseWriter, err error) {
	if errors.Is(err, errAnggotaRelationUnreadable) {
		writeError(w, 500, "DATA_UNREADABLE", "Data anggota tidak dapat dibaca")
		return
	}
	writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil relasi anggota")
}

func (a *API) attachAnggotaRelations(ctx context.Context, items []map[string]any) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, 0, len(items))
	byID := make(map[string]map[string]any, len(items))
	for _, item := range items {
		id, ok := item["id"].(string)
		if !ok {
			return errors.New("invalid anggota id")
		}
		ids = append(ids, id)
		byID[id] = item
		item["perkaderans"] = []map[string]any{}
		item["pendidikans"] = []map[string]any{}
	}

	pRows, err := a.db.Query(ctx, `SELECT "id","anggotaId","namaPerkaderan","tanggal","tempat","createdAt","updatedAt" FROM "Perkaderan" WHERE "anggotaId"=ANY($1) ORDER BY "tanggal","id"`, ids)
	if err != nil {
		return err
	}
	perkaderans, err := scanMaps(pRows)
	if err != nil {
		return err
	}
	for _, row := range perkaderans {
		if err := decryptFieldStrict(a.crypto, row, "namaPerkaderan", "tempat"); err != nil {
			return fmt.Errorf("%w: %v", errAnggotaRelationUnreadable, err)
		}
		id, _ := row["anggotaId"].(string)
		if item := byID[id]; item != nil {
			item["perkaderans"] = append(item["perkaderans"].([]map[string]any), row)
		}
	}

	eRows, err := a.db.Query(ctx, `SELECT "id","anggotaId","jenjang","namaSekolah","createdAt","updatedAt" FROM "Pendidikan" WHERE "anggotaId"=ANY($1) ORDER BY CASE "jenjang" WHEN 'SD' THEN 1 WHEN 'MI' THEN 2 WHEN 'SMP' THEN 3 WHEN 'MTs' THEN 4 WHEN 'SMA' THEN 5 WHEN 'SMK' THEN 6 WHEN 'MAN' THEN 7 WHEN 'KULIAH' THEN 8 ELSE 9 END,"id"`, ids)
	if err != nil {
		return err
	}
	pendidikans, err := scanMaps(eRows)
	if err != nil {
		return err
	}
	for _, row := range pendidikans {
		if err := decryptFieldStrict(a.crypto, row, "namaSekolah"); err != nil {
			return fmt.Errorf("%w: %v", errAnggotaRelationUnreadable, err)
		}
		id, _ := row["anggotaId"].(string)
		if item := byID[id]; item != nil {
			item["pendidikans"] = append(item["pendidikans"].([]map[string]any), row)
		}
	}
	return nil
}

func (a *API) createAnggota(w http.ResponseWriter, r *http.Request) { a.mutateAnggota(w, r, true) }
func (a *API) updateAnggota(w http.ResponseWriter, r *http.Request) { a.mutateAnggota(w, r, false) }

type kaderInput struct {
	NamaPerkaderan string `json:"namaPerkaderan"`
	Tanggal        string `json:"tanggal"`
	Tempat         string `json:"tempat"`
}
type pendidikanInput struct {
	Jenjang     string `json:"jenjang"`
	NamaSekolah string `json:"namaSekolah"`
}

func (a *API) mutateAnggota(w http.ResponseWriter, r *http.Request, creating bool) {
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		writeError(w, 400, "INVALID_FORM", "Form anggota tidak valid")
		return
	}
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	periodeID := ""
	ownerID := auth.User.ID
	oldPhoto := ""
	if creating {
		var err error
		periodeID, err = a.targetPeriode(r.Context(), auth.User.ID, "")
		if err != nil {
			if periodLookupFailed(w, err) {
				return
			}
			writeError(w, 400, "NO_ACTIVE_PERIODE", "Silakan aktifkan periode terlebih dahulu.")
			return
		}
		id = newID()
	} else {
		if err := a.db.QueryRow(r.Context(), `SELECT "periodeId","userId",COALESCE("foto",'') FROM "Anggota" WHERE "id"=$1 AND ("userId"=$2 OR $3='SEKRETARIS_CABANG')`, id, auth.User.ID, auth.User.Role).Scan(&periodeID, &ownerID, &oldPhoto); err != nil {
			writeLookupError(w, err, "NOT_FOUND", "Data tidak ditemukan")
			return
		}
	}
	name := strings.TrimSpace(r.FormValue("namaLengkap"))
	gender := r.FormValue("jenisKelamin")
	if name == "" || !oneOf(gender, "LAKI_LAKI", "PEREMPUAN") {
		writeError(w, 400, "VALIDATION_ERROR", "Nama lengkap dan jenis kelamin wajib diisi")
		return
	}
	nik := strings.ReplaceAll(strings.TrimSpace(r.FormValue("nik")), " ", "")
	if nik != "" && !anggotaNIKPattern.MatchString(nik) {
		writeError(w, 400, "INVALID_NIK", "NIK harus terdiri dari 16 digit angka")
		return
	}
	phone := strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(r.FormValue("noHp")))
	if phone != "" && !anggotaPhonePattern.MatchString(phone) {
		writeError(w, 400, "INVALID_PHONE", "Nomor handphone harus terdiri dari 8 sampai 15 digit angka")
		return
	}
	emailValue := strings.TrimSpace(r.FormValue("email"))
	if emailValue != "" {
		address, parseErr := mail.ParseAddress(emailValue)
		if parseErr != nil || !strings.EqualFold(address.Address, emailValue) {
			writeError(w, 400, "INVALID_EMAIL", "Format email tidak valid")
			return
		}
	}
	var birth any
	if raw := r.FormValue("tanggalLahir"); raw != "" {
		parsed, err := parseDate(raw)
		if err != nil {
			writeError(w, 400, "INVALID_DATE", "Tanggal lahir tidak valid")
			return
		}
		birth = parsed
	}
	kaders, educations, err := parseAnggotaChildren(r.FormValue("perkaderans"), r.FormValue("pendidikans"))
	if err != nil {
		writeError(w, 400, "INVALID_REQUEST", "Riwayat perkaderan atau pendidikan tidak valid")
		return
	}
	encryptedFields := make(map[string]any, len(anggotaEncryptedFields))
	for _, field := range anggotaEncryptedFields {
		value := strings.TrimSpace(r.FormValue(field))
		if field == "nik" {
			value = nik
		}
		if field == "noHp" {
			value = phone
		}
		if value == "" {
			encryptedFields[field] = nil
			continue
		}
		ciphertext, err := a.crypto.EncryptText(value)
		if err != nil {
			writeEncryptionError(w)
			return
		}
		encryptedFields[field] = ciphertext
	}
	enc := func(field string) any { return encryptedFields[field] }
	photo := oldPhoto
	if !creating && r.FormValue("removeFoto") == "true" {
		photo = ""
	}
	if file, header, err := r.FormFile("foto"); err == nil {
		if header.Size > 2<<20 {
			file.Close()
			writeError(w, 400, "FILE_TOO_LARGE", "Ukuran gambar maksimal 2MB")
			return
		}
		plain, e := readUpload(file, 2<<20)
		if e != nil {
			writeError(w, 400, "INVALID_FILE", "Gambar tidak dapat dibaca")
			return
		}
		contentType, e := validateUpload(header.Filename, plain, true)
		if e != nil {
			writeError(w, 400, "INVALID_FILE", e.Error())
			return
		}
		enc, e := a.crypto.EncryptFile(plain)
		if e != nil {
			writeError(w, 500, "ENCRYPTION_FAILED", "Gagal memproses gambar")
			return
		}
		photo = "anggota/" + security.EncryptedFilename(header.Filename)
		if e = a.r2.Put(r.Context(), photo, contentType, enc); e != nil {
			writeError(w, 502, "STORAGE_FAILED", "Gagal mengunggah gambar")
			return
		}
	}
	email := nullable(emailValue)
	newPhoto := photo != "" && photo != oldPhoto
	committed := false
	defer func() {
		if newPhoto && !committed {
			a.deleteStorageObject(photo)
		}
	}()
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menyimpan anggota")
		return
	}
	defer tx.Rollback(r.Context())
	if creating {
		_, err = tx.Exec(r.Context(), `INSERT INTO "Anggota"("id","userId","periodeId","nik","nia","email","foto","namaLengkap","tempatLahir","tanggalLahir","jenisKelamin","alamatLengkap","noHp","hobi","jabatan","noRfid","pekerjaan","jenjangPendidikan","namaInstansiPendidikan","createdAt","updatedAt") VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::"JenisKelamin",$12,$13,$14,$15,$16,$17,$18,$19,NOW(),NOW())`, id, auth.User.ID, periodeID, enc("nik"), enc("nia"), email, nullable(photo), enc("namaLengkap"), enc("tempatLahir"), birth, gender, enc("alamatLengkap"), enc("noHp"), enc("hobi"), enc("jabatan"), enc("noRfid"), enc("pekerjaan"), nullable(r.FormValue("jenjangPendidikan")), enc("namaInstansiPendidikan"))
	} else {
		result, updateErr := tx.Exec(r.Context(), `UPDATE "Anggota" SET "nik"=$4,"nia"=$5,"email"=$6,"foto"=$7,"namaLengkap"=$8,"tempatLahir"=$9,"tanggalLahir"=$10,"jenisKelamin"=$11::"JenisKelamin","alamatLengkap"=$12,"noHp"=$13,"hobi"=$14,"jabatan"=$15,"noRfid"=$16,"pekerjaan"=$17,"jenjangPendidikan"=$18,"namaInstansiPendidikan"=$19,"updatedAt"=NOW() WHERE "id"=$1 AND ("userId"=$2 OR $3='SEKRETARIS_CABANG')`, id, auth.User.ID, auth.User.Role, enc("nik"), enc("nia"), email, nullable(photo), enc("namaLengkap"), enc("tempatLahir"), birth, gender, enc("alamatLengkap"), enc("noHp"), enc("hobi"), enc("jabatan"), enc("noRfid"), enc("pekerjaan"), nullable(r.FormValue("jenjangPendidikan")), enc("namaInstansiPendidikan"))
		err = updateErr
		if err == nil {
			if result.RowsAffected() == 0 {
				writeError(w, 404, "NOT_FOUND", "Data anggota tidak ditemukan")
				return
			}
			if _, err = tx.Exec(r.Context(), `DELETE FROM "Perkaderan" WHERE "anggotaId"=$1`, id); err != nil {
				writeError(w, 500, "DATABASE_ERROR", "Gagal memperbarui perkaderan")
				return
			}
			if _, err = tx.Exec(r.Context(), `DELETE FROM "Pendidikan" WHERE "anggotaId"=$1`, id); err != nil {
				writeError(w, 500, "DATABASE_ERROR", "Gagal memperbarui pendidikan")
				return
			}
		}
	}
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menyimpan data anggota")
		return
	}
	for _, p := range kaders {
		date := time.Now()
		if v, e := parseDate(p.Tanggal); e == nil {
			date = v
		}
		encrypted, encryptErr := a.encryptStrings(p.NamaPerkaderan, defaultString(p.Tempat, "-"))
		if encryptErr != nil {
			writeEncryptionError(w)
			return
		}
		n, place := encrypted[0], encrypted[1]
		if _, err = tx.Exec(r.Context(), `INSERT INTO "Perkaderan"("id","anggotaId","namaPerkaderan","tanggal","tempat","createdAt","updatedAt") VALUES($1,$2,$3,$4,$5,NOW(),NOW())`, newID(), id, n, date, place); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menyimpan data perkaderan")
			return
		}
	}
	for _, p := range educations {
		school, encryptErr := a.crypto.EncryptText(defaultString(p.NamaSekolah, "-"))
		if encryptErr != nil {
			writeEncryptionError(w)
			return
		}
		if _, err = tx.Exec(r.Context(), `INSERT INTO "Pendidikan"("id","anggotaId","jenjang","namaSekolah","createdAt","updatedAt") VALUES($1,$2,$3,$4,NOW(),NOW())`, newID(), id, p.Jenjang, school); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menyimpan data pendidikan")
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menyimpan data anggota")
		return
	}
	committed = true
	if photo != oldPhoto && oldPhoto != "" {
		a.deleteAnggotaPhotoIfUnreferenced(r.Context(), oldPhoto)
	}
	action, message := "UPDATE", "Data anggota berhasil diperbarui!"
	if creating {
		action = "CREATE"
		message = "Data anggota berhasil disimpan!"
	}
	_ = a.createLog(r.Context(), r, auth.User, action, "ANGGOTA", action+" data anggota: "+name, id)
	a.notify(r.Context(), map[string]any{"type": "anggota.changed", "id": id, "userId": ownerID})
	writeJSON(w, map[bool]int{true: 201, false: 200}[creating], map[string]any{
		"success": message,
		"data":    map[string]any{"id": id},
	})
}

func parseAnggotaChildren(rawKaders, rawEducations string) ([]kaderInput, []pendidikanInput, error) {
	var kaders []kaderInput
	var educations []pendidikanInput
	if rawKaders != "" {
		if err := json.Unmarshal([]byte(rawKaders), &kaders); err != nil {
			return nil, nil, err
		}
	}
	if rawEducations != "" {
		if err := json.Unmarshal([]byte(rawEducations), &educations); err != nil {
			return nil, nil, err
		}
	}
	if len(kaders) > 20 || len(educations) > 20 {
		return nil, nil, fmt.Errorf("too many child records")
	}
	validCadres := map[string]bool{
		"Makesta": true, "Lakmud": true, "Lakut": true, "Diklatama": true,
		"Diklatmad": true, "Latin": true, "Latpel": true,
	}
	seenCadres := make(map[string]bool, len(kaders))
	for i := range kaders {
		kaders[i].NamaPerkaderan = strings.TrimSpace(kaders[i].NamaPerkaderan)
		kaders[i].Tanggal = strings.TrimSpace(kaders[i].Tanggal)
		kaders[i].Tempat = strings.TrimSpace(kaders[i].Tempat)
		if !validCadres[kaders[i].NamaPerkaderan] || seenCadres[kaders[i].NamaPerkaderan] || kaders[i].Tanggal == "" || kaders[i].Tempat == "" {
			return nil, nil, fmt.Errorf("invalid cadre record")
		}
		if _, err := parseDate(kaders[i].Tanggal); err != nil {
			return nil, nil, fmt.Errorf("invalid cadre date")
		}
		seenCadres[kaders[i].NamaPerkaderan] = true
	}
	validEducation := map[string]bool{
		"SD": true, "MI": true, "SMP": true, "MTs": true, "SMA": true,
		"SMK": true, "MAN": true, "KULIAH": true,
	}
	seenEducation := make(map[string]bool, len(educations))
	for i := range educations {
		educations[i].Jenjang = strings.TrimSpace(educations[i].Jenjang)
		educations[i].NamaSekolah = strings.TrimSpace(educations[i].NamaSekolah)
		if !validEducation[educations[i].Jenjang] || seenEducation[educations[i].Jenjang] || educations[i].NamaSekolah == "" {
			return nil, nil, fmt.Errorf("invalid education record")
		}
		seenEducation[educations[i].Jenjang] = true
	}
	return kaders, educations, nil
}

func (a *API) deleteAnggota(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	var owner, file, name string
	if err := a.db.QueryRow(r.Context(), `SELECT "userId",COALESCE("foto",''),"namaLengkap" FROM "Anggota" WHERE "id"=$1`, id).Scan(&owner, &file, &name); err != nil {
		writeLookupError(w, err, "NOT_FOUND", "Data anggota tidak ditemukan")
		return
	}
	if owner != auth.User.ID && auth.User.Role != "SEKRETARIS_CABANG" {
		writeError(w, 403, "FORBIDDEN", "Akses ditolak")
		return
	}
	_, err := a.db.Exec(r.Context(), `DELETE FROM "Anggota" WHERE "id"=$1`, id)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menghapus anggota")
		return
	}
	if file != "" {
		a.deleteAnggotaPhotoIfUnreferenced(r.Context(), file)
	}
	plain, _ := a.crypto.DecryptText(name)
	_ = a.createLog(r.Context(), r, auth.User, "DELETE", "ANGGOTA", "Menghapus data anggota: "+plain, id)
	a.notify(r.Context(), map[string]any{"type": "anggota.deleted", "id": id, "userId": owner})
	writeJSON(w, 200, map[string]any{"success": "Data anggota berhasil dihapus!"})
}

func (a *API) deleteAnggotaPhotoIfUnreferenced(ctx context.Context, key string) {
	if key == "" || a.r2 == nil {
		return
	}
	var referenced bool
	if err := a.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM "Anggota" WHERE "foto"=$1)`, key).Scan(&referenced); err != nil || referenced {
		return
	}
	_ = a.r2.Delete(ctx, key)
}

func (a *API) anggotaStats(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	targetID, err := a.targetPeriode(r.Context(), auth.User.ID, viewPeriode(r))
	if err != nil {
		if periodLookupFailed(w, err) {
			return
		}
		writeJSON(w, 200, nil)
		return
	}
	var periodName string
	if err := a.db.QueryRow(r.Context(), `SELECT "nama" FROM "Periode" WHERE "id"=$1`, targetID).Scan(&periodName); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca periode")
		return
	}
	selected := r.URL.Query().Get("userId")
	where := ""
	args := []any{}
	if auth.User.Role != "SEKRETARIS_CABANG" {
		where = `a."userId"=$1 AND a."periodeId"=$2`
		args = []any{auth.User.ID, targetID}
	} else if selected != "" && selected != "ALL" {
		where = `a."userId"=$1 AND p."nama"=$2`
		args = []any{selected, periodName}
	} else {
		where = `p."nama"=$1`
		args = []any{periodName}
	}
	query := `SELECT a."jenisKelamin"::text,k."namaPerkaderan" FROM "Anggota" a JOIN "Periode" p ON p."id"=a."periodeId" LEFT JOIN "Perkaderan" k ON k."anggotaId"=a."id" WHERE ` + where
	rows, err := a.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil statistik anggota")
		return
	}
	counts := map[string]int{"makesta": 0, "lakmud": 0, "latin": 0, "latpel": 0, "lakut": 0}
	for rows.Next() {
		var gender string
		var kader *string
		if err := rows.Scan(&gender, &kader); err != nil {
			rows.Close()
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca statistik anggota")
			return
		}
		if kader != nil {
			plain, err := a.crypto.DecryptText(*kader)
			if err != nil {
				rows.Close()
				writeError(w, 500, "DATA_UNREADABLE", "Data perkaderan tidak dapat dibaca")
				return
			}
			key := strings.ToLower(plain)
			if _, ok := counts[key]; ok {
				counts[key]++
			}
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca statistik anggota")
		return
	}
	var total, male, female int
	countQuery := `SELECT COUNT(*),COUNT(*) FILTER(WHERE a."jenisKelamin"='LAKI_LAKI'),COUNT(*) FILTER(WHERE a."jenisKelamin"='PEREMPUAN') FROM "Anggota" a JOIN "Periode" p ON p."id"=a."periodeId" WHERE ` + where
	if err := a.db.QueryRow(r.Context(), countQuery, args...).Scan(&total, &male, &female); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menghitung anggota")
		return
	}
	writeJSON(w, 200, map[string]int{"total": total, "lakiLaki": male, "perempuan": female, "makesta": counts["makesta"], "lakmud": counts["lakmud"], "latin": counts["latin"], "latpel": counts["latpel"], "lakut": counts["lakut"]})
}

func (a *API) activeUsers(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	rows, err := a.db.Query(r.Context(), `SELECT "id","name","role"::text FROM "User" WHERE "isActive"=true AND "emailVerified"=true ORDER BY "name"`)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil pengguna")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca pengguna")
		return
	}
	writeJSON(w, 200, map[string]any{"data": data})
}

func (a *API) copyAnggota(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if readJSON(r, &body) != nil || len(body.IDs) == 0 {
		writeError(w, 400, "INVALID_REQUEST", "Pilih anggota yang akan disalin")
		return
	}
	auth := authFrom(r)
	periodeID, err := a.targetPeriode(r.Context(), auth.User.ID, "")
	if err != nil {
		if periodLookupFailed(w, err) {
			return
		}
		writeError(w, 400, "NO_ACTIVE_PERIODE", "Silakan aktifkan periode tujuan terlebih dahulu.")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menyalin anggota")
		return
	}
	defer tx.Rollback(r.Context())
	count := 0
	for _, sourceID := range body.IDs {
		var sourcePeriod string
		var sourceRoot sql.NullString
		err := tx.QueryRow(r.Context(), `SELECT "periodeId","sourceAnggotaId" FROM "Anggota" WHERE "id"=$1 AND "userId"=$2`, sourceID, auth.User.ID).Scan(&sourcePeriod, &sourceRoot)
		if errors.Is(err, pgx.ErrNoRows) || sourcePeriod == periodeID {
			continue
		}
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca anggota asal")
			return
		}
		rootID := sourceID
		if sourceRoot.Valid && sourceRoot.String != "" {
			rootID = sourceRoot.String
		}
		newAnggotaID := newID()
		tag, err := tx.Exec(r.Context(), `INSERT INTO "Anggota"("id","userId","periodeId","sourceAnggotaId","nik","nia","email","foto","namaLengkap","tempatLahir","tanggalLahir","jenisKelamin","alamatLengkap","noHp","hobi","jabatan","noRfid","pekerjaan","jenjangPendidikan","namaInstansiPendidikan","createdAt","updatedAt") SELECT $1,$2,$3,$5,"nik","nia","email","foto","namaLengkap","tempatLahir","tanggalLahir","jenisKelamin","alamatLengkap","noHp","hobi","jabatan","noRfid","pekerjaan","jenjangPendidikan","namaInstansiPendidikan",NOW(),NOW() FROM "Anggota" WHERE "id"=$4 AND "userId"=$2 ON CONFLICT ("userId","periodeId","sourceAnggotaId") WHERE "sourceAnggotaId" IS NOT NULL DO NOTHING`, newAnggotaID, auth.User.ID, periodeID, sourceID, rootID)
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menyalin anggota")
			return
		}
		if tag.RowsAffected() == 0 {
			continue
		}
		if _, err := tx.Exec(r.Context(), `INSERT INTO "Perkaderan"("id","anggotaId","namaPerkaderan","tanggal","tempat","createdAt","updatedAt") SELECT 'c'||md5(random()::text||clock_timestamp()::text),$1,"namaPerkaderan","tanggal","tempat",NOW(),NOW() FROM "Perkaderan" WHERE "anggotaId"=$2`, newAnggotaID, sourceID); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menyalin perkaderan")
			return
		}
		if _, err := tx.Exec(r.Context(), `INSERT INTO "Pendidikan"("id","anggotaId","jenjang","namaSekolah","createdAt","updatedAt") SELECT 'c'||md5(random()::text||clock_timestamp()::text),$1,"jenjang","namaSekolah",NOW(),NOW() FROM "Pendidikan" WHERE "anggotaId"=$2`, newAnggotaID, sourceID); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal menyalin pendidikan")
			return
		}
		count++
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menyalin anggota")
		return
	}
	_ = a.createLog(r.Context(), r, auth.User, "CREATE", "ANGGOTA", "Menyalin "+itoa(count)+" anggota ke periode aktif", "")
	message := itoa(count) + " anggota berhasil disalin!"
	if count == 0 {
		message = "0 anggota disalin karena data sudah tersedia di periode aktif."
	}
	writeJSON(w, 200, map[string]any{"success": message, "copied": count, "skipped": len(body.IDs) - count})
}

func (a *API) anggotaImage(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	var key string
	if err := a.db.QueryRow(r.Context(), `SELECT COALESCE("foto",'') FROM "Anggota" WHERE "id"=$1 AND ("userId"=$2 OR $3='SEKRETARIS_CABANG')`, chi.URLParam(r, "id"), auth.User.ID, auth.User.Role).Scan(&key); err != nil {
		writeLookupError(w, err, "NOT_FOUND", "Gambar anggota tidak ditemukan")
		return
	}
	if key == "" {
		http.NotFound(w, r)
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
	w.Header().Set("Content-Type", mimeForExtension(security.OriginalExtension(key)))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = w.Write(plain)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"laci-backend/internal/config"
	"laci-backend/internal/domain"
	"laci-backend/internal/security"
)

func TestDataListsAgainstRestoredDatabase(t *testing.T) {
	if os.Getenv("LACI_RUN_DB_TESTS") != "1" {
		t.Skip("set LACI_RUN_DB_TESTS=1 to run read-only checks against the configured database")
	}
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	poolCfg.MaxConns = 1
	db, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	crypto, err := security.NewCrypto(cfg.EncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	var user domain.User
	var periodID string
	err = db.QueryRow(ctx, `SELECT u."id",u."role"::text,p."id" FROM "User" u JOIN "Periode" p ON p."userId"=u."id" WHERE u."isActive"=true AND p."isActive"=true ORDER BY u."id" LIMIT 1`).Scan(&user.ID, &user.Role, &periodID)
	if err != nil {
		t.Skip("no active user with an active period in the configured database")
	}
	a := New(cfg, db, crypto, nil)
	var admin domain.User
	var adminPeriodID string
	if err := db.QueryRow(ctx, `SELECT u."id",u."role"::text,p."id" FROM "User" u JOIN "Periode" p ON p."userId"=u."id" WHERE u."role"='SEKRETARIS_CABANG' AND u."isActive"=true AND p."isActive"=true LIMIT 1`).Scan(&admin.ID, &admin.Role, &adminPeriodID); err != nil {
		t.Skip("no active administrator with an active period in the configured database")
	}
	admin.IsActive = true
	assertRestoredTextReadable(t, ctx, db, crypto)
	t.Run("performance and integrity indexes", func(t *testing.T) {
		required := []string{
			"Periode_one_active_per_user_idx", "Periode_user_active_idx", "ArsipSurat_user_periode_tanggal_idx",
			"BerkasPimpinan_user_periode_tanggal_idx", "BerkasSP_user_periode_tanggal_idx", "Anggota_user_periode_created_idx",
			"AgendaKegiatan_user_periode_start_idx", "PengajuanBerkas_cabang_status_created_idx",
			"PengajuanBerkas_pac_status_created_idx", "Presensi_user_periode_tanggal_idx",
			"LogEmail_status_created_idx", "Session_expiresAt_idx",
		}
		var found int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM pg_indexes WHERE schemaname=current_schema() AND indexname=ANY($1)`, required).Scan(&found); err != nil {
			t.Fatal(err)
		}
		if found != len(required) {
			t.Fatalf("performance/integrity indexes present: %d of %d", found, len(required))
		}
	})
	t.Run("email identities are unambiguous", func(t *testing.T) {
		var duplicateGroups int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM (SELECT LOWER("email") FROM "User" GROUP BY LOWER("email") HAVING COUNT(*)>1) duplicates`).Scan(&duplicateGroups); err != nil {
			t.Fatal(err)
		}
		if duplicateGroups != 0 {
			t.Fatalf("ambiguous case-insensitive user emails: %d groups", duplicateGroups)
		}
	})
	tests := []struct {
		name, path string
		handler    http.HandlerFunc
	}{
		{"arsip metadata", "/api/arsip?jenis=KELUAR&organisasi=BERSAMA&sortKey=tanggal&sortDir=desc&page=1&limit=1", a.listArsip},
		{"arsip encrypted sort", "/api/arsip?sortKey=noSurat&sortDir=asc&page=1&limit=1", a.listArsip},
		{"berkas pimpinan", "/api/berkas-pimpinan?sortKey=tanggal&page=1&limit=1", a.listBerkasPimpinan},
		{"berkas sp", "/api/berkas-sp?sortKey=status&page=1&limit=1", a.listBerkasSP},
		{"pengajuan", "/api/pengajuan?sortKey=tanggal&status=PENDING&page=1&limit=1", a.listPengajuan},
		{"anggota creator sort", "/api/anggota?sortKey=dibuatOleh&page=1&limit=1", a.listAnggota},
		{"presensi", "/api/presensi?sortKey=tanggal&page=1&limit=1", a.listPresensi},
		{"agenda", "/api/agenda-kegiatan?sortKey=tanggalMulai&page=1&limit=1", a.listAgenda},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.path, nil)
			testUser := user
			if tt.name == "berkas sp" || tt.name == "agenda" {
				testUser = admin
			}
			r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: testUser}))
			w := httptest.NewRecorder()
			tt.handler(w, r)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			var payload struct {
				Data       []json.RawMessage `json:"data"`
				Total      int               `json:"total"`
				TotalPages int               `json:"totalPages"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if len(payload.Data) > 1 || payload.Total < len(payload.Data) || payload.TotalPages < 0 {
				t.Fatalf("invalid pagination metadata: items=%d total=%d pages=%d", len(payload.Data), payload.Total, payload.TotalPages)
			}
		})
	}
	t.Run("arsip filtered total and newest", func(t *testing.T) {
		var expectedTotal int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "ArsipSurat" WHERE "userId"=$1 AND "periodeId"=$2 AND "jenisSurat"='KELUAR' AND "organisasi"='BERSAMA'`, user.ID, periodID).Scan(&expectedTotal); err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(http.MethodGet, "/api/arsip?jenis=KELUAR&organisasi=BERSAMA&sortKey=tanggal&sortDir=desc&page=1&limit=1", nil)
		r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: user}))
		w := httptest.NewRecorder()
		a.listArsip(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		var payload struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
			Total int `json:"total"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Total != expectedTotal {
			t.Fatalf("filtered total = %d, want %d", payload.Total, expectedTotal)
		}
		if expectedTotal > 0 {
			var newestID string
			if err := db.QueryRow(ctx, `SELECT "id" FROM "ArsipSurat" WHERE "userId"=$1 AND "periodeId"=$2 AND "jenisSurat"='KELUAR' AND "organisasi"='BERSAMA' ORDER BY "tanggal" DESC,"id" DESC LIMIT 1`, user.ID, periodID).Scan(&newestID); err != nil {
				t.Fatal(err)
			}
			if len(payload.Data) != 1 || payload.Data[0].ID != newestID {
				t.Fatal("API did not return the newest matching archive")
			}
		}
	})
	t.Run("users filtered and paginated", func(t *testing.T) {
		var adminID string
		if err := db.QueryRow(ctx, `SELECT "id" FROM "User" WHERE "role"='SEKRETARIS_CABANG' AND "isActive"=true LIMIT 1`).Scan(&adminID); err != nil {
			t.Skip("no active administrator")
		}
		r := httptest.NewRequest(http.MethodGet, "/api/users?status=ACTIVE&emailStatus=VERIFIED&sortKey=name&sortDir=asc&page=1&limit=1", nil)
		r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: domain.User{ID: adminID, Role: "SEKRETARIS_CABANG"}}))
		w := httptest.NewRecorder()
		a.listUsers(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("users status = %d", w.Code)
		}
		var payload struct {
			Data  []json.RawMessage `json:"data"`
			Total int               `json:"total"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		var expected int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "User" WHERE "role"='SEKRETARIS_PAC' AND "isActive"=true AND "emailVerified"=true`).Scan(&expected); err != nil {
			t.Fatal(err)
		}
		if payload.Total != expected || len(payload.Data) > 1 {
			t.Fatalf("user pagination mismatch: total=%d expected=%d items=%d", payload.Total, expected, len(payload.Data))
		}
		if expected > 0 {
			var firstName string
			if err := db.QueryRow(ctx, `SELECT "name" FROM "User" WHERE "role"='SEKRETARIS_PAC' AND "isActive"=true AND "emailVerified"=true ORDER BY LOWER("name") ASC,"id" ASC LIMIT 1`).Scan(&firstName); err != nil {
				t.Fatal(err)
			}
			var first struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(payload.Data[0], &first); err != nil {
				t.Fatal(err)
			}
			if first.Name != firstName {
				t.Fatalf("first user = %q, want case-insensitive first %q", first.Name, firstName)
			}
		}
	})
	t.Run("anggota export includes complete relations", func(t *testing.T) {
		var adminID string
		if err := db.QueryRow(ctx, `SELECT u."id" FROM "User" u JOIN "Periode" p ON p."userId"=u."id" AND p."isActive"=true WHERE u."role"='SEKRETARIS_CABANG' AND u."isActive"=true LIMIT 1`).Scan(&adminID); err != nil {
			t.Skip("no active administrator")
		}
		r := httptest.NewRequest(http.MethodGet, "/api/anggota?includeRelations=true&page=1&limit=500", nil)
		r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: domain.User{ID: adminID, Role: "SEKRETARIS_CABANG"}}))
		w := httptest.NewRecorder()
		a.listAnggota(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("anggota export status = %d", w.Code)
		}
		var payload struct {
			Data []struct {
				ID          string            `json:"id"`
				Pendidikans []json.RawMessage `json:"pendidikans"`
				Perkaderans []json.RawMessage `json:"perkaderans"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.Data) == 0 {
			t.Skip("no anggota in active administrator period")
		}
		ids := make([]string, 0, len(payload.Data))
		var pendidikanCount, perkaderanCount int
		for _, item := range payload.Data {
			ids = append(ids, item.ID)
			pendidikanCount += len(item.Pendidikans)
			perkaderanCount += len(item.Perkaderans)
		}
		var expectedPendidikan, expectedPerkaderan int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "Pendidikan" WHERE "anggotaId"=ANY($1)`, ids).Scan(&expectedPendidikan); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "Perkaderan" WHERE "anggotaId"=ANY($1)`, ids).Scan(&expectedPerkaderan); err != nil {
			t.Fatal(err)
		}
		if pendidikanCount != expectedPendidikan || perkaderanCount != expectedPerkaderan {
			t.Fatalf("relation counts: pendidikan %d/%d, perkaderan %d/%d", pendidikanCount, expectedPendidikan, perkaderanCount, expectedPerkaderan)
		}
	})
	t.Run("email log rejects invalid filters", func(t *testing.T) {
		for _, path := range []string{
			"/api/email-logs?status=UNKNOWN",
			"/api/email-logs?type=UNKNOWN",
		} {
			r := httptest.NewRequest(http.MethodGet, path, nil)
			r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: admin}))
			w := httptest.NewRecorder()
			a.listEmailLogs(w, r)
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `"code":"INVALID_FILTER"`) {
				t.Fatalf("invalid filter %q = %d: %s", path, w.Code, w.Body.String())
			}
		}
	})
	t.Run("default lists put newest created records first", func(t *testing.T) {
		var adminID, adminPeriodID string
		if err := db.QueryRow(ctx, `SELECT u."id",p."id" FROM "User" u JOIN "Periode" p ON p."userId"=u."id" AND p."isActive"=true WHERE u."role"='SEKRETARIS_CABANG' AND u."isActive"=true LIMIT 1`).Scan(&adminID, &adminPeriodID); err != nil {
			t.Skip("no active administrator with an active period")
		}
		admin := domain.User{ID: adminID, Role: "SEKRETARIS_CABANG", IsActive: true}
		checks := []struct {
			name, path, expectedSQL string
			handler                 http.HandlerFunc
			args                    []any
		}{
			{"archive", "/api/arsip?page=1&limit=1", `SELECT "id" FROM "ArsipSurat" WHERE "userId"=$1 AND "periodeId"=$2 ORDER BY "createdAt" DESC,"id" DESC LIMIT 1`, a.listArsip, []any{adminID, adminPeriodID}},
			{"leadership documents", "/api/berkas-pimpinan?page=1&limit=1", `SELECT "id" FROM "BerkasPimpinan" WHERE "userId"=$1 AND "periodeId"=$2 ORDER BY "createdAt" DESC,"id" DESC LIMIT 1`, a.listBerkasPimpinan, []any{adminID, adminPeriodID}},
			{"SP documents", "/api/berkas-sp?page=1&limit=1", `SELECT "id" FROM "BerkasSP" WHERE "userId"=$1 AND "periodeId"=$2 ORDER BY "createdAt" DESC,"id" DESC LIMIT 1`, a.listBerkasSP, []any{adminID, adminPeriodID}},
			{"agenda", "/api/agenda-kegiatan?page=1&limit=1", `SELECT "id" FROM "AgendaKegiatan" WHERE "userId"=$1 AND "periodeId"=$2 ORDER BY "createdAt" DESC,"id" DESC LIMIT 1`, a.listAgenda, []any{adminID, adminPeriodID}},
			{"attendance", "/api/presensi?page=1&limit=1", `SELECT "id" FROM "Presensi" WHERE "userId"=$1 AND "periodeId"=$2 ORDER BY "createdAt" DESC,"id" DESC LIMIT 1`, a.listPresensi, []any{adminID, adminPeriodID}},
			{"submissions", "/api/pengajuan?page=1&limit=1", `SELECT "id" FROM "PengajuanBerkas" WHERE "periodeId"=$1 ORDER BY "createdAt" DESC,"id" DESC LIMIT 1`, a.listPengajuan, []any{adminPeriodID}},
			{"members", "/api/anggota?page=1&limit=1", `SELECT a."id" FROM "Anggota" a JOIN "Periode" p ON p."id"=a."periodeId" WHERE p."nama"=(SELECT "nama" FROM "Periode" WHERE "id"=$1) ORDER BY a."createdAt" DESC,a."id" DESC LIMIT 1`, a.listAnggota, []any{adminPeriodID}},
			{"users", "/api/users?page=1&limit=1", `SELECT "id" FROM "User" WHERE "role"='SEKRETARIS_PAC' ORDER BY "createdAt" DESC,"id" DESC LIMIT 1`, a.listUsers, nil},
			{"periods", "/api/periodes?page=1&limit=1", `SELECT "id" FROM "Periode" WHERE "userId"=$1 ORDER BY "createdAt" DESC,"id" DESC LIMIT 1`, a.listPeriodes, []any{adminID}},
			{"activity log", "/api/logs?scope=global&page=1&limit=1", `SELECT l."id" FROM "LogActivity" l JOIN "Periode" p ON p."id"=l."periodeId" WHERE p."isActive"=true ORDER BY l."createdAt" DESC,l."id" DESC LIMIT 1`, a.listLogs, nil},
			{"email log", "/api/email-logs?page=1&limit=1", `SELECT "id" FROM "LogEmail" ORDER BY "createdAt" DESC,"id" DESC LIMIT 1`, a.listEmailLogs, nil},
		}
		for _, check := range checks {
			t.Run(check.name, func(t *testing.T) {
				var expectedID string
				if err := db.QueryRow(ctx, check.expectedSQL, check.args...).Scan(&expectedID); err != nil {
					if err == pgx.ErrNoRows {
						t.Skip("no records in this list")
					}
					t.Fatal(err)
				}
				r := httptest.NewRequest(http.MethodGet, check.path, nil)
				r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: admin}))
				w := httptest.NewRecorder()
				check.handler(w, r)
				if w.Code != http.StatusOK {
					t.Fatalf("list status = %d: %s", w.Code, w.Body.String())
				}
				var payload struct {
					Data []struct {
						ID string `json:"id"`
					} `json:"data"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
					t.Fatal(err)
				}
				if len(payload.Data) == 0 || payload.Data[0].ID != expectedID {
					t.Fatalf("first ID = %v, want %s", payload.Data, expectedID)
				}
			})
		}
	})
	t.Run("multi query read endpoints with one database connection", func(t *testing.T) {
		var adminID string
		if err := db.QueryRow(ctx, `SELECT "id" FROM "User" WHERE "role"='SEKRETARIS_CABANG' AND "isActive"=true LIMIT 1`).Scan(&adminID); err != nil {
			t.Skip("no active administrator")
		}
		admin := domain.User{ID: adminID, Role: "SEKRETARIS_CABANG", IsActive: true}
		tests := []struct {
			name    string
			handler http.HandlerFunc
			path    string
			auth    domain.User
			id      string
		}{
			{"public stats", a.publicStats, "/api/public/stats", domain.User{}, ""},
			{"public readiness", a.publicReadiness, "/api/public/readiness", domain.User{}, ""},
			{"public data", a.publicData, "/api/public/data", domain.User{}, ""},
			{"dashboard", a.dashboardStats, "/api/dashboard/stats", admin, ""},
			{"user details", a.getUser, "/api/users/" + adminID, admin, adminID},
			{"user list", a.listUsers, "/api/users?page=1&limit=2", admin, ""},
			{"user stats", a.userStats, "/api/users/stats", admin, ""},
			{"period list", a.listPeriodes, "/api/periodes", user, ""},
			{"period detail", a.getPeriode, "/api/periodes/" + periodID, user, periodID},
			{"archive stats", a.arsipStats, "/api/arsip/stats", user, ""},
			{"leadership document stats", a.berkasPimpinanStats, "/api/berkas-pimpinan/stats", user, ""},
			{"SP document stats", a.berkasSPStats, "/api/berkas-sp/stats", admin, ""},
			{"anggota stats", a.anggotaStats, "/api/anggota/stats", admin, ""},
			{"active users", a.activeUsers, "/api/anggota/active-users", admin, ""},
			{"agenda stats", a.agendaStats, "/api/agenda-kegiatan/stats", admin, ""},
			{"pengajuan stats", a.pengajuanStats, "/api/pengajuan/stats", user, ""},
			{"active PAC users", a.activePACUsers, "/api/pengajuan/active-pac", admin, ""},
			{"activity logs", a.listLogs, "/api/logs?page=1&limit=2", admin, ""},
			{"activity stats", a.logStats, "/api/logs/stats", admin, ""},
			{"activity monitoring", a.logMonitoring, "/api/logs/monitoring", admin, ""},
			{"email logs", a.listEmailLogs, "/api/email-logs?page=1&limit=2", admin, ""},
			{"email stats", a.emailStats, "/api/email-logs/stats", admin, ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodGet, tt.path, nil)
				if tt.name == "public data" {
					r.Header.Set("X-API-Key", cfg.APIKey)
				}
				r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: tt.auth}))
				if tt.id != "" {
					routeCtx := chi.NewRouteContext()
					routeCtx.URLParams.Add("id", tt.id)
					r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
				}
				w := httptest.NewRecorder()
				tt.handler(w, r)
				if w.Code != http.StatusOK {
					t.Fatalf("%s status = %d", tt.name, w.Code)
				}
			})
		}
	})
	t.Run("pengajuan retry template reflects current data", func(t *testing.T) {
		var id, status string
		if err := db.QueryRow(ctx, `SELECT "id","status"::text FROM "PengajuanBerkas" ORDER BY "createdAt" DESC LIMIT 1`).Scan(&id, &status); err != nil {
			t.Skip("no pengajuan in configured database")
		}
		body, err := a.pengajuanRetryBody(ctx, "PENGAJUAN_STATUS", map[string]string{"id": id, "status": status})
		if err != nil || body == "" {
			t.Fatalf("current pengajuan status cannot render retry body: %v", err)
		}
		other := "DITERIMA"
		if status == other {
			other = "DITOLAK"
		}
		if _, err := a.pengajuanRetryBody(ctx, "PENGAJUAN_STATUS", map[string]string{"id": id, "status": other}); err == nil {
			t.Fatal("stale status email could be resent")
		}
	})
}

func assertRestoredTextReadable(t *testing.T, ctx context.Context, db *pgxpool.Pool, crypto *security.Crypto) {
	t.Helper()
	queries := []struct {
		name, sql string
	}{
		{"ArsipSurat", `SELECT "noSurat","pengirimPenerima","deskripsi","perihal" FROM "ArsipSurat"`},
		{"BerkasPimpinan", `SELECT "nama","catatan" FROM "BerkasPimpinan"`},
		{"BerkasSP", `SELECT "nama","catatan" FROM "BerkasSP"`},
		{"Anggota", `SELECT "namaLengkap","nik","nia","tempatLahir","alamatLengkap","noHp","hobi","jabatan","noRfid","pekerjaan","namaInstansiPendidikan" FROM "Anggota"`},
		{"Perkaderan", `SELECT "namaPerkaderan","tempat" FROM "Perkaderan"`},
		{"Pendidikan", `SELECT "namaSekolah" FROM "Pendidikan"`},
		{"AgendaKegiatan", `SELECT "judul","deskripsi","lokasi" FROM "AgendaKegiatan"`},
		{"PengajuanBerkas", `SELECT "noSurat","keperluan","deskripsi","alasanPenolakan" FROM "PengajuanBerkas"`},
	}
	for _, query := range queries {
		rows, err := db.Query(ctx, query.sql)
		if err != nil {
			t.Fatalf("read %s encrypted fields: %v", query.name, err)
		}
		unreadable := 0
		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				rows.Close()
				t.Fatalf("scan %s encrypted fields: %v", query.name, err)
			}
			for _, value := range values {
				if text, ok := value.(string); ok && text != "" {
					if _, err := crypto.DecryptText(text); err != nil {
						unreadable++
					}
				}
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			t.Fatalf("iterate %s encrypted fields: %v", query.name, err)
		}
		if unreadable > 0 {
			t.Errorf("%s has %d unreadable encrypted values", query.name, unreadable)
		}
	}
}

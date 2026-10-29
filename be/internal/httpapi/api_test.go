package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"laci-backend/internal/config"
	"laci-backend/internal/domain"
	"laci-backend/internal/security"
)

func TestReadJSONRequiresOneKnownObject(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{name: "valid", body: `{"name":"Laci"}`},
		{name: "unknown field", body: `{"name":"Laci","ignored":true}`, wantErr: true},
		{name: "second object", body: `{"name":"Laci"}{"name":"Other"}`, wantErr: true},
		{name: "trailing garbage", body: `{"name":"Laci"}garbage`, wantErr: true},
		{name: "oversized trailing input", body: `{"name":"Laci"}` + strings.Repeat(" ", 2<<20), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			var body struct {
				Name string `json:"name"`
			}
			err := readJSON(r, &body)
			if (err != nil) != tt.wantErr {
				t.Fatalf("readJSON() error = %v, want error %v", err, tt.wantErr)
			}
			if !tt.wantErr && body.Name != "Laci" {
				t.Fatalf("unexpected decoded name: %q", body.Name)
			}
		})
	}
}

func TestSortMapsIgnoresSurroundingWhitespace(t *testing.T) {
	items := []map[string]any{
		{"noSurat": "\u2060004/PAC/A"},
		{"noSurat": "002/PAC/A"},
		{"noSurat": " 001/PAC/A "},
	}
	sortMaps(items, "noSurat", "asc", "createdAt")
	for i, want := range []string{" 001/PAC/A ", "002/PAC/A", "\u2060004/PAC/A"} {
		if got := items[i]["noSurat"]; got != want {
			t.Fatalf("sorted item %d = %q, want %q", i, got, want)
		}
	}
}

func TestNormalizePeriodeName(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
		valid bool
	}{
		{name: "normalizes whitespace", input: "  Masa   Khidmat 2026-2028  ", want: "Masa Khidmat 2026-2028", valid: true},
		{name: "blank", input: " \t\n ", valid: false},
		{name: "one hundred unicode characters", input: strings.Repeat("é", 100), want: strings.Repeat("é", 100), valid: true},
		{name: "too long", input: strings.Repeat("a", 101), want: strings.Repeat("a", 101), valid: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, valid := normalizePeriodeName(test.input)
			if got != test.want || valid != test.valid {
				t.Fatalf("normalizePeriodeName(%q) = %q, %v; want %q, %v", test.input, got, valid, test.want, test.valid)
			}
		})
	}
}

func TestNormalizeAuditTimestampPreservesCalendarDates(t *testing.T) {
	stored := time.Date(2026, 10, 25, 21, 58, 26, 733000000, time.UTC)
	want := time.Date(2026, 10, 25, 14, 58, 26, 733000000, time.UTC)
	for _, name := range []string{"createdAt", "updatedAt"} {
		got, ok := normalizeAuditTimestamp(name, pgtype.TimestampOID, stored).(time.Time)
		if !ok || !got.Equal(want) {
			t.Fatalf("%s = %v, want %v", name, got, want)
		}
	}
	if got := normalizeAuditTimestamp("tanggal", pgtype.TimestampOID, stored); got != stored {
		t.Fatalf("calendar date changed: %v", got)
	}
	if got := normalizeAuditTimestamp("createdAt", pgtype.TimestamptzOID, stored); got != stored {
		t.Fatalf("timezone-aware timestamp changed: %v", got)
	}
}

func TestParseAnggotaChildrenRejectsMalformedInput(t *testing.T) {
	tests := []struct {
		name, kaders, educations   string
		wantKaders, wantEducations int
		wantErr                    bool
	}{
		{name: "empty fields"},
		{name: "valid arrays", kaders: `[{"namaPerkaderan":"Makesta","tanggal":"2026-10-28","tempat":"Magetan"}]`, educations: `[{"jenjang":"SD","namaSekolah":"SD QA"}]`, wantKaders: 1, wantEducations: 1},
		{name: "incomplete cadre", kaders: `[{"namaPerkaderan":"Makesta"}]`, wantErr: true},
		{name: "incomplete education", educations: `[{"jenjang":"SD"}]`, wantErr: true},
		{name: "malformed cadre JSON", kaders: `[`, wantErr: true},
		{name: "malformed education JSON", educations: `{`, wantErr: true},
		{name: "wrong shape", kaders: `{"namaPerkaderan":"Makesta"}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kaders, educations, err := parseAnggotaChildren(tt.kaders, tt.educations)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, want error %t", err, tt.wantErr)
			}
			if !tt.wantErr && (len(kaders) != tt.wantKaders || len(educations) != tt.wantEducations) {
				t.Fatalf("child counts = %d, %d", len(kaders), len(educations))
			}
		})
	}
}

func TestClientIPDoesNotTrustPrivatePeerForwarding(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.7:12345"
	r.Header.Set("X-Forwarded-For", "203.0.113.40")
	if got := clientIP(r); got != "10.0.0.7" {
		t.Fatalf("private peer spoofed client IP: %q", got)
	}
	r.RemoteAddr = "127.0.0.1:12345"
	if got := clientIP(r); got != "203.0.113.40" {
		t.Fatalf("loopback proxy client IP = %q", got)
	}
}

func TestRequestLocationDecodesCookieValue(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "user_address", Value: "Magetan%2C%20Magetan%2C%20Jawa%20Timur"})
	if got := requestLocation(r); got != "Magetan, Magetan, Jawa Timur" {
		t.Fatalf("requestLocation() = %q", got)
	}
}

func TestGoogleStateIsBoundToInitiatingBrowser(t *testing.T) {
	a := New(config.Config{FrontendURL: "http://localhost:3000", GoogleClientID: "test-client", GoogleRedirectURL: "http://localhost:3000/api/auth/callback/google", SessionSecret: "test-secret"}, nil, nil, nil)
	start := httptest.NewRecorder()
	a.googleStart(start, httptest.NewRequest(http.MethodGet, "/api/auth/google?callback=http%3A%2F%2Flocalhost%3A3000%2Fdashboard", nil))
	if start.Code != http.StatusTemporaryRedirect {
		t.Fatalf("google start status = %d", start.Code)
	}
	redirect, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state := redirect.Query().Get("state")
	cookies := start.Result().Cookies()
	if state == "" || len(cookies) != 1 || cookies[0].Name != googleStateCookie || cookies[0].Path != "/api/auth" || !cookies[0].HttpOnly || cookies[0].MaxAge != 600 {
		t.Fatalf("missing or invalid Google browser state: state=%t cookies=%v", state != "", cookies)
	}
	callback := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?state="+url.QueryEscape(state), nil)
	callback.AddCookie(cookies[0])
	if got, ok := a.validGoogleState(callback); !ok || got != "http://localhost:3000/dashboard" {
		t.Fatalf("valid browser state rejected: callback=%q valid=%t", got, ok)
	}
	foreign := httptest.NewRequest(http.MethodGet, callback.URL.String(), nil)
	if _, ok := a.validGoogleState(foreign); ok {
		t.Fatal("Google state accepted without browser cookie")
	}
	foreign.AddCookie(&http.Cookie{Name: googleStateCookie, Value: strings.Repeat("0", 64)})
	if _, ok := a.validGoogleState(foreign); ok {
		t.Fatal("Google state accepted with a different browser cookie")
	}
	response := httptest.NewRecorder()
	a.googleCallback(response, foreign)
	if response.Code != http.StatusTemporaryRedirect || !strings.Contains(response.Header().Get("Location"), "error=oauth_state") {
		t.Fatalf("foreign callback status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
	cleared := response.Result().Cookies()
	if len(cleared) != 1 || cleared[0].Name != googleStateCookie || cleared[0].Path != "/api/auth" || cleared[0].MaxAge != -1 {
		t.Fatalf("OAuth state cookie was not cleared: %v", cleared)
	}
}

func TestParsePageBoundsLargeValues(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?page=2000000000&limit=999999", nil)
	page, limit := parsePage(r)
	if page != 1_000_000 || limit != 500 {
		t.Fatalf("page=%d limit=%d", page, limit)
	}
}

func TestParseLogDateBoundaryUsesJakartaCalendarDay(t *testing.T) {
	start, err := parseLogDateBoundary("2026-10-25", false)
	if err != nil {
		t.Fatal(err)
	}
	end, err := parseLogDateBoundary("2026-10-25", true)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 10, 24, 17, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 10, 25, 16, 59, 59, int(time.Second-time.Nanosecond), time.UTC)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) {
		t.Fatalf("date boundaries = %s through %s, want %s through %s", start, end, wantStart, wantEnd)
	}
	if _, err := parseLogDateBoundary("2026-02-30", false); err == nil {
		t.Fatal("invalid calendar date was accepted")
	}
}

func TestUploadFailureDoesNotExposeStorageError(t *testing.T) {
	w := httptest.NewRecorder()
	writeUploadFailure(w, &storageUploadError{errors.New("bucket secret and endpoint details")})
	if w.Code != http.StatusBadGateway || strings.Contains(w.Body.String(), "bucket secret") {
		t.Fatalf("unsafe upload error response: %d %s", w.Code, w.Body.String())
	}
}

func TestDownloadTokensAreScopedToDocumentKind(t *testing.T) {
	crypt, err := security.NewCrypto("qa-download-token-key")
	if err != nil {
		t.Fatal(err)
	}
	a := New(config.Config{}, nil, crypt, nil)
	scoped, err := crypt.DownloadToken("arsip:same-id", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !a.validDownloadToken(scoped, "arsip", "same-id") || a.validDownloadToken(scoped, "berkas-sp", "same-id") || a.validDownloadToken(scoped, "arsip", "other-id") {
		t.Fatal("new download token is not restricted to the document kind and ID")
	}
	legacy, err := crypt.DownloadToken("same-id", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !a.validDownloadToken(legacy, "arsip", "same-id") {
		t.Fatal("unexpired legacy download token stopped working")
	}
}

func TestRealtimeStreamStopsOnShutdown(t *testing.T) {
	a := New(config.Config{}, nil, nil, nil)
	shutdown := make(chan struct{})
	a.SetStreamShutdown(shutdown)
	done := make(chan struct{})
	go func() {
		defer close(done)
		a.realtime(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/realtime", nil))
	}()
	close(shutdown)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("realtime stream did not stop during shutdown")
	}
	a.hub.mu.RLock()
	defer a.hub.mu.RUnlock()
	if len(a.hub.clients) != 0 {
		t.Fatal("realtime subscriber leaked after shutdown")
	}
}

func TestLogoutDoesNotClaimSuccessWhenDatabaseUnavailable(t *testing.T) {
	poolCfg, err := pgxpool.ParseConfig("postgres://localhost/laci_qa")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	a := New(config.Config{SessionCookie: "laci_session"}, pool, nil, nil)
	r := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	r.AddCookie(&http.Cookie{Name: "laci_session", Value: "g1_unexpired"})
	w := httptest.NewRecorder()
	a.logout(w, r)
	if w.Code != http.StatusInternalServerError || len(w.Result().Cookies()) != 0 {
		t.Fatalf("logout with unavailable database returned status=%d cookies=%v", w.Code, w.Result().Cookies())
	}
}

func TestAuthenticationReportsDatabaseOutage(t *testing.T) {
	poolCfg, err := pgxpool.ParseConfig("postgres://localhost/laci_qa")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	a := New(config.Config{SessionCookie: "laci_session"}, pool, nil, nil)
	r := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	r.AddCookie(&http.Cookie{Name: "laci_session", Value: "g1_unexpired"})
	w := httptest.NewRecorder()
	a.Router().ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "DATABASE_UNAVAILABLE") {
		t.Fatalf("authenticated request with unavailable database = %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	a.session(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("session lookup with unavailable database = %d: %s", w.Code, w.Body.String())
	}
	r = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"person@example.test","password":"password"}`))
	w = httptest.NewRecorder()
	a.login(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("login with unavailable database = %d: %s", w.Code, w.Body.String())
	}
}

func TestListDoesNotReportEmptyDataDuringDatabaseOutage(t *testing.T) {
	poolCfg, err := pgxpool.ParseConfig("postgres://localhost/laci_qa")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	a := New(config.Config{}, pool, nil, nil)
	r := httptest.NewRequest(http.MethodGet, "/api/arsip", nil)
	r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: domain.User{ID: "qa"}}))
	w := httptest.NewRecorder()
	a.listArsip(w, r)
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "DATABASE_UNAVAILABLE") {
		t.Fatalf("archive list with unavailable database = %d: %s", w.Code, w.Body.String())
	}
	r = httptest.NewRequest(http.MethodGet, "/api/arsip?periodeId=chosen", nil)
	r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: domain.User{ID: "qa"}}))
	w = httptest.NewRecorder()
	a.listArsip(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("archive list with selected period and unavailable database = %d: %s", w.Code, w.Body.String())
	}
}

func TestLookupFailuresKeepNotFoundSeparateFromDatabaseOutage(t *testing.T) {
	for _, tt := range []struct {
		err  error
		want int
	}{
		{pgx.ErrNoRows, http.StatusNotFound},
		{errors.New("database offline"), http.StatusServiceUnavailable},
	} {
		w := httptest.NewRecorder()
		writeLookupError(w, tt.err, "NOT_FOUND", "Data tidak ditemukan")
		if w.Code != tt.want {
			t.Fatalf("lookup error %v returned %d, want %d", tt.err, w.Code, tt.want)
		}
	}
	poolCfg, err := pgxpool.ParseConfig("postgres://localhost/laci_qa")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	crypt, err := security.NewCrypto("qa-download-test-key")
	if err != nil {
		t.Fatal(err)
	}
	a := New(config.Config{SessionCookie: "laci_session"}, pool, crypt, nil)
	downloadToken, err := crypt.DownloadToken("arsip:item", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, method, path, role string
		handler                  http.HandlerFunc
	}{
		{"period", http.MethodGet, "/api/periodes/item", "SEKRETARIS_PAC", a.getPeriode},
		{"member delete", http.MethodDelete, "/api/anggota/item", "SEKRETARIS_PAC", a.deleteAnggota},
		{"member image", http.MethodGet, "/api/anggota/item/image", "SEKRETARIS_PAC", a.anggotaImage},
		{"submission delete", http.MethodDelete, "/api/pengajuan/item", "SEKRETARIS_PAC", a.deletePengajuan},
		{"attendance delete", http.MethodDelete, "/api/presensi/item", "SEKRETARIS_PAC", a.deletePresensi},
		{"reference submission stats", http.MethodGet, "/api/pengajuan/reference/stats", "SEKRETARIS_PAC", a.referencePengajuanStats},
		{"email retry", http.MethodPost, "/api/email-logs/item/retry", "SEKRETARIS_CABANG", a.retryEmail},
		{"download", http.MethodGet, "/api/downloads/arsip/item", "SEKRETARIS_PAC", a.download},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.name == "download" {
				r.URL.RawQuery = "token=" + url.QueryEscape(downloadToken)
			}
			r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: domain.User{ID: "qa", Role: tt.role, IsActive: true}}))
			route := chi.NewRouteContext()
			route.URLParams.Add("id", "item")
			route.URLParams.Add("kind", "arsip")
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, route))
			w := httptest.NewRecorder()
			tt.handler(w, r)
			if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "DATABASE_UNAVAILABLE") {
				t.Fatalf("database outage returned %d: %s", w.Code, w.Body.String())
			}
		})
	}
	r := httptest.NewRequest(http.MethodPatch, "/api/presensi/item/status", strings.NewReader(`{"mode":"AUTO"}`))
	r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: domain.User{ID: "qa", Role: "SEKRETARIS_PAC", IsActive: true}}))
	route := chi.NewRouteContext()
	route.URLParams.Add("id", "item")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, route))
	w := httptest.NewRecorder()
	a.updatePresensiStatus(w, r)
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "DATABASE_UNAVAILABLE") {
		t.Fatalf("attendance status database outage returned %d: %s", w.Code, w.Body.String())
	}
}

func TestCabangOnlyReadEndpointsRejectPACBeforeDatabaseAccess(t *testing.T) {
	a := &API{}
	handlers := []struct {
		name    string
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{"list SP", http.MethodGet, "/api/berkas-sp", a.listBerkasSP},
		{"SP stats", http.MethodGet, "/api/berkas-sp/stats", a.berkasSPStats},
		{"get SP", http.MethodGet, "/api/berkas-sp/item", a.getBerkasSP},
		{"delete SP", http.MethodDelete, "/api/berkas-sp/item", a.deleteBerkasSP},
		{"SP download token", http.MethodPost, "/api/berkas-sp/item/download-token", a.berkasSPDownloadToken},
		{"list agenda", http.MethodGet, "/api/agenda-kegiatan", a.listAgenda},
		{"agenda stats", http.MethodGet, "/api/agenda-kegiatan/stats", a.agendaStats},
		{"get agenda", http.MethodGet, "/api/agenda-kegiatan/item", a.getAgenda},
	}

	for _, tt := range handlers {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.path, nil)
			r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: domain.User{ID: "pac", Role: "SEKRETARIS_PAC", IsActive: true}}))
			route := chi.NewRouteContext()
			route.URLParams.Add("id", "item")
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, route))
			w := httptest.NewRecorder()

			tt.handler(w, r)

			if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), `"code":"FORBIDDEN"`) {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestClientLogRejectsUnknownModuleBeforeDatabase(t *testing.T) {
	a := New(config.Config{}, nil, nil, nil)
	r := httptest.NewRequest(http.MethodPost, "/api/logs", strings.NewReader(`{"action":"EXPORT","module":"UNKNOWN","description":"QA"}`))
	r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: domain.User{ID: "qa"}}))
	w := httptest.NewRecorder()
	a.createClientLog(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown log module = %d: %s", w.Code, w.Body.String())
	}
}

func TestIsUniqueViolation(t *testing.T) {
	unique := &pgconn.PgError{Code: "23505"}
	if !isUniqueViolation(unique) || !isUniqueViolation(fmt.Errorf("wrapped: %w", unique)) {
		t.Fatal("unique constraint error should be recognized")
	}
	if isUniqueViolation(&pgconn.PgError{Code: "23503"}) || isUniqueViolation(errors.New("SQLSTATE 23505")) || isUniqueViolation(nil) {
		t.Fatal("non-unique errors should not be recognized")
	}
}

func TestArsipSortColumn(t *testing.T) {
	tests := []struct {
		key, column string
		valid       bool
	}{
		{"", `a."createdAt"`, true},
		{"createdAt", `a."createdAt"`, true},
		{"tanggal", `a."tanggal"`, true},
		{"organisasi", `a."organisasi"::text`, true},
		{"jenisSurat", `a."jenisSurat"::text`, true},
		{"noSurat", "", false},
		{`tanggal; DROP TABLE "ArsipSurat"`, "", false},
	}
	for _, tt := range tests {
		column, valid := arsipSortColumn(tt.key)
		if column != tt.column || valid != tt.valid {
			t.Errorf("arsipSortColumn(%q) = (%q, %t), want (%q, %t)", tt.key, column, valid, tt.column, tt.valid)
		}
	}
}

func TestPengajuanSortColumn(t *testing.T) {
	tests := []struct {
		key, column string
		valid       bool
	}{
		{"", `b."createdAt"`, true},
		{"tanggal", `b."tanggal"`, true},
		{"status", `b."status"::text`, true},
		{"penerima", `b."penerima"::text`, true},
		{"keperluan", "", false},
		{`tanggal; DELETE FROM "PengajuanBerkas"`, "", false},
	}
	for _, tt := range tests {
		column, valid := pengajuanSortColumn(tt.key)
		if column != tt.column || valid != tt.valid {
			t.Errorf("pengajuanSortColumn(%q) = (%q, %t), want (%q, %t)", tt.key, column, valid, tt.column, tt.valid)
		}
	}
}

func TestSimpleDocumentSortColumn(t *testing.T) {
	tests := []struct {
		table, key, column string
		valid              bool
	}{
		{"BerkasPimpinan", "", `b."createdAt"`, true},
		{"BerkasPimpinan", "tanggal", `b."tanggal"`, true},
		{"BerkasPimpinan", "nama", "", false},
		{"BerkasSP", "", `b."createdAt"`, true},
		{"BerkasSP", "tanggalMulai", `b."tanggalMulai"`, true},
		{"BerkasSP", "status", `b."tanggalBerakhir"`, true},
		{"BerkasSP", "organisasi", `b."organisasi"::text`, true},
		{"BerkasSP", `nama; DROP TABLE "BerkasSP"`, "", false},
	}
	for _, tt := range tests {
		column, valid := simpleDocumentSortColumn(tt.table, tt.key)
		if column != tt.column || valid != tt.valid {
			t.Errorf("simpleDocumentSortColumn(%q, %q) = (%q, %t), want (%q, %t)", tt.table, tt.key, column, valid, tt.column, tt.valid)
		}
	}
}

func TestPaginateMapsLargePage(t *testing.T) {
	data := []map[string]any{{"id": "a"}, {"id": "b"}, {"id": "c"}}
	result := paginateMaps(data, math.MaxInt, 500)
	if result["total"] != 3 || result["totalPages"] != 1 || len(result["data"].([]map[string]any)) != 0 {
		t.Fatalf("large page must be empty without overflowing: %#v", result)
	}
	result = paginateMaps(data, 2, 2)
	if result["total"] != 3 || result["totalPages"] != 2 || len(result["data"].([]map[string]any)) != 1 {
		t.Fatalf("second page must contain remaining row: %#v", result)
	}
}

func TestDataOpenAPISpecDoesNotInventMethods(t *testing.T) {
	a := &API{cfg: config.Config{FrontendURL: "http://localhost:3000", SessionCookie: "laci_session"}}
	spec := a.dataOpenAPISpec()
	paths := spec["paths"].(map[string]any)
	arsip := paths["/api/arsip"].(map[string]any)
	if _, ok := arsip["get"]; !ok {
		t.Fatal("arsip list GET must be documented")
	}
	if _, ok := arsip["post"]; ok {
		t.Fatal("undocumented mutation must not be advertised")
	}
	if _, ok := paths["/api/auth/login"]; ok {
		t.Fatal("login must not be incorrectly documented as GET")
	}
}

func TestDecryptFieldStrictRejectsUnreadableData(t *testing.T) {
	crypto, err := security.NewCrypto("unit-test-key")
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := crypto.EncryptText("nomor surat")
	if err != nil {
		t.Fatal(err)
	}
	item := map[string]any{"noSurat": cipher}
	if err := decryptFieldStrict(crypto, item, "noSurat"); err != nil || item["noSurat"] != "nomor surat" {
		t.Fatalf("valid encrypted field must be decrypted: %v", err)
	}
	item["noSurat"] = "not encrypted"
	if err := decryptFieldStrict(crypto, item, "noSurat"); err == nil {
		t.Fatal("invalid encrypted field must be rejected")
	}
}

func TestRealtimeAnonymousConnectionOnlySeesPublicEvents(t *testing.T) {
	a := New(config.Config{SessionCookie: "laci_session"}, nil, nil, nil)
	w := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a.realtime(w, httptest.NewRequest(http.MethodGet, "/api/realtime", nil).WithContext(ctx))
	if got := w.Result().Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("anonymous realtime content type = %q", got)
	}
	if _, ok := visibleRealtimeEvent(`{"type":"arsip.created","id":"secret"}`, domain.User{}); ok {
		t.Fatal("anonymous user saw a private archive event")
	}
	public, ok := visibleRealtimeEvent(`{"type":"log","module":"PRESENSI","entityId":"public-id","_origin":"internal"}`, domain.User{})
	if !ok || !strings.Contains(public, "public-id") || strings.Contains(public, "_origin") {
		t.Fatal("public presensi event was not safely delivered")
	}
	public, ok = visibleRealtimeEvent(`{"type":"log","module":"PRESENSI","entityId":"public-id","userId":"private-user","description":"private note","email":"private@example.test"}`, domain.User{})
	if !ok || !strings.Contains(public, "public-id") || strings.Contains(public, "private") {
		t.Fatalf("anonymous presensi event disclosed private fields: %s", public)
	}
	other := domain.User{ID: "pac-1", Role: "SEKRETARIS_PAC"}
	filtered, ok := visibleRealtimeEvent(`{"type":"mutation","model":"Anggota","id":"foreign-record","userId":"pac-2"}`, other)
	if !ok || strings.Contains(filtered, "foreign-record") || !strings.Contains(filtered, "Anggota") {
		t.Fatal("foreign record ID was exposed or invalidation was lost")
	}
}

func TestPublicDataRejectsMissingAPIKeyWithoutCaching(t *testing.T) {
	a := New(config.Config{APIKey: "qa-test-key"}, nil, nil, nil)
	w := httptest.NewRecorder()
	a.publicData(w, httptest.NewRequest(http.MethodGet, "/api/public/data", nil))
	if w.Code != http.StatusUnauthorized || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("public data without key = %d, cache=%q", w.Code, w.Header().Get("Cache-Control"))
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPublicUpstreamEndpointsValidateAndMapResponses(t *testing.T) {
	previous := outboundClient
	outboundClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := ""
		switch r.URL.Host {
		case "www.google.com":
			payload, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			form, err := url.ParseQuery(string(payload))
			if err != nil || form.Get("secret") != "qa-recaptcha-secret" {
				return nil, errors.New("unexpected recaptcha request")
			}
			body = `{"success":true,"score":0.9}`
			if form.Get("response") == "low" {
				body = `{"success":true,"score":0.2}`
			}
		case "api-hari-libur.vercel.app":
			if r.URL.Query().Get("year") != "2026" {
				return nil, errors.New("unexpected holiday year")
			}
			body = `{"data":[{"date":"2026-01-01","name":"QA Holiday"}]}`
		default:
			return nil, errors.New("unexpected upstream host")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	t.Cleanup(func() { outboundClient = previous })
	a := New(config.Config{AppEnv: "production", RecaptchaSecret: "qa-recaptcha-secret"}, nil, nil, nil)
	for _, check := range []struct {
		token string
		valid bool
	}{{"good", true}, {"low", false}} {
		w := httptest.NewRecorder()
		a.verifyRecaptcha(w, httptest.NewRequest(http.MethodPost, "/api/recaptcha/verify", strings.NewReader(`{"token":"`+check.token+`"}`)))
		if w.Code != http.StatusOK || strings.Contains(w.Body.String(), `"valid":true`) != check.valid {
			t.Fatalf("recaptcha token %q = %d: %s", check.token, w.Code, w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	a.publicPHBI(w, httptest.NewRequest(http.MethodGet, "/api/public/phbi?year=2026", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "QA Holiday") {
		t.Fatalf("public holidays = %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	a.publicPHBI(w, httptest.NewRequest(http.MethodGet, "/api/public/phbi?year=2026%20OR%201%3D1", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid holiday year = %d: %s", w.Code, w.Body.String())
	}
}

func TestClientLogRejectsOversizedDescriptionBeforeDatabaseWrite(t *testing.T) {
	a := New(config.Config{}, nil, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/logs", strings.NewReader(`{"action":"EXPORT","module":"ARSIP_SURAT","description":"`+strings.Repeat("x", 2001)+`"}`))
	w := httptest.NewRecorder()
	a.createClientLog(w, request)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("oversized client log = %d: %s", w.Code, w.Body.String())
	}
}

func TestNotifyIsNonBlockingAndMapsLegacyEvents(t *testing.T) {
	a := New(config.Config{}, nil, nil, nil)
	ch := make(chan string, 1)
	a.hub.add(ch)
	defer a.hub.remove(ch)
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.notify(context.Background(), map[string]any{"type": "arsip.created", "id": "record"})
		}()
	}
	wg.Wait()
	if time.Since(start) > 3*time.Second {
		t.Fatal("slow subscriber blocked realtime publishers")
	}
	if len(a.notifyQueue) != 200 {
		t.Fatalf("queued notifications = %d, want 200", len(a.notifyQueue))
	}
	if got := mutationModel(map[string]any{"type": "arsip.created"}); got != "ArsipSurat" {
		t.Fatalf("legacy archive event mapped to %q", got)
	}
	if got := mutationModel(map[string]any{"type": "log", "module": "PRESENSI"}); got != "Presensi" {
		t.Fatalf("presensi event mapped to %q", got)
	}
}

func TestRateLimitScopeCannotBeBypassedWithDifferentIDs(t *testing.T) {
	a := New(config.Config{}, nil, nil, nil)
	handler := a.rateLimit(1, time.Minute, "presensi-absensi")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/api/presensi/id-1/absensi", nil))
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/api/presensi/id-2/absensi", nil))
	if first.Code != http.StatusOK || second.Code != http.StatusTooManyRequests {
		t.Fatalf("statuses = %d, %d", first.Code, second.Code)
	}
}

func TestRandomDigits(t *testing.T) {
	for i := 0; i < 100; i++ {
		otp, err := randomDigits(6)
		if err != nil {
			t.Fatal(err)
		}
		if len(otp) != 6 {
			t.Fatalf("OTP length = %d", len(otp))
		}
		for _, digit := range otp {
			if digit < '0' || digit > '9' {
				t.Fatal("OTP contains a non-digit")
			}
		}
	}
}

func TestPrivateRoutesRejectAnonymousRequests(t *testing.T) {
	a := New(config.Config{SessionCookie: "laci_session", FrontendURL: "http://localhost:3000"}, nil, nil, nil)
	routes := `
GET /api/me
PATCH /api/me
GET /api/periodes
POST /api/periodes
GET /api/periodes/id
PATCH /api/periodes/id
POST /api/periodes/id/activate
DELETE /api/periodes/id
GET /api/arsip
POST /api/arsip
GET /api/arsip/stats
GET /api/arsip/id
PATCH /api/arsip/id
DELETE /api/arsip/id
POST /api/arsip/id/download-token
GET /api/berkas-pimpinan
POST /api/berkas-pimpinan
GET /api/berkas-pimpinan/stats
GET /api/berkas-pimpinan/id
PATCH /api/berkas-pimpinan/id
DELETE /api/berkas-pimpinan/id
POST /api/berkas-pimpinan/id/download-token
GET /api/berkas-sp
POST /api/berkas-sp
GET /api/berkas-sp/stats
GET /api/berkas-sp/id
PATCH /api/berkas-sp/id
DELETE /api/berkas-sp/id
POST /api/berkas-sp/id/download-token
GET /api/anggota
POST /api/anggota
GET /api/anggota/stats
GET /api/anggota/active-users
POST /api/anggota/copy
GET /api/anggota/id
PATCH /api/anggota/id
DELETE /api/anggota/id
GET /api/anggota/id/image
GET /api/agenda-kegiatan
POST /api/agenda-kegiatan
GET /api/agenda-kegiatan/stats
GET /api/agenda-kegiatan/id
PATCH /api/agenda-kegiatan/id
DELETE /api/agenda-kegiatan/id
GET /api/presensi
POST /api/presensi
GET /api/presensi/id
PATCH /api/presensi/id
PATCH /api/presensi/id/status
DELETE /api/presensi/id
GET /api/pengajuan
POST /api/pengajuan
GET /api/pengajuan/stats
GET /api/pengajuan/active-pac
GET /api/pengajuan/reference
GET /api/pengajuan/reference/stats
GET /api/pengajuan/reference/id
GET /api/pengajuan/id
PATCH /api/pengajuan/id
PATCH /api/pengajuan/id/status
DELETE /api/pengajuan/id
POST /api/pengajuan/id/download-token
GET /api/dashboard/stats
GET /api/logs
GET /api/logs/stats
GET /api/logs/monitoring
GET /api/logs/id
POST /api/logs
GET /api/users
GET /api/users/stats
GET /api/users/id
PATCH /api/users/id/status
POST /api/users/id/reset-password
DELETE /api/users/id
GET /api/email-logs
GET /api/email-logs/stats
POST /api/email-logs/id/retry
GET /api/backups
POST /api/backups
GET /api/backups/download
GET /api/backups/file
DELETE /api/backups`
	for _, line := range strings.Split(strings.TrimSpace(routes), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			t.Fatalf("bad route test case: %q", line)
		}
		w := httptest.NewRecorder()
		a.Router().ServeHTTP(w, httptest.NewRequest(parts[0], parts[1], nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s: status %d, want 401", line, w.Code)
		}
	}
}

func TestRemovedImportRoutesRejectPost(t *testing.T) {
	a := New(config.Config{SessionCookie: "laci_session", FrontendURL: "http://localhost:3000"}, nil, nil, nil)
	for _, path := range []string{
		"/api/arsip/import",
		"/api/berkas-pimpinan/import",
		"/api/berkas-sp/import",
	} {
		w := httptest.NewRecorder()
		a.Router().ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s: status %d, want 405", path, w.Code)
		}
	}
}

func TestClientImportLogIsRejected(t *testing.T) {
	a := New(config.Config{}, nil, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/logs", strings.NewReader(`{"action":"IMPORT","module":"ARSIP_SURAT","description":"legacy"}`))
	r.Header.Set("Content-Type", "application/json")
	a.createClientLog(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("client import log status = %d, want 400", w.Code)
	}
}

func BenchmarkRealtimePublishWithSlowSubscribers(b *testing.B) {
	hub := newEventHub()
	for i := 0; i < 100; i++ {
		hub.add(make(chan string, 1))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.publish(`{"type":"mutation","model":"ArsipSurat"}`)
	}
}

func TestValidPresensiTimeRange(t *testing.T) {
	tests := []struct {
		name  string
		start string
		end   string
		want  bool
	}{
		{name: "valid", start: "08:00", end: "12:00", want: true},
		{name: "same time", start: "08:00", end: "08:00", want: false},
		{name: "end before start", start: "12:00", end: "08:00", want: false},
		{name: "invalid start", start: "25:00", end: "26:00", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validPresensiTimeRange(tt.start, tt.end); got != tt.want {
				t.Fatalf("validPresensiTimeRange(%q, %q) = %v, want %v", tt.start, tt.end, got, tt.want)
			}
		})
	}
}

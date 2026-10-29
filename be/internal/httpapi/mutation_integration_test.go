package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"

	"laci-backend/internal/config"
	"laci-backend/internal/domain"
	"laci-backend/internal/security"
	"laci-backend/internal/storage"
)

func TestMutationFlowsInIsolatedSchema(t *testing.T) {
	if os.Getenv("LACI_RUN_ISOLATED_DB_TESTS") != "1" {
		t.Skip("set LACI_RUN_ISOLATED_DB_TESTS=1 to create and remove a dedicated test schema")
	}
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	adminCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := pgxpool.NewWithConfig(ctx, adminCfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	schema := "laci_qa_" + newID()
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Errorf("remove isolated test schema: %v", err)
		}
	})
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	poolCfg.ConnConfig.RuntimeParams["search_path"] = schema
	poolCfg.MaxConns = 2
	db, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	for _, path := range []string{"../../migrations/001_baseline.sql", "../../migrations/002_performance_and_integrity.sql"} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, string(migration), pgx.QueryExecModeSimpleProtocol); err != nil {
			t.Fatalf("apply isolated migration %s: %v", path, err)
		}
	}
	crypt, err := security.NewCrypto(cfg.EncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	a := New(cfg, db, crypt, nil)
	t.Cleanup(a.CloseBackgroundMail)
	users := []domain.User{
		{ID: "qa_user_a", Name: "User A", Role: "SEKRETARIS_PAC", IsActive: true},
		{ID: "qa_user_b", Name: "User B", Role: "SEKRETARIS_PAC", IsActive: true},
		{ID: "qa_admin", Name: "Admin", Role: "SEKRETARIS_CABANG", IsActive: true},
	}
	for _, user := range users {
		if _, err := db.Exec(ctx, `INSERT INTO "User"("id","name","email","role","isActive","emailVerified","createdAt","updatedAt") VALUES($1,$2,$3,$4::"Role",true,true,NOW(),NOW())`, user.ID, user.Name, user.ID+"@example.test", user.Role); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, `INSERT INTO "Periode"("id","nama","userId","isActive","createdAt","updatedAt") VALUES($1,'QA',$2,true,NOW(),NOW())`, user.ID+"_period", user.ID); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("health and API specification are served through the router", func(t *testing.T) {
		for _, check := range []struct {
			path string
			body string
		}{
			{path: "/healthz", body: `"status":"ok"`},
			{path: "/api/docs", body: `"openapi"`},
		} {
			w := httptest.NewRecorder()
			a.Router().ServeHTTP(w, httptest.NewRequest(http.MethodGet, check.path, nil))
			if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), check.body) {
				t.Fatalf("GET %s = %d: %s", check.path, w.Code, w.Body.String())
			}
		}
	})

	t.Run("submission statistics return a complete zero shape without a period", func(t *testing.T) {
		user := domain.User{ID: "qa_no_period_" + newID(), Role: "SEKRETARIS_PAC", IsActive: true}
		if _, err := db.Exec(ctx, `INSERT INTO "User"("id","name","email","role","isActive","emailVerified","createdAt","updatedAt") VALUES($1,'No Period',$2,'SEKRETARIS_PAC',true,true,NOW(),NOW())`, user.ID, user.ID+"@example.test"); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		a.pengajuanStats(w, qaMultipartRequest(t, http.MethodGet, "/api/pengajuan/stats", user, "", nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"total":0`) || !strings.Contains(w.Body.String(), `"ditolak":0`) {
			t.Fatalf("statistics without a period = %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("realtime crosses backend instances and releases its database listener", func(t *testing.T) {
		listener := New(cfg, db, crypt, nil)
		publisher := New(cfg, db, crypt, nil)
		defer listener.CloseBackgroundMail()
		defer publisher.CloseBackgroundMail()
		stream := make(chan string, 16)
		listener.hub.add(stream)
		defer listener.hub.remove(stream)
		realtimeCtx, stop := context.WithCancel(context.Background())
		defer stop()
		listener.StartRealtimeListener(realtimeCtx)
		go publisher.publishNotifications(realtimeCtx)
		deadline := time.NewTimer(4 * time.Second)
		defer deadline.Stop()
		probe := time.NewTicker(50 * time.Millisecond)
		defer probe.Stop()
		for {
			select {
			case event := <-stream:
				if !strings.Contains(event, `"type":"qa.realtime"`) {
					t.Fatalf("unexpected realtime event: %s", event)
				}
				stop()
				for releaseDeadline := time.Now().Add(time.Second); db.Stat().AcquiredConns() != 0 && time.Now().Before(releaseDeadline); {
					time.Sleep(10 * time.Millisecond)
				}
				if db.Stat().AcquiredConns() != 0 {
					t.Fatal("realtime listener retained a database connection after cancellation")
				}
				return
			case <-probe.C:
				publisher.notify(realtimeCtx, map[string]any{"type": "qa.realtime"})
			case <-deadline.C:
				t.Fatal("notification did not reach another backend instance")
			}
		}
	})

	t.Run("email retry validates state and records delivery failure", func(t *testing.T) {
		noSMTP := *a
		noSMTP.cfg.SMTPHost = ""
		cases := []struct {
			kind, status string
			want         int
		}{
			{kind: "VERIFIED_SUCCESS", status: "SENT", want: http.StatusBadRequest},
			{kind: "VERIFIED_SUCCESS", status: "FAILED", want: http.StatusBadGateway},
		}
		for _, check := range cases {
			id := newID()
			if _, err := db.Exec(ctx, `INSERT INTO "LogEmail"("id","to","subject","type","status","createdAt","updatedAt") VALUES($1,$2,'QA email',$3::"EmailType",$4::"EmailStatus",NOW(),NOW())`, id, users[2].ID+"@example.test", check.kind, check.status); err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			noSMTP.retryEmail(w, qaMultipartRequest(t, http.MethodPost, "/api/email-logs/"+id+"/retry", users[2], id, nil))
			if w.Code != check.want {
				t.Fatalf("retry %s email = %d: %s", check.status, w.Code, w.Body.String())
			}
			if check.status == "FAILED" {
				var status string
				var retries int
				if err := db.QueryRow(ctx, `SELECT "status"::text,"retryCount" FROM "LogEmail" WHERE "id"=$1`, id).Scan(&status, &retries); err != nil || status != "FAILED" || retries != 1 {
					t.Fatalf("retry failure log = %s, count=%d: %v", status, retries, err)
				}
			}
		}
	})

	t.Run("Google callback exchanges token and creates a registered session", func(t *testing.T) {
		oauthAPI := *a
		oauthAPI.cfg.FrontendURL = "http://localhost:3000"
		oauthAPI.cfg.GoogleClientID = "qa-client"
		oauthAPI.cfg.GoogleClientSecret = "qa-secret"
		oauthAPI.cfg.GoogleRedirectURL = "http://localhost:3000/api/auth/callback/google"
		oauthAPI.cfg.SessionSecret = "qa-oauth-session-secret"
		fakeClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			payload := `{"access_token":"qa-access-token","token_type":"Bearer","expires_in":3600}`
			if r.URL.Host == "openidconnect.googleapis.com" {
				if r.Header.Get("Authorization") != "Bearer qa-access-token" {
					t.Errorf("userinfo did not receive the exchanged access token")
				}
				payload = `{"email":"` + users[2].ID + `@example.test","email_verified":true,"picture":"https://example.test/qa-avatar.png"}`
			}
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(payload)), Request: r}, nil
		})}
		previousOutbound := outboundClient
		outboundClient = fakeClient
		defer func() { outboundClient = previousOutbound }()
		start := httptest.NewRecorder()
		oauthAPI.googleStart(start, httptest.NewRequest(http.MethodGet, "/api/auth/google?callback=http%3A%2F%2Flocalhost%3A3000%2Fdashboard%3Fsource%3Dqa", nil))
		if start.Code != http.StatusTemporaryRedirect {
			t.Fatalf("Google start = %d: %s", start.Code, start.Body.String())
		}
		authorizationURL, err := url.Parse(start.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		callback := httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?code=qa-code&state="+url.QueryEscape(authorizationURL.Query().Get("state")), nil)
		callback.AddCookie(start.Result().Cookies()[0])
		callback = callback.WithContext(context.WithValue(callback.Context(), oauth2.HTTPClient, fakeClient))
		result := httptest.NewRecorder()
		oauthAPI.googleCallback(result, callback)
		if result.Code != http.StatusTemporaryRedirect || result.Header().Get("Location") != "http://localhost:3000/dashboard?source=qa" {
			t.Fatalf("Google callback = %d, redirect=%q", result.Code, result.Header().Get("Location"))
		}
		var sessionCookie *http.Cookie
		for _, cookie := range result.Result().Cookies() {
			if cookie.Name == oauthAPI.cfg.SessionCookie {
				sessionCookie = cookie
			}
		}
		if sessionCookie == nil || !sessionCookie.HttpOnly {
			t.Fatal("Google callback did not issue an HTTP-only session cookie")
		}
		var sessions int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "Session" WHERE "userId"=$1`, users[2].ID).Scan(&sessions); err != nil || sessions != 1 {
			t.Fatalf("Google session count=%d: %v", sessions, err)
		}
		var loginLogs int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "LogActivity" WHERE "userId"=$1 AND "action"='LOGIN' AND "module"='AUTH' AND "description" LIKE '%via Google%'`, users[2].ID).Scan(&loginLogs); err != nil || loginLogs != 1 {
			t.Fatalf("Google login activity count=%d: %v", loginLogs, err)
		}
	})

	t.Run("profile read and update preserve credential integrity", func(t *testing.T) {
		id := "qa_profile_" + newID()
		originalEmail := id + "@example.test"
		user := domain.User{ID: id, Name: "Profil Lama", Email: originalEmail, Role: "SEKRETARIS_PAC", IsActive: true, EmailVerified: true}
		if _, err := db.Exec(ctx, `INSERT INTO "User"("id","name","email","role","isActive","emailVerified","createdAt","updatedAt") VALUES($1,$2,$3,'SEKRETARIS_PAC',true,true,NOW(),NOW())`, id, user.Name, originalEmail); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, `INSERT INTO "Periode"("id","nama","userId","isActive","createdAt","updatedAt") VALUES($1,'QA',$2,true,NOW(),NOW())`, id+"_period", id); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, `INSERT INTO "Account"("id","userId","accountId","providerId","password","createdAt","updatedAt") VALUES($1,$2,$3,'credential',$4,NOW(),NOW())`, newID(), id, id, "$2a$10$placeholder"); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		a.me(w, qaMultipartRequest(t, http.MethodGet, "/api/me", user, "", nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), originalEmail) {
			t.Fatalf("profile read = %d: %s", w.Code, w.Body.String())
		}
		newEmail := id + "_new@example.test"
		request := qaMultipartRequest(t, http.MethodPatch, "/api/me", user, "", map[string]string{"name": "Profil Baru", "email": newEmail, "password": "password-qa-strong"})
		w = httptest.NewRecorder()
		a.updateMe(w, request)
		if w.Code != http.StatusOK {
			t.Fatalf("profile update = %d: %s", w.Code, w.Body.String())
		}
		var name, email string
		var verified bool
		if err := db.QueryRow(ctx, `SELECT "name","email","emailVerified" FROM "User" WHERE "id"=$1`, id).Scan(&name, &email, &verified); err != nil || name != "Profil Baru" || email != newEmail || verified {
			t.Fatalf("updated profile name=%q email=%q verified=%t error=%v", name, email, verified, err)
		}
		var hash string
		if err := db.QueryRow(ctx, `SELECT "password" FROM "Account" WHERE "userId"=$1 AND "providerId"='credential'`, id).Scan(&hash); err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte("password-qa-strong")) != nil {
			t.Fatalf("profile password was not updated: %v", err)
		}
		w = httptest.NewRecorder()
		a.verificationStatus(w, httptest.NewRequest(http.MethodGet, "/api/auth/verification-status?email="+newEmail, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"verified":false`) {
			t.Fatalf("updated email verification status = %d: %s", w.Code, w.Body.String())
		}
		store := &backupObjectStore{}
		withStore := *a
		withStore.r2 = store
		user.Name, user.Email, user.EmailVerified = "Profil Baru", newEmail, false
		png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR\x00\x00\x00\x01\x00\x00\x00\x01")
		w = httptest.NewRecorder()
		withStore.updateMe(w, qaMultipartNamedFileRequest(t, http.MethodPatch, "/api/me", user, "", map[string]string{"name": user.Name, "email": user.Email}, "image", "profil.png", png))
		if w.Code != http.StatusOK || !strings.HasPrefix(store.putKey, "profile/") || bytes.Equal(store.value, png) {
			t.Fatalf("encrypted profile image upload = %d: key=%q", w.Code, store.putKey)
		}
		sessionToken := "g1_qa_profile_" + newID()
		if _, err := db.Exec(ctx, `INSERT INTO "Session"("id","userId","token","expiresAt","createdAt","updatedAt") VALUES($1,$2,$3,NOW()+INTERVAL '1 hour',NOW(),NOW())`, newID(), id, a.sessionDigest(sessionToken)); err != nil {
			t.Fatal(err)
		}
		profileRequest := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		profileRequest.AddCookie(&http.Cookie{Name: cfg.SessionCookie, Value: sessionToken})
		w = httptest.NewRecorder()
		withStore.Router().ServeHTTP(w, profileRequest)
		if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "private, no-store" || !strings.Contains(w.Body.String(), newEmail) {
			t.Fatalf("authenticated profile response = %d, cache=%q: %s", w.Code, w.Header().Get("Cache-Control"), w.Body.String())
		}
		imageRequest := httptest.NewRequest(http.MethodGet, "/api/manajemen-user/"+id+"/image", nil)
		imageRequest.AddCookie(&http.Cookie{Name: cfg.SessionCookie, Value: sessionToken})
		route := chi.NewRouteContext()
		route.URLParams.Add("id", id)
		imageRequest = imageRequest.WithContext(context.WithValue(imageRequest.Context(), chi.RouteCtxKey, route))
		w = httptest.NewRecorder()
		withStore.userImage(w, imageRequest)
		if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/png" || !bytes.Equal(w.Body.Bytes(), png) {
			t.Fatalf("encrypted profile image download = %d: %s", w.Code, w.Body.String())
		}
		for _, check := range []struct {
			user domain.User
			want int
		}{
			{user: users[1], want: http.StatusNotFound},
			{user: users[2], want: http.StatusOK},
		} {
			token := "g1_qa_image_" + newID()
			if _, err := db.Exec(ctx, `INSERT INTO "Session"("id","userId","token","expiresAt","createdAt","updatedAt") VALUES($1,$2,$3,NOW()+INTERVAL '1 hour',NOW(),NOW())`, newID(), check.user.ID, a.sessionDigest(token)); err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodGet, "/api/manajemen-user/"+id+"/image", nil)
			r.AddCookie(&http.Cookie{Name: cfg.SessionCookie, Value: token})
			route := chi.NewRouteContext()
			route.URLParams.Add("id", id)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, route))
			w := httptest.NewRecorder()
			withStore.userImage(w, r)
			if w.Code != check.want {
				t.Fatalf("%s read another user's image = %d: %s", check.user.Role, w.Code, w.Body.String())
			}
		}
	})

	t.Run("backup jobs are exclusive and release their lock", func(t *testing.T) {
		lockConn, err := db.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer lockConn.Release()
		var locked bool
		if err := lockConn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, backupLockKey).Scan(&locked); err != nil {
			t.Fatal(err)
		}
		if !locked {
			t.Skip("another backup is already running")
		}
		r := httptest.NewRequest(http.MethodPost, "/api/backups", nil)
		r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: users[2]}))
		w := httptest.NewRecorder()
		a.createBackup(w, r)
		if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "BACKUP_IN_PROGRESS") {
			t.Fatalf("concurrent backup status = %d: %s", w.Code, w.Body.String())
		}
		var released bool
		if err := lockConn.QueryRow(ctx, `SELECT pg_advisory_unlock($1)`, backupLockKey).Scan(&released); err != nil || !released {
			t.Fatalf("release test lock: %v", err)
		}
		broken := *a
		broken.cfg.DatabaseURL = "invalid database URL"
		if _, err := broken.executeBackup(ctx); err == nil {
			t.Fatal("backup unexpectedly succeeded with an invalid database URL")
		}
		if err := lockConn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, backupLockKey).Scan(&locked); err != nil || !locked {
			t.Fatalf("failed backup kept its advisory lock: acquired=%t error=%v", locked, err)
		}
		if err := lockConn.QueryRow(ctx, `SELECT pg_advisory_unlock($1)`, backupLockKey).Scan(&released); err != nil || !released {
			t.Fatalf("release verification lock: %v", err)
		}
	})

	t.Run("email retries serialize across database connections", func(t *testing.T) {
		release, locked, err := a.acquireEmailRetryLock(ctx, "qa-email-retry")
		if err != nil || !locked {
			t.Fatalf("first retry lock: acquired=%t error=%v", locked, err)
		}
		_, locked, err = a.acquireEmailRetryLock(ctx, "qa-email-retry")
		if err != nil || locked {
			release()
			t.Fatalf("concurrent retry lock: acquired=%t error=%v", locked, err)
		}
		release()
		release, locked, err = a.acquireEmailRetryLock(ctx, "qa-email-retry")
		if err != nil || !locked {
			t.Fatalf("released retry lock could not be reacquired: acquired=%t error=%v", locked, err)
		}
		release()
	})

	t.Run("email log finalization survives request cancellation", func(t *testing.T) {
		id := newID()
		if _, err := db.Exec(ctx, `INSERT INTO "LogEmail"("id","to","subject","type","status","createdAt","updatedAt") VALUES($1,'qa@example.test','QA','VERIFICATION','PENDING',NOW(),NOW())`, id); err != nil {
			t.Fatal(err)
		}
		canceled, stop := context.WithCancel(ctx)
		stop()
		if err := a.setEmailLogStatus(canceled, id, "FAILED", "SMTP unavailable"); err != nil {
			t.Fatal(err)
		}
		var status, message string
		if err := db.QueryRow(ctx, `SELECT "status"::text,"errorMessage" FROM "LogEmail" WHERE "id"=$1`, id).Scan(&status, &message); err != nil || status != "FAILED" || message != "SMTP unavailable" {
			t.Fatalf("email final status=%q message=%q error=%v", status, message, err)
		}
	})

	t.Run("pg_dump uploads authenticated ciphertext", func(t *testing.T) {
		store := &backupObjectStore{}
		for i := 0; i < 10; i++ {
			store.objects = append(store.objects, storage.Object{Key: "backups/older-" + string(rune('a'+i)) + ".dump", LastModified: time.Unix(int64(i), 0)})
		}
		withStore := *a
		withStore.r2 = store
		backupCtx, backupCancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer backupCancel()
		name, err := withStore.executeBackup(backupCtx)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(name, ".dump.enc") || store.putKey != "backups/"+name || store.putSize != int64(len(store.value)) || bytes.HasPrefix(store.value, []byte("PGDMP")) {
			t.Fatalf("backup upload mismatch: name=%q key=%q bytes=%d declared=%d", name, store.putKey, len(store.value), store.putSize)
		}
		if len(store.deleted) != 1 || store.deleted[0] != "backups/older-a.dump" {
			t.Fatalf("backup retention deleted %v", store.deleted)
		}
		store.putErr = context.DeadlineExceeded
		store.deleted = nil
		if _, err := withStore.executeBackup(backupCtx); err == nil || len(store.deleted) != 0 {
			t.Fatalf("failed upload removed existing backups: error=%v deleted=%v", err, store.deleted)
		}
		store.putErr = nil
		var decrypted bytes.Buffer
		if err := crypt.DecryptBackup(backupCtx, &decrypted, bytes.NewReader(store.value)); err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(decrypted.Bytes(), []byte("PGDMP")) {
			t.Fatal("decrypted backup is not a PostgreSQL custom-format dump")
		}
		verify := exec.CommandContext(backupCtx, "pg_restore", "--list")
		verify.Stdin = bytes.NewReader(decrypted.Bytes())
		if output, err := verify.CombinedOutput(); err != nil || !bytes.Contains(output, []byte("TOC Entries")) {
			t.Fatalf("pg_restore cannot read decrypted backup: %v", err)
		}
		token := "g1_qa_backup_" + newID()
		if _, err := db.Exec(backupCtx, `INSERT INTO "Session"("id","userId","token","expiresAt","createdAt","updatedAt") VALUES($1,$2,$3,NOW()+INTERVAL '1 hour',NOW(),NOW())`, newID(), users[2].ID, a.sessionDigest(token)); err != nil {
			t.Fatal(err)
		}
		path := "/api/backups/file?key=" + store.putKey
		unauthenticated := httptest.NewRecorder()
		withStore.Router().ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, path, nil))
		if unauthenticated.Code != http.StatusUnauthorized {
			t.Fatalf("backup route without session = %d", unauthenticated.Code)
		}
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.AddCookie(&http.Cookie{Name: cfg.SessionCookie, Value: token})
		response := httptest.NewRecorder()
		withStore.Router().ServeHTTP(response, request)
		if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), decrypted.Bytes()) {
			t.Fatalf("authenticated backup route: status=%d bytes=%d", response.Code, response.Body.Len())
		}
	})

	t.Run("cron backup authorization and backup deletion", func(t *testing.T) {
		store := &backupObjectStore{}
		withStore := *a
		withStore.r2 = store
		withStore.cfg.CronSecret = "qa-cron-secret"
		request := httptest.NewRequest(http.MethodGet, "/api/cron/backup", nil)
		w := httptest.NewRecorder()
		withStore.cronBackup(w, request)
		if w.Code != http.StatusUnauthorized || store.putKey != "" {
			t.Fatalf("unauthorized cron backup = %d, key=%q", w.Code, store.putKey)
		}
		request = httptest.NewRequest(http.MethodGet, "/api/cron/backup", nil)
		request.Header.Set("Authorization", "Bearer qa-cron-secret")
		w = httptest.NewRecorder()
		withStore.cronBackup(w, request)
		if w.Code != http.StatusOK || !strings.HasSuffix(store.putKey, ".dump.enc") {
			t.Fatalf("authorized cron backup = %d, key=%q: %s", w.Code, store.putKey, w.Body.String())
		}
		w = httptest.NewRecorder()
		withStore.deleteBackup(w, qaMultipartRequest(t, http.MethodDelete, "/api/backups?key="+store.putKey, users[0], "", nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("PAC deleted backup: %d", w.Code)
		}
		w = httptest.NewRecorder()
		withStore.deleteBackup(w, qaMultipartRequest(t, http.MethodDelete, "/api/backups?key="+store.putKey, users[2], "", nil))
		if w.Code != http.StatusOK || len(store.deleted) != 1 || store.deleted[0] != store.putKey {
			t.Fatalf("admin backup delete = %d: %v", w.Code, store.deleted)
		}
	})

	t.Run("archive write and encrypted read", func(t *testing.T) {
		r := qaMultipartRequest(t, http.MethodPost, "/api/arsip", users[0], "", map[string]string{
			"noSurat": "QA/2026/001", "jenisSurat": "KELUAR", "tanggal": "2026-10-24", "pengirimPenerima": "Tujuan QA", "perihal": "Uji", "organisasi": "BERSAMA",
		})
		w := httptest.NewRecorder()
		a.createArsip(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("create archive status = %d: %s", w.Code, w.Body.String())
		}
		var encrypted string
		if err := db.QueryRow(ctx, `SELECT "noSurat" FROM "ArsipSurat" WHERE "userId"=$1`, users[0].ID).Scan(&encrypted); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(encrypted, "v2:") {
			t.Fatal("new archive text is not AES-GCM")
		}
		plain, err := crypt.DecryptText(encrypted)
		if err != nil || plain != "QA/2026/001" {
			t.Fatalf("archive decryption = %q, %v", plain, err)
		}
	})

	t.Run("encryption failure does not write records", func(t *testing.T) {
		broken := *a
		broken.crypto = &security.Crypto{}
		var before, after int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "ArsipSurat"`).Scan(&before); err != nil {
			t.Fatal(err)
		}
		r := qaMultipartRequest(t, http.MethodPost, "/api/arsip", users[0], "", map[string]string{
			"noSurat": "QA-INVALID-CRYPTO", "jenisSurat": "KELUAR", "tanggal": "2026-10-24", "pengirimPenerima": "Tujuan QA", "perihal": "Uji", "organisasi": "BERSAMA",
		})
		w := httptest.NewRecorder()
		broken.createArsip(w, r)
		if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "ENCRYPTION_FAILED") {
			t.Fatalf("archive encryption failure = %d: %s", w.Code, w.Body.String())
		}
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "ArsipSurat"`).Scan(&after); err != nil {
			t.Fatal(err)
		}
		if after != before {
			t.Fatalf("archive count changed after encryption failure: %d -> %d", before, after)
		}
	})

	t.Run("archive sort, pagination and ownership", func(t *testing.T) {
		for _, entry := range []struct {
			user   domain.User
			number string
			date   string
		}{
			{users[0], "QA/2026/003", "2026-10-26"},
			{users[0], "QA/2026/002", "2026-10-25"},
			{users[1], "OTHER/2026/001", "2026-10-27"},
		} {
			r := qaMultipartRequest(t, http.MethodPost, "/api/arsip", entry.user, "", map[string]string{
				"noSurat": entry.number, "jenisSurat": "KELUAR", "tanggal": entry.date, "pengirimPenerima": "Tujuan QA", "perihal": "Uji", "organisasi": "BERSAMA",
			})
			w := httptest.NewRecorder()
			a.createArsip(w, r)
			if w.Code != http.StatusCreated {
				t.Fatalf("create archive %q = %d: %s", entry.number, w.Code, w.Body.String())
			}
		}
		for page, want := range map[int]string{1: "QA/2026/001", 2: "QA/2026/002", 3: "QA/2026/003"} {
			r := httptest.NewRequest(http.MethodGet, "/api/arsip?sortKey=noSurat&sortDir=asc&page="+itoa(page)+"&limit=1", nil)
			r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: users[0]}))
			w := httptest.NewRecorder()
			a.listArsip(w, r)
			var result struct {
				Data []struct {
					NoSurat string `json:"noSurat"`
				} `json:"data"`
				Total      int `json:"total"`
				TotalPages int `json:"totalPages"`
			}
			if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &result) != nil || result.Total != 3 || result.TotalPages != 3 || len(result.Data) != 1 || result.Data[0].NoSurat != want {
				t.Fatalf("archive page %d = %d: %s", page, w.Code, w.Body.String())
			}
		}
		var foreignID string
		if err := db.QueryRow(ctx, `SELECT "id" FROM "ArsipSurat" WHERE "userId"=$1 LIMIT 1`, users[1].ID).Scan(&foreignID); err != nil {
			t.Fatal(err)
		}
		r := qaMultipartRequest(t, http.MethodGet, "/api/arsip/"+foreignID, users[0], foreignID, nil)
		w := httptest.NewRecorder()
		a.getArsip(w, r)
		if w.Code != http.StatusNotFound {
			t.Fatalf("foreign archive detail = %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("archive stats include CBP KPP", func(t *testing.T) {
		r := qaMultipartRequest(t, http.MethodPost, "/api/arsip", users[0], "", map[string]string{
			"noSurat": "QA/CBP-KPP/001", "jenisSurat": "MASUK", "tanggal": "2026-10-27", "pengirimPenerima": "CBP KPP", "perihal": "Uji statistik", "organisasi": "CBP_KPP",
		})
		w := httptest.NewRecorder()
		a.createArsip(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("create CBP/KPP archive = %d: %s", w.Code, w.Body.String())
		}

		w = httptest.NewRecorder()
		a.arsipStats(w, qaMultipartRequest(t, http.MethodGet, "/api/arsip/stats", users[0], "", nil))
		var stats map[string]int
		if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &stats) != nil {
			t.Fatalf("archive stats = %d: %s", w.Code, w.Body.String())
		}
		if stats["cbpkpp"] != 1 {
			t.Fatalf("CBP/KPP archive count = %d, want 1", stats["cbpkpp"])
		}
	})

	t.Run("archive update and deletion", func(t *testing.T) {
		fields := map[string]string{"noSurat": "QA/MUTATE/001", "jenisSurat": "MASUK", "tanggal": "2026-10-25", "pengirimPenerima": "Pengirim QA", "perihal": "Perihal QA", "organisasi": "IPNU", "deskripsi": "Catatan awal"}
		w := httptest.NewRecorder()
		a.createArsip(w, qaMultipartRequest(t, http.MethodPost, "/api/arsip", users[0], "", fields))
		if w.Code != http.StatusCreated {
			t.Fatalf("archive create = %d: %s", w.Code, w.Body.String())
		}
		var created struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.Data.ID == "" {
			t.Fatalf("archive create response: %v", err)
		}
		id := created.Data.ID
		fields["noSurat"] = "QA/MUTATE/002"
		fields["perihal"] = "Perihal Diperbarui"
		w = httptest.NewRecorder()
		a.updateArsip(w, qaMultipartRequest(t, http.MethodPatch, "/api/arsip/"+id, users[1], id, fields))
		if w.Code != http.StatusNotFound {
			t.Fatalf("foreign archive update = %d", w.Code)
		}
		w = httptest.NewRecorder()
		a.updateArsip(w, qaMultipartRequest(t, http.MethodPatch, "/api/arsip/"+id, users[0], id, fields))
		if w.Code != http.StatusOK {
			t.Fatalf("owner archive update = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.getArsip(w, qaMultipartRequest(t, http.MethodGet, "/api/arsip/"+id, users[0], id, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "QA/MUTATE/002") || !strings.Contains(w.Body.String(), "Perihal Diperbarui") {
			t.Fatalf("updated archive detail = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.arsipDownloadToken(w, qaMultipartRequest(t, http.MethodPost, "/api/arsip/"+id+"/download-token", users[1], id, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("foreign archive token = %d", w.Code)
		}
		w = httptest.NewRecorder()
		a.arsipDownloadToken(w, qaMultipartRequest(t, http.MethodPost, "/api/arsip/"+id+"/download-token", users[0], id, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"token":`) {
			t.Fatalf("owner archive token = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.deleteArsip(w, qaMultipartRequest(t, http.MethodDelete, "/api/arsip/"+id, users[1], id, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("foreign archive delete = %d", w.Code)
		}
		w = httptest.NewRecorder()
		a.deleteArsip(w, qaMultipartRequest(t, http.MethodDelete, "/api/arsip/"+id, users[0], id, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("owner archive delete = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.getArsip(w, qaMultipartRequest(t, http.MethodGet, "/api/arsip/"+id, users[0], id, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("deleted archive remained readable = %d", w.Code)
		}
	})

	t.Run("submission detail, status and ownership", func(t *testing.T) {
		id := "qa_submission_" + newID()
		number, err := crypt.EncryptText("QA/SURAT/001")
		if err != nil {
			t.Fatal(err)
		}
		purpose, err := crypt.EncryptText("Keperluan QA")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, `INSERT INTO "PengajuanBerkas"("id","userId","periodeId","periodeIdPac","noSurat","penerima","tanggal","keperluan","file","status","createdAt","updatedAt") VALUES($1,$2,$3,$4,$5,'BERSAMA',NOW(),$6,'qa/test-file.enc','PENDING',NOW(),NOW())`, id, users[0].ID, users[2].ID+"_period", users[0].ID+"_period", number, purpose); err != nil {
			t.Fatal(err)
		}
		for _, check := range []struct {
			user domain.User
			want int
		}{
			{users[0], http.StatusOK},
			{users[1], http.StatusNotFound},
			{users[2], http.StatusOK},
		} {
			r := qaMultipartRequest(t, http.MethodGet, "/api/pengajuan/"+id, check.user, id, nil)
			w := httptest.NewRecorder()
			a.getPengajuan(w, r)
			if w.Code != check.want {
				t.Fatalf("submission detail for %s = %d, want %d: %s", check.user.ID, w.Code, check.want, w.Body.String())
			}
		}
		w := httptest.NewRecorder()
		a.listReferencePengajuan(w, qaMultipartRequest(t, http.MethodGet, "/api/pengajuan/reference?q=QA%2FSURAT%2F001", users[1], "", nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "QA/SURAT/001") {
			t.Fatalf("PAC reference submissions = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.getReferencePengajuan(w, qaMultipartRequest(t, http.MethodGet, "/api/pengajuan/reference/"+id, users[1], id, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "QA/SURAT/001") {
			t.Fatalf("PAC reference detail = %d: %s", w.Code, w.Body.String())
		}
		for _, check := range []struct {
			user domain.User
			want int
		}{{users[0], http.StatusOK}, {users[1], http.StatusForbidden}, {users[2], http.StatusOK}} {
			w = httptest.NewRecorder()
			a.pengajuanDownloadToken(w, qaMultipartRequest(t, http.MethodPost, "/api/pengajuan/"+id+"/download-token", check.user, id, nil))
			if w.Code != check.want {
				t.Fatalf("submission download token for %s = %d: %s", check.user.ID, w.Code, w.Body.String())
			}
		}
		statusRequest := func(user domain.User) *http.Request {
			r := httptest.NewRequest(http.MethodPatch, "/api/pengajuan/"+id+"/status", strings.NewReader(`{"status":"DITOLAK","alasanPenolakan":"Data kurang lengkap"}`))
			r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: user}))
			route := chi.NewRouteContext()
			route.URLParams.Add("id", id)
			return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, route))
		}
		w = httptest.NewRecorder()
		a.updatePengajuanStatus(w, statusRequest(users[0]))
		if w.Code != http.StatusForbidden {
			t.Fatalf("PAC changed submission status: %d", w.Code)
		}
		noSMTP := *a
		noSMTP.cfg.SMTPHost = ""
		tooLongReason := httptest.NewRequest(http.MethodPatch, "/api/pengajuan/"+id+"/status", strings.NewReader(`{"status":"DITOLAK","alasanPenolakan":"`+strings.Repeat("a", pengajuanRejectReasonMaxLength+1)+`"}`))
		tooLongReason.Header.Set("Content-Type", "application/json")
		tooLongReason = tooLongReason.WithContext(context.WithValue(tooLongReason.Context(), userContextKey, authContext{User: users[2]}))
		tooLongReasonRoute := chi.NewRouteContext()
		tooLongReasonRoute.URLParams.Add("id", id)
		tooLongReason = tooLongReason.WithContext(context.WithValue(tooLongReason.Context(), chi.RouteCtxKey, tooLongReasonRoute))
		w = httptest.NewRecorder()
		noSMTP.updatePengajuanStatus(w, tooLongReason)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "maksimal 2000 karakter") {
			t.Fatalf("overlong rejection reason = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		noSMTP.updatePengajuanStatus(w, statusRequest(users[2]))
		if w.Code != http.StatusOK {
			t.Fatalf("admin submission status = %d: %s", w.Code, w.Body.String())
		}
		var status, reason string
		if err := db.QueryRow(ctx, `SELECT "status"::text,"alasanPenolakan" FROM "PengajuanBerkas" WHERE "id"=$1`, id).Scan(&status, &reason); err != nil {
			t.Fatal(err)
		}
		plainReason, err := crypt.DecryptText(reason)
		if err != nil || status != "DITOLAK" || plainReason != "Data kurang lengkap" {
			t.Fatalf("submission status=%s reason=%q error=%v", status, plainReason, err)
		}
	})

	t.Run("submission create, pending edit, approval and object cleanup", func(t *testing.T) {
		store := &backupObjectStore{}
		withStore := *a
		withStore.r2 = store
		withStore.cfg.SMTPHost = ""
		withStore.cfg.MailFromAddress = ""
		file := []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF\n")
		fields := map[string]string{"noSurat": "QA/PENGAJUAN/001", "penerima": "BERSAMA", "tanggal": "2026-10-25", "keperluan": "Keperluan QA", "deskripsi": "Catatan QA"}
		invalidFields := map[string]string{"noSurat": strings.Repeat("a", pengajuanNumberMaxLength+1), "penerima": " BERSAMA ", "tanggal": "2026-10-25", "keperluan": "Keperluan QA"}
		w := httptest.NewRecorder()
		withStore.createPengajuan(w, qaMultipartFileRequest(t, http.MethodPost, "/api/pengajuan", users[0], "", invalidFields, "pengajuan.pdf", file))
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "maksimal 255 karakter") || store.putKey != "" {
			t.Fatalf("overlong submission number = %d: %s (stored=%q)", w.Code, w.Body.String(), store.putKey)
		}
		w = httptest.NewRecorder()
		withStore.createPengajuan(w, qaMultipartFileRequest(t, http.MethodPost, "/api/pengajuan", users[0], "", fields, "pengajuan.pdf", file))
		if w.Code != http.StatusCreated {
			t.Fatalf("submission create = %d: %s", w.Code, w.Body.String())
		}
		var created struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.Data.ID == "" {
			t.Fatalf("submission create response: %v", err)
		}
		id := created.Data.ID
		var encryptedNumber, key, status string
		if err := db.QueryRow(ctx, `SELECT "noSurat","file","status"::text FROM "PengajuanBerkas" WHERE "id"=$1`, id).Scan(&encryptedNumber, &key, &status); err != nil || !strings.HasPrefix(encryptedNumber, "v2:") || key != store.putKey || status != "PENDING" || bytes.Equal(store.value, file) {
			t.Fatalf("submission storage state: key=%q status=%q error=%v", key, status, err)
		}
		fields["noSurat"] = "QA/PENGAJUAN/002"
		w = httptest.NewRecorder()
		withStore.updatePengajuan(w, qaMultipartRequest(t, http.MethodPatch, "/api/pengajuan/"+id, users[1], id, fields))
		if w.Code != http.StatusNotFound {
			t.Fatalf("foreign submission edit = %d", w.Code)
		}
		w = httptest.NewRecorder()
		withStore.updatePengajuan(w, qaMultipartRequest(t, http.MethodPatch, "/api/pengajuan/"+id, users[0], id, fields))
		if w.Code != http.StatusOK {
			t.Fatalf("owner pending submission edit = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		withStore.getPengajuan(w, qaMultipartRequest(t, http.MethodGet, "/api/pengajuan/"+id, users[0], id, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "QA/PENGAJUAN/002") {
			t.Fatalf("updated submission detail = %d: %s", w.Code, w.Body.String())
		}
		statusRequest := httptest.NewRequest(http.MethodPatch, "/api/pengajuan/"+id+"/status", strings.NewReader(`{"status":"DITERIMA"}`))
		statusRequest.Header.Set("Content-Type", "application/json")
		statusRequest = statusRequest.WithContext(context.WithValue(statusRequest.Context(), userContextKey, authContext{User: users[2]}))
		statusRoute := chi.NewRouteContext()
		statusRoute.URLParams.Add("id", id)
		statusRequest = statusRequest.WithContext(context.WithValue(statusRequest.Context(), chi.RouteCtxKey, statusRoute))
		w = httptest.NewRecorder()
		withStore.updatePengajuanStatus(w, statusRequest)
		if w.Code != http.StatusOK {
			t.Fatalf("admin approved submission = %d: %s", w.Code, w.Body.String())
		}
		statusRequest = httptest.NewRequest(http.MethodPatch, "/api/pengajuan/"+id+"/status", strings.NewReader(`{"status":"DITOLAK","alasanPenolakan":"Tidak boleh diproses dua kali"}`))
		statusRequest.Header.Set("Content-Type", "application/json")
		statusRequest = statusRequest.WithContext(context.WithValue(statusRequest.Context(), userContextKey, authContext{User: users[2]}))
		statusRoute = chi.NewRouteContext()
		statusRoute.URLParams.Add("id", id)
		statusRequest = statusRequest.WithContext(context.WithValue(statusRequest.Context(), chi.RouteCtxKey, statusRoute))
		w = httptest.NewRecorder()
		withStore.updatePengajuanStatus(w, statusRequest)
		if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "NOT_PENDING") {
			t.Fatalf("processed submission status changed twice = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		withStore.updatePengajuan(w, qaMultipartRequest(t, http.MethodPatch, "/api/pengajuan/"+id, users[0], id, fields))
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "NOT_PENDING") {
			t.Fatalf("approved submission was editable = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		withStore.deletePengajuan(w, qaMultipartRequest(t, http.MethodDelete, "/api/pengajuan/"+id, users[0], id, nil))
		if w.Code != http.StatusOK || len(store.deleted) != 1 || store.deleted[0] != key {
			t.Fatalf("submission delete and object cleanup = %d: %v", w.Code, store.deleted)
		}
	})

	t.Run("agenda create, update, list and delete enforce ownership", func(t *testing.T) {
		makeRequest := func(method, path, id, body string, user domain.User) *http.Request {
			r := httptest.NewRequest(method, path, strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: user}))
			route := chi.NewRouteContext()
			route.URLParams.Add("id", id)
			return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, route))
		}
		body := `{"judul":"Agenda QA","warna":"#123456","tanggalMulai":"2026-10-24","lokasi":"Kantor"}`
		w := httptest.NewRecorder()
		a.createAgenda(w, makeRequest(http.MethodPost, "/api/agenda-kegiatan", "", body, users[0]))
		if w.Code != http.StatusForbidden {
			t.Fatalf("PAC created agenda: %d", w.Code)
		}
		invalidRange := `{"judul":"Agenda QA","warna":"#123456","tanggalMulai":"2026-10-24T18:00:00Z","tanggalSelesai":"2026-10-24T08:00:00Z"}`
		w = httptest.NewRecorder()
		a.createAgenda(w, makeRequest(http.MethodPost, "/api/agenda-kegiatan", "", invalidRange, users[2]))
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "INVALID_DATE_RANGE") {
			t.Fatalf("agenda accepted an invalid date range = %d: %s", w.Code, w.Body.String())
		}
		invalidColor := `{"judul":"Agenda QA","warna":"blue","tanggalMulai":"2026-10-24"}`
		w = httptest.NewRecorder()
		a.createAgenda(w, makeRequest(http.MethodPost, "/api/agenda-kegiatan", "", invalidColor, users[2]))
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "INVALID_COLOR") {
			t.Fatalf("agenda accepted an invalid color = %d: %s", w.Code, w.Body.String())
		}
		longTitle := `{"judul":"` + strings.Repeat("A", 256) + `","warna":"#123456","tanggalMulai":"2026-10-24"}`
		w = httptest.NewRecorder()
		a.createAgenda(w, makeRequest(http.MethodPost, "/api/agenda-kegiatan", "", longTitle, users[2]))
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "VALIDATION_ERROR") {
			t.Fatalf("agenda accepted an oversized title = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.createAgenda(w, makeRequest(http.MethodPost, "/api/agenda-kegiatan", "", body, users[2]))
		if w.Code != http.StatusCreated {
			t.Fatalf("admin agenda create = %d: %s", w.Code, w.Body.String())
		}
		var created struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.Data.ID == "" {
			t.Fatalf("agenda create response: %v", err)
		}
		id := created.Data.ID
		w = httptest.NewRecorder()
		a.getAgenda(w, makeRequest(http.MethodGet, "/api/agenda-kegiatan/"+id, id, "", users[0]))
		if w.Code != http.StatusForbidden {
			t.Fatalf("PAC read foreign agenda: %d", w.Code)
		}
		w = httptest.NewRecorder()
		a.getAgenda(w, makeRequest(http.MethodGet, "/api/agenda-kegiatan/"+id, id, "", users[2]))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Agenda QA") {
			t.Fatalf("admin agenda detail = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.updateAgenda(w, makeRequest(http.MethodPatch, "/api/agenda-kegiatan/"+id, id, `{"judul":"Agenda QA Baru","warna":"#123456","tanggalMulai":"2026-10-24"}`, users[2]))
		if w.Code != http.StatusOK {
			t.Fatalf("agenda update = %d: %s", w.Code, w.Body.String())
		}
		var encryptedTitle string
		if err := db.QueryRow(ctx, `SELECT "judul" FROM "AgendaKegiatan" WHERE "id"=$1`, id).Scan(&encryptedTitle); err != nil || !strings.HasPrefix(encryptedTitle, "v2:") {
			t.Fatalf("agenda title was not encrypted: %v", err)
		}
		w = httptest.NewRecorder()
		a.listAgenda(w, makeRequest(http.MethodGet, "/api/agenda-kegiatan?q=Baru&page=1&limit=1", "", "", users[2]))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Agenda QA Baru") {
			t.Fatalf("agenda filtered list = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.deleteAgenda(w, makeRequest(http.MethodDelete, "/api/agenda-kegiatan/"+id, id, "", users[0]))
		if w.Code != http.StatusForbidden {
			t.Fatalf("PAC deleted foreign agenda: %d", w.Code)
		}
		w = httptest.NewRecorder()
		a.deleteAgenda(w, makeRequest(http.MethodDelete, "/api/agenda-kegiatan/"+id, id, "", users[2]))
		if w.Code != http.StatusOK {
			t.Fatalf("admin agenda delete = %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("SP document lifecycle and ownership", func(t *testing.T) {
		fields := map[string]string{"nama": "Berkas SP QA", "tanggalMulai": "2026-10-24", "tanggalBerakhir": "2026-10-25", "organisasi": "IPNU", "catatan": "Catatan QA"}
		r := qaMultipartRequest(t, http.MethodPost, "/api/berkas-sp", users[0], "", fields)
		w := httptest.NewRecorder()
		a.createBerkasSP(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("PAC created SP document: %d", w.Code)
		}
		invalidFields := map[string]string{"nama": "Berkas SP Rentang Salah", "tanggalMulai": "2026-10-25", "tanggalBerakhir": "2026-10-24", "organisasi": "IPNU"}
		w = httptest.NewRecorder()
		a.createBerkasSP(w, qaMultipartRequest(t, http.MethodPost, "/api/berkas-sp", users[2], "", invalidFields))
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "INVALID_DATE_RANGE") {
			t.Fatalf("SP accepted invalid date range = %d: %s", w.Code, w.Body.String())
		}
		invalidFields["tanggalBerakhir"] = "2026-10-26"
		invalidFields["organisasi"] = "BERSAMA"
		w = httptest.NewRecorder()
		a.createBerkasSP(w, qaMultipartRequest(t, http.MethodPost, "/api/berkas-sp", users[2], "", invalidFields))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("SP accepted unsupported organisation = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.createBerkasSP(w, qaMultipartRequest(t, http.MethodPost, "/api/berkas-sp", users[2], "", fields))
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "FILE_REQUIRED") {
			t.Fatalf("SP accepted missing file = %d: %s", w.Code, w.Body.String())
		}
		store := &backupObjectStore{}
		withStore := *a
		withStore.r2 = store
		file := []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF\n")
		r = qaMultipartFileRequest(t, http.MethodPost, "/api/berkas-sp", users[2], "", fields, "qa.pdf", file)
		w = httptest.NewRecorder()
		withStore.createBerkasSP(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("admin SP create = %d: %s", w.Code, w.Body.String())
		}
		var created struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.Data.ID == "" {
			t.Fatalf("SP create response: %v", err)
		}
		id := created.Data.ID
		var stored string
		if err := db.QueryRow(ctx, `SELECT "nama" FROM "BerkasSP" WHERE "id"=$1`, id).Scan(&stored); err != nil || !strings.HasPrefix(stored, "v2:") {
			t.Fatalf("SP document name is not encrypted: %v", err)
		}
		w = httptest.NewRecorder()
		withStore.getBerkasSP(w, qaMultipartRequest(t, http.MethodGet, "/api/berkas-sp/"+id, users[0], id, nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("PAC read foreign SP document: %d", w.Code)
		}
		w = httptest.NewRecorder()
		withStore.getBerkasSP(w, qaMultipartRequest(t, http.MethodGet, "/api/berkas-sp/"+id, users[2], id, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Berkas SP QA") {
			t.Fatalf("admin SP detail = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		withStore.berkasSPDownloadToken(w, qaMultipartRequest(t, http.MethodPost, "/api/berkas-sp/"+id+"/download-token", users[0], id, nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("PAC SP download token = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		withStore.berkasSPDownloadToken(w, qaMultipartRequest(t, http.MethodPost, "/api/berkas-sp/"+id+"/download-token", users[2], id, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"token":`) {
			t.Fatalf("admin SP download token = %d: %s", w.Code, w.Body.String())
		}
		fields["nama"] = "Berkas SP Diperbarui"
		w = httptest.NewRecorder()
		withStore.updateBerkasSP(w, qaMultipartRequest(t, http.MethodPatch, "/api/berkas-sp/"+id, users[2], id, fields))
		if w.Code != http.StatusOK {
			t.Fatalf("SP update = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		withStore.listBerkasSP(w, qaMultipartRequest(t, http.MethodGet, "/api/berkas-sp?q=Diperbarui&organisasi=IPNU&page=1&limit=1", users[2], "", nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Berkas SP Diperbarui") {
			t.Fatalf("SP filtered list = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		withStore.deleteBerkasSP(w, qaMultipartRequest(t, http.MethodDelete, "/api/berkas-sp/"+id, users[0], id, nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("PAC deleted foreign SP document: %d", w.Code)
		}
		w = httptest.NewRecorder()
		withStore.deleteBerkasSP(w, qaMultipartRequest(t, http.MethodDelete, "/api/berkas-sp/"+id, users[2], id, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("admin SP delete = %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("leadership document upload, download and cleanup", func(t *testing.T) {
		store := &backupObjectStore{}
		withStore := *a
		withStore.r2 = store
		file := []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF\n")
		fields := map[string]string{"nama": "Berkas Pimpinan QA", "tanggal": "2026-10-24", "catatan": "Rahasia QA"}
		w := httptest.NewRecorder()
		withStore.createBerkasPimpinan(w, qaMultipartRequest(t, http.MethodPost, "/api/berkas-pimpinan", users[0], "", fields))
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "FILE_REQUIRED") {
			t.Fatalf("leadership document accepted missing file = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		withStore.createBerkasPimpinan(w, qaMultipartFileRequest(t, http.MethodPost, "/api/berkas-pimpinan", users[0], "", fields, "qa.txt", []byte("not allowed")))
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "UPLOAD_FAILED") {
			t.Fatalf("leadership document accepted unsupported file = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		withStore.createBerkasPimpinan(w, qaMultipartFileRequest(t, http.MethodPost, "/api/berkas-pimpinan", users[0], "", fields, "qa.pdf", nil))
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "file kosong") {
			t.Fatalf("leadership document accepted empty file = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		withStore.createBerkasPimpinan(w, qaMultipartFileRequest(t, http.MethodPost, "/api/berkas-pimpinan", users[0], "", fields, "qa.pdf", make([]byte, (5<<20)+1)))
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "melebihi batas") {
			t.Fatalf("leadership document accepted oversized file = %d: %s", w.Code, w.Body.String())
		}
		r := qaMultipartFileRequest(t, http.MethodPost, "/api/berkas-pimpinan", users[0], "", fields, "qa.pdf", file)
		w = httptest.NewRecorder()
		withStore.createBerkasPimpinan(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("leadership document create = %d: %s", w.Code, w.Body.String())
		}
		var created struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.Data.ID == "" {
			t.Fatalf("leadership document create response: %v", err)
		}
		id := created.Data.ID
		if bytes.Equal(store.value, file) || !strings.HasPrefix(store.putKey, "berkas-pimpinan/") {
			t.Fatalf("leadership document was not stored as encrypted object: key=%q", store.putKey)
		}
		var storedName, storedFile string
		if err := db.QueryRow(ctx, `SELECT "nama","file" FROM "BerkasPimpinan" WHERE "id"=$1`, id).Scan(&storedName, &storedFile); err != nil || !strings.HasPrefix(storedName, "v2:") || storedFile != store.putKey {
			t.Fatalf("leadership document database values: name=%q file=%q error=%v", storedName, storedFile, err)
		}
		fields["nama"] = "Berkas Pimpinan Diperbarui"
		w = httptest.NewRecorder()
		withStore.updateBerkasPimpinan(w, qaMultipartRequest(t, http.MethodPatch, "/api/berkas-pimpinan/"+id, users[1], id, fields))
		if w.Code != http.StatusNotFound {
			t.Fatalf("foreign leadership document update = %d", w.Code)
		}
		w = httptest.NewRecorder()
		withStore.updateBerkasPimpinan(w, qaMultipartRequest(t, http.MethodPatch, "/api/berkas-pimpinan/"+id, users[0], id, fields))
		if w.Code != http.StatusOK {
			t.Fatalf("owner leadership document update = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		withStore.getBerkasPimpinan(w, qaMultipartRequest(t, http.MethodGet, "/api/berkas-pimpinan/"+id, users[0], id, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Berkas Pimpinan Diperbarui") {
			t.Fatalf("updated leadership document detail = %d: %s", w.Code, w.Body.String())
		}
		for _, candidate := range []string{id, "nonexistent"} {
			request := httptest.NewRequest(http.MethodGet, "/api/downloads/berkas-pimpinan/"+candidate, nil)
			route := chi.NewRouteContext()
			route.URLParams.Add("kind", "berkas-pimpinan")
			route.URLParams.Add("id", candidate)
			request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
			w = httptest.NewRecorder()
			withStore.download(w, request)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("unauthorized download revealed file existence for %q: %d", candidate, w.Code)
			}
		}
		w = httptest.NewRecorder()
		withStore.berkasPimpinanDownloadToken(w, qaMultipartRequest(t, http.MethodPost, "/api/berkas-pimpinan/"+id+"/download-token", users[1], id, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("foreign download token = %d", w.Code)
		}
		w = httptest.NewRecorder()
		withStore.berkasPimpinanDownloadToken(w, qaMultipartRequest(t, http.MethodPost, "/api/berkas-pimpinan/"+id+"/download-token", users[0], id, nil))
		var tokenBody struct {
			Token string `json:"token"`
		}
		if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &tokenBody) != nil || tokenBody.Token == "" {
			t.Fatalf("owner download token = %d: %s", w.Code, w.Body.String())
		}
		request := httptest.NewRequest(http.MethodGet, "/api/downloads/berkas-pimpinan/"+id+"?token="+tokenBody.Token, nil)
		route := chi.NewRouteContext()
		route.URLParams.Add("kind", "berkas-pimpinan")
		route.URLParams.Add("id", id)
		request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
		w = httptest.NewRecorder()
		withStore.download(w, request)
		if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "private, no-store" || !bytes.Equal(w.Body.Bytes(), file) {
			t.Fatalf("downloaded leadership document = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		withStore.deleteBerkasPimpinan(w, qaMultipartRequest(t, http.MethodDelete, "/api/berkas-pimpinan/"+id, users[1], id, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("foreign delete = %d", w.Code)
		}
		w = httptest.NewRecorder()
		withStore.deleteBerkasPimpinan(w, qaMultipartRequest(t, http.MethodDelete, "/api/berkas-pimpinan/"+id, users[0], id, nil))
		if w.Code != http.StatusOK || len(store.deleted) != 1 || store.deleted[0] != storedFile {
			t.Fatalf("owner delete and object cleanup = %d: %v", w.Code, store.deleted)
		}
	})

	t.Run("admin user list, activation, password reset and deletion", func(t *testing.T) {
		id := "qa_managed_" + newID()
		if _, err := db.Exec(ctx, `INSERT INTO "User"("id","name","email","role","isActive","emailVerified","createdAt","updatedAt") VALUES($1,'Managed QA',$2,'SEKRETARIS_PAC',true,true,NOW(),NOW())`, id, id+"@example.test"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, `INSERT INTO "Account"("id","userId","accountId","providerId","password","createdAt","updatedAt") VALUES($1,$2,$3,'credential',$4,NOW(),NOW())`, newID(), id, id, "$2a$10$placeholder"); err != nil {
			t.Fatal(err)
		}
		sessionToken := "g1_qa_managed_" + newID()
		if _, err := db.Exec(ctx, `INSERT INTO "Session"("id","userId","token","expiresAt","createdAt","updatedAt") VALUES($1,$2,$3,NOW()+INTERVAL '1 hour',NOW(),NOW())`, newID(), id, a.sessionDigest(sessionToken)); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		a.listUsers(w, qaMultipartRequest(t, http.MethodGet, "/api/users?q=Managed+QA&status=ACTIVE&page=1&limit=5", users[0], "", nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("PAC listed users: %d", w.Code)
		}
		w = httptest.NewRecorder()
		a.listUsers(w, qaMultipartRequest(t, http.MethodGet, "/api/users?q=Managed+QA&status=ACTIVE&page=1&limit=5", users[2], "", nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), id) {
			t.Fatalf("admin filtered users = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.toggleUserStatus(w, qaMultipartRequest(t, http.MethodPatch, "/api/users/"+id+"/status", users[2], id, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"isActive":false`) {
			t.Fatalf("deactivate user = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.toggleUserStatus(w, qaMultipartRequest(t, http.MethodPatch, "/api/users/"+id+"/status", users[2], id, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"isActive":true`) {
			t.Fatalf("reactivate user = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.resetUserPassword(w, qaMultipartRequest(t, http.MethodPost, "/api/users/"+id+"/reset-password", users[0], id, nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("PAC reset password: %d", w.Code)
		}
		w = httptest.NewRecorder()
		a.resetUserPassword(w, qaMultipartRequest(t, http.MethodPost, "/api/users/"+id+"/reset-password", users[2], id, nil))
		var passwordBody struct {
			TemporaryPassword string `json:"temporaryPassword"`
		}
		if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" || json.Unmarshal(w.Body.Bytes(), &passwordBody) != nil || passwordBody.TemporaryPassword == "" {
			t.Fatalf("admin password reset = %d: %s", w.Code, w.Body.String())
		}
		var hash string
		if err := db.QueryRow(ctx, `SELECT "password" FROM "Account" WHERE "userId"=$1 AND "providerId"='credential'`, id).Scan(&hash); err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(passwordBody.TemporaryPassword)) != nil {
			t.Fatalf("temporary password was not persisted: %v", err)
		}
		var sessions int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "Session" WHERE "userId"=$1`, id).Scan(&sessions); err != nil || sessions != 0 {
			t.Fatalf("old sessions survived password reset: count=%d error=%v", sessions, err)
		}
		w = httptest.NewRecorder()
		a.deleteUser(w, qaMultipartRequest(t, http.MethodDelete, "/api/users/"+id, users[2], id, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("admin delete user = %d: %s", w.Code, w.Body.String())
		}
		var remaining int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "User" WHERE "id"=$1`, id).Scan(&remaining); err != nil || remaining != 0 {
			t.Fatalf("deleted user remains: count=%d error=%v", remaining, err)
		}
	})

	t.Run("client activity log detail respects ownership", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/logs", strings.NewReader(`{"action":"EXPORT","module":"ARSIP_SURAT","description":"QA export activity"}`))
		request.Header.Set("Content-Type", "application/json")
		request = request.WithContext(context.WithValue(request.Context(), userContextKey, authContext{User: users[0]}))
		w := httptest.NewRecorder()
		a.createClientLog(w, request)
		if w.Code != http.StatusOK {
			t.Fatalf("client activity log create = %d: %s", w.Code, w.Body.String())
		}
		var id string
		if err := db.QueryRow(ctx, `SELECT "id" FROM "LogActivity" WHERE "description"='QA export activity' AND "userId"=$1`, users[0].ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		for _, check := range []struct {
			user domain.User
			want int
		}{{users[0], http.StatusOK}, {users[1], http.StatusNotFound}, {users[2], http.StatusOK}} {
			w = httptest.NewRecorder()
			a.getLog(w, qaMultipartRequest(t, http.MethodGet, "/api/logs/"+id, check.user, id, nil))
			if w.Code != check.want {
				t.Fatalf("activity log detail for %s = %d: %s", check.user.ID, w.Code, w.Body.String())
			}
		}
	})

	t.Run("branch member list reads only its requested historical period", func(t *testing.T) {
		periodID := users[2].ID + "_history"
		if _, err := db.Exec(ctx, `INSERT INTO "Periode"("id","nama","userId","isActive","createdAt","updatedAt") VALUES($1,'QA History',$2,false,NOW(),NOW())`, periodID, users[2].ID); err != nil {
			t.Fatal(err)
		}
		name, err := crypt.EncryptText("Admin Historical Member")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, `INSERT INTO "Anggota"("id","userId","periodeId","namaLengkap","jenisKelamin","createdAt","updatedAt") VALUES($1,$2,$3,$4,'LAKI_LAKI',NOW(),NOW())`, newID(), users[2].ID, periodID, name); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/api/anggota?periodeId="+url.QueryEscape(periodID), nil)
		req = req.WithContext(context.WithValue(req.Context(), userContextKey, authContext{User: users[2]}))
		w := httptest.NewRecorder()
		a.listAnggota(w, req)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Admin Historical Member") || !strings.Contains(w.Body.String(), `"total":1`) {
			t.Fatalf("branch historical member list = %d: %s", w.Code, w.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/api/anggota?periodeId="+url.QueryEscape(users[0].ID+"_period"), nil)
		req = req.WithContext(context.WithValue(req.Context(), userContextKey, authContext{User: users[2]}))
		w = httptest.NewRecorder()
		a.listAnggota(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("branch read foreign historical period = %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("member update rejects corrupt child JSON and copy cannot cross users", func(t *testing.T) {
		for _, test := range []struct {
			name  string
			field string
			value string
			code  string
		}{
			{name: "invalid NIK", field: "nik", value: "123", code: "INVALID_NIK"},
			{name: "invalid phone", field: "noHp", value: "nomor-salah", code: "INVALID_PHONE"},
			{name: "invalid email", field: "email", value: "email-salah", code: "INVALID_EMAIL"},
		} {
			fields := map[string]string{
				"namaLengkap": "Anggota Invalid", "jenisKelamin": "LAKI_LAKI", "perkaderans": `[]`, "pendidikans": `[]`,
			}
			fields[test.field] = test.value
			w := httptest.NewRecorder()
			a.createAnggota(w, qaMultipartRequest(t, http.MethodPost, "/api/anggota", users[0], "", fields))
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), test.code) {
				t.Fatalf("%s accepted = %d: %s", test.name, w.Code, w.Body.String())
			}
		}
		r := qaMultipartRequest(t, http.MethodPost, "/api/anggota", users[0], "", map[string]string{
			"namaLengkap": "Anggota QA", "jenisKelamin": "LAKI_LAKI", "perkaderans": `[{"namaPerkaderan":"Makesta","tanggal":"2026-10-24","tempat":"QA"}]`, "pendidikans": `[]`,
		})
		w := httptest.NewRecorder()
		a.createAnggota(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("create member status = %d: %s", w.Code, w.Body.String())
		}
		var memberID string
		if err := db.QueryRow(ctx, `SELECT "id" FROM "Anggota" WHERE "userId"=$1`, users[0].ID).Scan(&memberID); err != nil {
			t.Fatal(err)
		}
		const sharedPhoto = "anggota/qa-shared-photo-jpg.enc"
		if _, err := db.Exec(ctx, `UPDATE "Anggota" SET "foto"=$2 WHERE "id"=$1`, memberID, sharedPhoto); err != nil {
			t.Fatal(err)
		}
		w = httptest.NewRecorder()
		a.getAnggota(w, qaMultipartRequest(t, http.MethodGet, "/api/anggota/"+memberID, users[1], memberID, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("foreign member detail = %d", w.Code)
		}
		w = httptest.NewRecorder()
		a.getAnggota(w, qaMultipartRequest(t, http.MethodGet, "/api/anggota/"+memberID, users[0], memberID, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Anggota QA") || !strings.Contains(w.Body.String(), "Makesta") {
			t.Fatalf("owner member detail = %d: %s", w.Code, w.Body.String())
		}
		r = qaMultipartRequest(t, http.MethodPatch, "/api/anggota/"+memberID, users[1], memberID, map[string]string{
			"namaLengkap": "Foreign Update", "jenisKelamin": "LAKI_LAKI", "perkaderans": `[]`, "pendidikans": `[]`,
		})
		w = httptest.NewRecorder()
		a.updateAnggota(w, r)
		if w.Code != http.StatusNotFound {
			t.Fatalf("foreign PAC member update = %d: %s", w.Code, w.Body.String())
		}
		r = qaMultipartRequest(t, http.MethodPatch, "/api/anggota/"+memberID, users[2], memberID, map[string]string{
			"namaLengkap": "Anggota QA Diperbarui Cabang", "jenisKelamin": "LAKI_LAKI", "perkaderans": `[{"namaPerkaderan":"Makesta","tanggal":"2026-10-24","tempat":"QA"}]`, "pendidikans": `[]`,
		})
		w = httptest.NewRecorder()
		a.updateAnggota(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("branch member update = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.getAnggota(w, qaMultipartRequest(t, http.MethodGet, "/api/anggota/"+memberID, users[0], memberID, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Anggota QA Diperbarui Cabang") {
			t.Fatalf("branch member update was not persisted = %d: %s", w.Code, w.Body.String())
		}
		r = qaMultipartRequest(t, http.MethodPatch, "/api/anggota/"+memberID, users[0], memberID, map[string]string{
			"namaLengkap": "Anggota QA", "jenisKelamin": "LAKI_LAKI", "perkaderans": `[{"namaPerkaderan":"Makesta","tanggal":"","tempat":"QA"}]`, "pendidikans": `[]`,
		})
		w = httptest.NewRecorder()
		a.updateAnggota(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("incomplete member cadre update status = %d: %s", w.Code, w.Body.String())
		}
		r = qaMultipartRequest(t, http.MethodPatch, "/api/anggota/"+memberID, users[0], memberID, map[string]string{
			"namaLengkap": "Anggota QA", "jenisKelamin": "LAKI_LAKI", "perkaderans": `[`, "pendidikans": `[]`,
		})
		w = httptest.NewRecorder()
		a.updateAnggota(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("malformed member update status = %d", w.Code)
		}
		var cadreCount int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "Perkaderan" WHERE "anggotaId"=$1`, memberID).Scan(&cadreCount); err != nil || cadreCount != 1 {
			t.Fatalf("existing cadre data changed: count=%d error=%v", cadreCount, err)
		}
		r = httptest.NewRequest(http.MethodPost, "/api/anggota/copy", strings.NewReader(`{"ids":["`+memberID+`"]}`))
		r.Header.Set("Content-Type", "application/json")
		r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: users[1]}))
		w = httptest.NewRecorder()
		a.copyAnggota(w, r)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "0 anggota") {
			t.Fatalf("foreign member copy response = %d: %s", w.Code, w.Body.String())
		}
		var copied int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "Anggota" WHERE "userId"=$1`, users[1].ID).Scan(&copied); err != nil || copied != 0 {
			t.Fatalf("foreign member was copied: count=%d error=%v", copied, err)
		}

		targetPeriodID := users[0].ID + "_copy_period"
		if _, err := db.Exec(ctx, `INSERT INTO "Periode"("id","nama","userId","isActive","createdAt","updatedAt") VALUES($1,'QA Copy Target',$2,false,NOW(),NOW())`, targetPeriodID, users[0].ID); err != nil {
			t.Fatal(err)
		}
		w = httptest.NewRecorder()
		a.activatePeriode(w, qaMultipartRequest(t, http.MethodPost, "/api/periodes/"+targetPeriodID+"/activate", users[0], targetPeriodID, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("activate member copy target = %d: %s", w.Code, w.Body.String())
		}
		copyRequest := func() *http.Request {
			req := httptest.NewRequest(http.MethodPost, "/api/anggota/copy", strings.NewReader(`{"ids":["`+memberID+`"]}`))
			req.Header.Set("Content-Type", "application/json")
			return req.WithContext(context.WithValue(req.Context(), userContextKey, authContext{User: users[0]}))
		}
		w = httptest.NewRecorder()
		a.copyAnggota(w, copyRequest())
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"copied":1`) {
			t.Fatalf("member copy response = %d: %s", w.Code, w.Body.String())
		}
		var copiedMemberID string
		if err := db.QueryRow(ctx, `SELECT "id" FROM "Anggota" WHERE "userId"=$1 AND "periodeId"=$2`, users[0].ID, targetPeriodID).Scan(&copiedMemberID); err != nil {
			t.Fatal(err)
		}
		var copiedCadres int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "Perkaderan" WHERE "anggotaId"=$1`, copiedMemberID).Scan(&copiedCadres); err != nil || copiedCadres != 1 {
			t.Fatalf("copied member cadres = %d: %v", copiedCadres, err)
		}
		w = httptest.NewRecorder()
		a.copyAnggota(w, copyRequest())
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"copied":0`) {
			t.Fatalf("duplicate member copy response = %d: %s", w.Code, w.Body.String())
		}
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "Anggota" WHERE "userId"=$1 AND "periodeId"=$2`, users[0].ID, targetPeriodID).Scan(&copied); err != nil || copied != 1 {
			t.Fatalf("duplicate member copy count = %d: %v", copied, err)
		}
		store := &backupObjectStore{}
		withStore := *a
		withStore.r2 = store
		w = httptest.NewRecorder()
		withStore.deleteAnggota(w, qaMultipartRequest(t, http.MethodDelete, "/api/anggota/"+copiedMemberID, users[0], copiedMemberID, nil))
		if w.Code != http.StatusOK || len(store.deleted) != 0 {
			t.Fatalf("deleting copied member removed shared photo = %d: %v", w.Code, store.deleted)
		}
		w = httptest.NewRecorder()
		withStore.deleteAnggota(w, qaMultipartRequest(t, http.MethodDelete, "/api/anggota/"+memberID, users[0], memberID, nil))
		if w.Code != http.StatusOK || len(store.deleted) != 1 || store.deleted[0] != sharedPhoto {
			t.Fatalf("last member reference did not remove photo = %d: %v", w.Code, store.deleted)
		}
		w = httptest.NewRecorder()
		a.activatePeriode(w, qaMultipartRequest(t, http.MethodPost, "/api/periodes/"+users[0].ID+"_period/activate", users[0], users[0].ID+"_period", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("restore member source period = %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("attendance submit and duplicate prevention", func(t *testing.T) {
		loc, _ := time.LoadLocation("Asia/Jakarta")
		today := time.Now().In(loc).Format("2006-01-02")
		r := qaMultipartRequest(t, http.MethodPost, "/api/presensi", users[0], "", map[string]string{
			"namaKegiatan": "QA Presensi", "tempat": "QA", "penyelenggara": "QA", "tanggal": today, "jamMulai": "00:00", "jamSelesai": "23:59",
		})
		w := httptest.NewRecorder()
		a.createPresensi(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("create attendance status = %d: %s", w.Code, w.Body.String())
		}
		var sessionID string
		if err := db.QueryRow(ctx, `SELECT "id" FROM "Presensi" WHERE "userId"=$1`, users[0].ID).Scan(&sessionID); err != nil {
			t.Fatal(err)
		}
		fields := map[string]string{"namaLengkap": "Peserta QA", "email": "peserta@example.test", "noHp": "081234567890", "organisasi": "IPNU"}
		var participantID, accessToken string
		for attempt, want := range []int{http.StatusCreated, http.StatusConflict} {
			r = qaMultipartRequest(t, http.MethodPost, "/api/presensi/"+sessionID+"/absensi", domain.User{}, sessionID, fields)
			w = httptest.NewRecorder()
			a.submitPresensi(w, r)
			if w.Code != want {
				t.Fatalf("attendance attempt %d status = %d, want %d: %s", attempt+1, w.Code, want, w.Body.String())
			}
			if attempt == 0 {
				var payload struct {
					ParticipantID string `json:"participantId"`
					AccessToken   string `json:"accessToken"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil || payload.ParticipantID == "" || payload.AccessToken == "" {
					t.Fatalf("attendance result is incomplete: %v", err)
				}
				participantID, accessToken = payload.ParticipantID, payload.AccessToken
			}
		}
		concurrentFields := map[string]string{"namaLengkap": "Peserta Serentak", "email": "serentak@example.test", "noHp": "081234567891", "organisasi": "IPNU"}
		const attempts = 12
		requests := make([]*http.Request, attempts)
		for i := range requests {
			requests[i] = qaMultipartRequest(t, http.MethodPost, "/api/presensi/"+sessionID+"/absensi", domain.User{}, sessionID, concurrentFields)
		}
		statuses := make(chan int, attempts)
		var group sync.WaitGroup
		for _, request := range requests {
			group.Add(1)
			go func(r *http.Request) {
				defer group.Done()
				w := httptest.NewRecorder()
				a.submitPresensi(w, r)
				statuses <- w.Code
			}(request)
		}
		group.Wait()
		close(statuses)
		created, conflicts := 0, 0
		for status := range statuses {
			switch status {
			case http.StatusCreated:
				created++
			case http.StatusConflict:
				conflicts++
			default:
				t.Fatalf("unexpected concurrent attendance status = %d", status)
			}
		}
		if created != 1 || conflicts != attempts-1 {
			t.Fatalf("concurrent attendance outcomes: created=%d conflicts=%d", created, conflicts)
		}
		w = httptest.NewRecorder()
		a.publicPresensi(w, qaMultipartRequest(t, http.MethodGet, "/api/presensi/"+sessionID+"/public", domain.User{}, sessionID, nil))
		if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "peserta@example.test") || !strings.Contains(w.Body.String(), `"dataPresensi":[]`) {
			t.Fatalf("public attendance exposed participant data: %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.getPresensi(w, qaMultipartRequest(t, http.MethodGet, "/api/presensi/"+sessionID, users[1], sessionID, nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("foreign user read attendance participants: %d", w.Code)
		}
		w = httptest.NewRecorder()
		a.getPresensi(w, qaMultipartRequest(t, http.MethodGet, "/api/presensi/"+sessionID, users[0], sessionID, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "peserta@example.test") {
			t.Fatalf("owner attendance detail = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.participantDetail(w, qaMultipartRequest(t, http.MethodGet, "/api/participants/"+participantID, domain.User{}, participantID, nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("participant details without token = %d", w.Code)
		}
		w = httptest.NewRecorder()
		a.participantDetail(w, qaMultipartRequest(t, http.MethodGet, "/api/participants/"+participantID+"?token="+accessToken, domain.User{}, participantID, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "peserta@example.test") {
			t.Fatalf("participant token detail = %d: %s", w.Code, w.Body.String())
		}
		statusRequest := httptest.NewRequest(http.MethodPatch, "/api/presensi/"+sessionID+"/status", strings.NewReader(`{"mode":"MANUAL_CLOSE"}`))
		statusRequest.Header.Set("Content-Type", "application/json")
		statusRequest = statusRequest.WithContext(context.WithValue(statusRequest.Context(), userContextKey, authContext{User: users[0]}))
		statusRoute := chi.NewRouteContext()
		statusRoute.URLParams.Add("id", sessionID)
		statusRequest = statusRequest.WithContext(context.WithValue(statusRequest.Context(), chi.RouteCtxKey, statusRoute))
		w = httptest.NewRecorder()
		a.updatePresensiStatus(w, statusRequest)
		if w.Code != http.StatusOK {
			t.Fatalf("close attendance = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.submitPresensi(w, qaMultipartRequest(t, http.MethodPost, "/api/presensi/"+sessionID+"/absensi", domain.User{}, sessionID, map[string]string{"namaLengkap": "Peserta Terlambat", "email": "terlambat@example.test", "noHp": "081234567892", "organisasi": "IPNU"}))
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "PRESENSI_CLOSED") {
			t.Fatalf("closed attendance accepted participant = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.listPresensi(w, qaMultipartRequest(t, http.MethodGet, "/api/presensi?q=QA&status=CLOSED&sortKey=isActive&page=1&limit=1", users[0], "", nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), sessionID) || !strings.Contains(w.Body.String(), `"total":1`) {
			t.Fatalf("closed attendance filtered list = %d: %s", w.Code, w.Body.String())
		}
		updateFields := map[string]string{"namaKegiatan": "QA Presensi Diperbarui", "tempat": "QA", "penyelenggara": "QA", "tanggal": today, "jamMulai": "00:00", "jamSelesai": "23:59"}
		w = httptest.NewRecorder()
		a.updatePresensi(w, qaMultipartRequest(t, http.MethodPatch, "/api/presensi/"+sessionID, users[1], sessionID, updateFields))
		if w.Code != http.StatusNotFound {
			t.Fatalf("foreign attendance update = %d", w.Code)
		}
		w = httptest.NewRecorder()
		a.updatePresensi(w, qaMultipartRequest(t, http.MethodPatch, "/api/presensi/"+sessionID, users[0], sessionID, updateFields))
		if w.Code != http.StatusOK {
			t.Fatalf("owner attendance update = %d: %s", w.Code, w.Body.String())
		}
		w = httptest.NewRecorder()
		a.deletePresensi(w, qaMultipartRequest(t, http.MethodDelete, "/api/presensi/"+sessionID, users[0], sessionID, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("delete attendance = %d: %s", w.Code, w.Body.String())
		}
		var remaining int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "PresensiData" WHERE "presensiId"=$1`, sessionID).Scan(&remaining); err != nil || remaining != 0 {
			t.Fatalf("attendance participants survived deletion: count=%d error=%v", remaining, err)
		}
	})

	t.Run("period activation is isolated by owner", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/periodes", strings.NewReader(`{"nama":"Periode Baru"}`))
		r.Header.Set("Content-Type", "application/json")
		r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: users[0]}))
		w := httptest.NewRecorder()
		a.createPeriode(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("create period status = %d: %s", w.Code, w.Body.String())
		}
		var newID string
		if err := db.QueryRow(ctx, `SELECT "id" FROM "Periode" WHERE "userId"=$1 AND "nama"='Periode Baru'`, users[0].ID).Scan(&newID); err != nil {
			t.Fatal(err)
		}
		for _, check := range []struct {
			user domain.User
			want int
		}{{users[1], http.StatusNotFound}, {users[0], http.StatusOK}} {
			update := httptest.NewRequest(http.MethodPatch, "/api/periodes/"+newID, strings.NewReader(`{"nama":"Periode Diperbarui"}`))
			update.Header.Set("Content-Type", "application/json")
			update = update.WithContext(context.WithValue(update.Context(), userContextKey, authContext{User: check.user}))
			route := chi.NewRouteContext()
			route.URLParams.Add("id", newID)
			update = update.WithContext(context.WithValue(update.Context(), chi.RouteCtxKey, route))
			w = httptest.NewRecorder()
			a.updatePeriode(w, update)
			if w.Code != check.want {
				t.Fatalf("period update for %s = %d: %s", check.user.ID, w.Code, w.Body.String())
			}
		}
		r = httptest.NewRequest(http.MethodPost, "/api/periodes/"+newID+"/activate", nil)
		r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: users[1]}))
		route := chi.NewRouteContext()
		route.URLParams.Add("id", newID)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, route))
		w = httptest.NewRecorder()
		a.activatePeriode(w, r)
		if w.Code != http.StatusNotFound {
			t.Fatalf("cross-user period activation status = %d", w.Code)
		}
		r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: users[0]}))
		w = httptest.NewRecorder()
		a.activatePeriode(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("own period activation status = %d: %s", w.Code, w.Body.String())
		}
		var activeCount int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "Periode" WHERE "userId"=$1 AND "isActive"=true`, users[0].ID).Scan(&activeCount); err != nil || activeCount != 1 {
			t.Fatalf("active period count = %d, error = %v", activeCount, err)
		}
	})

	t.Run("concurrent period creation and activation versus deletion", func(t *testing.T) {
		const userID = "qa_period_race"
		user := domain.User{ID: userID, Name: "Period Race", Role: "SEKRETARIS_PAC", IsActive: true}
		if _, err := db.Exec(ctx, `INSERT INTO "User"("id","name","email","role","isActive","emailVerified","createdAt","updatedAt") VALUES($1,$2,$3,'SEKRETARIS_PAC',true,true,NOW(),NOW())`, userID, user.Name, userID+"@example.test"); err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		statuses := make(chan int, 2)
		var group sync.WaitGroup
		for _, name := range []string{"Periode A", "Periode B"} {
			group.Add(1)
			go func() {
				defer group.Done()
				<-start
				r := httptest.NewRequest(http.MethodPost, "/api/periodes", strings.NewReader(`{"nama":"`+name+`"}`))
				r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: user}))
				w := httptest.NewRecorder()
				a.createPeriode(w, r)
				statuses <- w.Code
			}()
		}
		close(start)
		group.Wait()
		close(statuses)
		for status := range statuses {
			if status != http.StatusCreated {
				t.Fatalf("concurrent period creation status = %d", status)
			}
		}
		var activeID, inactiveID, pointer string
		if err := db.QueryRow(ctx, `SELECT "id" FROM "Periode" WHERE "userId"=$1 AND "isActive"=true`, userID).Scan(&activeID); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(ctx, `SELECT "id" FROM "Periode" WHERE "userId"=$1 AND "isActive"=false`, userID).Scan(&inactiveID); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(ctx, `SELECT "periodeAktifId" FROM "User" WHERE "id"=$1`, userID).Scan(&pointer); err != nil || pointer != activeID {
			t.Fatalf("active period pointer = %q, active row = %q, error = %v", pointer, activeID, err)
		}
		start = make(chan struct{})
		outcomes := make(chan int, 2)
		for _, op := range []struct {
			method, path string
			handler      http.HandlerFunc
		}{
			{http.MethodPost, "/api/periodes/" + inactiveID + "/activate", a.activatePeriode},
			{http.MethodDelete, "/api/periodes/" + inactiveID, a.deletePeriode},
		} {
			group.Add(1)
			go func() {
				defer group.Done()
				<-start
				r := httptest.NewRequest(op.method, op.path, nil)
				r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: user}))
				route := chi.NewRouteContext()
				route.URLParams.Add("id", inactiveID)
				r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, route))
				w := httptest.NewRecorder()
				op.handler(w, r)
				outcomes <- w.Code
			}()
		}
		close(start)
		group.Wait()
		close(outcomes)
		var okCount, expectedFailure int
		for status := range outcomes {
			if status == http.StatusOK {
				okCount++
			} else if status == http.StatusBadRequest || status == http.StatusNotFound {
				expectedFailure++
			} else {
				t.Fatalf("period activation/deletion race status = %d", status)
			}
		}
		if okCount != 1 || expectedFailure != 1 {
			t.Fatalf("period race outcomes: successes=%d rejected=%d", okCount, expectedFailure)
		}
		var activeCount int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "Periode" p JOIN "User" u ON u."periodeAktifId"=p."id" WHERE p."userId"=$1 AND p."isActive"=true AND u."id"=$1`, userID).Scan(&activeCount); err != nil || activeCount != 1 {
			t.Fatalf("period state is inconsistent: active pointer matches=%d, error=%v", activeCount, err)
		}
	})

	t.Run("admin-only agenda and SP document", func(t *testing.T) {
		agendaJSON := `{"judul":"Agenda QA","deskripsi":"Rapat","lokasi":"Kantor","warna":"#123456","tanggalMulai":"2026-10-24"}`
		r := httptest.NewRequest(http.MethodPost, "/api/agenda-kegiatan", strings.NewReader(agendaJSON))
		r.Header.Set("Content-Type", "application/json")
		r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: users[0]}))
		w := httptest.NewRecorder()
		a.createAgenda(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("PAC agenda creation status = %d", w.Code)
		}
		r = httptest.NewRequest(http.MethodPost, "/api/agenda-kegiatan", strings.NewReader(agendaJSON))
		r.Header.Set("Content-Type", "application/json")
		r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: users[2]}))
		w = httptest.NewRecorder()
		a.createAgenda(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("admin agenda creation status = %d: %s", w.Code, w.Body.String())
		}
		var encryptedTitle string
		if err := db.QueryRow(ctx, `SELECT "judul" FROM "AgendaKegiatan" WHERE "userId"=$1`, users[2].ID).Scan(&encryptedTitle); err != nil {
			t.Fatal(err)
		}
		if title, err := crypt.DecryptText(encryptedTitle); err != nil || title != "Agenda QA" {
			t.Fatalf("agenda title = %q, error = %v", title, err)
		}
		spFields := map[string]string{
			"nama": "Sertifikat QA", "tanggalMulai": "2026-10-24", "tanggalBerakhir": "2027-10-24", "organisasi": "IPNU",
		}
		spStore := &backupObjectStore{}
		withSPStore := *a
		withSPStore.r2 = spStore
		r = qaMultipartFileRequest(t, http.MethodPost, "/api/berkas-sp", users[2], "", spFields, "sertifikat.pdf", []byte("%PDF-1.4\n%%EOF\n"))
		w = httptest.NewRecorder()
		withSPStore.createBerkasSP(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("SP document creation status = %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("registration activation login and logout", func(t *testing.T) {
		credentials := `{"name":"Akun QA","email":"akun.qa@example.test","password":"QASecure123!"}`
		r := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(credentials))
		w := httptest.NewRecorder()
		a.register(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("registration status = %d: %s", w.Code, w.Body.String())
		}
		var registeredID string
		if err := db.QueryRow(ctx, `SELECT "id" FROM "User" WHERE "email"='akun.qa@example.test'`).Scan(&registeredID); err != nil {
			t.Fatal(err)
		}
		loginBody := `{"email":"akun.qa@example.test","password":"QASecure123!"}`
		r = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(loginBody))
		w = httptest.NewRecorder()
		a.login(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("inactive account login status = %d", w.Code)
		}
		r = httptest.NewRequest(http.MethodPatch, "/api/users/"+registeredID+"/status", nil)
		r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: users[2]}))
		route := chi.NewRouteContext()
		route.URLParams.Add("id", registeredID)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, route))
		w = httptest.NewRecorder()
		a.toggleUserStatus(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("account activation status = %d: %s", w.Code, w.Body.String())
		}
		r = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(loginBody))
		w = httptest.NewRecorder()
		a.login(w, r)
		if w.Code != http.StatusOK || len(w.Result().Cookies()) == 0 {
			t.Fatalf("active account login status = %d: %s", w.Code, w.Body.String())
		}
		cookie := w.Result().Cookies()[0]
		var storedToken string
		if err := db.QueryRow(ctx, `SELECT "token" FROM "Session" WHERE "userId"=$1`, registeredID).Scan(&storedToken); err != nil {
			t.Fatal(err)
		}
		if storedToken == cookie.Value || !strings.HasPrefix(storedToken, "s1:") {
			t.Fatal("new session token is stored in plaintext")
		}
		r = httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
		r.AddCookie(cookie)
		if user, _, err := a.currentUser(r); err != nil || user.ID != registeredID {
			t.Fatalf("new session identity = %q, error = %v", user.ID, err)
		}
		legacyToken := "g1_legacy_session_qa"
		if _, err := db.Exec(ctx, `INSERT INTO "Session"("id","userId","token","expiresAt","createdAt","updatedAt") VALUES($1,$2,$3,NOW()+INTERVAL '1 hour',NOW(),NOW())`, newID(), registeredID, legacyToken); err != nil {
			t.Fatal(err)
		}
		legacyRequest := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
		legacyRequest.AddCookie(&http.Cookie{Name: a.cfg.SessionCookie, Value: legacyToken})
		if user, _, err := a.currentUser(legacyRequest); err != nil || user.ID != registeredID {
			t.Fatalf("legacy session identity = %q, error = %v", user.ID, err)
		}
		r = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		r.AddCookie(cookie)
		w = httptest.NewRecorder()
		a.logout(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("logout status = %d: %s", w.Code, w.Body.String())
		}
		r = httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
		r.AddCookie(cookie)
		if _, _, err := a.currentUser(r); err == nil {
			t.Fatal("logout did not invalidate database session")
		}
	})

	t.Run("OTP is stored as a digest and consumed once", func(t *testing.T) {
		const otp = "123456"
		email := users[0].ID + "@example.test"
		digest := a.otpDigest(email, otp)
		if strings.Contains(digest, otp) || !strings.HasPrefix(digest, "h1:") {
			t.Fatal("OTP digest is not protected")
		}
		if _, err := db.Exec(ctx, `INSERT INTO "Verification"("id","identifier","value","expiresAt","createdAt","updatedAt") VALUES($1,$2,$3,NOW()+INTERVAL '5 minutes',NOW(),NOW())`, newID(), email, digest); err != nil {
			t.Fatal(err)
		}
		noSMTP := *a
		noSMTP.cfg.SMTPHost = ""
		const attempts = 8
		statuses := make(chan int, attempts)
		var group sync.WaitGroup
		for i := 0; i < attempts; i++ {
			group.Add(1)
			go func() {
				defer group.Done()
				r := httptest.NewRequest(http.MethodPost, "/api/auth/otp/verify", strings.NewReader(`{"email":"`+email+`","otp":"`+otp+`"}`))
				w := httptest.NewRecorder()
				noSMTP.verifyOTP(w, r)
				statuses <- w.Code
			}()
		}
		group.Wait()
		close(statuses)
		verified, rejected := 0, 0
		for status := range statuses {
			switch status {
			case http.StatusOK:
				verified++
			case http.StatusBadRequest:
				rejected++
			default:
				t.Fatalf("unexpected OTP verification status: %d", status)
			}
		}
		if verified != 1 || rejected != attempts-1 {
			t.Fatalf("OTP verification outcomes: verified=%d rejected=%d", verified, rejected)
		}
		deadline := time.Now().Add(5 * time.Second)
		for {
			var notifications int
			if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "LogEmail" WHERE "to"=$1 AND "type"='VERIFIED_SUCCESS'`, email).Scan(&notifications); err != nil {
				t.Fatal(err)
			}
			if notifications == 1 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("OTP success notification did not finish")
			}
			time.Sleep(10 * time.Millisecond)
		}
	})

	t.Run("OTP sending never stores plaintext", func(t *testing.T) {
		const email = "otp.send.qa@example.test"
		if _, err := db.Exec(ctx, `INSERT INTO "User"("id","name","email","role","isActive","emailVerified","createdAt","updatedAt") VALUES($1,'OTP QA',$2,'SEKRETARIS_PAC',false,false,NOW(),NOW())`, newID(), email); err != nil {
			t.Fatal(err)
		}
		noSMTP := *a
		noSMTP.cfg.SMTPHost = ""
		r := httptest.NewRequest(http.MethodPost, "/api/auth/otp/send", strings.NewReader(`{"email":"`+email+`"}`))
		w := httptest.NewRecorder()
		noSMTP.sendOTP(w, r)
		if w.Code != http.StatusBadGateway {
			t.Fatalf("OTP send without SMTP = %d: %s", w.Code, w.Body.String())
		}
		var stored string
		if err := db.QueryRow(ctx, `SELECT "value" FROM "Verification" WHERE "identifier"=$1`, email).Scan(&stored); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(stored, "h1:") || len(stored) != 67 {
			t.Fatalf("OTP is not stored as a digest: %q", stored)
		}
	})

	t.Run("concurrent OTP sends leave one active code", func(t *testing.T) {
		const email = "otp.concurrent.qa@example.test"
		if _, err := db.Exec(ctx, `INSERT INTO "User"("id","name","email","role","isActive","emailVerified","createdAt","updatedAt") VALUES($1,'OTP Concurrent QA',$2,'SEKRETARIS_PAC',false,false,NOW(),NOW())`, newID(), email); err != nil {
			t.Fatal(err)
		}
		noSMTP := *a
		noSMTP.cfg.SMTPHost = ""
		const attempts = 8
		statuses := make(chan int, attempts)
		var group sync.WaitGroup
		for i := 0; i < attempts; i++ {
			group.Add(1)
			go func() {
				defer group.Done()
				r := httptest.NewRequest(http.MethodPost, "/api/auth/otp/send", strings.NewReader(`{"email":"`+email+`"}`))
				w := httptest.NewRecorder()
				noSMTP.sendOTP(w, r)
				statuses <- w.Code
			}()
		}
		group.Wait()
		close(statuses)
		for status := range statuses {
			if status != http.StatusBadGateway {
				t.Fatalf("concurrent OTP send status = %d", status)
			}
		}
		var active int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "Verification" WHERE "identifier"=$1 AND "expiresAt">NOW()`, email).Scan(&active); err != nil {
			t.Fatal(err)
		}
		if active != 1 {
			t.Fatalf("active OTP rows after %d concurrent sends = %d", attempts, active)
		}
	})

	t.Run("OTP send and verify do not deadlock", func(t *testing.T) {
		const email, otp = "otp.race.qa@example.test", "876543"
		if _, err := db.Exec(ctx, `INSERT INTO "User"("id","name","email","role","isActive","emailVerified","createdAt","updatedAt") VALUES($1,'OTP Race QA',$2,'SEKRETARIS_PAC',false,false,NOW(),NOW())`, newID(), email); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, `INSERT INTO "Verification"("id","identifier","value","expiresAt","createdAt","updatedAt") VALUES($1,$2,$3,NOW()+INTERVAL '5 minutes',NOW(),NOW())`, newID(), email, a.otpDigest(email, otp)); err != nil {
			t.Fatal(err)
		}
		noSMTP := *a
		noSMTP.cfg.SMTPHost = ""
		start := make(chan struct{})
		statuses := make(chan int, 2)
		var group sync.WaitGroup
		for _, test := range []struct {
			path, body string
			handler    http.HandlerFunc
		}{
			{"/api/auth/otp/send", `{"email":"` + email + `"}`, noSMTP.sendOTP},
			{"/api/auth/otp/verify", `{"email":"` + email + `","otp":"` + otp + `"}`, noSMTP.verifyOTP},
		} {
			group.Add(1)
			go func() {
				defer group.Done()
				<-start
				requestCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				r := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body)).WithContext(requestCtx)
				w := httptest.NewRecorder()
				test.handler(w, r)
				statuses <- w.Code
			}()
		}
		close(start)
		group.Wait()
		close(statuses)
		var success, rejected, mailFailure int
		for status := range statuses {
			switch status {
			case http.StatusOK:
				success++
			case http.StatusBadRequest:
				rejected++
			case http.StatusBadGateway:
				mailFailure++
			default:
				t.Fatalf("unexpected OTP race response: %d", status)
			}
		}
		if !((success == 1 && rejected == 1) || (rejected == 1 && mailFailure == 1)) {
			t.Fatalf("unexpected OTP race outcomes: success=%d rejected=%d mailFailure=%d", success, rejected, mailFailure)
		}
	})

	t.Run("legacy plaintext OTP remains usable once", func(t *testing.T) {
		const otp = "654321"
		email := users[1].ID + "@example.test"
		if _, err := db.Exec(ctx, `INSERT INTO "Verification"("id","identifier","value","expiresAt","createdAt","updatedAt") VALUES($1,$2,$3,NOW()+INTERVAL '5 minutes',NOW(),NOW())`, newID(), email, otp); err != nil {
			t.Fatal(err)
		}
		noSMTP := *a
		noSMTP.cfg.SMTPHost = ""
		for attempt, want := range []int{http.StatusOK, http.StatusBadRequest} {
			r := httptest.NewRequest(http.MethodPost, "/api/auth/otp/verify", strings.NewReader(`{"email":"`+email+`","otp":"`+otp+`"}`))
			w := httptest.NewRecorder()
			noSMTP.verifyOTP(w, r)
			if w.Code != want {
				t.Fatalf("legacy OTP attempt %d = %d: %s", attempt+1, w.Code, w.Body.String())
			}
		}
		deadline := time.Now().Add(2 * time.Second)
		for {
			var notifications int
			if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM "LogEmail" WHERE "to"=$1 AND "type"='VERIFIED_SUCCESS'`, email).Scan(&notifications); err != nil {
				t.Fatal(err)
			}
			if notifications == 1 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("legacy OTP notification did not finish")
			}
			time.Sleep(10 * time.Millisecond)
		}
	})
}

func qaMultipartRequest(t *testing.T, method, path string, user domain.User, routeID string, fields map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(method, path, &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: user}))
	if routeID != "" {
		route := chi.NewRouteContext()
		route.URLParams.Add("id", routeID)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, route))
	}
	return r
}

func qaMultipartFileRequest(t *testing.T, method, path string, user domain.User, routeID string, fields map[string]string, filename string, content []byte) *http.Request {
	return qaMultipartNamedFileRequest(t, method, path, user, routeID, fields, "file", filename, content)
}

func qaMultipartNamedFileRequest(t *testing.T, method, path string, user domain.User, routeID string, fields map[string]string, field, filename string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	file, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(method, path, &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r = r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: user}))
	if routeID != "" {
		route := chi.NewRouteContext()
		route.URLParams.Add("id", routeID)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, route))
	}
	return r
}

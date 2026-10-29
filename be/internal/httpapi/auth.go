package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"laci-backend/internal/domain"
	"laci-backend/internal/security"
)

var validName = regexp.MustCompile(`^[\p{L}\s.',-]+$`)

const googleStateCookie = "laci_google_state"

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var body loginRequest
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Data login tidak valid")
		return
	}
	body.Email = strings.ToLower(strings.TrimSpace(body.Email))
	if body.Email == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, "INVALID_CREDENTIALS", "Email dan password wajib diisi")
		return
	}

	var user domain.User
	var passwordHash string
	err := a.db.QueryRow(r.Context(), `
		SELECT u."id",u."name",u."email",u."emailVerified",u."image",u."role"::text,u."isActive",
		       u."periodeAktifId",u."createdAt",u."updatedAt",u."lastLogoutAt",a."password"
		FROM "User" u JOIN "Account" a ON a."userId"=u."id" AND a."providerId"='credential'
		WHERE LOWER(u."email")=$1 LIMIT 1`, body.Email).Scan(
		&user.ID, &user.Name, &user.Email, &user.EmailVerified, &user.Image, &user.Role, &user.IsActive,
		&user.PeriodeAktifID, &user.CreatedAt, &user.UpdatedAt, &user.LastLogoutAt, &passwordHash,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Layanan login tidak tersedia")
		return
	}
	if err != nil || !security.VerifyPassword(passwordHash, body.Password) {
		writeError(w, http.StatusUnauthorized, "INVALID_EMAIL_OR_PASSWORD", "Email atau password salah")
		return
	}
	if !user.IsActive {
		writeError(w, http.StatusForbidden, "ACCOUNT_INACTIVE", "Akun Anda belum diaktifkan oleh Sekretaris Cabang")
		return
	}
	session, err := a.createSession(r.Context(), r, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "SESSION_FAILED", "Gagal membuat sesi login")
		return
	}
	a.setSessionCookie(w, session.Token, session.ExpiresAt)
	_ = a.createLog(r.Context(), r, user, "LOGIN", "AUTH", "User login ke sistem: "+user.Name, "")
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"user": user, "session": session}})
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var body registerRequest
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Data pendaftaran tidak valid")
		return
	}
	body.Name = strings.Join(strings.Fields(strings.TrimSpace(body.Name)), " ")
	body.Email = strings.ToLower(strings.TrimSpace(body.Email))
	if len([]rune(body.Name)) < 2 || len([]rune(body.Name)) > 100 || !validName.MatchString(body.Name) {
		writeError(w, http.StatusBadRequest, "INVALID_NAME", "Nama tidak valid")
		return
	}
	address, err := mail.ParseAddress(body.Email)
	if err != nil || len(address.Address) > 255 {
		writeError(w, http.StatusBadRequest, "INVALID_EMAIL", "Format email tidak valid")
		return
	}
	body.Email = strings.ToLower(address.Address)
	if len(body.Password) < 6 || len(body.Password) > 128 {
		writeError(w, http.StatusBadRequest, "INVALID_PASSWORD", "Password minimal 6 karakter")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 10)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "HASH_FAILED", "Gagal memproses password")
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Gagal memulai pendaftaran")
		return
	}
	defer tx.Rollback(r.Context())

	userID, accountID := newID(), newID()
	now := time.Now().UTC()
	_, err = tx.Exec(r.Context(), `INSERT INTO "User"
		("id","name","email","emailVerified","role","isActive","createdAt","updatedAt")
		VALUES($1,$2,$3,false,'SEKRETARIS_PAC',false,$4,$4)`, userID, body.Name, body.Email, now)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "USER_ALREADY_EXISTS", "Email sudah terdaftar")
			return
		}
		writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Gagal membuat akun")
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO "Account"
		("id","userId","accountId","providerId","password","createdAt","updatedAt")
		VALUES($1,$2,$3,'credential',$4,$5,$5)`, accountID, userID, body.Email, string(hash), now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Gagal membuat akun")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Gagal menyelesaikan pendaftaran")
		return
	}
	a.notify(r.Context(), map[string]any{"type": "mutation", "model": "User", "id": userID})

	writeJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{"id": userID, "name": body.Name, "email": body.Email}})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	user, session, err := a.currentUser(r)
	if err != nil && !errors.Is(err, errMissingSession) && !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengakhiri sesi")
		return
	}
	if err == nil {
		tx, err := a.db.Begin(r.Context())
		if err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengakhiri sesi")
			return
		}
		defer tx.Rollback(r.Context())
		if _, err := tx.Exec(r.Context(), `DELETE FROM "Session" WHERE "id"=$1`, session.ID); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengakhiri sesi")
			return
		}
		if _, err := tx.Exec(r.Context(), `UPDATE "User" SET "lastLogoutAt"=NOW(),"updatedAt"=NOW() WHERE "id"=$1`, user.ID); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengakhiri sesi")
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			writeError(w, 500, "DATABASE_ERROR", "Gagal mengakhiri sesi")
			return
		}
		_ = a.createLog(r.Context(), r, user, "LOGOUT", "AUTH", "User logout dari sistem: "+user.Name, "")
	}
	http.SetCookie(w, &http.Cookie{Name: a.cfg.SessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: a.cfg.AppEnv == "production", SameSite: http.SameSiteLaxMode})
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *API) session(w http.ResponseWriter, r *http.Request) {
	user, session, err := a.currentUser(r)
	if err != nil && !errors.Is(err, errMissingSession) && !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Layanan sesi tidak tersedia")
		return
	}
	if err != nil || !user.IsActive {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	writeJSON(w, http.StatusOK, domain.SessionResponse{Session: session, User: user})
}

func (a *API) createSession(ctx context.Context, r *http.Request, userID string) (domain.Session, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return domain.Session{}, err
	}
	session := domain.Session{ID: newID(), UserID: userID, Token: "g1_" + hex.EncodeToString(tokenBytes), ExpiresAt: time.Now().Add(a.cfg.SessionTTL), CreatedAt: time.Now().UTC()}
	_, _ = a.db.Exec(ctx, `DELETE FROM "Session" WHERE "expiresAt"<=NOW()`)
	_, err := a.db.Exec(ctx, `INSERT INTO "Session"("id","userId","expiresAt","token","ipAddress","userAgent","createdAt","updatedAt") VALUES($1,$2,$3,$4,$5,$6,$7,$7)`,
		session.ID, userID, session.ExpiresAt, a.sessionDigest(session.Token), clientIP(r), r.UserAgent(), session.CreatedAt)
	return session, err
}

func (a *API) setSessionCookie(w http.ResponseWriter, value string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{Name: a.cfg.SessionCookie, Value: value, Path: "/", Expires: expires, MaxAge: int(time.Until(expires).Seconds()), HttpOnly: true, Secure: a.cfg.AppEnv == "production", SameSite: http.SameSiteLaxMode})
}

type otpRequest struct {
	Email string `json:"email"`
}

func (a *API) sendOTP(w http.ResponseWriter, r *http.Request) {
	var body otpRequest
	if err := readJSON(r, &body); err != nil {
		writeError(w, 400, "INVALID_REQUEST", "Email tidak valid")
		return
	}
	body.Email = strings.ToLower(strings.TrimSpace(body.Email))
	var name string
	var verified bool
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Layanan verifikasi tidak tersedia")
		return
	}
	defer tx.Rollback(r.Context())
	if err := tx.QueryRow(r.Context(), `SELECT "name","emailVerified" FROM "User" WHERE LOWER("email")=$1 FOR UPDATE`, body.Email).Scan(&name, &verified); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "User tidak ditemukan")
		} else {
			writeError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Layanan verifikasi tidak tersedia")
		}
		return
	}
	if verified {
		writeError(w, http.StatusBadRequest, "ALREADY_VERIFIED", "Email sudah terverifikasi")
		return
	}

	otp, err := randomDigits(6)
	if err != nil {
		writeError(w, 500, "OTP_FAILED", "Gagal membuat OTP")
		return
	}
	if _, err := tx.Exec(r.Context(), `DELETE FROM "Verification" WHERE "identifier"=$1`, body.Email); err != nil {
		writeError(w, 500, "OTP_FAILED", "Gagal membuat OTP")
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO "Verification"("id","identifier","value","expiresAt","createdAt","updatedAt") VALUES($1,$2,$3,NOW()+INTERVAL '5 minutes',NOW(),NOW())`, newID(), body.Email, a.otpDigest(body.Email, otp))
	if err != nil {
		writeError(w, 500, "OTP_FAILED", "Gagal membuat OTP")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "OTP_FAILED", "Gagal membuat OTP")
		return
	}
	subject := "Kode Verifikasi Laci Digital"
	bodyHTML := verificationEmailHTML(name, otp)
	if err := a.sendMail(r.Context(), body.Email, subject, bodyHTML, "VERIFICATION", map[string]any{"name": name}); err != nil {
		writeError(w, http.StatusBadGateway, "EMAIL_FAILED", "Gagal mengirim email verifikasi")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

type verifyOTPRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

func (a *API) verifyOTP(w http.ResponseWriter, r *http.Request) {
	var body verifyOTPRequest
	if err := readJSON(r, &body); err != nil {
		writeError(w, 400, "INVALID_REQUEST", "Data OTP tidak valid")
		return
	}
	body.Email = strings.ToLower(strings.TrimSpace(body.Email))
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal memverifikasi email")
		return
	}
	defer tx.Rollback(r.Context())
	var id string
	otp := strings.TrimSpace(body.OTP)
	var userID string
	if err := tx.QueryRow(r.Context(), `SELECT "id" FROM "User" WHERE LOWER("email")=$1 FOR UPDATE`, body.Email).Scan(&userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "INVALID_OTP", "Kode OTP salah atau sudah kedaluwarsa")
		} else {
			writeError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Layanan verifikasi tidak tersedia")
		}
		return
	}
	if err := tx.QueryRow(r.Context(), `DELETE FROM "Verification" WHERE "identifier"=$1 AND "value" IN ($2,$3) AND "expiresAt">NOW() RETURNING "id"`, body.Email, a.otpDigest(body.Email, otp), otp).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "INVALID_OTP", "Kode OTP salah atau sudah kedaluwarsa")
		} else {
			writeError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Layanan verifikasi tidak tersedia")
		}
		return
	}
	result, err := tx.Exec(r.Context(), `UPDATE "User" SET "emailVerified"=true,"updatedAt"=NOW() WHERE "id"=$1`, userID)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal memverifikasi email")
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, 404, "USER_NOT_FOUND", "User tidak ditemukan")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal memverifikasi email")
		return
	}
	var name string
	if a.db.QueryRow(r.Context(), `SELECT "name" FROM "User" WHERE LOWER("email")=$1`, body.Email).Scan(&name) == nil {
		a.sendBackgroundMail(r.Context(), func(ctx context.Context) {
			if err := a.sendMail(ctx, body.Email, "Email Berhasil Diverifikasi - Laci Digital", verifiedSuccessEmailHTML(name), "VERIFIED_SUCCESS", map[string]any{"name": name}); err != nil {
				slog.Warn("verification confirmation email failed", "error", err)
			}
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *API) verificationStatus(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("email")))
	var verified bool
	if err := a.db.QueryRow(r.Context(), `SELECT "emailVerified" FROM "User" WHERE LOWER("email")=$1`, email).Scan(&verified); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "User tidak ditemukan")
		} else {
			writeError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Layanan verifikasi tidak tersedia")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"verified": verified})
}

func (a *API) googleConfig() *oauth2.Config {
	return &oauth2.Config{ClientID: a.cfg.GoogleClientID, ClientSecret: a.cfg.GoogleClientSecret, RedirectURL: a.cfg.GoogleRedirectURL, Scopes: []string{"openid", "email", "profile"}, Endpoint: google.Endpoint}
}

func (a *API) googleStart(w http.ResponseWriter, r *http.Request) {
	if a.cfg.GoogleClientID == "" {
		writeError(w, 503, "GOOGLE_DISABLED", "Login Google belum dikonfigurasi")
		return
	}
	callback := r.URL.Query().Get("callback")
	if !validCallbackURL(callback, a.cfg.FrontendURL) {
		callback = a.cfg.FrontendURL + "/dashboard?login=success"
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		writeError(w, http.StatusInternalServerError, "OAUTH_STATE_FAILED", "Gagal memulai login Google")
		return
	}
	stateNonce := hex.EncodeToString(nonce)
	claims := jwt.MapClaims{"callback": callback, "nonce": stateNonce, "exp": time.Now().Add(10 * time.Minute).Unix()}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	state, err := token.SignedString([]byte(a.cfg.SessionSecret))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "OAUTH_STATE_FAILED", "Gagal memulai login Google")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: googleStateCookie, Value: stateNonce, Path: "/api/auth", MaxAge: 600, HttpOnly: true, Secure: a.cfg.AppEnv == "production", SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, a.googleConfig().AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func (a *API) validGoogleState(r *http.Request) (string, bool) {
	parsed, err := jwt.Parse(r.URL.Query().Get("state"), func(token *jwt.Token) (any, error) { return []byte(a.cfg.SessionSecret), nil }, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !parsed.Valid {
		return "", false
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return "", false
	}
	nonce, ok := claims["nonce"].(string)
	if !ok || len(nonce) != 64 {
		return "", false
	}
	cookie, err := r.Cookie(googleStateCookie)
	if err != nil || subtle.ConstantTimeCompare([]byte(nonce), []byte(cookie.Value)) != 1 {
		return "", false
	}
	callback, _ := claims["callback"].(string)
	return callback, true
}

func (a *API) googleCallback(w http.ResponseWriter, r *http.Request) {
	callback, valid := a.validGoogleState(r)
	http.SetCookie(w, &http.Cookie{Name: googleStateCookie, Value: "", Path: "/api/auth", MaxAge: -1, HttpOnly: true, Secure: a.cfg.AppEnv == "production", SameSite: http.SameSiteLaxMode})
	if !valid {
		http.Redirect(w, r, a.cfg.FrontendURL+"/login?error=oauth_state", http.StatusTemporaryRedirect)
		return
	}
	oauthToken, err := a.googleConfig().Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Redirect(w, r, a.cfg.FrontendURL+"/login?error=oauth", http.StatusTemporaryRedirect)
		return
	}
	request, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	request.Header.Set("Authorization", "Bearer "+oauthToken.AccessToken)
	response, err := outboundClient.Do(request)
	if err != nil {
		http.Redirect(w, r, a.cfg.FrontendURL+"/login?error=oauth", http.StatusTemporaryRedirect)
		return
	}
	defer response.Body.Close()
	var info struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Picture       string `json:"picture"`
	}
	if json.NewDecoder(response.Body).Decode(&info) != nil || !info.EmailVerified {
		http.Redirect(w, r, a.cfg.FrontendURL+"/login?error=oauth", http.StatusTemporaryRedirect)
		return
	}
	var user domain.User
	err = a.db.QueryRow(r.Context(), `SELECT "id","name","email","emailVerified","image","role"::text,"isActive","periodeAktifId","createdAt","updatedAt","lastLogoutAt" FROM "User" WHERE LOWER("email")=$1`, strings.ToLower(info.Email)).Scan(
		&user.ID, &user.Name, &user.Email, &user.EmailVerified, &user.Image, &user.Role, &user.IsActive, &user.PeriodeAktifID, &user.CreatedAt, &user.UpdatedAt, &user.LastLogoutAt)
	if err != nil {
		http.Redirect(w, r, a.cfg.FrontendURL+"/login?error=unregistered", http.StatusTemporaryRedirect)
		return
	}
	if !user.IsActive {
		http.Redirect(w, r, a.cfg.FrontendURL+"/login?error=account_inactive", http.StatusTemporaryRedirect)
		return
	}
	if _, err := a.db.Exec(r.Context(), `UPDATE "User" SET "image"=COALESCE("image",NULLIF($2,'')),"emailVerified"=true,"updatedAt"=NOW() WHERE "id"=$1`, user.ID, info.Picture); err != nil {
		http.Redirect(w, r, a.cfg.FrontendURL+"/login?error=oauth", http.StatusTemporaryRedirect)
		return
	}
	session, err := a.createSession(r.Context(), r, user.ID)
	if err != nil {
		http.Redirect(w, r, a.cfg.FrontendURL+"/login?error=session", http.StatusTemporaryRedirect)
		return
	}
	a.setSessionCookie(w, session.Token, session.ExpiresAt)
	_ = a.createLog(r.Context(), r, user, "LOGIN", "AUTH", "User login ke sistem via Google: "+user.Name, "")
	if !validCallbackURL(callback, a.cfg.FrontendURL) {
		callback = a.cfg.FrontendURL + "/dashboard?login=success"
	}
	http.Redirect(w, r, callback, http.StatusTemporaryRedirect)
}

func (a *API) sendMail(ctx context.Context, to, subject, htmlBody, emailType string, metadata map[string]any) error {
	logID := newID()
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	if _, err := a.db.Exec(ctx, `INSERT INTO "LogEmail"("id","to","subject","type","status","metadata","createdAt","updatedAt") VALUES($1,$2,$3,$4::"EmailType",'PENDING',$5,NOW(),NOW())`, logID, to, subject, emailType, string(metadataJSON)); err != nil {
		return err
	}
	if a.cfg.SMTPHost == "" || a.cfg.MailFromAddress == "" {
		err := fmt.Errorf("SMTP configuration missing")
		if updateErr := a.setEmailLogStatus(ctx, logID, "FAILED", err.Error()); updateErr != nil {
			slog.Error("email status update failed", "error", updateErr)
		}
		return err
	}
	from := a.cfg.MailFromAddress
	cleanHeader := strings.NewReplacer("\r", " ", "\n", " ")
	message := []byte("From: " + cleanHeader.Replace(a.cfg.MailFromName) + " <" + cleanHeader.Replace(from) + ">\r\n" + "To: " + cleanHeader.Replace(to) + "\r\n" + "Subject: " + cleanHeader.Replace(subject) + "\r\n" + "MIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n" + htmlBody)
	err = sendSMTP(ctx, a.cfg.SMTPHost, a.cfg.SMTPPort, a.cfg.SMTPUser, a.cfg.SMTPPass, from, to, message)
	if err != nil {
		if updateErr := a.setEmailLogStatus(ctx, logID, "FAILED", err.Error()); updateErr != nil {
			slog.Error("email status update failed", "error", updateErr)
		}
		return err
	}
	if err := a.setEmailLogStatus(ctx, logID, "SENT", ""); err != nil {
		slog.Error("email sent but status update failed", "error", err)
	}
	return nil
}

func (a *API) setEmailLogStatus(ctx context.Context, id, status, message string) error {
	updateCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	_, err := a.db.Exec(updateCtx, `UPDATE "LogEmail" SET "status"=$2::"EmailStatus","errorMessage"=NULLIF($3,''),"updatedAt"=NOW() WHERE "id"=$1`, id, status, message)
	return err
}

func validCallbackURL(callback, frontend string) bool {
	candidate, err := url.Parse(callback)
	if err != nil || !candidate.IsAbs() {
		return false
	}
	allowed, err := url.Parse(frontend)
	return err == nil && candidate.Scheme == allowed.Scheme && candidate.Host == allowed.Host
}

func newID() string {
	b := make([]byte, 15)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Errorf("generate random ID: %w", err))
	}
	return "c" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b))
}

func randomDigits(n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		for {
			var raw [1]byte
			if _, err := rand.Read(raw[:]); err != nil {
				return "", err
			}
			if raw[0] < 250 {
				b[i] = '0' + raw[0]%10
				break
			}
		}
	}
	return string(b), nil
}

func (a *API) otpDigest(email, otp string) string {
	mac := hmac.New(sha256.New, []byte(a.cfg.SessionSecret))
	_, _ = mac.Write([]byte(email + ":" + otp))
	return "h1:" + hex.EncodeToString(mac.Sum(nil))
}

func (a *API) sessionDigest(token string) string {
	mac := hmac.New(sha256.New, []byte(a.cfg.SessionSecret))
	_, _ = mac.Write([]byte("session:" + token))
	return "s1:" + hex.EncodeToString(mac.Sum(nil))
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer := net.ParseIP(host)
	if peer == nil {
		return host
	}
	if peer.IsLoopback() {
		forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
		if ip := net.ParseIP(forwarded); ip != nil {
			return ip.String()
		}
	}
	return peer.String()
}

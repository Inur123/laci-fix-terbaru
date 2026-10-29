package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

var serviceKeys = []string{
	"API_KEY", "CRON_SECRET", "GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET",
	"RECAPTCHA_SECRET_KEY", "SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASS",
	"MAIL_FROM_ADDRESS", "MAIL_FROM_NAME", "ADMIN_NOTIFICATION_EMAIL",
	"R2_ACCOUNT_ID", "R2_ACCESS_KEY_ID", "R2_SECRET_ACCESS_KEY", "R2_BUCKET_NAME",
}

func main() {
	source := flag.String("source", "", "original Next.js .env")
	backendPath := flag.String("backend", ".env", "Go backend .env")
	frontendPath := flag.String("frontend", "../fe/.env.local", "Next.js frontend .env.local")
	flag.Parse()
	if *source == "" {
		fail(errors.New("-source is required"))
	}

	legacy, err := godotenv.Read(*source)
	if err != nil {
		fail(err)
	}
	backend, err := godotenv.Read(*backendPath)
	if err != nil {
		fail(err)
	}
	if backend["DATABASE_URL"] == "" || backend["DATABASE_URL"] == legacy["DATABASE_URL"] {
		fail(errors.New("backend must point to its restored database, not the original database"))
	}
	if backend["ENCRYPTION_KEY"] == "" || backend["ENCRYPTION_KEY"] != legacy["ENCRYPTION_KEY"] {
		fail(errors.New("backend ENCRYPTION_KEY does not match the original application"))
	}
	if len(backend["SESSION_SECRET"]) < 32 {
		fail(errors.New("backend SESSION_SECRET is missing or too short"))
	}
	localDatabase, localSession := backend["DATABASE_URL"], backend["SESSION_SECRET"]
	for _, key := range serviceKeys {
		if legacy[key] == "" {
			fail(fmt.Errorf("original .env is missing %s", key))
		}
		backend[key] = legacy[key]
	}
	if legacy["NEXT_PUBLIC_RECAPTCHA_SITE_KEY"] == "" {
		fail(errors.New("original .env is missing NEXT_PUBLIC_RECAPTCHA_SITE_KEY"))
	}
	frontendURL := strings.TrimRight(backend["FRONTEND_URL"], "/")
	if frontendURL == "" {
		fail(errors.New("backend FRONTEND_URL is required"))
	}
	backend["GOOGLE_REDIRECT_URL"] = frontendURL + "/api/auth/callback/google"

	frontend := map[string]string{
		"BACKEND_URL":                    "http://127.0.0.1:8080",
		"API_KEY":                        legacy["API_KEY"],
		"NEXT_PUBLIC_RECAPTCHA_SITE_KEY": legacy["NEXT_PUBLIC_RECAPTCHA_SITE_KEY"],
	}
	if current, err := godotenv.Read(*frontendPath); err == nil {
		for key, value := range current {
			if _, managed := frontend[key]; !managed {
				frontend[key] = value
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		fail(err)
	}
	if err := writePrivate(*backendPath, backend); err != nil {
		fail(err)
	}
	if err := writePrivate(*frontendPath, frontend); err != nil {
		fail(err)
	}
	updatedBackend, err := godotenv.Read(*backendPath)
	if err != nil {
		fail(err)
	}
	updatedFrontend, err := godotenv.Read(*frontendPath)
	if err != nil {
		fail(err)
	}
	if updatedBackend["DATABASE_URL"] != localDatabase || updatedBackend["SESSION_SECRET"] != localSession {
		fail(errors.New("local database or session secret changed unexpectedly"))
	}
	for _, key := range serviceKeys {
		if updatedBackend[key] != legacy[key] {
			fail(fmt.Errorf("backend %s did not round-trip", key))
		}
	}
	if updatedFrontend["API_KEY"] != legacy["API_KEY"] || updatedFrontend["NEXT_PUBLIC_RECAPTCHA_SITE_KEY"] != legacy["NEXT_PUBLIC_RECAPTCHA_SITE_KEY"] {
		fail(errors.New("frontend service settings did not round-trip"))
	}
	fmt.Println("Service settings synchronized; backend database and session secret were preserved.")
}

func writePrivate(path string, values map[string]string) error {
	content, err := godotenv.Marshal(values)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".env-sync-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.WriteString(content + "\n"); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

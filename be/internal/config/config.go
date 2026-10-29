package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                 string
	HTTPAddr               string
	FrontendURL            string
	DatabaseURL            string
	APIKey                 string
	SessionSecret          string
	SessionCookie          string
	SessionTTL             time.Duration
	EncryptionKey          string
	GoogleClientID         string
	GoogleClientSecret     string
	GoogleRedirectURL      string
	RecaptchaSecret        string
	SMTPHost               string
	SMTPPort               int
	SMTPUser               string
	SMTPPass               string
	MailFromAddress        string
	MailFromName           string
	AdminNotificationEmail string
	R2AccountID            string
	R2AccessKeyID          string
	R2SecretAccessKey      string
	R2BucketName           string
	CronSecret             string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	ttl, err := time.ParseDuration(value("SESSION_TTL", "6h"))
	if err != nil {
		return Config{}, errors.New("SESSION_TTL must be a Go duration such as 6h")
	}
	port, err := strconv.Atoi(value("SMTP_PORT", "587"))
	if err != nil {
		return Config{}, errors.New("SMTP_PORT must be numeric")
	}

	cfg := Config{
		AppEnv:                 value("APP_ENV", "development"),
		HTTPAddr:               value("HTTP_ADDR", ":8080"),
		FrontendURL:            strings.TrimRight(value("FRONTEND_URL", "http://localhost:3000"), "/"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		APIKey:                 os.Getenv("API_KEY"),
		SessionSecret:          os.Getenv("SESSION_SECRET"),
		SessionCookie:          value("SESSION_COOKIE", "laci_session"),
		SessionTTL:             ttl,
		EncryptionKey:          os.Getenv("ENCRYPTION_KEY"),
		GoogleClientID:         os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:     os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:      os.Getenv("GOOGLE_REDIRECT_URL"),
		RecaptchaSecret:        os.Getenv("RECAPTCHA_SECRET_KEY"),
		SMTPHost:               os.Getenv("SMTP_HOST"),
		SMTPPort:               port,
		SMTPUser:               os.Getenv("SMTP_USER"),
		SMTPPass:               os.Getenv("SMTP_PASS"),
		MailFromAddress:        os.Getenv("MAIL_FROM_ADDRESS"),
		MailFromName:           value("MAIL_FROM_NAME", "Laci Digital"),
		AdminNotificationEmail: os.Getenv("ADMIN_NOTIFICATION_EMAIL"),
		R2AccountID:            os.Getenv("R2_ACCOUNT_ID"),
		R2AccessKeyID:          os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretAccessKey:      os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2BucketName:           os.Getenv("R2_BUCKET_NAME"),
		CronSecret:             os.Getenv("CRON_SECRET"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if len(cfg.SessionSecret) < 32 {
		return Config{}, errors.New("SESSION_SECRET is required and must contain at least 32 characters")
	}
	if cfg.EncryptionKey == "" {
		return Config{}, errors.New("ENCRYPTION_KEY is required and must match the old application")
	}
	return cfg, nil
}

func value(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

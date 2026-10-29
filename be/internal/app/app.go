package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"laci-backend/internal/config"
	"laci-backend/internal/httpapi"
	"laci-backend/internal/security"
	"laci-backend/internal/storage"
)

type App struct {
	db             *pgxpool.Pool
	router         http.Handler
	api            *httpapi.API
	cancelListener context.CancelFunc
	cancelStreams  context.CancelFunc
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	poolCfg, err := databasePoolConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	db, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.Ping(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect database: %w", err)
	}

	crypto, err := security.NewCrypto(cfg.EncryptionKey)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize encryption: %w", err)
	}
	r2, err := storage.NewR2(ctx, cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2BucketName)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize R2: %w", err)
	}

	api := httpapi.New(cfg, db, crypto, r2)
	streamCtx, cancelStreams := context.WithCancel(context.Background())
	api.SetStreamShutdown(streamCtx.Done())
	listenerCtx, cancelListener := context.WithCancel(ctx)
	api.StartRealtimeListener(listenerCtx)
	return &App{db: db, router: api.Router(), api: api, cancelListener: cancelListener, cancelStreams: cancelStreams}, nil
}

func databasePoolConfig(databaseURL string) (*pgxpool.Config, error) {
	poolCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = 20
	poolCfg.MinConns = 1
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.MaxConnLifetimeJitter = 5 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	if poolCfg.ConnConfig.ConnectTimeout == 0 {
		poolCfg.ConnConfig.ConnectTimeout = 5 * time.Second
	}
	if poolCfg.ConnConfig.RuntimeParams == nil {
		poolCfg.ConnConfig.RuntimeParams = make(map[string]string)
	}
	if _, ok := poolCfg.ConnConfig.RuntimeParams["statement_timeout"]; !ok {
		poolCfg.ConnConfig.RuntimeParams["statement_timeout"] = "60000"
	}
	if _, ok := poolCfg.ConnConfig.RuntimeParams["lock_timeout"]; !ok {
		poolCfg.ConnConfig.RuntimeParams["lock_timeout"] = "10000"
	}
	return poolCfg, nil
}

func (a *App) Router() http.Handler { return a.router }

func (a *App) StopStreams() { a.cancelStreams() }

func (a *App) Close() {
	a.StopStreams()
	a.cancelListener()
	a.api.CloseBackgroundMail()
	a.db.Close()
}

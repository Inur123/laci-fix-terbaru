package app

import (
	"testing"
	"time"
)

func TestDatabasePoolConfigBoundsResourcesAndPreservesOverrides(t *testing.T) {
	tests := []struct {
		name, databaseURL string
		connectTimeout    time.Duration
		statementTimeout  string
		lockTimeout       string
	}{
		{"defaults", "postgres://localhost/laci", 5 * time.Second, "60000", "10000"},
		{"explicit overrides", "postgres://localhost/laci?connect_timeout=9&statement_timeout=120000&lock_timeout=25000", 9 * time.Second, "120000", "25000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := databasePoolConfig(tt.databaseURL)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.MaxConns != 20 || cfg.MinConns != 1 || cfg.MaxConnLifetime != 30*time.Minute || cfg.MaxConnLifetimeJitter != 5*time.Minute || cfg.MaxConnIdleTime != 5*time.Minute {
				t.Fatalf("unexpected pool bounds: max=%d min=%d lifetime=%s jitter=%s idle=%s", cfg.MaxConns, cfg.MinConns, cfg.MaxConnLifetime, cfg.MaxConnLifetimeJitter, cfg.MaxConnIdleTime)
			}
			if cfg.ConnConfig.ConnectTimeout != tt.connectTimeout || cfg.ConnConfig.RuntimeParams["statement_timeout"] != tt.statementTimeout || cfg.ConnConfig.RuntimeParams["lock_timeout"] != tt.lockTimeout {
				t.Fatalf("unexpected timeouts: connect=%s statement=%q lock=%q", cfg.ConnConfig.ConnectTimeout, cfg.ConnConfig.RuntimeParams["statement_timeout"], cfg.ConnConfig.RuntimeParams["lock_timeout"])
			}
		})
	}
}

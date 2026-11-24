package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"laci-backend/internal/config"
	"laci-backend/internal/security"
)

func TestPublicKegiatanCORS(t *testing.T) {
	a := New(config.Config{FrontendURL: "http://localhost:3000"}, nil, nil, nil)
	router := a.Router()

	// 1. Test OPTIONS preflight
	req := httptest.NewRequest(http.MethodOptions, "/api/kegiatan", nil)
	req.Header.Set("Origin", "https://pelajarnumagetan.blogspot.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 for OPTIONS /api/kegiatan, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "GET, OPTIONS" {
		t.Errorf("expected Access-Control-Allow-Methods: 'GET, OPTIONS', got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type" {
		t.Errorf("expected Access-Control-Allow-Headers: 'Content-Type', got %q", got)
	}
}

func TestPublicKegiatanJSONSerialization(t *testing.T) {
	selesaiStr := "2026-11-25T15:00:00Z"
	lokasiStr := "Gedung PCNU Magetan"

	// Item with all fields
	itemFull := publicKegiatanItem{
		ID:             "cm12345",
		Judul:          "Konfercab IPNU IPPNU",
		TanggalMulai:   "2026-11-25T08:00:00Z",
		TanggalSelesai: &selesaiStr,
		Warna:          "#22c55e",
		Lokasi:         &lokasiStr,
		User: publicKegiatanUser{
			Name: "Sekretaris Cabang",
		},
	}

	// Item with null optional fields
	itemNulls := publicKegiatanItem{
		ID:             "cm67890",
		Judul:          "Rapat Koordinasi",
		TanggalMulai:   "2026-12-01T10:00:00Z",
		TanggalSelesai: nil,
		Warna:          "#3b82f6",
		Lokasi:         nil,
		User: publicKegiatanUser{
			Name: "PAC Sukomoro",
		},
	}

	res := publicKegiatanResponse{
		Success: true,
		Data:    []publicKegiatanItem{itemFull, itemNulls},
	}

	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	jsonStr := string(raw)

	if !strings.Contains(jsonStr, `"success":true`) {
		t.Errorf("missing success:true in %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"lokasi":null`) {
		t.Errorf("expected null lokasi for second item in %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"tanggal_selesai":null`) {
		t.Errorf("expected null tanggal_selesai for second item in %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"tanggal_mulai":"2026-11-25T08:00:00Z"`) {
		t.Errorf("missing formatted tanggal_mulai in %s", jsonStr)
	}
}

func TestPublicKegiatanEndpoint(t *testing.T) {
	_ = godotenv.Load("../../.env")
	cfg, err := config.Load()
	if err != nil || cfg.DatabaseURL == "" {
		t.Skip("skipping integration test: database configuration not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		t.Skipf("cannot parse db config: %v", err)
	}
	poolCfg.MaxConns = 2
	db, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		t.Skipf("cannot connect to db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		t.Skipf("db ping failed: %v", err)
	}

	crypto, err := security.NewCrypto(cfg.EncryptionKey)
	if err != nil {
		t.Fatalf("crypto init failed: %v", err)
	}

	a := New(cfg, db, crypto, nil)
	router := a.Router()

	req := httptest.NewRequest(http.MethodGet, "/api/kegiatan", nil)
	req.Header.Set("Origin", "https://pelajarnumagetan.blogspot.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *, got %q", got)
	}

	var resp publicKegiatanResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode json response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success: true, got false")
	}

	// Verify all items are properly decrypted and formatted
	for i, item := range resp.Data {
		if item.ID == "" {
			t.Errorf("item[%d] ID is empty", i)
		}
		if item.Judul == "" {
			t.Errorf("item[%d] Judul is empty", i)
		}
		// Encrypted string typically contains ":" and hex ciphertext, ensure it is decrypted
		if strings.Contains(item.Judul, ":") && len(item.Judul) > 60 {
			t.Errorf("item[%d] Judul seems encrypted: %s", i, item.Judul)
		}
		if _, err := time.Parse(time.RFC3339, item.TanggalMulai); err != nil {
			t.Errorf("item[%d] invalid tanggal_mulai RFC3339 format %q: %v", i, item.TanggalMulai, err)
		}
		if item.TanggalSelesai != nil {
			if _, err := time.Parse(time.RFC3339, *item.TanggalSelesai); err != nil {
				t.Errorf("item[%d] invalid tanggal_selesai RFC3339 format %q: %v", i, *item.TanggalSelesai, err)
			}
		}
		if item.Lokasi != nil && strings.Contains(*item.Lokasi, ":") && len(*item.Lokasi) > 60 {
			t.Errorf("item[%d] Lokasi seems encrypted: %s", i, *item.Lokasi)
		}
		if item.User.Name == "" {
			t.Errorf("item[%d] User.Name is empty", i)
		}
	}
}

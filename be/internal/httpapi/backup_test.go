package httpapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"laci-backend/internal/domain"
	"laci-backend/internal/security"
	"laci-backend/internal/storage"
)

type backupObjectStore struct {
	*storage.R2
	value       []byte
	openErr     error
	opened      int
	putKey      string
	putSize     int64
	putErr      error
	objects     []storage.Object
	deleted     []string
	signedKey   string
	signedValue string
}

func (s *backupObjectStore) Put(_ context.Context, key, _ string, value []byte) error {
	if s.putErr != nil {
		return s.putErr
	}
	s.putKey = key
	s.value = bytes.Clone(value)
	return nil
}

func (s *backupObjectStore) Get(_ context.Context, key string) ([]byte, error) {
	if key != s.putKey {
		return nil, errors.New("object not found")
	}
	return bytes.Clone(s.value), nil
}

func (s *backupObjectStore) PutReader(_ context.Context, key, _ string, body io.Reader, size int64) error {
	if s.putErr != nil {
		return s.putErr
	}
	s.putKey = key
	s.putSize = size
	value, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	s.value = value
	return nil
}

func (s *backupObjectStore) List(_ context.Context, _ string) ([]storage.Object, error) {
	return s.objects, nil
}

func (s *backupObjectStore) Delete(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return nil
}

func (s *backupObjectStore) Open(_ context.Context, _ string) (io.ReadCloser, error) {
	s.opened++
	if s.openErr != nil {
		return nil, s.openErr
	}
	return io.NopCloser(bytes.NewReader(s.value)), nil
}

func (s *backupObjectStore) SignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	s.signedKey = key
	return s.signedValue, nil
}

func backupRequest(path, role string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	return r.WithContext(context.WithValue(r.Context(), userContextKey, authContext{User: domain.User{Role: role}}))
}

func TestBackupDownloadURLPreservesLegacyAndSecuresEncryptedBackups(t *testing.T) {
	store := &backupObjectStore{signedValue: "https://example.test/signed-legacy-backup"}
	a := &API{r2: store}
	legacy := "backups/laci-old.dump"
	w := httptest.NewRecorder()
	a.backupDownloadURL(w, backupRequest("/api/backups/download?key="+url.QueryEscape(legacy), "SEKRETARIS_CABANG"))
	if w.Code != http.StatusOK || store.signedKey != legacy || !strings.Contains(w.Body.String(), store.signedValue) {
		t.Fatalf("legacy download: status=%d, key=%q, body=%s", w.Code, store.signedKey, w.Body.String())
	}
	w = httptest.NewRecorder()
	a.backupDownloadURL(w, backupRequest("/api/backups/download?key=backups%2Flaci-new.dump.enc", "SEKRETARIS_CABANG"))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "/api/backups/file?key=backups%2Flaci-new.dump.enc") {
		t.Fatalf("encrypted download URL: status=%d, body=%s", w.Code, w.Body.String())
	}
	if store.signedKey != legacy {
		t.Fatal("encrypted backup must not receive a public R2 signed URL")
	}
	for _, key := range []string{"backups/../secret.dump.enc", "backups/nested/file.dump.enc", "elsewhere/file.dump.enc", "backups/file\".dump.enc", "backups/file\t.dump.enc"} {
		w = httptest.NewRecorder()
		a.backupDownloadURL(w, backupRequest("/api/backups/download?key="+url.QueryEscape(key), "SEKRETARIS_CABANG"))
		if w.Code != http.StatusBadRequest {
			t.Errorf("invalid key %q accepted: %d", key, w.Code)
		}
	}
}

func TestEncryptedBackupDownloadRequiresRoleAndAuthenticatesContent(t *testing.T) {
	c, err := security.NewCrypto("backup-download-test-key")
	if err != nil {
		t.Fatal(err)
	}
	plain := bytes.Repeat([]byte("pgdump"), 200000)
	var encrypted bytes.Buffer
	if err := c.EncryptBackup(context.Background(), &encrypted, bytes.NewReader(plain)); err != nil {
		t.Fatal(err)
	}
	store := &backupObjectStore{value: encrypted.Bytes()}
	a := &API{r2: store, crypto: c}
	path := "/api/backups/file?key=backups%2Flaci-new.dump.enc"
	w := httptest.NewRecorder()
	a.downloadEncryptedBackup(w, backupRequest(path, "ANGGOTA"))
	if w.Code != http.StatusForbidden || store.opened != 0 {
		t.Fatalf("wrong role: status=%d opens=%d", w.Code, store.opened)
	}
	w = httptest.NewRecorder()
	a.downloadEncryptedBackup(w, backupRequest(path, "SEKRETARIS_CABANG"))
	if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), plain) {
		t.Fatalf("valid download: status=%d, bytes=%d", w.Code, w.Body.Len())
	}
	if w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Header().Get("Content-Disposition"), "laci-new.dump") {
		t.Fatalf("download headers: %v", w.Header())
	}
	store.value = bytes.Clone(store.value)
	store.value[len(store.value)-1] ^= 1
	w = httptest.NewRecorder()
	a.downloadEncryptedBackup(w, backupRequest(path, "SEKRETARIS_CABANG"))
	if w.Code != http.StatusBadGateway || bytes.Contains(w.Body.Bytes(), plain[:10]) {
		t.Fatalf("tampered backup leaked plaintext: status=%d body=%s", w.Code, w.Body.String())
	}
	store.openErr = errors.New("storage unavailable")
	w = httptest.NewRecorder()
	a.downloadEncryptedBackup(w, backupRequest(path, "SEKRETARIS_CABANG"))
	if w.Code != http.StatusBadGateway {
		t.Fatalf("storage error status=%d", w.Code)
	}
}

func TestOldestBackupKeysKeepsTenIncludingNewBackup(t *testing.T) {
	objects := []storage.Object{{Key: "backups/", LastModified: time.Unix(0, 0)}}
	for i := 0; i < 12; i++ {
		objects = append(objects, storage.Object{Key: "backups/backup-" + string(rune('a'+i)) + ".dump", LastModified: time.Unix(int64(i), 0)})
	}
	keys := oldestBackupKeys(objects)
	if len(keys) != 3 || keys[0] != "backups/backup-a.dump" || keys[1] != "backups/backup-b.dump" || keys[2] != "backups/backup-c.dump" {
		t.Fatalf("rotation selected %v", keys)
	}
	if keys := oldestBackupKeys(objects[:10]); len(keys) != 0 {
		t.Fatalf("fewer than ten existing backups selected for deletion: %v", keys)
	}
}

func TestBackupListIsNewestFirst(t *testing.T) {
	store := &backupObjectStore{objects: []storage.Object{
		{Key: "backups/older.dump", LastModified: time.Unix(10, 0)},
		{Key: "backups/newer.dump.enc", LastModified: time.Unix(20, 0)},
		{Key: "backups/"},
	}}
	a := &API{r2: store}
	w := httptest.NewRecorder()
	a.listBackups(w, backupRequest("/api/backups", "SEKRETARIS_CABANG"))
	newIndex := strings.Index(w.Body.String(), "newer.dump.enc")
	oldIndex := strings.Index(w.Body.String(), "older.dump")
	if w.Code != http.StatusOK || newIndex < 0 || oldIndex < 0 || newIndex >= oldIndex || strings.Contains(w.Body.String(), `"filename":""`) {
		t.Fatalf("backup list: status=%d body=%s", w.Code, w.Body.String())
	}
}

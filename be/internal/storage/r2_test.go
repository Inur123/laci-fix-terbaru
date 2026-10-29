package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
	"laci-backend/internal/security"
)

func TestObjectRoundTripAgainstS3CompatibleEndpoint(t *testing.T) {
	stored := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/qa/arsip/example.enc" {
			t.Errorf("unexpected object path: %s", r.URL.Path)
		}
		switch r.Method {
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			stored = string(body)
			w.Header().Set("ETag", `"qa"`)
		case http.MethodGet:
			_, _ = io.WriteString(w, stored)
		case http.MethodDelete:
			stored = ""
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method: %s", r.Method)
		}
	}))
	defer server.Close()
	cfg := aws.Config{Region: "auto", Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider("test", "test", "")), HTTPClient: server.Client()}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(server.URL)
		options.UsePathStyle = true
	})
	store := &R2{client: client, bucket: "qa", ready: true}
	const key = "arsip/example.enc"
	const payload = "encrypted test bytes"
	if err := store.Put(context.Background(), key, "application/octet-stream", []byte(payload)); err != nil {
		t.Fatal(err)
	}
	if stored != payload {
		t.Fatalf("stored body = %q", stored)
	}
	got, err := store.Get(context.Background(), key)
	if err != nil || string(got) != payload {
		t.Fatalf("read body = %q, error = %v", got, err)
	}
	stream, err := store.Open(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	streamed, readErr := io.ReadAll(stream)
	closeErr := stream.Close()
	if readErr != nil || closeErr != nil || string(streamed) != payload {
		t.Fatalf("streamed body = %q, read error = %v, close error = %v", streamed, readErr, closeErr)
	}
	url, err := store.SignedURL(context.Background(), key, 5*time.Minute)
	if err != nil || !strings.Contains(url, "X-Amz-Signature=") {
		t.Fatalf("signed URL = %q, error = %v", url, err)
	}
	if err := store.Delete(context.Background(), key); err != nil || stored != "" {
		t.Fatalf("delete object: stored=%q error=%v", stored, err)
	}
}

func TestListReadsEveryObjectPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("prefix") != "backups/" {
			t.Errorf("unexpected prefix: %q", r.URL.Query().Get("prefix"))
		}
		w.Header().Set("Content-Type", "application/xml")
		switch r.URL.Query().Get("continuation-token") {
		case "":
			_, _ = fmt.Fprint(w, `<ListBucketResult><IsTruncated>true</IsTruncated><NextContinuationToken>next</NextContinuationToken><Contents><Key>backups/first.dump</Key><Size>1</Size><LastModified>2026-10-24T00:00:00.000Z</LastModified></Contents></ListBucketResult>`)
		case "next":
			_, _ = fmt.Fprint(w, `<ListBucketResult><IsTruncated>false</IsTruncated><Contents><Key>backups/second.dump</Key><Size>2</Size><LastModified>2026-10-24T00:00:00.000Z</LastModified></Contents></ListBucketResult>`)
		default:
			t.Errorf("unexpected continuation token: %q", r.URL.Query().Get("continuation-token"))
		}
	}))
	defer server.Close()
	cfg := aws.Config{Region: "auto", Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider("test", "test", "")), HTTPClient: server.Client()}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(server.URL)
		options.UsePathStyle = true
	})
	store := &R2{client: client, bucket: "qa", ready: true}
	objects, err := store.List(context.Background(), "backups/")
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 2 || objects[0].Key != "backups/first.dump" || objects[1].Key != "backups/second.dump" {
		t.Fatalf("unexpected objects: %#v", objects)
	}
}

func TestLiveR2CanListBackupsReadOnly(t *testing.T) {
	if os.Getenv("LACI_RUN_EXTERNAL_TESTS") != "1" {
		t.Skip("set LACI_RUN_EXTERNAL_TESTS=1 to check the configured R2 bucket")
	}
	_ = godotenv.Load("../../.env")
	accountID := os.Getenv("R2_ACCOUNT_ID")
	accessKey := os.Getenv("R2_ACCESS_KEY_ID")
	secretKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucket := os.Getenv("R2_BUCKET_NAME")
	if accountID == "" || accessKey == "" || secretKey == "" || bucket == "" {
		t.Skip("R2 credentials are not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	store, err := NewR2(ctx, accountID, accessKey, secretKey, bucket)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.List(ctx, "backups/"); err != nil {
		t.Fatalf("read-only backup listing failed: %v", err)
	}
}

func TestLiveR2EncryptedBackupRoundTrip(t *testing.T) {
	if os.Getenv("LACI_RUN_EXTERNAL_WRITE_TESTS") != "1" {
		t.Skip("set LACI_RUN_EXTERNAL_WRITE_TESTS=1 for a temporary encrypted R2 object test")
	}
	_ = godotenv.Load("../../.env")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, err := NewR2(ctx, os.Getenv("R2_ACCOUNT_ID"), os.Getenv("R2_ACCESS_KEY_ID"), os.Getenv("R2_SECRET_ACCESS_KEY"), os.Getenv("R2_BUCKET_NAME"))
	if err != nil {
		t.Fatal(err)
	}
	crypto, err := security.NewCrypto("r2-backup-roundtrip-test-key")
	if err != nil {
		t.Fatal(err)
	}
	key := fmt.Sprintf("qa/backup-encryption-%d.dump.enc", time.Now().UnixNano())
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if err := store.Delete(cleanupCtx, key); err != nil {
			t.Errorf("remove temporary R2 object %s: %v", key, err)
		}
	}()
	plain := bytes.Repeat([]byte("synthetic-backup-data"), 60000)
	var encrypted bytes.Buffer
	if err := crypto.EncryptBackup(ctx, &encrypted, bytes.NewReader(plain)); err != nil {
		t.Fatal(err)
	}
	if err := store.PutReader(ctx, key, "application/octet-stream", bytes.NewReader(encrypted.Bytes()), int64(encrypted.Len())); err != nil {
		t.Fatal(err)
	}
	object, err := store.Open(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	var decrypted bytes.Buffer
	decryptErr := crypto.DecryptBackup(ctx, &decrypted, object)
	closeErr := object.Close()
	if decryptErr != nil || closeErr != nil || !bytes.Equal(decrypted.Bytes(), plain) {
		t.Fatalf("R2 encrypted backup round trip: decrypt=%v close=%v", decryptErr, closeErr)
	}
}

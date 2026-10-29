package security

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
)

func TestBackupEncryptionRoundTripAndAuthentication(t *testing.T) {
	c, err := NewCrypto("backup-test-key")
	if err != nil {
		t.Fatal(err)
	}
	other, err := NewCrypto("wrong-key")
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{0, 1, backupChunkSize, backupChunkSize + 13, 2 * backupChunkSize} {
		t.Run(fmt.Sprintf("size-%d", size), func(t *testing.T) {
			plain := bytes.Repeat([]byte("data"), (size+3)/4)[:size]
			var encrypted bytes.Buffer
			if err := c.EncryptBackup(context.Background(), &encrypted, bytes.NewReader(plain)); err != nil {
				t.Fatal(err)
			}
			var decrypted bytes.Buffer
			if err := c.DecryptBackup(context.Background(), &decrypted, bytes.NewReader(encrypted.Bytes())); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(decrypted.Bytes(), plain) {
				t.Fatal("backup round trip changed data")
			}
			if err := other.DecryptBackup(context.Background(), io.Discard, bytes.NewReader(encrypted.Bytes())); err == nil {
				t.Fatal("wrong encryption key accepted")
			}
			for _, cut := range []int{0, 8, 16, len(encrypted.Bytes()) - 1} {
				if err := c.DecryptBackup(context.Background(), io.Discard, bytes.NewReader(encrypted.Bytes()[:cut])); err == nil {
					t.Fatalf("truncated backup accepted at %d bytes", cut)
				}
			}
			tampered := bytes.Clone(encrypted.Bytes())
			tampered[len(tampered)-1] ^= 1
			if err := c.DecryptBackup(context.Background(), io.Discard, bytes.NewReader(tampered)); err == nil {
				t.Fatal("modified authentication tag accepted")
			}
			withTrailing := append(bytes.Clone(encrypted.Bytes()), 1)
			if err := c.DecryptBackup(context.Background(), io.Discard, bytes.NewReader(withTrailing)); err == nil {
				t.Fatal("trailing data accepted")
			}
		})
	}
}

func TestBackupEncryptionHonorsCancellationAndWriterFailure(t *testing.T) {
	c, err := NewCrypto("backup-test-key")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.EncryptBackup(ctx, io.Discard, bytes.NewReader(nil)); !errors.Is(err, context.Canceled) {
		t.Fatalf("encryption cancellation = %v", err)
	}
	var encrypted bytes.Buffer
	if err := c.EncryptBackup(context.Background(), &encrypted, bytes.NewReader([]byte("data"))); err != nil {
		t.Fatal(err)
	}
	if err := c.DecryptBackup(ctx, io.Discard, bytes.NewReader(encrypted.Bytes())); !errors.Is(err, context.Canceled) {
		t.Fatalf("decryption cancellation = %v", err)
	}
	if err := c.DecryptBackup(context.Background(), shortWriter{}, bytes.NewReader(encrypted.Bytes())); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write = %v", err)
	}
}

type shortWriter struct{}

func (shortWriter) Write(value []byte) (int, error) { return len(value) - 1, nil }

func BenchmarkBackupEncrypt1MiB(b *testing.B) {
	c, err := NewCrypto("backup-benchmark-key")
	if err != nil {
		b.Fatal(err)
	}
	plain := bytes.Repeat([]byte("data"), backupChunkSize/4)
	b.SetBytes(int64(len(plain)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := c.EncryptBackup(context.Background(), io.Discard, bytes.NewReader(plain)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBackupDecrypt1MiB(b *testing.B) {
	c, err := NewCrypto("backup-benchmark-key")
	if err != nil {
		b.Fatal(err)
	}
	plain := bytes.Repeat([]byte("data"), backupChunkSize/4)
	var encrypted bytes.Buffer
	if err := c.EncryptBackup(context.Background(), &encrypted, bytes.NewReader(plain)); err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(plain)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := c.DecryptBackup(context.Background(), io.Discard, bytes.NewReader(encrypted.Bytes())); err != nil {
			b.Fatal(err)
		}
	}
}

package security

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

func TestTextAndFileRoundTrip(t *testing.T) {
	crypto, err := NewCrypto("test-encryption-key")
	if err != nil {
		t.Fatal(err)
	}
	plain := "Data anggota: こんにちは"
	encrypted, err := crypto.EncryptText(plain)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := crypto.DecryptText(encrypted)
	if err != nil || decrypted != plain {
		t.Fatalf("text round trip failed: %v", err)
	}
	if !strings.HasPrefix(encrypted, textVersion) {
		t.Fatal("new text must use authenticated format")
	}
	file, err := crypto.EncryptFile([]byte(plain))
	if err != nil {
		t.Fatal(err)
	}
	content, err := crypto.DecryptFile(file)
	if err != nil || string(content) != plain {
		t.Fatalf("file round trip failed: %v", err)
	}
	if !bytes.HasPrefix(file, []byte(fileVersion)) {
		t.Fatal("new file must use authenticated format")
	}
}

func TestAuthenticatedEncryptionRejectsTampering(t *testing.T) {
	c, err := NewCrypto("test-encryption-key")
	if err != nil {
		t.Fatal(err)
	}
	textValue, err := c.EncryptText("sensitive")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(textValue, textVersion))
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)-1] ^= 1
	if _, err := c.DecryptText(textVersion + base64.RawURLEncoding.EncodeToString(raw)); err == nil {
		t.Fatal("tampered text accepted")
	}
	file, err := c.EncryptFile([]byte("sensitive"))
	if err != nil {
		t.Fatal(err)
	}
	file[len(file)-1] ^= 1
	if _, err := c.DecryptFile(file); err == nil {
		t.Fatal("tampered file accepted")
	}
}

func TestLegacyCBCDataRemainsReadable(t *testing.T) {
	c, err := NewCrypto("test-encryption-key")
	if err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		t.Fatal(err)
	}
	iv := bytes.Repeat([]byte{7}, aes.BlockSize)
	plain := []byte("data lama")
	textCipher := make([]byte, len(pkcs7Pad(append([]byte(nil), plain...), aes.BlockSize)))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(textCipher, pkcs7Pad(append([]byte(nil), plain...), aes.BlockSize))
	legacyText := hex.EncodeToString(iv) + ":" + hex.EncodeToString(textCipher)
	if got, err := c.DecryptText(legacyText); err != nil || got != string(plain) {
		t.Fatalf("legacy text unreadable: %v", err)
	}
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	if _, err := zw.Write(plain); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	padded := pkcs7Pad(compressed.Bytes(), aes.BlockSize)
	fileCipher := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(fileCipher, padded)
	legacyFile := append(append([]byte(nil), iv...), fileCipher...)
	if got, err := c.DecryptFile(legacyFile); err != nil || string(got) != string(plain) {
		t.Fatalf("legacy file unreadable: %v", err)
	}
}

func TestDownloadTokenRejectsTamperingAndExpiry(t *testing.T) {
	crypto, err := NewCrypto("test-encryption-key")
	if err != nil {
		t.Fatal(err)
	}
	token, err := crypto.DownloadToken("document-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if id, ok := crypto.VerifyDownloadToken(token); !ok || id != "document-1" {
		t.Fatal("valid token was rejected")
	}
	decoded, _ := base64.RawURLEncoding.DecodeString(token)
	tampered := base64.RawURLEncoding.EncodeToString([]byte(strings.Replace(string(decoded), "document-1", "document-2", 1)))
	if _, ok := crypto.VerifyDownloadToken(tampered); ok {
		t.Fatal("tampered token was accepted")
	}
	expired, err := crypto.DownloadToken("document-1", -time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := crypto.VerifyDownloadToken(expired); ok {
		t.Fatal("expired token was accepted")
	}
}

func BenchmarkEncryptText256Bytes(b *testing.B) {
	c, err := NewCrypto("benchmark-encryption-key")
	if err != nil {
		b.Fatal(err)
	}
	plain := strings.Repeat("x", 256)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.EncryptText(plain); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecryptText256Bytes(b *testing.B) {
	c, err := NewCrypto("benchmark-encryption-key")
	if err != nil {
		b.Fatal(err)
	}
	value, err := c.EncryptText(strings.Repeat("x", 256))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.DecryptText(value); err != nil {
			b.Fatal(err)
		}
	}
}

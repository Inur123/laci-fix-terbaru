package security

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/scrypt"
)

const encryptionSalt = "laci-ipnu-ippnu-salt-2025"
const textVersion = "v2:"
const fileVersion = "LACI2GCM"
const maxDecryptedFileSize = 64 << 20

type Crypto struct {
	key []byte
}

func NewCrypto(secret string) (*Crypto, error) {
	key, err := scrypt.Key([]byte(secret), []byte(encryptionSalt), 16384, 8, 1, 32)
	if err != nil {
		return nil, err
	}
	return &Crypto{key: key}, nil
}

func (c *Crypto) EncryptText(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	aead, err := c.newAEAD()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return textVersion + base64.RawURLEncoding.EncodeToString(aead.Seal(nonce, nonce, []byte(plain), nil)), nil
}

func (c *Crypto) DecryptText(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, textVersion) {
		encoded, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, textVersion))
		if err != nil {
			return "", errors.New("invalid encrypted text")
		}
		aead, err := c.newAEAD()
		if err != nil {
			return "", err
		}
		if len(encoded) < aead.NonceSize()+aead.Overhead() {
			return "", errors.New("invalid encrypted text")
		}
		plain, err := aead.Open(nil, encoded[:aead.NonceSize()], encoded[aead.NonceSize():], nil)
		if err != nil {
			return "", errors.New("encrypted text authentication failed")
		}
		return string(plain), nil
	}
	return c.decryptLegacyText(value)
}

func (c *Crypto) newAEAD() (cipher.AEAD, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (c *Crypto) decryptLegacyText(value string) (string, error) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid encrypted text")
	}
	iv, err := hex.DecodeString(parts[0])
	if err != nil || len(iv) != aes.BlockSize {
		return "", errors.New("invalid encryption iv")
	}
	encrypted, err := hex.DecodeString(parts[1])
	if err != nil || len(encrypted)%aes.BlockSize != 0 {
		return "", errors.New("invalid encrypted payload")
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	plain := make([]byte, len(encrypted))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, encrypted)
	plain, err = pkcs7Unpad(plain, aes.BlockSize)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (c *Crypto) EncryptFile(plain []byte) ([]byte, error) {
	if len(plain) > maxDecryptedFileSize {
		return nil, errors.New("file too large")
	}
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	if _, err := zw.Write(plain); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}

	aead, err := c.newAEAD()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	encrypted := aead.Seal(nil, nonce, compressed.Bytes(), nil)
	result := make([]byte, 0, len(fileVersion)+len(nonce)+len(encrypted))
	result = append(result, fileVersion...)
	result = append(result, nonce...)
	return append(result, encrypted...), nil
}

func (c *Crypto) DecryptFile(value []byte) ([]byte, error) {
	var compressed []byte
	if bytes.HasPrefix(value, []byte(fileVersion)) {
		aead, err := c.newAEAD()
		if err != nil {
			return nil, err
		}
		value = value[len(fileVersion):]
		if len(value) < aead.NonceSize()+aead.Overhead() {
			return nil, errors.New("invalid encrypted file")
		}
		compressed, err = aead.Open(nil, value[:aead.NonceSize()], value[aead.NonceSize():], nil)
		if err != nil {
			return nil, errors.New("encrypted file authentication failed")
		}
	} else {
		var err error
		compressed, err = c.decryptLegacyFile(value)
		if err != nil {
			return nil, err
		}
	}
	zr, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	plain, err := io.ReadAll(io.LimitReader(zr, maxDecryptedFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(plain) > maxDecryptedFileSize {
		return nil, errors.New("decrypted file too large")
	}
	return plain, nil
}

func (c *Crypto) decryptLegacyFile(value []byte) ([]byte, error) {
	if len(value) < aes.BlockSize || (len(value)-aes.BlockSize)%aes.BlockSize != 0 {
		return nil, errors.New("invalid encrypted file")
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}
	iv := value[:aes.BlockSize]
	encrypted := value[aes.BlockSize:]
	decrypted := make([]byte, len(encrypted))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(decrypted, encrypted)
	decrypted, err = pkcs7Unpad(decrypted, aes.BlockSize)
	if err != nil {
		return nil, err
	}
	return decrypted, nil
}

func (c *Crypto) DownloadToken(id string, validFor time.Duration) (string, error) {
	payload := fmt.Sprintf("%s:%d", id, time.Now().Add(validFor).UnixMilli())
	signature := hmac.New(sha256.New, c.key)
	_, _ = signature.Write([]byte(payload))
	value := payload + "." + hex.EncodeToString(signature.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(value)), nil
}

func (c *Crypto) VerifyDownloadToken(token string) (string, bool) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", false
	}
	value := string(decoded)
	dot := strings.LastIndex(value, ".")
	if dot <= 0 {
		return "", false
	}
	payload, signatureHex := value[:dot], value[dot+1:]
	provided, err := hex.DecodeString(signatureHex)
	if err != nil {
		return "", false
	}
	expected := hmac.New(sha256.New, c.key)
	_, _ = expected.Write([]byte(payload))
	if !hmac.Equal(provided, expected.Sum(nil)) {
		return "", false
	}
	idx := strings.LastIndex(payload, ":")
	if idx <= 0 {
		return "", false
	}
	var expires int64
	if _, err := fmt.Sscanf(payload[idx+1:], "%d", &expires); err != nil || expires < time.Now().UnixMilli() {
		return "", false
	}
	return payload[:idx], true
}

func Hash(value string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(value))))
	return hex.EncodeToString(sum[:])
}

func EncryptedFilename(original string) string {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		panic(fmt.Errorf("generate encrypted filename: %w", err))
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(original), "."))
	if ext == "" {
		ext = "bin"
	}
	return fmt.Sprintf("%d-%s-%s.enc", time.Now().UnixMilli(), hex.EncodeToString(random), ext)
}

func OriginalExtension(key string) string {
	name := strings.TrimSuffix(filepath.Base(key), ".enc")
	parts := strings.Split(name, "-")
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return "bin"
}

func pkcs7Pad(value []byte, blockSize int) []byte {
	padding := blockSize - len(value)%blockSize
	return append(value, bytes.Repeat([]byte{byte(padding)}, padding)...)
}

func pkcs7Unpad(value []byte, blockSize int) ([]byte, error) {
	if len(value) == 0 || len(value)%blockSize != 0 {
		return nil, errors.New("invalid padding")
	}
	padding := int(value[len(value)-1])
	if padding == 0 || padding > blockSize || padding > len(value) {
		return nil, errors.New("invalid padding")
	}
	for _, b := range value[len(value)-padding:] {
		if int(b) != padding {
			return nil, errors.New("invalid padding")
		}
	}
	return value[:len(value)-padding], nil
}

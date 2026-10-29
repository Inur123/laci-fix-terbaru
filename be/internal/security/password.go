package security

import (
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/scrypt"
	"golang.org/x/text/unicode/norm"
)

// VerifyPassword accepts hashes created by Better Auth 1.5.6 and new bcrypt hashes.
func VerifyPassword(hash, password string) bool {
	if strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$") || strings.HasPrefix(hash, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}

	salt, encodedKey, ok := strings.Cut(hash, ":")
	if !ok || len(salt) != 32 || len(encodedKey) != 128 {
		return false
	}
	if _, err := hex.DecodeString(salt); err != nil {
		return false
	}
	expected, err := hex.DecodeString(encodedKey)
	if err != nil {
		return false
	}

	// Better Auth passes the 32-character hex salt as UTF-8 text to scrypt.
	derived, err := scrypt.Key([]byte(norm.NFKC.String(password)), []byte(salt), 16384, 16, 1, 64)
	return err == nil && subtle.ConstantTimeCompare(derived, expected) == 1
}

package security

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestVerifyPasswordBetterAuth(t *testing.T) {
	// Fixture generated independently with Node.js crypto.scryptSync and NFKC.
	const hash = "0123456789abcdef0123456789abcdef:5b34bd5d097940a3a198cf728ae9ec7b1836edc2792420d90e459dce91777c57cdf4ad2812d882e8d8126db81702f9f83a549d34a1e3aeac9a4a9685b25edf2e"
	if !VerifyPassword(hash, "Pässwörd１23") {
		t.Fatal("Better Auth password should match")
	}
	if VerifyPassword(hash, "wrong password") {
		t.Fatal("incorrect password should not match")
	}
	if VerifyPassword("invalid:hash", "anything") {
		t.Fatal("malformed hash should not match")
	}
}

func TestVerifyPasswordBcrypt(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password baru"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(string(hash), "password baru") || VerifyPassword(string(hash), "wrong password") {
		t.Fatal("bcrypt verification failed")
	}
}

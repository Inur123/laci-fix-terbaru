package httpapi

import (
	"strings"
	"testing"
)

func TestEmailTemplatesEscapeUserData(t *testing.T) {
	name := `<script>alert("x")</script>`
	data := pengajuanEmailData{
		UserName:  name,
		PACName:   name,
		Number:    "A-1",
		Date:      "23/10/2026",
		DetailURL: "javascript:alert(1)",
		Status:    "DITOLAK",
		Reason:    name,
	}
	for _, body := range []string{
		verificationEmailHTML(name, "123456"),
		verifiedSuccessEmailHTML(name),
		pengajuanCreatedUserEmailHTML(data),
		pengajuanCreatedAdminEmailHTML(data),
		pengajuanStatusEmailHTML(data),
	} {
		if body == "" || !strings.Contains(body, "Laci Digital") {
			t.Fatal("email template was not rendered")
		}
		if strings.Contains(body, name) || strings.Contains(body, `href="javascript:`) {
			t.Fatal("email template contains unescaped user input")
		}
	}
}

func TestVerificationEmailMatchesOTPValidity(t *testing.T) {
	body := verificationEmailHTML("Rekan", "123456")
	if !strings.Contains(body, "5 menit") || strings.Contains(body, "10 menit") {
		t.Fatal("email expiration differs from OTP database validity")
	}
}

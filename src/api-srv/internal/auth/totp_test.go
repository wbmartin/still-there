package auth

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

// ValidateTOTPCodeAt accepts the step before now, now's and the next, and
// says which one matched; the replay check in /mfaVerify depends on it.
func TestValidateTOTPCodeAt(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_015, 0) // mid-step
	current := now.Unix() / 30
	codeAt := func(step int64) string {
		t.Helper()
		code, err := totp.GenerateCode(secret, time.Unix(step*30, 0))
		if err != nil {
			t.Fatal(err)
		}
		return code
	}

	for _, offset := range []int64{-1, 0, 1} {
		step, valid := ValidateTOTPCodeAt(codeAt(current+offset), secret, now)
		if !valid || step != current+offset {
			t.Errorf("offset %d: got (%d, %v), want (%d, true)",
				offset, step, valid, current+offset)
		}
	}
	for _, offset := range []int64{-2, 2} {
		if _, valid := ValidateTOTPCodeAt(
			codeAt(current+offset), secret, now,
		); valid {
			t.Errorf("offset %d: want refused", offset)
		}
	}
	code := codeAt(current)
	for _, bad := range []string{code[:5], code + "0", "12345a", "", "abcdef"} {
		if _, valid := ValidateTOTPCodeAt(bad, secret, now); valid {
			t.Errorf("%q: want refused", bad)
		}
	}
	if _, valid := ValidateTOTPCodeAt(code, "not base32!", now); valid {
		t.Error("bad secret: want refused")
	}
}

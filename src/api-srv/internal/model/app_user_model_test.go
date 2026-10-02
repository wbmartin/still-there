package model

import "testing"

// MFAEnrolled decides what /auth tells the web app and whether /mfaSetup
// refuses: a stored seed that a code has been verified against.
func TestAppUserModelMFAEnrolled(t *testing.T) {
	seed, empty := "ciphertext", ""
	testCaseList := []struct {
		name     string
		mfaSetup bool
		seed     *string
		want     bool
	}{
		{"verified seed", true, &seed, true},
		{"seed not yet verified", false, &seed, false},
		{"admin reset: empty seed", true, &empty, false},
		{"no seed", false, nil, false},
		{"mfa_setup with no seed", true, nil, false},
	}
	for _, testCase := range testCaseList {
		user := AppUserModel{
			MFASetup: testCase.mfaSetup, TOTPSeed: testCase.seed,
		}
		if got := user.MFAEnrolled(); got != testCase.want {
			t.Errorf(
				"%s: MFAEnrolled() = %v, want %v",
				testCase.name, got, testCase.want,
			)
		}
	}
}

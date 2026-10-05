package model

import (
	"encoding/json"
	"strings"
	"testing"
)

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

// The admin shape (/userList, /user) never carries the preferences
// document or the two secrets; the /me shape carries the preferences and
// nothing administrative.
func TestAppUserModelWireShapes(t *testing.T) {
	seed := "encrypted-seed-value"
	user := AppUserModel{
		AppUserID:       7,
		UserName:        "someone",
		PwHash:          "argon2id$hash",
		TOTPSeed:        &seed,
		SecurityContext: 2,
		AccountStatus:   AccountStatusActive,
		PreferenceJSONB: json.RawMessage(`{"colorScheme":"dark"}`),
		UpdateTS:        1.5,
	}

	adminEncoded, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("marshal admin shape: %v", err)
	}
	adminBody := string(adminEncoded)
	for _, forbidden := range []string{
		"preferenceJsonb", "colorScheme", "pwHash", user.PwHash,
		"totpSeed", seed,
	} {
		if strings.Contains(adminBody, forbidden) {
			t.Errorf("%q in the admin shape: %s", forbidden, adminBody)
		}
	}

	ownEncoded, err := json.Marshal(user.OwnProfile())
	if err != nil {
		t.Fatalf("marshal own profile: %v", err)
	}
	ownBody := string(ownEncoded)
	for _, expected := range []string{
		`"preferenceJsonb":{"colorScheme":"dark"}`, `"appUserId":7`,
		`"userName":"someone"`, `"securityContext":2`, `"updateTs":1.5`,
	} {
		if !strings.Contains(ownBody, expected) {
			t.Errorf("%q missing from the own profile: %s", expected, ownBody)
		}
	}
	for _, forbidden := range []string{
		"pwHash", user.PwHash, "totpSeed", seed, "accountStatus",
		"mfaExempt", "appUserGroupMembershipList",
	} {
		if strings.Contains(ownBody, forbidden) {
			t.Errorf("%q in the own profile: %s", forbidden, ownBody)
		}
	}

	// None saved (SQL NULL) leaves the key off.
	user.PreferenceJSONB = nil
	ownEncoded, err = json.Marshal(user.OwnProfile())
	if err != nil {
		t.Fatalf("marshal own profile: %v", err)
	}
	if strings.Contains(string(ownEncoded), "preferenceJsonb") {
		t.Errorf("preferenceJsonb present with none saved: %s", ownEncoded)
	}
}

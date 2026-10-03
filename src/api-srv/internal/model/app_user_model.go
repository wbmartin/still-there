package model

import "encoding/json"

// AppUserModel represents an application user
type AppUserModel struct {
	AppUserID                  int     `json:"appUserId"                  db:"app_user_id"`
	UserName                   string  `json:"userName"                   db:"user_name"`
	PwHash                     string  `json:"-"                          db:"pw_hash"` // Never serialize
	FirstName                  string  `json:"firstName"                  db:"first_name"`
	LastName                   string  `json:"lastName"                   db:"last_name"`
	Email                      string  `json:"email"                      db:"email"`
	SecurityContext            int     `json:"securityContext"            db:"security_context_id"`
	AppUserGroupMembershipList []int   `json:"appUserGroupMembershipList" db:"app_user_group_membership_list"`
	MFAExempt                  bool    `json:"mfaExempt"                  db:"mfa_exempt"`
	MFASetup                   bool    `json:"mfaSetup"                   db:"mfa_setup"`
	TOTPSeed                   *string `json:"-"                          db:"totp_seed"` // Never serialize
	AccountStatus              string  `json:"accountStatus"              db:"account_status"`
	// Free-form per-user preferences, owned by the user through PUT /me.
	PreferenceJSONB json.RawMessage `json:"preferenceJsonb,omitempty" db:"preference_jsonb"`
	CreateTS        float64         `json:"createTs"                   db:"create_ts"`
	CreateUserID    int             `json:"createUserId"               db:"create_user_id"`
	UpdateTS        float64         `json:"updateTs"                   db:"update_ts"`
	UpdateUserID    int             `json:"updateUserId"               db:"update_user_id"`
	DeleteTS        *float64        `json:"deleteTs,omitempty"         db:"delete_ts"`
	DeleteUserID    *int            `json:"deleteUserId,omitempty"     db:"delete_user_id"`
}

// MFAEnrolled reports a completed TOTP enrolment: a seed is stored and a code
// from it has been verified once (mfa_setup). A seed without mfa_setup is an
// enrolment still in progress, which /mfaSetup may show again; a completed
// one it never shows.
func (u *AppUserModel) MFAEnrolled() bool {
	return u.MFASetup && u.TOTPSeed != nil && *u.TOTPSeed != ""
}

// The account_status values (B034). An active login signs in; a blocked one
// was active until an admin blocked it, and neither signs in (/auth refuses
// anything not active) nor, through the BLOCKED* KV flag, keeps a live
// session until unblocked.
const (
	AccountStatusActive  = "active"
	AccountStatusBlocked = "blocked"
)

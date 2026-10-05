package model

import "encoding/json"

// AppUserModel represents an application user. It is the administration
// wire shape (/userList, /user). PreferenceJSONB is `json:"-"`: nothing in
// it is secret, but READ_USER is no reason to read someone else's UI
// settings, so they travel only on the caller's own /me
// (AppUserOwnProfileModel).
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
	PreferenceJSONB json.RawMessage `json:"-"                          db:"preference_jsonb"` // Own UI state, /me only
	CreateTS        float64         `json:"createTs"                   db:"create_ts"`
	CreateUserID    int             `json:"createUserId"               db:"create_user_id"`
	UpdateTS        float64         `json:"updateTs"                   db:"update_ts"`
	UpdateUserID    int             `json:"updateUserId"               db:"update_user_id"`
	DeleteTS        *float64        `json:"deleteTs,omitempty"         db:"delete_ts"`
	DeleteUserID    *int            `json:"deleteUserId,omitempty"     db:"delete_user_id"`
}

// AppUserOwnProfileModel is what GET and PUT /me return: the caller's own
// row, with the preferences document the admin shape leaves off. The API
// checks only that the document is a JSON object within its size bound;
// the web app owns the keys inside (types/profile-type.ts). None saved is
// SQL NULL, which leaves the key off the wire.
type AppUserOwnProfileModel struct {
	AppUserID       int             `json:"appUserId"`
	UserName        string          `json:"userName"`
	FirstName       string          `json:"firstName"`
	LastName        string          `json:"lastName"`
	Email           string          `json:"email"`
	SecurityContext int             `json:"securityContext"`
	PreferenceJSONB json.RawMessage `json:"preferenceJsonb,omitempty"`
	UpdateTS        float64         `json:"updateTs"`
	UpdateUserID    int             `json:"updateUserId"`
}

// OwnProfile is the /me view of the row.
func (u *AppUserModel) OwnProfile() AppUserOwnProfileModel {
	return AppUserOwnProfileModel{
		AppUserID:       u.AppUserID,
		UserName:        u.UserName,
		FirstName:       u.FirstName,
		LastName:        u.LastName,
		Email:           u.Email,
		SecurityContext: u.SecurityContext,
		PreferenceJSONB: u.PreferenceJSONB,
		UpdateTS:        u.UpdateTS,
		UpdateUserID:    u.UpdateUserID,
	}
}

// MFAEnrolled reports a completed TOTP enrolment: a seed is stored and a code
// from it has been verified once (mfa_setup). A seed without mfa_setup is an
// enrolment still in progress, which /mfaSetup may show again; a completed
// one it never shows.
func (u *AppUserModel) MFAEnrolled() bool {
	return u.MFASetup && u.TOTPSeed != nil && *u.TOTPSeed != ""
}

// The account_status values. An active login signs in. A blocked one (B034)
// was active until an admin blocked it, and neither signs in (/auth refuses
// anything not active) nor, through the BLOCKED* KV flag, keeps a live
// session until unblocked. An invited one (B035) was created by an admin
// with no password; it cannot sign in until its holder redeems the invite
// link, which makes it active.
const (
	AccountStatusActive  = "active"
	AccountStatusBlocked = "blocked"
	AccountStatusInvited = "invited"
)

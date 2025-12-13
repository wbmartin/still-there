package model

// AppUserModel represents an application user
type AppUserModel struct {
	AppUserID               int      `json:"appUserId" db:"app_user_id"`
	UserName                string   `json:"userName" db:"user_name"`
	PwHash                  string   `json:"-" db:"pw_hash"` // Never serialize
	FirstName               string   `json:"firstName" db:"first_name"`
	LastName                string   `json:"lastName" db:"last_name"`
	Email                   string   `json:"email" db:"email"`
	SecurityContext         int      `json:"securityContext" db:"security_context"`
	UserGroupMembershipList []int    `json:"userGroupMembershipList" db:"user_group_membership_list"`
	MFAExempt               bool     `json:"mfaExempt" db:"mfa_exempt"`
	CreateTS                float64  `json:"createTs" db:"create_ts"`
	CreateUserID            int      `json:"createUserId" db:"create_user_id"`
	UpdateTS                float64  `json:"updateTs" db:"update_ts"`
	UpdateUserID            int      `json:"updateUserId" db:"update_user_id"`
	DeleteTS                *float64 `json:"deleteTs,omitempty" db:"delete_ts"`
	DeleteUserID            *int     `json:"deleteUserId,omitempty" db:"delete_user_id"`
}

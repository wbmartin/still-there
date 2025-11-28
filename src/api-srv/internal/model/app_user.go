package model

// AppUser represents an application user
type AppUser struct {
	AppUserID               int      `json:"app_user_id" db:"app_user_id"`
	Username                string   `json:"username" db:"username"`
	PwHash                  string   `json:"-" db:"pw_hash"` // Never serialize password hash
	FirstName               string   `json:"first_name" db:"first_name"`
	LastName                string   `json:"last_name" db:"last_name"`
	Email                   string   `json:"email" db:"email"`
	SecurityContext         int      `json:"security_context" db:"security_context"`
	UserGroupMembershipList []int    `json:"user_group_membership_list" db:"user_group_membership_list"`
	MFAExempt               bool     `json:"mfa_exempt" db:"mfa_exempt"`
	CreateTS                float64  `json:"create_ts" db:"create_ts"`
	CreateUserID            int      `json:"create_user_id" db:"create_user_id"`
	UpdateTS                float64  `json:"update_ts" db:"update_ts"`
	UpdateUserID            int      `json:"update_user_id" db:"update_user_id"`
	DeleteTS                *float64 `json:"delete_ts,omitempty" db:"delete_ts"`
	DeleteUserID            *int     `json:"delete_user_id,omitempty" db:"delete_user_id"`
}

// UserGrants maps grant short codes to their IDs for user-related operations
var UserGrants = map[string]int{
	"CHG_OWN_PW":        GrantChangeOwnPassword,
	"ADD_USER":          GrantAddUser,
	"BLOCK_USER":        GrantBlockUser,
	"CHG_OTHER_USER_PW": GrantChangeOtherUserPassword,
}

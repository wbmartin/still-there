package model

// UserGrant represents a user grant record
type UserGrant struct {
	UserGrantID      int      `json:"user_grant_id" db:"user_grant_id"`
	GrantShortCode   string   `json:"grant_short_code" db:"grant_short_code"`
	GrantDescription string   `json:"grant_description" db:"grant_description"`
	SecurityContext  int      `json:"security_context" db:"security_context"`
	CreateTS         float64  `json:"create_ts" db:"create_ts"`
	CreateUserID     int      `json:"create_user_id" db:"create_user_id"`
	UpdateTS         float64  `json:"update_ts" db:"update_ts"`
	UpdateUserID     int      `json:"update_user_id" db:"update_user_id"`
	DeleteTS         *float64 `json:"delete_ts,omitempty" db:"delete_ts"`
	DeleteUserID     *int     `json:"delete_user_id,omitempty" db:"delete_user_id"`
}

// Grant constants (entity-specific and application-wide)
const (
	// Example entity grants
	GrantExampleGet    = 1001
	GrantExampleAdd    = 1002
	GrantExampleChange = 1003
	GrantExampleDelete = 1004

	// Application-wide grants
	GrantChangeOwnPassword       = 8001
	GrantAddUser                 = 9001
	GrantBlockUser               = 9002
	GrantChangeOtherUserPassword = 9003
)

// AllKnownGrants maps grant short codes to their IDs
var AllKnownGrants = map[string]int{
	"GET_EXAMPLE":       GrantExampleGet,
	"ADD_EXAMPLE":       GrantExampleAdd,
	"CHG_EXAMPLE":       GrantExampleChange,
	"DEL_EXAMPLE":       GrantExampleDelete,
	"CHG_OWN_PW":        GrantChangeOwnPassword,
	"ADD_USER":          GrantAddUser,
	"BLOCK_USER":        GrantBlockUser,
	"CHG_OTHER_USER_PW": GrantChangeOtherUserPassword,
}

// GrantIDToShortCodeMap maps grant IDs to their short codes
var GrantIDToShortCodeMap = map[int]string{
	GrantExampleGet:              "GET_EXAMPLE",
	GrantExampleAdd:              "ADD_EXAMPLE",
	GrantExampleChange:           "CHG_EXAMPLE",
	GrantExampleDelete:           "DEL_EXAMPLE",
	GrantChangeOwnPassword:       "CHG_OWN_PW",
	GrantAddUser:                 "ADD_USER",
	GrantBlockUser:               "BLOCK_USER",
	GrantChangeOtherUserPassword: "CHG_OTHER_USER_PW",
}

func init() {
	// This init function ensures the maps are populated with all grant constants
	// It's called automatically when the package is imported
}

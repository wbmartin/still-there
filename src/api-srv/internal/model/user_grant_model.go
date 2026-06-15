package model

// UserGrantModel represents a user grant record
type UserGrantModel struct {
	UserGrantID      int      `json:"user_grant_id"            db:"user_grant_id"`
	GrantShortCode   string   `json:"grant_short_code"         db:"grant_short_code"`
	GrantDescription string   `json:"grant_description"        db:"grant_description"`
	SecurityContext  int      `json:"security_context_id"         db:"security_context_id"`
	CreateTS         float64  `json:"create_ts"                db:"create_ts"`
	CreateUserID     int      `json:"create_user_id"           db:"create_user_id"`
	UpdateTS         float64  `json:"update_ts"                db:"update_ts"`
	UpdateUserID     int      `json:"update_user_id"           db:"update_user_id"`
	DeleteTS         *float64 `json:"delete_ts,omitempty"      db:"delete_ts"`
	DeleteUserID     *int     `json:"delete_user_id,omitempty" db:"delete_user_id"`
}

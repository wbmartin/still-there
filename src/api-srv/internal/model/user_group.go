package model

// UserGroup represents a user group
type UserGroup struct {
	UserGroupID     int      `json:"user_group_id" db:"user_group_id"`
	GroupName       string   `json:"group_name" db:"group_name"`
	UserGrantList   []int    `json:"user_grant_list" db:"user_grant_list"`
	SecurityContext int      `json:"security_context" db:"security_context"`
	CreateTS        float64  `json:"create_ts" db:"create_ts"`
	CreateUserID    int      `json:"create_user_id" db:"create_user_id"`
	UpdateTS        float64  `json:"update_ts" db:"update_ts"`
	UpdateUserID    int      `json:"update_user_id" db:"update_user_id"`
	DeleteTS        *float64 `json:"delete_ts,omitempty" db:"delete_ts"`
	DeleteUserID    *int     `json:"delete_user_id,omitempty" db:"delete_user_id"`
}

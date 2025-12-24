package model

// AppUserGroupModel represents a user group
type AppUserGroupModel struct {
	AppUserGroupID  int      `json:"appUserGroupId"         db:"app_user_group_id"`
	GroupName       string   `json:"groupName"              db:"group_name"`
	UserGrantList   []int    `json:"userGrantList"          db:"user_grant_list"`
	SecurityContext int      `json:"securityContext"        db:"security_context"`
	CreateTS        float64  `json:"createTs"               db:"create_ts"`
	CreateUserID    int      `json:"createUserId"           db:"create_user_id"`
	UpdateTS        float64  `json:"updateTs"               db:"update_ts"`
	UpdateUserID    int      `json:"updateUserId"           db:"update_user_id"`
	DeleteTS        *float64 `json:"deleteTs,omitempty"     db:"delete_ts"`
	DeleteUserID    *int     `json:"deleteUserId,omitempty" db:"delete_user_id"`
}

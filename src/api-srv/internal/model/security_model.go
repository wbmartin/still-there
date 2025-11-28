package model

// AuthRequestModel represents the request body for initial authentication
type AuthRequestModel struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponseModel represents the response for initial authentication
type AuthResponseModel struct {
	AppUserID              int              `json:"app_user_id"`
	Username               string           `json:"username"`
	FirstName              string           `json:"first_name"`
	LastName               string           `json:"last_name"`
	SecurityContext        int              `json:"security_context"`
	UserGrantShortCodeList map[int][]string `json:"user_grant_short_code_list"` // indexed by security_context
	TempToken              string           `json:"temp_token,omitempty"`       // Only present if MFA required
}

// Auth2RequestModel represents the request body for MFA authentication
type Auth2RequestModel struct {
	TempToken string `json:"temp_token"`
	TOTP      string `json:"totp"`
}

// Auth2ResponseModel represents the response for successful MFA authentication
type Auth2ResponseModel struct {
	AppUserID              int              `json:"app_user_id"`
	Username               string           `json:"username"`
	FirstName              string           `json:"first_name"`
	LastName               string           `json:"last_name"`
	SecurityContext        int              `json:"security_context"`
	UserGrantShortCodeList map[int][]string `json:"user_grant_short_code_list"`
}

// LogoffRequest represents the request body for logoff
type LogoffRequest struct {
	// No body required for logoff
}

// ChangeSecurityContextRequest represents the request body for changing security context
type ChangeSecurityContextRequest struct {
	TargetSecurityContext int `json:"target_security_context"`
}

// ChangeSecurityContextResponse represents the response for changing security context
type ChangeSecurityContextResponse struct {
	AppUserID              int              `json:"app_user_id"`
	Username               string           `json:"username"`
	FirstName              string           `json:"first_name"`
	LastName               string           `json:"last_name"`
	SecurityContext        int              `json:"security_context"`
	UserGrantShortCodeList map[int][]string `json:"user_grant_short_code_list"`
}

// ChangeCurrentUserPasswordRequest represents the request body for changing password
type ChangeCurrentUserPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// BlockUserRequest represents the request body for blocking a user
type BlockUserRequest struct {
	Username string `json:"username"`
}

// ResetUserPasswordRequest represents the request body for resetting another user's password
type ResetUserPasswordRequest struct {
	Username    string `json:"username"`
	NewPassword string `json:"new_password"`
}

// AddUserRequest represents the request body for adding a new user
type AddUserRequest struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Email           string `json:"email"`
	SecurityContext int    `json:"security_context"`
	MFAExempt       bool   `json:"mfa_exempt"`
}

package model

// AuthRequestModel represents the request body for initial authentication
type AuthRequestModel struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
}

// AuthResponseModel represents the response for initial authentication
type AuthResponseModel struct {
	AppUserID                    int    `json:"appUserId"`
	UserName                     string `json:"userName"`
	FirstName                    string `json:"firstName"`
	LastName                     string `json:"lastName"`
	SecurityContext              int    `json:"securityContext"`
	GrantList                    []int  `json:"grantList"`
	AvailableSecurityContextList []int  `json:"availableSecurityContextList"`
	TempToken                    string `json:"tempToken,omitempty"` // MFA required
	// With a temp token: true sends the web app to verify a code, false to
	// enrol. The client never asks /mfaSetup to find out.
	MFAEnrolled bool `json:"mfaEnrolled"`
}

// AuthResponseWrapper wraps the auth response for consistency
type AuthResponseWrapper struct {
	User      AuthResponseModel `json:"user"`
	CsrfToken string            `json:"csrfToken,omitempty"`
}

// Auth2RequestModel represents the request body for MFA authentication
type Auth2RequestModel struct {
	TempToken string `json:"tempToken"`
	TOTP      string `json:"totp"`
}

// Auth2ResponseModel represents the response for successful MFA authentication
type Auth2ResponseModel struct {
	AppUserID                    int    `json:"appUserId"`
	UserName                     string `json:"userName"`
	FirstName                    string `json:"firstName"`
	LastName                     string `json:"lastName"`
	SecurityContext              int    `json:"securityContext"`
	GrantList                    []int  `json:"grantList"`
	AvailableSecurityContextList []int  `json:"availableSecurityContextList"`
	MFARequired                  bool   `json:"mfaRequired"`
}

// Auth2ResponseWrapper wraps the auth2 response for consistency
type Auth2ResponseWrapper struct {
	User      Auth2ResponseModel `json:"user"`
	CsrfToken string             `json:"csrfToken,omitempty"`
}

// LogoffRequest represents the request body for logoff
type LogoffRequest struct {
	// No body required for logoff
}

// ChangeSecurityContextRequest represents the request to change security context
type ChangeSecurityContextRequest struct {
	SecurityContext int `json:"securityContext"`
}

// ChangeSecurityContextResponse for changing security context
type ChangeSecurityContextResponse struct {
	AppUserID              int              `json:"appUserId"`
	UserName               string           `json:"userName"`
	FirstName              string           `json:"firstName"`
	LastName               string           `json:"lastName"`
	SecurityContext        int              `json:"securityContext"`
	UserGrantShortCodeList map[int][]string `json:"userGrantShortCodeList"`
}

// ChangeSecurityContextResponseWrapper wraps change response
type ChangeSecurityContextResponseWrapper struct {
	User ChangeSecurityContextResponse `json:"user"`
}

// DataWrapper wraps non-user data responses for consistency
type DataWrapper struct {
	Data interface{} `json:"data"`
}

// ChangeCurrentUserPasswordRequest for changing password
type ChangeCurrentUserPasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// BlockUserRequest represents the request body for blocking a user
type BlockUserRequest struct {
	UserName string `json:"userName"`
}

// ResetUserPasswordRequest for resetting another user's password
type ResetUserPasswordRequest struct {
	UserName    string `json:"userName"`
	NewPassword string `json:"newPassword"`
}

// AddUserRequest represents the request body for adding a new user
type AddUserRequest struct {
	UserName        string `json:"userName"`
	Password        string `json:"password"`
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	Email           string `json:"email"`
	SecurityContext int    `json:"securityContext"`
	MFAExempt       bool   `json:"mfaExempt"`
}

// CreateUserRequest represents the request body for creating a new user
type CreateUserRequest struct {
	UserName string `json:"userName"`
	// Password is the raw password; hashed by the handler, never stored.
	Password                   string `json:"password"`
	FirstName                  string `json:"firstName"`
	LastName                   string `json:"lastName"`
	Email                      string `json:"email"`
	AppUserGroupMembershipList []int  `json:"appUserGroupMembershipList"`
	MFAExempt                  bool   `json:"mfaExempt"`
}

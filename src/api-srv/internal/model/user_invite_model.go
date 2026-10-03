package model

// B035: invite a user with a one-time link. The user_invite row itself never
// leaves the repository: it holds only the token's SHA-256, and the token is
// returned once, by the call that issues it (UserInviteIssuedModel).

// CreateInvitedUserRequest is the body of POST /userInvite: the user row an
// admin would otherwise send to POST /user, without a password.
type CreateInvitedUserRequest struct {
	UserName                   string `json:"userName"`
	FirstName                  string `json:"firstName"`
	LastName                   string `json:"lastName"`
	Email                      string `json:"email"`
	AppUserGroupMembershipList []int  `json:"appUserGroupMembershipList"`
	MFAExempt                  bool   `json:"mfaExempt"`
}

// UserInviteTargetRequest is the body of POST /userInviteReissue and
// /userInviteRevoke.
type UserInviteTargetRequest struct {
	AppUserID int `json:"appUserId"`
}

// UserInviteTokenRequest is the body of POST /inviteCheck; POST
// /inviteRedeem adds the password.
type UserInviteTokenRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// UserInviteIssuedModel answers POST /userInvite (with the created User) and
// POST /userInviteReissue (without). InviteToken is the only copy of the
// token anywhere: the web app builds the link from it and shows it once.
type UserInviteIssuedModel struct {
	User        *AppUserModel `json:"user,omitempty"`
	InviteToken string        `json:"inviteToken"`
	ExpiresTS   float64       `json:"expiresTs"`
}

// UserInvitePreviewModel answers POST /inviteCheck: enough for the invited
// person to recognise the invite is theirs, and nothing more.
type UserInvitePreviewModel struct {
	UserName  string  `json:"userName"`
	FirstName string  `json:"firstName"`
	ExpiresTS float64 `json:"expiresTs"`
}

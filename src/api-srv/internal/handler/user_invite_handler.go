package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
	"github.com/wbmartin/starter-api/internal/security"
)

// B035: invite a user with a one-time link instead of an admin-set
// password. It is a second admin path beside POST /user, and both sit
// beside public self-registration (POST /userSelfCreate); a derived app
// keeps the ones it wants.
//
// Three admin routes (POST /userInvite, /userInviteReissue,
// /userInviteRevoke) run the full five checks. Two public routes are the
// invited person's half: POST /inviteCheck reads who a link is for, and
// POST /inviteRedeem sets the password and makes the login active. The web
// app then signs in with that password, so TOTP enrolment is the ordinary
// first-login path, not a second copy of it.
//
// Token rules: 256 random bits (auth.NewInviteToken), only the SHA-256
// stored, single use, INVITE_TTL_HOURS to expiry, the per-address rate limit
// and a per-address plus per-token lockout on both public routes, and the
// redeeming client's address recorded on the row.

// UserInviteRepositoryInterface is what the invite routes need from
// repository.UserInviteRepository; an interface so the check order can be
// proved with a repository that panics on contact.
type UserInviteRepositoryInterface interface {
	CreateInvitedUser(
		ctx context.Context,
		user *model.AppUserModel,
		securityContext int,
		tokenHash string,
		expiresTS float64,
		actorUserID int,
	) (*model.AppUserModel, error)
	ReissueInvite(
		ctx context.Context,
		appUserID int,
		securityContext int,
		tokenHash string,
		expiresTS float64,
		actorUserID int,
	) error
	RevokeInvite(
		ctx context.Context,
		appUserID int,
		securityContext int,
		actorUserID int,
	) error
	RetrieveLiveInvite(
		ctx context.Context, tokenHash string, now float64,
	) (*model.UserInvitePreviewModel, error)
	RedeemInvite(
		ctx context.Context, tokenHash, pwHash, sourceIP string,
	) (string, error)
}

// UserInviteHandler serves the five invite routes.
type UserInviteHandler struct {
	appsrv               *app.Services
	appUserGroupRepo     AppUserGroupRepositoryInterface
	userInviteRepository UserInviteRepositoryInterface
}

// NewUserInviteHandler creates the invite handler.
func NewUserInviteHandler(
	appServices *app.Services,
	appUserGroupRepo AppUserGroupRepositoryInterface,
	userInviteRepository UserInviteRepositoryInterface,
) *UserInviteHandler {
	return &UserInviteHandler{
		appsrv:               appServices,
		appUserGroupRepo:     appUserGroupRepo,
		userInviteRepository: userInviteRepository,
	}
}

// inviteTokenPattern is the only shape auth.NewInviteToken produces.
// Anything else is refused before it reaches the database.
var inviteTokenPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// inviteLockoutKey is the per-token half of the lockout the public routes
// share with /auth (isAccountLocked / incrementFailedAttempt): the
// per-address half is the same key /auth counts, so an address guessing
// tokens is locked out of signing in too, and the reverse. The prefix holds
// a ':', which a user name cannot (repository userNamePattern), so it never
// lands in a real login's counter; 16 hex characters of the hash are plenty
// to tell tokens apart.
func inviteLockoutKey(tokenHash string) string {
	return "invite:" + tokenHash[:16]
}

// invalidInviteMessage is the one answer for every unusable link (unknown,
// malformed, expired, redeemed, revoked, or its login deleted or active), so
// the answer says nothing about which.
const invalidInviteMessage = "This invite link is not valid or has " +
	"expired. Ask your administrator for a new one."

// minPasswordLength is the floor POST /user and the password routes apply.
const minPasswordLength = 8

// CreateInvitedUser handles POST /userInvite: create a login with no
// password and issue its first link. Same grant and field rules as POST
// /user.
func (h *UserInviteHandler) CreateInvitedUser(
	w http.ResponseWriter, r *http.Request,
) {
	app := security.NewAppHelper(w, r, h.appsrv, "CreateInvitedUser")
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserCreate)
	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("USR_INVITE_START", "CreateInvitedUser invoked")
	jwtClaims := app.GetClaims()

	var req model.CreateInvitedUserRequest
	if !decodeJSONBody(app, w, r, maxLoginLayerBodyBytes, &req) {
		return
	}
	if req.UserName == "" || req.FirstName == "" || req.LastName == "" ||
		req.Email == "" {
		app.WriteError(w, http.StatusBadRequest,
			"userName, firstName, lastName and email are required",
			"BRQ0", nil)
		return
	}

	valid, err := groupListInSecurityContext(
		r.Context(), h.appUserGroupRepo,
		req.AppUserGroupMembershipList, jwtClaims.SecurityContext,
	)
	if !valid || err != nil {
		app.WriteError(
			w,
			http.StatusForbidden,
			"Access denied: all groups must be in the same security context as the user",
			"SEC_CTX_VIOLATION",
			err,
		)
		return
	}

	token, tokenHash, expiresTS, ok := h.newInvite(app, w)
	if !ok {
		return
	}

	now := model.Now()
	user := model.AppUserModel{
		UserName:                   req.UserName,
		FirstName:                  req.FirstName,
		LastName:                   req.LastName,
		Email:                      req.Email,
		AppUserGroupMembershipList: req.AppUserGroupMembershipList,
		MFAExempt:                  req.MFAExempt,
		SecurityContext:            jwtClaims.SecurityContext,
		CreateTS:                   now,
		CreateUserID:               jwtClaims.UserID,
		UpdateTS:                   now,
		UpdateUserID:               jwtClaims.UserID,
	}
	created, err := h.userInviteRepository.CreateInvitedUser(
		r.Context(), &user, jwtClaims.SecurityContext, tokenHash,
		expiresTS, jwtClaims.UserID,
	)
	if err != nil {
		mapRepoError(app, w, err,
			fmt.Sprintf("Failed to invite user: username=%s", req.UserName),
			"USR_INVITE_FAIL")
		return
	}

	app.WriteJSON(w, http.StatusCreated, model.DataWrapper{
		Data: model.UserInviteIssuedModel{
			User: created, InviteToken: token, ExpiresTS: expiresTS,
		},
	})
	app.Info("USR_INVITE_DONE",
		fmt.Sprintf("Invited userId=%d, username='%s'",
			created.AppUserID, created.UserName))
}

// ReissueInvite handles POST /userInviteReissue: a new link for an invited
// login, revoking any it still holds. The token is shown once, so a lost
// link is replaced, never read back.
func (h *UserInviteHandler) ReissueInvite(
	w http.ResponseWriter, r *http.Request,
) {
	app := security.NewAppHelper(w, r, h.appsrv, "ReissueInvite")
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserUpdate)
	if !app.CheckAllChecksComplete() {
		return
	}

	jwtClaims := app.GetClaims()
	req, ok := decodeInviteTarget(app, w, r)
	if !ok {
		return
	}
	token, tokenHash, expiresTS, ok := h.newInvite(app, w)
	if !ok {
		return
	}
	err := h.userInviteRepository.ReissueInvite(
		r.Context(), req.AppUserID, jwtClaims.SecurityContext, tokenHash,
		expiresTS, jwtClaims.UserID,
	)
	if err != nil {
		mapRepoError(app, w, err,
			fmt.Sprintf("Failed to reissue invite: appUserId=%d",
				req.AppUserID),
			"USR_INVITE_REISSUE_FAIL")
		return
	}

	app.WriteJSON(w, http.StatusOK, model.DataWrapper{
		Data: model.UserInviteIssuedModel{
			InviteToken: token, ExpiresTS: expiresTS,
		},
	})
	app.Info("USR_INVITE_REISSUED",
		fmt.Sprintf("Invite reissued for appUserId=%d", req.AppUserID))
}

// RevokeInvite handles POST /userInviteRevoke: the login's live links stop
// working; the login stays invited.
func (h *UserInviteHandler) RevokeInvite(
	w http.ResponseWriter, r *http.Request,
) {
	app := security.NewAppHelper(w, r, h.appsrv, "RevokeInvite")
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserUpdate)
	if !app.CheckAllChecksComplete() {
		return
	}

	jwtClaims := app.GetClaims()
	req, ok := decodeInviteTarget(app, w, r)
	if !ok {
		return
	}
	err := h.userInviteRepository.RevokeInvite(
		r.Context(), req.AppUserID, jwtClaims.SecurityContext,
		jwtClaims.UserID,
	)
	if err != nil {
		mapRepoError(app, w, err,
			fmt.Sprintf("Failed to revoke invite: appUserId=%d",
				req.AppUserID),
			"USR_INVITE_REVOKE_FAIL")
		return
	}

	app.WriteJSON(w, http.StatusOK, model.DataWrapper{
		Data: map[string]string{"message": "Invite revoked"},
	})
	app.Info("USR_INVITE_REVOKED",
		fmt.Sprintf("Invite revoked for appUserId=%d", req.AppUserID))
}

// CheckInvite handles POST /inviteCheck (public): who a live link is for,
// so the page can greet the person before they choose a password. A POST,
// not a GET, so the token never sits in a URL an access log records.
func (h *UserInviteHandler) CheckInvite(
	w http.ResponseWriter, r *http.Request,
) {
	app := security.NewAppHelper(w, r, h.appsrv, "CheckInvite")
	// The link is the credential. Each exception is stated here:
	// no JWT: the person holding the link has never signed in
	app.SkipJWT()
	// the per-address bucket; the per-token and per-address lockout in
	// decodeInviteToken is the stricter throttle on guessing
	app.CheckRateLimit()
	// the block list is keyed by JWT subject, and there is none
	app.SkipUserBlock()
	// CSRF needs a session to derive the expected token from. The forgery
	// it guards against cannot happen here: a cross-site page cannot know
	// the 256-bit token the body must carry, and the answer sets no cookie
	app.SkipCSRF()
	// the token is the authorization: RetrieveLiveInvite
	app.SkipAuthorization()
	if !app.CheckAllChecksComplete() {
		return
	}

	_, tokenHash, ok := h.decodeInviteToken(app, w, r)
	if !ok {
		return
	}
	preview, err := h.userInviteRepository.RetrieveLiveInvite(
		r.Context(), tokenHash, model.Now(),
	)
	if err != nil {
		h.refuseInvite(app, w, r, tokenHash, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: preview})
}

// RedeemInvite handles POST /inviteRedeem (public): set the password, make
// the login active, spend the link. The answer carries the user name, and
// the web app signs in with it and the password just chosen.
func (h *UserInviteHandler) RedeemInvite(
	w http.ResponseWriter, r *http.Request,
) {
	app := security.NewAppHelper(w, r, h.appsrv, "RedeemInvite")
	// The same exceptions as CheckInvite, for the same reasons: no session
	// exists, the per-address bucket and the invite lockout throttle, and
	// the token is the credential (RetrieveLiveInvite, RedeemInvite).
	app.SkipJWT()
	app.CheckRateLimit()
	app.SkipUserBlock()
	app.SkipCSRF()
	app.SkipAuthorization()
	if !app.CheckAllChecksComplete() {
		return
	}

	req, tokenHash, ok := h.decodeInviteToken(app, w, r)
	if !ok {
		return
	}
	// The password rule first: a password no login could have costs no
	// database read.
	if len(req.Password) < minPasswordLength {
		app.WriteError(w, http.StatusBadRequest,
			fmt.Sprintf("Password must be at least %d characters long",
				minPasswordLength),
			"BRQ0", nil)
		return
	}
	// Look before hashing: Argon2id is the expensive step, and a stream of
	// dead tokens must not buy one each. RedeemInvite looks again under a
	// row lock, so this is a filter, not the decision.
	if _, err := h.userInviteRepository.RetrieveLiveInvite(
		r.Context(), tokenHash, model.Now(),
	); err != nil {
		h.refuseInvite(app, w, r, tokenHash, err)
		return
	}
	pwHash, err := auth.HashPassword(req.Password)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError,
			"Failed to process password", "UCH0", err)
		return
	}
	userName, err := h.userInviteRepository.RedeemInvite(
		r.Context(), tokenHash, pwHash,
		security.ClientIP(r, h.appsrv.Config.TrustedProxyCount),
	)
	if err != nil {
		h.refuseInvite(app, w, r, tokenHash, err)
		return
	}

	app.WriteJSON(w, http.StatusOK, model.DataWrapper{
		Data: map[string]string{"userName": userName},
	})
	app.Info("INVITE_REDEEMED", "Invite redeemed for user: "+userName)
}

// decodeInviteToken reads the body, refuses a locked-out address or token,
// and refuses a token of the wrong shape (counted as a failure). It answers
// the request, the token's hash and whether to go on.
func (h *UserInviteHandler) decodeInviteToken(
	app *security.AppHelper, w http.ResponseWriter, r *http.Request,
) (*model.UserInviteTokenRequest, string, bool) {
	var req model.UserInviteTokenRequest
	if !decodeJSONBody(app, w, r, maxAuthBodyBytes, &req) {
		return nil, "", false
	}
	tokenHash := auth.HashInviteToken(req.Token)

	locked, err := isAccountLocked(
		h.appsrv.KV,
		security.ClientIP(r, h.appsrv.Config.TrustedProxyCount),
		inviteLockoutKey(tokenHash),
	)
	if err != nil {
		// Fail closed, as /auth does: without the lockout store there is no
		// throttle on guessing.
		app.WriteError(w, http.StatusInternalServerError,
			"Invites are temporarily unavailable", "LOCKOUT_CHECK_FAIL", err)
		return nil, "", false
	}
	if locked {
		app.WriteError(w, http.StatusTooManyRequests,
			"Too many attempts; try again in a few minutes",
			"ACCOUNT_LOCKED", nil)
		return nil, "", false
	}

	if !inviteTokenPattern.MatchString(req.Token) {
		h.refuseInvite(app, w, r, tokenHash, repository.ErrNotFound)
		return nil, "", false
	}
	return &req, tokenHash, true
}

// refuseInvite answers a link that cannot be used. A not-found link counts
// against the address and the token and gets the one generic 404; anything
// else is a server fault.
func (h *UserInviteHandler) refuseInvite(
	app *security.AppHelper,
	w http.ResponseWriter,
	r *http.Request,
	tokenHash string,
	err error,
) {
	if !errors.Is(err, repository.ErrNotFound) {
		app.WriteError(w, http.StatusInternalServerError,
			"The invite could not be read", "INVITE_READ_FAIL", err)
		return
	}
	if countErr := incrementFailedAttempt(
		h.appsrv.KV, h.appsrv.Config,
		security.ClientIP(r, h.appsrv.Config.TrustedProxyCount),
		inviteLockoutKey(tokenHash),
	); countErr != nil {
		app.WriteError(w, http.StatusInternalServerError,
			"Invites are temporarily unavailable",
			"FAILED_ATTEMPT_INCR_FAIL", countErr)
		return
	}
	app.WriteError(w, http.StatusNotFound, invalidInviteMessage,
		"INVITE_INVALID", nil)
}

// newInvite mints a token and its expiry, answering 500 itself on failure.
func (h *UserInviteHandler) newInvite(
	app *security.AppHelper, w http.ResponseWriter,
) (token, tokenHash string, expiresTS float64, ok bool) {
	token, err := auth.NewInviteToken()
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError,
			"Failed to generate the invite link", "INVITE_TOKEN_GEN", err)
		return "", "", 0, false
	}
	ttl := time.Duration(h.appsrv.Config.InviteTTLHours) * time.Hour
	expiresTS = model.EpochSeconds(time.Now().Add(ttl))
	return token, auth.HashInviteToken(token), expiresTS, true
}

// decodeInviteTarget reads the body of the reissue and revoke routes.
func decodeInviteTarget(
	app *security.AppHelper, w http.ResponseWriter, r *http.Request,
) (*model.UserInviteTargetRequest, bool) {
	var req model.UserInviteTargetRequest
	if !decodeJSONBody(app, w, r, maxLoginLayerBodyBytes, &req) {
		return nil, false
	}
	if req.AppUserID <= 0 {
		app.WriteError(w, http.StatusBadRequest,
			"appUserId is required", "BRQ0", nil)
		return nil, false
	}
	return &req, true
}

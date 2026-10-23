package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"stillthere-api/internal/app"
	"stillthere-api/internal/auth"
	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
)

// B035: the invite routes. The admin half runs the full five checks; the
// public half runs the rate limit and the invite lockout, and nothing
// reaches the repository, let alone Argon2id, for a token that cannot be
// live. The admin is login 7 of the test tenant (2).

const inviteTenant = 2

// recordingInviteRepository records what reached it. Methods it does not
// define are the embedded nil interface and panic.
type recordingInviteRepository struct {
	UserInviteRepositoryInterface
	liveTokenHash string // the one hash the public reads accept

	createdUser      *model.AppUserModel
	createdTokenHash string
	createdExpiresTS float64
	createdActorID   int
	redeemedPwHash   string
	redeemedSourceIP string
}

func (s *recordingInviteRepository) CreateInvitedUser(
	_ context.Context,
	user *model.AppUserModel,
	_ int,
	tokenHash string,
	expiresTS float64,
	actorUserID int,
) (*model.AppUserModel, error) {
	s.createdUser, s.createdTokenHash = user, tokenHash
	s.createdExpiresTS, s.createdActorID = expiresTS, actorUserID
	user.AppUserID = 51
	user.AccountStatus = model.AccountStatusInvited
	return user, nil
}

func (s *recordingInviteRepository) RetrieveLiveInvite(
	_ context.Context, tokenHash string, _ float64,
) (*model.UserInvitePreviewModel, error) {
	if tokenHash != s.liveTokenHash {
		return nil, repository.ErrNotFound
	}
	return &model.UserInvitePreviewModel{
		UserName: "invitee", FirstName: "Ivy", ExpiresTS: 1,
	}, nil
}

func (s *recordingInviteRepository) RedeemInvite(
	_ context.Context, tokenHash, pwHash, sourceIP string,
) (string, error) {
	if tokenHash != s.liveTokenHash {
		return "", repository.ErrNotFound
	}
	s.redeemedPwHash, s.redeemedSourceIP = pwHash, sourceIP
	return "invitee", nil
}

// panicInviteRepository fails the test by panicking on any contact.
type panicInviteRepository struct {
	UserInviteRepositoryInterface
}

// liveInviteToken is a token of the right shape.
var liveInviteToken = strings.Repeat("ab", 32)

const inviteSessionID = "session-inv"

func inviteServices(t *testing.T) *app.Services {
	t.Helper()
	appServices := lockoutServices(t)
	appServices.Config.InviteTTLHours = 72
	return appServices
}

// inviteAdminRequest drives one admin invite route as login 7 holding
// grantList; withCookie false sends no session, csrf false no CSRF header.
func inviteAdminRequest(
	t *testing.T,
	appServices *app.Services,
	userInviteRepository UserInviteRepositoryInterface,
	target, body string,
	grantList []int,
	withCookie, csrf bool,
) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, target,
		strings.NewReader(body))
	r.RemoteAddr = "192.0.2.40:1234"
	if withCookie {
		token, err := auth.GenerateJWT(
			"Test", "inviter", 7, inviteSessionID, inviteTenant, grantList,
			appServices.Config.JWTSecret,
			appServices.Config.JWTSessionExpireHours,
		)
		require.NoError(t, err)
		r.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	}
	if csrf {
		r.Header.Set("X-CSRF-Token",
			inviteSessionID[len(inviteSessionID)-3:])
	}

	handler := NewUserInviteHandler(appServices, nil, userInviteRepository)
	w := httptest.NewRecorder()
	switch target {
	case "/userInvite":
		handler.CreateInvitedUser(w, r)
	case "/userInviteReissue":
		handler.ReissueInvite(w, r)
	case "/userInviteRevoke":
		handler.RevokeInvite(w, r)
	default:
		t.Fatalf("no route wired for %s", target)
	}
	return w
}

// invitePublicRequest drives one public invite route from remoteAddr.
func invitePublicRequest(
	appServices *app.Services,
	userInviteRepository UserInviteRepositoryInterface,
	target, remoteAddr, body string,
) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, target,
		strings.NewReader(body))
	r.RemoteAddr = remoteAddr
	handler := NewUserInviteHandler(appServices, nil, userInviteRepository)
	w := httptest.NewRecorder()
	if target == "/inviteCheck" {
		handler.CheckInvite(w, r)
	} else {
		handler.RedeemInvite(w, r)
	}
	return w
}

const inviteBody = `{"userName":"invitee","firstName":"Ivy",` +
	`"lastName":"Vine","email":"ivy@example.com",` +
	`"appUserGroupMembershipList":[],"mfaExempt":false}`

// Each refused check answers before the repository, which panics if
// reached.
func TestInviteAdminRouteCheckSequence(t *testing.T) {
	testCaseList := []struct {
		name       string
		target     string
		grantList  []int
		withCookie bool
		csrf       bool
		wantStatus int
	}{
		{"no cookie is 401", "/userInvite",
			[]int{model.GrantUserCreate}, false, true,
			http.StatusUnauthorized},
		{"no CSRF header is 403", "/userInvite",
			[]int{model.GrantUserCreate}, true, false,
			http.StatusForbidden},
		{"invite needs CREATE_USER", "/userInvite",
			[]int{model.GrantUserUpdate}, true, true,
			http.StatusForbidden},
		{"reissue needs UPDATE_USER", "/userInviteReissue",
			[]int{model.GrantUserCreate}, true, true,
			http.StatusForbidden},
		{"revoke needs UPDATE_USER", "/userInviteRevoke",
			[]int{model.GrantUserCreate}, true, true,
			http.StatusForbidden},
	}
	for _, testCase := range testCaseList {
		t.Run(testCase.name, func(t *testing.T) {
			w := inviteAdminRequest(t, inviteServices(t),
				panicInviteRepository{}, testCase.target,
				`{"appUserId":51}`, testCase.grantList,
				testCase.withCookie, testCase.csrf)
			assert.Equal(t, testCase.wantStatus, w.Code, w.Body.String())
		})
	}
}

// The answer carries the token once, the repository gets its SHA-256 and an
// INVITE_TTL_HOURS expiry, and the actor is the admin.
func TestCreateInvitedUser_IssuesATokenAndStoresOnlyItsHash(t *testing.T) {
	userInviteRepository := &recordingInviteRepository{}
	before := float64(time.Now().Unix())

	w := inviteAdminRequest(t, inviteServices(t), userInviteRepository,
		"/userInvite", inviteBody, []int{model.GrantUserCreate}, true, true)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var answer struct {
		Data model.UserInviteIssuedModel `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &answer))
	token := answer.Data.InviteToken
	assert.Regexp(t, `^[0-9a-f]{64}$`, token)
	assert.Equal(t, auth.HashInviteToken(token),
		userInviteRepository.createdTokenHash)
	assert.NotContains(t, w.Body.String(),
		userInviteRepository.createdTokenHash,
		"the answer never carries the stored hash")
	assert.InDelta(t, before+72*3600, userInviteRepository.createdExpiresTS,
		5)
	assert.Equal(t, 7, userInviteRepository.createdActorID)
	assert.Equal(t, inviteTenant,
		userInviteRepository.createdUser.SecurityContext,
		"the tenant comes from the session")
	require.NotNil(t, answer.Data.User)
	assert.Equal(t, model.AccountStatusInvited,
		answer.Data.User.AccountStatus)
}

// An invite carries mfa_exempt into the login it creates, so it is held to
// POST /user's rule: true only where ALLOW_MFA_EXEMPT_VIA_API=true (B097).
func TestCreateInvitedUser_MFAExemptNeedsTheConfigKey(t *testing.T) {
	exemptBody := strings.Replace(inviteBody, `"mfaExempt":false`,
		`"mfaExempt":true`, 1)

	t.Run("not allowed → 403 before the repository", func(t *testing.T) {
		w := inviteAdminRequest(t, inviteServices(t),
			panicInviteRepository{}, "/userInvite", exemptBody,
			[]int{model.GrantUserCreate}, true, true)
		assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), "MFA exemption cannot")
	})
	t.Run("allowed → 201, stored exempt", func(t *testing.T) {
		appServices := inviteServices(t)
		appServices.Config.AllowMFAExemptViaAPI = true
		userInviteRepository := &recordingInviteRepository{}
		w := inviteAdminRequest(t, appServices, userInviteRepository,
			"/userInvite", exemptBody, []int{model.GrantUserCreate},
			true, true)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		assert.True(t, userInviteRepository.createdUser.MFAExempt)
	})
}

// The same required fields as POST /user, refused before the repository.
func TestCreateInvitedUser_RequiresEveryField(t *testing.T) {
	body := strings.Replace(inviteBody, `"ivy@example.com"`, `""`, 1)
	w := inviteAdminRequest(t, inviteServices(t), panicInviteRepository{},
		"/userInvite", body, []int{model.GrantUserCreate}, true, true)
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

// A malformed token never reaches the repository, an unknown one does, and
// both get the one generic 404.
func TestInvitePublicRoutes_RefuseADeadLinkAlike(t *testing.T) {
	for _, target := range []string{"/inviteCheck", "/inviteRedeem"} {
		t.Run(target, func(t *testing.T) {
			appServices := inviteServices(t)
			malformed := invitePublicRequest(appServices,
				panicInviteRepository{}, target, "192.0.2.41:1234",
				`{"token":"not-a-token","password":"long-enough"}`)
			unknown := invitePublicRequest(appServices,
				&recordingInviteRepository{liveTokenHash: "none"}, target,
				"192.0.2.42:1234",
				`{"token":"`+liveInviteToken+`","password":"long-enough"}`)
			for _, w := range []*httptest.ResponseRecorder{
				malformed, unknown,
			} {
				assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
				assert.Contains(t, w.Body.String(), "not valid or has expired")
			}
			assert.Equal(t, malformed.Body.String(), unknown.Body.String())
		})
	}
}

// After FAILED_LOGIN_ATTEMPT_THRESHOLD dead tokens from one address the next
// request is 429 before the repository, and the lock is the one /auth
// reads.
func TestInvitePublicRoutes_LockOutAGuessingAddress(t *testing.T) {
	appServices := inviteServices(t)
	const remoteAddr = "192.0.2.43:1234"

	threshold := appServices.Config.FailedLoginAttemptThreshold
	for attempt := 0; attempt < threshold; attempt++ {
		guess := strings.Repeat("0", 63) + string(rune('a'+attempt))
		w := invitePublicRequest(appServices, &recordingInviteRepository{},
			"/inviteCheck", remoteAddr, `{"token":"`+guess+`"}`)
		require.Equal(t, http.StatusNotFound, w.Code, "attempt %d", attempt)
	}

	w := invitePublicRequest(appServices, panicInviteRepository{},
		"/inviteCheck", remoteAddr, `{"token":"`+liveInviteToken+`"}`)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, w.Body.String())
	locked, err := isAccountLocked(appServices.KV, "192.0.2.43", "someone")
	require.NoError(t, err)
	assert.True(t, locked, "the address is locked out of /auth too")
}

// The per-address request bucket runs before anything else.
func TestInvitePublicRoutes_AreRateLimited(t *testing.T) {
	appServices := inviteServices(t)
	appServices.Config.MaxUnauthReqPerMin = 0

	w := invitePublicRequest(appServices, panicInviteRepository{},
		"/inviteCheck", "192.0.2.44:1234",
		`{"token":"`+liveInviteToken+`"}`)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, w.Body.String())
}

// No cookie and no CSRF header, and a live token is answered with who it
// is for.
func TestCheckInvite_NeedsNoSession(t *testing.T) {
	userInviteRepository := &recordingInviteRepository{
		liveTokenHash: auth.HashInviteToken(liveInviteToken),
	}
	w := invitePublicRequest(inviteServices(t), userInviteRepository,
		"/inviteCheck", "192.0.2.45:1234",
		`{"token":"`+liveInviteToken+`"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"userName":"invitee"`)
	assert.Contains(t, w.Body.String(), `"firstName":"Ivy"`)
}

// The password rule is checked before the token is looked up, so the
// repository is never reached.
func TestRedeemInvite_RefusesAShortPasswordFirst(t *testing.T) {
	w := invitePublicRequest(inviteServices(t), panicInviteRepository{},
		"/inviteRedeem", "192.0.2.46:1234",
		`{"token":"`+liveInviteToken+`","password":"short"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

// The repository gets a hash that verifies against the chosen password
// (never the password), and the client address that redeemed the link.
func TestRedeemInvite_StoresAnArgonHashAndTheClient(t *testing.T) {
	userInviteRepository := &recordingInviteRepository{
		liveTokenHash: auth.HashInviteToken(liveInviteToken),
	}
	const password = "a-chosen-password"

	w := invitePublicRequest(inviteServices(t), userInviteRepository,
		"/inviteRedeem", "192.0.2.47:1234",
		`{"token":"`+liveInviteToken+`","password":"`+password+`"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"userName":"invitee"`)
	assert.NotEqual(t, password, userInviteRepository.redeemedPwHash)
	assert.True(t, auth.CheckPasswordHash(password,
		userInviteRepository.redeemedPwHash))
	assert.Equal(t, "192.0.2.47", userInviteRepository.redeemedSourceIP)
}

// An admin-set password would leave an invited login invited and unusable,
// so it is a 409 and no write.
func TestChangeOtherUserPassword_RefusesAnInvitedLogin(t *testing.T) {
	appServices := inviteServices(t)
	repo := new(MockUserRepository)
	repo.On("RetrieveUserList", mock.Anything, inviteTenant, []int{51},
		float64(0)).
		Return([]*model.AppUserModel{{
			AppUserID: 51, UserName: "invitee",
			SecurityContext: inviteTenant,
			AccountStatus:   model.AccountStatusInvited,
		}}, nil)
	token, err := auth.GenerateJWT(
		"Test", "inviter", 7, inviteSessionID, inviteTenant,
		[]int{model.GrantChangeOtherUserPassword},
		appServices.Config.JWTSecret,
		appServices.Config.JWTSessionExpireHours,
	)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	NewUserHandler(appServices, repo, nil).ChangeOtherUserPassword(w,
		sessionRequest(http.MethodPost, "/changeOtherUserPassword",
			`{"appUserId":51,"newPassword":"long-enough"}`, token,
			inviteSessionID))
	assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	repo.AssertNotCalled(t, "UpdateUserPassword", mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything)
}

// /auth refuses an invited login (no password yet) as a bad password.
func TestAuthHandler_InvitedLoginIsRefusedLikeABadPassword(t *testing.T) {
	appServices := inviteServices(t)
	repo := new(MockUserRepository)
	repo.On("RetrieveUserByUserName", mock.Anything, "invitee").
		Return(&model.AppUserModel{
			AppUserID: 51, UserName: "invitee", PwHash: "",
			SecurityContext: inviteTenant, MFAExempt: true,
			AccountStatus: model.AccountStatusInvited,
		}, nil)

	w := authAttempt(t, AuthHandler(appServices, repo), "192.0.2.49:1",
		"invitee", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid credentials")
	assert.Empty(t, w.Result().Cookies())
}

package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/model"
)

// The check contract, route by route (B039). Every route cmd/server/main.go
// registers is a row below saying which of the explicit checks its handler
// runs: CheckJWT -> CheckRateLimitAndUserBlock -> CheckCSRF ->
// CheckAuthorization -> CheckAllChecksComplete. The test sends each row a
// request that fails exactly one check (and every later one) and asserts
// that check answers, in that order, before anything else runs; a check the
// row says is skipped must not answer. A skipped check needs a reason in the
// row, so an exception is read here, not inferred.
//
// Handlers are built over repositories that fail on contact: testify mocks
// with no expectations (an unexpected call panics), the invite fake whose
// methods are a nil interface, and a nil group repository. A refusal that
// reached the repository fails the test. A new route in main.go fails
// TestRouteContract_EveryRouteIsListed until it has a row.

type routeContract struct {
	route   string // as main.go registers it: "POST /userInvite"
	handler func(appServices *app.Services) http.HandlerFunc
	session bool // CheckJWT: no auth cookie → 401
	rate    bool // a rate limit: spent → 429
	block   bool // the BLOCKED* flag → 403
	csrf    bool // CheckCSRF on an unsafe method: no header → 403
	grant   int  // CheckAuthorization grant; 0 when skipped
	skip    string
}

func userHandlerOf(appServices *app.Services) *UserHandler {
	return NewUserHandler(appServices, new(MockUserRepository),
		new(MockAppUserGroupRepository))
}

func exampleHandlerOf(appServices *app.Services) *ExampleHandler {
	return NewExampleHandler(appServices, new(MockExampleRepository))
}

func profileHandlerOf(appServices *app.Services) *ProfileHandler {
	return NewProfileHandler(appServices, new(MockProfileRepository))
}

func inviteHandlerOf(appServices *app.Services) *UserInviteHandler {
	return NewUserInviteHandler(appServices,
		new(MockAppUserGroupRepository), &panicInviteRepository{})
}

// groupHandlerOf passes a nil repository: any call dereferences it.
func groupHandlerOf(appServices *app.Services) *AppUserGroupHandler {
	return NewAppUserGroupHandler(appServices, nil)
}

func withUserRepo(
	build func(*app.Services, UserRepositoryInterface) http.HandlerFunc,
) func(*app.Services) http.HandlerFunc {
	return func(appServices *app.Services) http.HandlerFunc {
		return build(appServices, new(MockUserRepository))
	}
}

func fullChecks(
	route string,
	grant int,
	handler func(*app.Services) http.HandlerFunc,
) routeContract {
	return routeContract{route: route, handler: handler, session: true,
		rate: true, block: true, csrf: !strings.HasPrefix(route, "GET "),
		grant: grant}
}

var routeContractList = []routeContract{
	{route: "GET /health",
		handler: func(s *app.Services) http.HandlerFunc {
			return HealthHandler(s)
		},
		skip: "liveness probe: no session, no data"},
	{route: "POST /auth", handler: withUserRepo(AuthHandler),
		skip: "login: no session yet; the per-address and per-name " +
			"lockout stands in for the rate limit " +
			"(TestAuthHandler_LockoutAnswersBeforeTheRepository)"},
	{route: "GET /refresh", handler: withUserRepo(RefreshHandler),
		session: true, rate: true, block: true,
		skip: "any live session refreshes; no grant"},
	{route: "POST /changeSecurityContext",
		handler: withUserRepo(ChangeSecurityContextHandler),
		session: true, rate: true, block: true,
		skip: "no CSRF by design: it changes no server-side data " +
			"(B005 skipped); the grant is checked against the " +
			"target context after the gate"},
	fullChecks("POST /changeOwnPassword", model.GrantChangeOwnPassword,
		withUserRepo(ChangeOwnPasswordHandler)),
	fullChecks("GET /me", model.GrantLogon,
		func(s *app.Services) http.HandlerFunc {
			return profileHandlerOf(s).RetrieveOwnProfile
		}),
	fullChecks("PUT /me", model.GrantLogon,
		func(s *app.Services) http.HandlerFunc {
			return profileHandlerOf(s).UpdateOwnProfile
		}),
	{route: "POST /logoff",
		handler: func(s *app.Services) http.HandlerFunc {
			return LogoffHandler(s)
		},
		session: true, rate: true, block: true,
		skip: "ending one's own session needs no CSRF and no grant"},

	fullChecks("GET /example", model.GrantExampleRead,
		func(s *app.Services) http.HandlerFunc {
			return exampleHandlerOf(s).RetrieveExample
		}),
	fullChecks("POST /example", model.GrantExampleCreate,
		func(s *app.Services) http.HandlerFunc {
			return exampleHandlerOf(s).CreateExample
		}),
	fullChecks("PUT /example", model.GrantExampleUpdate,
		func(s *app.Services) http.HandlerFunc {
			return exampleHandlerOf(s).UpdateExample
		}),
	fullChecks("DELETE /example", model.GrantExampleDelete,
		func(s *app.Services) http.HandlerFunc {
			return exampleHandlerOf(s).DeleteExample
		}),
	fullChecks("GET /exampleList", model.GrantExampleRead,
		func(s *app.Services) http.HandlerFunc {
			return exampleHandlerOf(s).RetrieveExampleList
		}),
	fullChecks("POST /exampleList", model.GrantExampleCreate,
		func(s *app.Services) http.HandlerFunc {
			return exampleHandlerOf(s).CreateExampleList
		}),
	fullChecks("PUT /exampleList", model.GrantExampleUpdate,
		func(s *app.Services) http.HandlerFunc {
			return exampleHandlerOf(s).UpdateExampleList
		}),
	fullChecks("DELETE /exampleList", model.GrantExampleDelete,
		func(s *app.Services) http.HandlerFunc {
			return exampleHandlerOf(s).DeleteExampleList
		}),

	{route: "POST /userSelfCreate",
		handler: withUserRepo(UserSelfCreateHandler), rate: true,
		skip: "public self-registration: per-address rate limit only"},
	{route: "POST /inviteCheck",
		handler: func(s *app.Services) http.HandlerFunc {
			return inviteHandlerOf(s).CheckInvite
		},
		rate: true,
		skip: "public invite half: rate limit, then the per-address " +
			"and per-token lockout"},
	{route: "POST /inviteRedeem",
		handler: func(s *app.Services) http.HandlerFunc {
			return inviteHandlerOf(s).RedeemInvite
		},
		rate: true,
		skip: "public invite half: rate limit, then the per-address " +
			"and per-token lockout"},
	{route: "POST /mfaSetup", handler: withUserRepo(MFASetupHandler),
		session: true, rate: true,
		skip: "temp token or session; the rate check runs before the " +
			"JWT is read, so the block flag is not consulted"},
	{route: "POST /mfaVerify", handler: withUserRepo(MFAVerifyHandler),
		session: true, rate: true,
		skip: "temp token or session; the rate check runs before the " +
			"JWT is read, so the block flag is not consulted"},

	fullChecks("GET /userList", model.GrantUserRead,
		func(s *app.Services) http.HandlerFunc {
			return userHandlerOf(s).RetrieveUserList
		}),
	fullChecks("POST /user", model.GrantUserCreate,
		func(s *app.Services) http.HandlerFunc {
			return userHandlerOf(s).CreateUser
		}),
	fullChecks("PUT /user", model.GrantUserUpdate,
		func(s *app.Services) http.HandlerFunc {
			return userHandlerOf(s).UpdateUser
		}),
	fullChecks("DELETE /user", model.GrantUserDelete,
		func(s *app.Services) http.HandlerFunc {
			return userHandlerOf(s).DeleteUser
		}),
	fullChecks("POST /changeOtherUserPassword",
		model.GrantChangeOtherUserPassword,
		func(s *app.Services) http.HandlerFunc {
			return userHandlerOf(s).ChangeOtherUserPassword
		}),
	fullChecks("POST /addUserListToGroup", model.GrantUserGroupUpdate,
		func(s *app.Services) http.HandlerFunc {
			return userHandlerOf(s).AddUserListToGroupHandler
		}),
	fullChecks("POST /removeUserListFromGroup", model.GrantUserGroupUpdate,
		func(s *app.Services) http.HandlerFunc {
			return userHandlerOf(s).RemoveUserListFromGroupHandler
		}),

	fullChecks("POST /userInvite", model.GrantUserCreate,
		func(s *app.Services) http.HandlerFunc {
			return inviteHandlerOf(s).CreateInvitedUser
		}),
	fullChecks("POST /userInviteReissue", model.GrantUserUpdate,
		func(s *app.Services) http.HandlerFunc {
			return inviteHandlerOf(s).ReissueInvite
		}),
	fullChecks("POST /userInviteRevoke", model.GrantUserUpdate,
		func(s *app.Services) http.HandlerFunc {
			return inviteHandlerOf(s).RevokeInvite
		}),

	fullChecks("POST /admin/resetUserMFA", model.GrantResetUserMFA,
		withUserRepo(AdminResetUserMFAHandler)),
	fullChecks("POST /admin/blockUser", model.GrantUserBlock,
		func(s *app.Services) http.HandlerFunc {
			return AdminBlockUserHandler(s, new(MockUserRepository), true)
		}),
	fullChecks("POST /admin/unblockUser", model.GrantUserBlock,
		func(s *app.Services) http.HandlerFunc {
			return AdminBlockUserHandler(s, new(MockUserRepository), false)
		}),

	fullChecks("GET /grantList", model.GrantUserCreate,
		func(s *app.Services) http.HandlerFunc {
			return userHandlerOf(s).RetrieveGrantList
		}),

	fullChecks("GET /appUserGroupList", model.GrantUserGroupRead,
		func(s *app.Services) http.HandlerFunc {
			return groupHandlerOf(s).RetrieveAppUserGroupList
		}),
	fullChecks("POST /appUserGroup", model.GrantUserGroupCreate,
		func(s *app.Services) http.HandlerFunc {
			return groupHandlerOf(s).CreateAppUserGroup
		}),
	fullChecks("PUT /appUserGroup", model.GrantUserGroupUpdate,
		func(s *app.Services) http.HandlerFunc {
			return groupHandlerOf(s).UpdateAppUserGroup
		}),
	fullChecks("DELETE /appUserGroup", model.GrantUserGroupDelete,
		func(s *app.Services) http.HandlerFunc {
			return groupHandlerOf(s).DeleteAppUserGroup
		}),

	fullChecks("POST /error-report", model.GrantLogon,
		func(s *app.Services) http.HandlerFunc {
			return NewErrorReportHandler(s).HandleErrorReport
		}),
}

// The body text of each check's refusal (app_helper.go).
const (
	refusedNoSession = "Missing auth cookie"
	refusedRate      = "Rate limit exceeded"
	refusedBlocked   = "User account is blocked"
	refusedCSRF      = "Missing CSRF token"
	refusedGrant     = "Permission denied (grant"
	refusedGate      = "check not completed"
)

const contractSessionID = "session-contract"

// contractRequest is the row's route as the test tenant's alice (login 7).
// withCookie false sends no session, csrf false no CSRF header.
func contractRequest(
	t *testing.T,
	appServices *app.Services,
	route string,
	withCookie, csrf bool,
	grantList []int,
) *http.Request {
	t.Helper()
	method, target, _ := strings.Cut(route, " ")
	body := ""
	if method != http.MethodGet {
		body = "{}"
	}
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.RemoteAddr = "192.0.2.77:1234"
	if withCookie {
		token, err := auth.GenerateJWT(
			"Test", "alice", 7, contractSessionID, 2, grantList,
			appServices.Config.JWTSecret,
			appServices.Config.JWTSessionExpireHours,
		)
		require.NoError(t, err)
		r.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	}
	if csrf {
		r.Header.Set("X-CSRF-Token",
			contractSessionID[len(contractSessionID)-3:])
	}
	return r
}

// callRoute runs handler once. reached is true when something past the
// checks panicked: the fail-on-contact repositories.
func callRoute(
	handler http.HandlerFunc, r *http.Request,
) (w *httptest.ResponseRecorder, reached bool) {
	w = httptest.NewRecorder()
	defer func() {
		if recover() != nil {
			reached = true
		}
	}()
	handler(w, r)
	return w, false
}

// TestRouteContract_EveryRouteIsListed: the table and main.go name the same
// routes, once each.
func TestRouteContract_EveryRouteIsListed(t *testing.T) {
	source, err := os.ReadFile("../../cmd/server/main.go")
	require.NoError(t, err)
	pattern := regexp.MustCompile(
		`mux\.HandleFunc\(\s*"([A-Z]+ /[^"]*)"`)
	registered := map[string]bool{}
	for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
		registered[match[1]] = true
	}
	require.NotEmpty(t, registered)

	listed := map[string]bool{}
	for _, row := range routeContractList {
		assert.False(t, listed[row.route], "listed twice: %s", row.route)
		listed[row.route] = true
		assert.True(t, registered[row.route],
			"%s is not registered in main.go", row.route)
	}
	for route := range registered {
		assert.True(t, listed[route],
			"%s has no row in routeContractList", route)
	}
}

// TestRouteContract_SkipsAreExplained: a row that skips any check says why;
// a route with no session can have no block, CSRF or grant check.
func TestRouteContract_SkipsAreExplained(t *testing.T) {
	for _, row := range routeContractList {
		safe := strings.HasPrefix(row.route, "GET ")
		full := row.session && row.rate && row.block &&
			(row.csrf || safe) && row.grant != 0
		if !full {
			assert.NotEmpty(t, row.skip, "%s skips a check", row.route)
		}
		if !row.session {
			assert.False(t, row.block || row.csrf || row.grant != 0,
				"%s: no session, so nothing to block or authorize",
				row.route)
		}
		if safe {
			assert.False(t, row.csrf, "%s: CSRF skips safe methods",
				row.route)
		}
	}
}

// TestRouteContract_ChecksAnswerBeforeTheRepository walks each row through
// one failing check at a time, in handler order.
func TestRouteContract_ChecksAnswerBeforeTheRepository(t *testing.T) {
	type checkCase struct {
		name       string
		enforced   func(row routeContract) bool
		needs      func(row routeContract) bool // the case applies
		prepare    func(t *testing.T, appServices *app.Services)
		withCookie func(row routeContract) bool
		csrf       bool
		wantStatus int
		wantBody   string
	}
	always := func(routeContract) bool { return true }
	sessionOnly := func(row routeContract) bool { return row.session }
	checkCaseList := []checkCase{
		{
			name:       "no session",
			enforced:   func(row routeContract) bool { return row.session },
			needs:      always,
			withCookie: func(routeContract) bool { return false },
			wantStatus: http.StatusUnauthorized,
			wantBody:   refusedNoSession,
		},
		{
			name:     "rate limit spent",
			enforced: func(row routeContract) bool { return row.rate },
			needs:    always,
			prepare: func(t *testing.T, appServices *app.Services) {
				appServices.Config.MaxUserReqPerMin = 0
				appServices.Config.MaxUnauthReqPerMin = 0
			},
			withCookie: sessionOnly,
			wantStatus: http.StatusTooManyRequests,
			wantBody:   refusedRate,
		},
		{
			name:     "login blocked",
			enforced: func(row routeContract) bool { return row.block },
			needs:    sessionOnly,
			prepare: func(t *testing.T, appServices *app.Services) {
				require.NoError(t,
					appServices.KV.Set(blockedKey("alice"), "1", ""))
			},
			withCookie: sessionOnly,
			wantStatus: http.StatusForbidden,
			wantBody:   refusedBlocked,
		},
		{
			name:       "no CSRF header",
			enforced:   func(row routeContract) bool { return row.csrf },
			needs:      sessionOnly,
			withCookie: sessionOnly,
			wantStatus: http.StatusForbidden,
			wantBody:   refusedCSRF,
		},
		{
			name:       "no grant",
			enforced:   func(row routeContract) bool { return row.grant != 0 },
			needs:      sessionOnly,
			withCookie: sessionOnly,
			csrf:       true,
			wantStatus: http.StatusForbidden,
			wantBody:   refusedGrant,
		},
	}

	for _, row := range routeContractList {
		for _, check := range checkCaseList {
			if !check.needs(row) {
				continue
			}
			enforced := check.enforced(row)
			verb := " → skipped"
			if enforced {
				verb = " → refused"
			}
			t.Run(row.route+" "+check.name+verb, func(t *testing.T) {
				appServices := testServices(t)
				if check.prepare != nil {
					check.prepare(t, appServices)
				}
				r := contractRequest(t, appServices, row.route,
					check.withCookie(row), check.csrf, nil)
				w, reached := callRoute(row.handler(appServices), r)
				if !enforced {
					assert.NotContains(t, w.Body.String(), check.wantBody)
					return
				}
				assert.False(t, reached, "repository reached")
				assert.Equal(t, check.wantStatus, w.Code, w.Body.String())
				assert.Contains(t, w.Body.String(), check.wantBody)
				if check.wantBody == refusedGrant {
					// the JSON-escaped short code: \"READ_USER\"
					assert.Contains(t, w.Body.String(),
						`\"`+model.GrantIDToShortCodeMap[row.grant]+`\"`)
				}
			})
		}

		t.Run(row.route+" every check passes → past the gate",
			func(t *testing.T) {
				appServices := testServices(t)
				var grantList []int
				if row.grant != 0 {
					grantList = []int{row.grant}
				}
				r := contractRequest(t, appServices, row.route,
					row.session, true, grantList)
				w, _ := callRoute(row.handler(appServices), r)
				for _, refusal := range []string{
					refusedNoSession, refusedRate, refusedBlocked,
					refusedCSRF, refusedGrant, refusedGate,
				} {
					assert.NotContains(t, w.Body.String(), refusal)
				}
			})
	}
}

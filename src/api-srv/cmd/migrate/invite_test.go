package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"stillthere-api/internal/app"
	"stillthere-api/internal/auth"
	"stillthere-api/internal/cache"
	"stillthere-api/internal/config"
	"stillthere-api/internal/handler"
	"stillthere-api/internal/logging"
	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
)

// B099: `invite` against the scratch database bin/300-run-tests names in
// TEST_DATABASE_URL. Tenant 3 stands in for a real tenant (the scratch
// database refuses tenant 1); tenant 2 is the test tenant the command
// refuses. Every login here is named MigrateTestInvitee*; the tests remove
// them and their invites before and after.

const (
	inviteTestOrigin      = "https://app.example.com"
	inviteTestRealTenant  = 3
	inviteTestSeedActorID = 1
)

// inviteLinkPattern is the link the web /invite page reads: the origin,
// the route, and the token in the fragment (web-app utils/invite-link.ts).
var inviteLinkPattern = regexp.MustCompile(
	`^` + regexp.QuoteMeta(inviteTestOrigin) + `/invite#([0-9a-f]{64})$`)

// inviteTestPool connects to the scratch database. The skip message is
// the one bin/300-run-tests fails on, so the gate never passes without it.
func inviteTestPool(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("skipping integration tests: TEST_DATABASE_URL unset " +
			"(run bin/300-run-tests go)")
	}
	parsed, err := url.Parse(databaseURL)
	require.NoError(t, err)
	databaseName := strings.TrimPrefix(parsed.Path, "/")
	if !strings.HasSuffix(databaseName, "_test_db") &&
		!strings.HasSuffix(databaseName, "_e2e_db") {
		t.Fatalf("refusing to run against %q: not a scratch database",
			databaseName)
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skipping integration tests: no reachable test database "+
			"(%v)", err)
	}
	wipeInviteTestLogins(t, pool)
	t.Cleanup(func() {
		wipeInviteTestLogins(t, pool)
		pool.Close()
	})
	return pool, databaseURL
}

func wipeInviteTestLogins(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		DELETE FROM user_invite WHERE app_user_id IN (
		  SELECT app_user_id FROM app_user
		  WHERE user_name LIKE 'MigrateTestInvitee%')`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		"DELETE FROM app_user WHERE user_name LIKE 'MigrateTestInvitee%'")
	require.NoError(t, err)
}

// seedInvitedLogin creates an invited login the way POST /userInvite does,
// holding one live invite for priorToken, and answers its id.
func seedInvitedLogin(
	t *testing.T, pool *pgxpool.Pool, userName string, tenant int,
	priorToken string,
) int {
	t.Helper()
	now := model.Now()
	created, err := repository.NewUserInviteRepository(pool).
		CreateInvitedUser(context.Background(), &model.AppUserModel{
			UserName: userName, FirstName: "Ivy", LastName: "Vine",
			Email:                      userName + "@example.com",
			AppUserGroupMembershipList: []int{},
			SecurityContext:            tenant,
			CreateTS:                   now, CreateUserID: inviteTestSeedActorID,
			UpdateTS: now, UpdateUserID: inviteTestSeedActorID,
		}, tenant, auth.HashInviteToken(priorToken), now+3600,
			inviteTestSeedActorID)
	require.NoError(t, err)
	return created.AppUserID
}

func newTestToken(t *testing.T) string {
	t.Helper()
	token, err := auth.NewInviteToken()
	require.NoError(t, err)
	return token
}

// inviteCheck drives the API's POST /inviteCheck handler, with the real
// repository, as the web /invite page would.
func inviteCheck(
	t *testing.T, pool *pgxpool.Pool, token string,
) *httptest.ResponseRecorder {
	t.Helper()
	kv := cache.NewKV()
	t.Cleanup(kv.Close)
	appServices := &app.Services{
		KV: kv,
		Config: &config.Config{
			JWTSecret:                        "test-secret",
			JWTSessionExpireHours:            1,
			MaxUserReqPerMin:                 1000,
			MaxUnauthReqPerMin:               1000,
			FailedLoginAttemptThreshold:      7,
			FailedLoginAttemptWindowDuration: "5min",
			FailedLoginLockoutDuration:       "5min",
			InviteTTLHours:                   72,
		},
		Logger: logging.NewLogger("test", &bytes.Buffer{},
			slog.LevelDebug),
	}
	inviteHandler := handler.NewUserInviteHandler(appServices, nil,
		repository.NewUserInviteRepository(pool))
	r := httptest.NewRequest(http.MethodPost, "/inviteCheck",
		strings.NewReader(`{"token":"`+token+`"}`))
	r.RemoteAddr = "192.0.2.90:1234"
	w := httptest.NewRecorder()
	inviteHandler.CheckInvite(w, r)
	return w
}

type inviteRow struct {
	createUserID int
	expiresTS    float64
	revoked      bool
	tokenHash    string
}

func inviteRowList(
	t *testing.T, pool *pgxpool.Pool, appUserID int,
) []inviteRow {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT create_user_id, expires_ts, revoked_ts IS NOT NULL,
		       token_hash
		FROM user_invite WHERE app_user_id = $1
		ORDER BY user_invite_id`, appUserID)
	require.NoError(t, err)
	defer rows.Close()
	var rowList []inviteRow
	for rows.Next() {
		var row inviteRow
		require.NoError(t, rows.Scan(&row.createUserID, &row.expiresTS,
			&row.revoked, &row.tokenHash))
		rowList = append(rowList, row)
	}
	require.NoError(t, rows.Err())
	return rowList
}

// The printed link carries a token the API's /inviteCheck accepts, the
// login's earlier link stops working, and the redeem path then activates
// the login.
func TestInvite_LinkPassesInviteCheckAndRevokesThePriorLink(t *testing.T) {
	pool, _ := inviteTestPool(t)
	ctx := context.Background()
	const userName = "MigrateTestInviteeFirst"
	priorToken := newTestToken(t)
	appUserID := seedInvitedLogin(t, pool, userName, inviteTestRealTenant,
		priorToken)
	require.Equal(t, http.StatusOK, inviteCheck(t, pool, priorToken).Code,
		"the seeded link is live before the command runs")

	now := time.Now()
	issued, err := issueInvite(ctx, pool, userName, inviteTestOrigin+"/",
		48, now)
	require.NoError(t, err)
	assert.Equal(t, userName, issued.userName)
	assert.Equal(t, inviteTestRealTenant, issued.securityContext)
	match := inviteLinkPattern.FindStringSubmatch(issued.link)
	require.NotNil(t, match, "link %q", issued.link)
	token := match[1]

	w := inviteCheck(t, pool, token)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"userName":"`+userName+`"`)
	assert.Equal(t, http.StatusNotFound,
		inviteCheck(t, pool, priorToken).Code, "the prior link is revoked")

	rowList := inviteRowList(t, pool, appUserID)
	require.Len(t, rowList, 2)
	assert.True(t, rowList[0].revoked)
	assert.False(t, rowList[1].revoked)
	assert.Equal(t, auth.HashInviteToken(token), rowList[1].tokenHash,
		"only the hash is stored")
	assert.Equal(t, inviteActorUserID, rowList[1].createUserID)
	assert.InDelta(t, model.EpochSeconds(now.Add(48*time.Hour)),
		rowList[1].expiresTS, 1)

	redeemedUserName, err := repository.NewUserInviteRepository(pool).
		RedeemInvite(ctx, auth.HashInviteToken(token), "x", "192.0.2.91")
	require.NoError(t, err)
	assert.Equal(t, userName, redeemedUserName)
	_, err = issueInvite(ctx, pool, userName, inviteTestOrigin, 48, now)
	assert.ErrorContains(t, err, "is active, not invited",
		"a redeemed login takes no second first-sign-in link")
}

// Each refusal writes nothing: the login's invite rows are unchanged.
func TestInvite_RefusesAndWritesNothing(t *testing.T) {
	pool, _ := inviteTestPool(t)
	ctx := context.Background()
	idOf := map[string]int{}
	for _, userName := range []string{
		"MigrateTestInviteeReal", "MigrateTestInviteeActive",
		"MigrateTestInviteeBlocked", "MigrateTestInviteeDeleted",
	} {
		idOf[userName] = seedInvitedLogin(t, pool, userName,
			inviteTestRealTenant, newTestToken(t))
	}
	idOf["MigrateTestInviteeTestTenant"] = seedInvitedLogin(t, pool,
		"MigrateTestInviteeTestTenant", inviteTestTenant, newTestToken(t))
	_, err := pool.Exec(ctx, `
		UPDATE app_user SET account_status = 'active'
		WHERE user_name = 'MigrateTestInviteeActive'`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		UPDATE app_user SET account_status = 'blocked'
		WHERE user_name = 'MigrateTestInviteeBlocked'`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		UPDATE app_user SET delete_ts = 1, delete_user_id = 1
		WHERE user_name = 'MigrateTestInviteeDeleted'`)
	require.NoError(t, err)

	caseList := []struct {
		name, userName, baseURL string
		ttlHours                int
		want                    string
	}{
		{"unknown login", "MigrateTestInviteeNobody", inviteTestOrigin, 72,
			"no live login named"},
		{"deleted login", "MigrateTestInviteeDeleted", inviteTestOrigin, 72,
			"no live login named"},
		{"test tenant", "MigrateTestInviteeTestTenant", inviteTestOrigin,
			72, "the test tenant"},
		{"active login", "MigrateTestInviteeActive", inviteTestOrigin, 72,
			"is active, not invited"},
		{"blocked login", "MigrateTestInviteeBlocked", inviteTestOrigin, 72,
			"is blocked, not invited"},
		{"no user name", "", inviteTestOrigin, 72, "needs a user name"},
		{"no base url", "MigrateTestInviteeReal", "", 72,
			"-base-url must be"},
		{"base url with a path", "MigrateTestInviteeReal",
			inviteTestOrigin + "/invite", 72, "-base-url must be"},
		{"zero ttl", "MigrateTestInviteeReal", inviteTestOrigin, 0,
			"-invite-ttl-hours"},
	}
	for _, tc := range caseList {
		t.Run(tc.name, func(t *testing.T) {
			before := map[int][]inviteRow{}
			for _, appUserID := range idOf {
				before[appUserID] = inviteRowList(t, pool, appUserID)
			}
			issued, err := issueInvite(ctx, pool, tc.userName, tc.baseURL,
				tc.ttlHours, time.Now())
			assert.Nil(t, issued)
			assert.ErrorContains(t, err, tc.want)
			for _, appUserID := range idOf {
				assert.Equal(t, before[appUserID],
					inviteRowList(t, pool, appUserID))
			}
		})
	}
}

// run wires the subcommand: the schema check passes on a migrated
// database, and stdout is the link alone, for a script to capture.
func TestInvite_RunPrintsOnlyTheLink(t *testing.T) {
	pool, databaseURL := inviteTestPool(t)
	const userName = "MigrateTestInviteeRun"
	appUserID := seedInvitedLogin(t, pool, userName, inviteTestRealTenant,
		newTestToken(t))

	stdout := captureStdout(t, func() error {
		return run("", databaseURL, inviteOptions{
			baseURL: inviteTestOrigin, ttlHours: 72,
		}, []string{"invite", userName})
	})
	match := inviteLinkPattern.FindStringSubmatch(
		strings.TrimSuffix(stdout, "\n"))
	require.NotNil(t, match, "stdout %q", stdout)
	assert.Equal(t, http.StatusOK, inviteCheck(t, pool, match[1]).Code)
	assert.Len(t, inviteRowList(t, pool, appUserID), 2)

	err := run("", databaseURL, inviteOptions{
		baseURL: inviteTestOrigin, ttlHours: 72,
	}, []string{"invite"})
	assert.ErrorContains(t, err, "exactly one user name")
}

func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	saved := os.Stdout
	os.Stdout = writer
	runErr := fn()
	os.Stdout = saved
	require.NoError(t, writer.Close())
	out, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, runErr)
	return string(out)
}

func TestInviteOrigin(t *testing.T) {
	for _, accepted := range []string{
		"https://app.example.com", "https://app.example.com/",
		"http://localhost:5173",
	} {
		origin, err := inviteOrigin(accepted)
		assert.NoError(t, err, accepted)
		assert.Equal(t, strings.TrimSuffix(accepted, "/"), origin)
	}
	for _, refused := range []string{
		"", "app.example.com", "ftp://app.example.com",
		"https://app.example.com/base", "https://app.example.com?x=1",
		"https://app.example.com#frag", "https://user:pw@app.example.com",
		"https://",
	} {
		_, err := inviteOrigin(refused)
		assert.Error(t, err, refused)
	}
}

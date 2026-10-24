package main

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"stillthere-api/internal/auth"
	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
)

// `bootstrap-admin` against the scratch database. It needs a tenant with
// no login at all, and the scratch database seeds 2 and 3 (and refuses
// 1), so tenant 4 stands in for a real tenant migrated empty. The tests
// own every row in tenant 4 and remove them before and after.
const bootstrapTestTenant = 4

func bootstrapTestPool(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	pool, databaseURL := inviteTestPool(t)
	wipeBootstrapTestTenant(t, pool)
	t.Cleanup(func() { wipeBootstrapTestTenant(t, pool) })
	return pool, databaseURL
}

func wipeBootstrapTestTenant(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, table := range []string{"user_invite", "app_user",
		"app_user_group"} {
		_, err := pool.Exec(context.Background(),
			"DELETE FROM "+table+" WHERE security_context_id = $1",
			bootstrapTestTenant)
		require.NoError(t, err)
	}
}

type bootstrapTenantRows struct {
	groupCount, loginCount, inviteCount int
}

func bootstrapRowCount(
	t *testing.T, pool *pgxpool.Pool,
) bootstrapTenantRows {
	t.Helper()
	var rows bootstrapTenantRows
	require.NoError(t, pool.QueryRow(context.Background(), `
		SELECT
		  (SELECT count(*) FROM app_user_group
		   WHERE security_context_id = $1),
		  (SELECT count(*) FROM app_user WHERE security_context_id = $1),
		  (SELECT count(*) FROM user_invite
		   WHERE security_context_id = $1)`,
		bootstrapTestTenant,
	).Scan(&rows.groupCount, &rows.loginCount, &rows.inviteCount))
	return rows
}

// Every grant the API checks, the device grants among them.
func TestBootstrapAdminGrantList(t *testing.T) {
	grantList := bootstrapAdminGrantList()
	assert.Len(t, grantList, len(model.AllKnownGrants))
	assert.IsIncreasing(t, grantList)
	for _, grantID := range []int{model.GrantLogon,
		model.GrantUserCreate, model.GrantDeviceRead,
		model.GrantDeviceCreate, model.GrantDeviceIssueToken} {
		assert.Contains(t, grantList, grantID)
	}
}

// The link passes the API's /inviteCheck, the login is an invited person
// in an Admin group holding every grant, and the redeem path activates
// it. A second run refuses and writes nothing.
func TestBootstrapAdmin_MakesGroupAndInvitedLogin(t *testing.T) {
	pool, _ := bootstrapTestPool(t)
	ctx := context.Background()
	const userName = "MigrateTestBootstrapAdmin"
	require.Equal(t, bootstrapTenantRows{}, bootstrapRowCount(t, pool),
		"tenant 4 starts empty")

	now := time.Now()
	issued, err := bootstrapAdmin(ctx, pool, bootstrapTestTenant, userName,
		"admin@example.com", inviteTestOrigin, 24, now)
	require.NoError(t, err)
	assert.Equal(t, userName, issued.userName)
	assert.Equal(t, bootstrapTestTenant, issued.securityContext)
	assert.WithinDuration(t, now.Add(24*time.Hour), issued.expires,
		time.Second)
	match := inviteLinkPattern.FindStringSubmatch(issued.link)
	require.NotNil(t, match, "link %q", issued.link)
	w := inviteCheck(t, pool, match[1])
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"userName":"`+userName+`"`)

	user, err := repository.NewUserRepository(pool).
		RetrieveUserByUserName(ctx, userName)
	require.NoError(t, err)
	assert.Equal(t, bootstrapTestTenant, user.SecurityContext)
	assert.Equal(t, model.AccountStatusInvited, user.AccountStatus)
	assert.False(t, user.MFAExempt, "the admin enrols TOTP")
	require.Len(t, user.AppUserGroupMembershipList, 1)
	groupList, err := repository.NewAppUserGroupRepository(pool).
		RetrieveAppUserGroupListByIDList(ctx,
			user.AppUserGroupMembershipList, bootstrapTestTenant)
	require.NoError(t, err)
	require.Len(t, groupList, 1)
	assert.Equal(t, bootstrapAdminGroupName, groupList[0].GroupName)
	assert.ElementsMatch(t, bootstrapAdminGrantList(),
		groupList[0].UserGrantList)
	assert.Equal(t, bootstrapTenantRows{1, 1, 1},
		bootstrapRowCount(t, pool))

	_, err = bootstrapAdmin(ctx, pool, bootstrapTestTenant,
		"MigrateTestBootstrapSecond", "second@example.com",
		inviteTestOrigin, 24, now)
	assert.ErrorContains(t, err, "already has 1 live login")
	assert.Equal(t, bootstrapTenantRows{1, 1, 1},
		bootstrapRowCount(t, pool), "the refusal wrote nothing")

	redeemedUserName, err := repository.NewUserInviteRepository(pool).
		RedeemInvite(ctx, auth.HashInviteToken(match[1]), "x",
			"192.0.2.92")
	require.NoError(t, err)
	assert.Equal(t, userName, redeemedUserName)
}

// Each refusal before the first write leaves tenant 4 empty.
func TestBootstrapAdmin_RefusesAndWritesNothing(t *testing.T) {
	pool, _ := bootstrapTestPool(t)
	ctx := context.Background()
	caseList := []struct {
		name, userName, email, baseURL string
		tenant, ttlHours               int
		want                           string
	}{
		{"test tenant", "MigrateTestBootstrapA", "a@example.com",
			inviteTestOrigin, inviteTestTenant, 72, "the test tenant"},
		{"tenant zero", "MigrateTestBootstrapA", "a@example.com",
			inviteTestOrigin, 0, 72, "-tenant must be"},
		{"no user name", "", "a@example.com", inviteTestOrigin,
			bootstrapTestTenant, 72, "needs a user name"},
		{"no email", "MigrateTestBootstrapA", "", inviteTestOrigin,
			bootstrapTestTenant, 72, "bare email address"},
		{"named email", "MigrateTestBootstrapA", "A <a@example.com>",
			inviteTestOrigin, bootstrapTestTenant, 72, "bare email address"},
		{"no base url", "MigrateTestBootstrapA", "a@example.com", "",
			bootstrapTestTenant, 72, "-base-url must be"},
		{"zero ttl", "MigrateTestBootstrapA", "a@example.com",
			inviteTestOrigin, bootstrapTestTenant, 0, "-invite-ttl-hours"},
	}
	for _, tc := range caseList {
		t.Run(tc.name, func(t *testing.T) {
			issued, err := bootstrapAdmin(ctx, pool, tc.tenant, tc.userName,
				tc.email, tc.baseURL, tc.ttlHours, time.Now())
			assert.Nil(t, issued)
			assert.ErrorContains(t, err, tc.want)
			assert.Equal(t, bootstrapTenantRows{},
				bootstrapRowCount(t, pool))
		})
	}
}

// A run that fails after making the group (here: a user name the login
// insert refuses) leaves it, and the next run reuses it rather than
// failing on the duplicate. A group of that name with other grants is
// someone else's and is refused.
func TestBootstrapAdmin_ReusesOnlyItsOwnGroup(t *testing.T) {
	pool, _ := bootstrapTestPool(t)
	ctx := context.Background()
	_, err := bootstrapAdmin(ctx, pool, bootstrapTestTenant,
		"not a valid name", "a@example.com", inviteTestOrigin, 72,
		time.Now())
	require.ErrorContains(t, err, "username must contain only")
	require.Equal(t, bootstrapTenantRows{groupCount: 1},
		bootstrapRowCount(t, pool), "the group outlived the failed run")

	issued, err := bootstrapAdmin(ctx, pool, bootstrapTestTenant,
		"MigrateTestBootstrapRetry", "a@example.com", inviteTestOrigin, 72,
		time.Now())
	require.NoError(t, err)
	assert.Equal(t, "MigrateTestBootstrapRetry", issued.userName)
	assert.Equal(t, bootstrapTenantRows{1, 1, 1},
		bootstrapRowCount(t, pool), "one group, reused")

	wipeBootstrapTestTenant(t, pool)
	_, err = pool.Exec(ctx, `
		INSERT INTO app_user_group (group_name, user_grant_list,
		  security_context_id, create_ts, create_user_id, update_ts,
		  update_user_id)
		VALUES ($1, $2, $3, 1, 0, 1, 0)`,
		bootstrapAdminGroupName, []int{model.GrantLogon},
		bootstrapTestTenant)
	require.NoError(t, err)
	_, err = bootstrapAdmin(ctx, pool, bootstrapTestTenant,
		"MigrateTestBootstrapOther", "a@example.com", inviteTestOrigin, 72,
		time.Now())
	assert.ErrorContains(t, err, "holds other grants")
	assert.Equal(t, bootstrapTenantRows{groupCount: 1},
		bootstrapRowCount(t, pool))
}

// run wires the subcommand: stdout is the link alone, and a wrong
// argument count is refused before any write.
func TestBootstrapAdmin_RunPrintsOnlyTheLink(t *testing.T) {
	pool, databaseURL := bootstrapTestPool(t)
	options := inviteOptions{baseURL: inviteTestOrigin, ttlHours: 72,
		tenant: bootstrapTestTenant}

	err := run("", databaseURL, options,
		[]string{"bootstrap-admin", "MigrateTestBootstrapRun"})
	assert.ErrorContains(t, err, "a user name and an email address")
	require.Equal(t, bootstrapTenantRows{}, bootstrapRowCount(t, pool))

	stdout := captureStdout(t, func() error {
		return run("", databaseURL, options, []string{"bootstrap-admin",
			"MigrateTestBootstrapRun", "run@example.com"})
	})
	match := inviteLinkPattern.FindStringSubmatch(
		strings.TrimSuffix(stdout, "\n"))
	require.NotNil(t, match, "stdout %q", stdout)
	assert.Equal(t, http.StatusOK, inviteCheck(t, pool, match[1]).Code)
	assert.Equal(t, bootstrapTenantRows{1, 1, 1},
		bootstrapRowCount(t, pool))
}

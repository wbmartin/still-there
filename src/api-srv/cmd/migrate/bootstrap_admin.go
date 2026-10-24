package main

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"stillthere-api/internal/auth"
	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
)

// `bootstrap-admin` makes the first login of a tenant whose database was
// migrated empty (doc/320, First admin): nothing else can, since nobody
// can sign in to invite anyone. It writes an Admin group holding every
// grant the API knows (model.AllKnownGrants, so the device grants
// 1010-1014 come with it) and one invited login in that group with
// its first invite, the way POST /userInvite does
// (UserInviteRepository.CreateInvitedUser), then answers the link as
// `invite` does. The person opens it, sets a password and enrols TOTP;
// everyone after is invited from the web app.
//
// A tenant that already has a live login is refused: its admins invite
// from the web app, and `invite` reissues a lost link. An Admin group left
// by a run that failed after making it is reused when it holds exactly
// these grants, so the command can be run again.

// bootstrapAdminGroupName is the group the first login joins.
const bootstrapAdminGroupName = "Admin"

// bootstrapAdminGrantList is every grant the API checks, ascending.
func bootstrapAdminGrantList() []int {
	grantList := make([]int, 0, len(model.AllKnownGrants))
	for _, grantID := range model.AllKnownGrants {
		grantList = append(grantList, grantID)
	}
	slices.Sort(grantList)
	return grantList
}

// bootstrapAdmin writes the Admin group (or reuses a matching one) and the
// invited login userName in tenant, and answers its first sign-in link.
func bootstrapAdmin(
	ctx context.Context,
	pool *pgxpool.Pool,
	tenant int,
	userName, email, baseURL string,
	ttlHours int,
	now time.Time,
) (*issuedInvite, error) {
	origin, err := inviteOrigin(baseURL)
	if err != nil {
		return nil, err
	}
	if ttlHours <= 0 {
		return nil, fmt.Errorf("-invite-ttl-hours must be a positive " +
			"number of hours")
	}
	if tenant < 1 {
		return nil, fmt.Errorf("-tenant must be a tenant id, 1 or more "+
			"(got %d)", tenant)
	}
	if tenant == inviteTestTenant {
		return nil, fmt.Errorf("tenant %d is the test tenant: refusing to "+
			"make a login there", inviteTestTenant)
	}
	if userName == "" {
		return nil, fmt.Errorf("bootstrap-admin needs a user name")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return nil, fmt.Errorf("bootstrap-admin needs a bare email "+
			"address, e.g. admin@example.com (got %q)", email)
	}

	var liveLoginCount int
	err = pool.QueryRow(ctx, `
		SELECT count(*) FROM app_user
		WHERE security_context_id = $1 AND delete_ts IS NULL`,
		tenant).Scan(&liveLoginCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count logins: %w", err)
	}
	if liveLoginCount > 0 {
		return nil, fmt.Errorf("tenant %d already has %d live login(s): "+
			"invite from the web app, or reissue a link with `invite`",
			tenant, liveLoginCount)
	}

	groupID, err := bootstrapAdminGroup(ctx, pool, tenant, now)
	if err != nil {
		return nil, err
	}

	token, err := auth.NewInviteToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate the invite token: %w",
			err)
	}
	expires := now.Add(time.Duration(ttlHours) * time.Hour)
	stamp := model.EpochSeconds(now)
	created, err := repository.NewUserInviteRepository(pool).
		CreateInvitedUser(ctx, &model.AppUserModel{
			UserName:                   userName,
			FirstName:                  "Site",
			LastName:                   "Administrator",
			Email:                      email,
			SecurityContext:            tenant,
			AppUserGroupMembershipList: []int{groupID},
			MFAExempt:                  false,
			CreateTS:                   stamp,
			CreateUserID:               inviteActorUserID,
			UpdateTS:                   stamp,
			UpdateUserID:               inviteActorUserID,
		}, tenant, auth.HashInviteToken(token), model.EpochSeconds(expires),
			inviteActorUserID)
	if err != nil {
		return nil, fmt.Errorf("bootstrap-admin %q: %w", userName, err)
	}
	return &issuedInvite{
		link:            origin + inviteLinkPath + token,
		userName:        created.UserName,
		securityContext: tenant,
		expires:         expires,
	}, nil
}

// bootstrapAdminGroup answers the id of tenant's Admin group, making it
// when there is none. An existing one is used only when it is live and
// holds exactly bootstrapAdminGrantList: anything else is someone's
// group, not this command's.
func bootstrapAdminGroup(
	ctx context.Context, pool *pgxpool.Pool, tenant int, now time.Time,
) (int, error) {
	grantList := bootstrapAdminGrantList()
	var groupID int
	var existingGrantList []int
	var deleteTS *float64
	err := pool.QueryRow(ctx, `
		SELECT app_user_group_id, user_grant_list, delete_ts
		FROM app_user_group
		WHERE security_context_id = $1 AND group_name = $2`,
		tenant, bootstrapAdminGroupName,
	).Scan(&groupID, &existingGrantList, &deleteTS)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		stamp := model.EpochSeconds(now)
		group, err := repository.NewAppUserGroupRepository(pool).
			CreateAppUserGroup(ctx, &model.AppUserGroupModel{
				GroupName:       bootstrapAdminGroupName,
				UserGrantList:   grantList,
				SecurityContext: tenant,
				CreateTS:        stamp,
				CreateUserID:    inviteActorUserID,
				UpdateTS:        stamp,
				UpdateUserID:    inviteActorUserID,
			}, tenant)
		if err != nil {
			return 0, fmt.Errorf("failed to create the %s group: %w",
				bootstrapAdminGroupName, err)
		}
		return group.AppUserGroupID, nil
	case err != nil:
		return 0, fmt.Errorf("failed to read the %s group: %w",
			bootstrapAdminGroupName, err)
	}
	sortedExisting := slices.Clone(existingGrantList)
	slices.Sort(sortedExisting)
	if deleteTS != nil || !slices.Equal(sortedExisting, grantList) {
		return 0, fmt.Errorf("tenant %d already has a group named %q that "+
			"is deleted or holds other grants: refusing to reuse it",
			tenant, bootstrapAdminGroupName)
	}
	return groupID, nil
}

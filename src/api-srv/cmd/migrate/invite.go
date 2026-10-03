package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
)

// `invite` issues the first sign-in link on a host whose logins are all
// invited (B099; doc/330 A9): the data load is a bin/620 export, which
// leaves no password anywhere, so nobody can sign in to invite anyone
// else. It is the API's reissue (POST /userInviteReissue) without a
// caller: the same token code (auth.NewInviteToken, auth.HashInviteToken)
// and the same repository write (UserInviteRepository.ReissueInvite), which
// revokes the login's open links before it stores the new hash.

// inviteTestTenant is the reserved test tenant (CLAUDE.md): no real
// person's first sign-in happens there, so a link into it is refused, as
// the export scrub refuses its rows.
const inviteTestTenant = 2

// inviteActorUserID stamps create_user_id/update_user_id on the rows the
// command writes: no login is signed in, so 0, as the hand INSERT it
// replaces did.
const inviteActorUserID = 0

// inviteLinkPath is the web app's invite route (web-app ROUTES.INVITE);
// the token rides in the fragment (web-app utils/invite-link.ts).
const inviteLinkPath = "/invite#"

// issuedInvite is what the command reports: the link (the only copy of
// the token) and who it is for.
type issuedInvite struct {
	link            string
	userName        string
	securityContext int
	expires         time.Time
}

// issueInvite issues a new link for the live invited login userName,
// revoking any open link it holds, and answers the link built on baseURL.
// Nothing but the user_invite rows is written.
func issueInvite(
	ctx context.Context,
	pool *pgxpool.Pool,
	userName, baseURL string,
	ttlHours int,
	now time.Time,
) (*issuedInvite, error) {
	if userName == "" {
		return nil, fmt.Errorf("invite needs a user name")
	}
	origin, err := inviteOrigin(baseURL)
	if err != nil {
		return nil, err
	}
	if ttlHours <= 0 {
		return nil, fmt.Errorf("-invite-ttl-hours must be a positive " +
			"number of hours")
	}

	userRepository := repository.NewUserRepository(pool)
	user, err := userRepository.RetrieveUserByUserName(ctx, userName)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("no live login named %q", userName)
	}
	if err != nil {
		return nil, err
	}
	if user.SecurityContext == inviteTestTenant {
		return nil, fmt.Errorf("%q is in tenant %d, the test tenant: "+
			"refusing to issue a link there", userName, inviteTestTenant)
	}
	if user.AccountStatus != model.AccountStatusInvited {
		return nil, fmt.Errorf("%q is %s, not invited: only an invited "+
			"login takes an invite link", userName, user.AccountStatus)
	}

	token, err := auth.NewInviteToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate the invite token: %w",
			err)
	}
	expires := now.Add(time.Duration(ttlHours) * time.Hour)
	// ReissueInvite locks the login and checks again that it is live and
	// invited, so a redeem or delete in between is refused, not raced.
	err = repository.NewUserInviteRepository(pool).ReissueInvite(
		ctx, user.AppUserID, user.SecurityContext,
		auth.HashInviteToken(token), model.EpochSeconds(expires),
		inviteActorUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("invite %q: %w", userName, err)
	}
	return &issuedInvite{
		link:            origin + inviteLinkPath + token,
		userName:        user.UserName,
		securityContext: user.SecurityContext,
		expires:         expires,
	}, nil
}

// inviteOrigin checks -base-url is a bare http(s) origin, such as
// https://<SITE_NAME>, and answers it without a trailing slash.
func inviteOrigin(baseURL string) (string, error) {
	origin := strings.TrimRight(baseURL, "/")
	parsed, err := url.Parse(origin)
	if origin == "" || err != nil ||
		(parsed.Scheme != "https" && parsed.Scheme != "http") ||
		parsed.Host == "" || parsed.Path != "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.ForceQuery ||
		parsed.Fragment != "" {
		return "", fmt.Errorf("-base-url must be the site's origin, "+
			"e.g. https://app.example.com (got %q)", baseURL)
	}
	return origin, nil
}

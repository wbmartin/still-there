package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wbmartin/starter-api/internal/model"
)

// UserInviteRepository owns user_invite and the app_user writes an invite
// makes (B035). Tokens never reach this package: the handler hashes the
// token (auth.HashInviteToken) and every method here takes or stores the
// hash.
type UserInviteRepository struct {
	db *pgxpool.Pool
}

// NewUserInviteRepository creates the invite repository.
func NewUserInviteRepository(db *pgxpool.Pool) *UserInviteRepository {
	return &UserInviteRepository{db: db}
}

// liveUserInvitePredicate is what makes a user_invite row usable: not
// redeemed, not revoked, not deleted. Expiry is compared separately, against
// the caller's clock. prefix is the table alias with its dot ("ui."), or ""
// in a query that reads user_invite alone.
func liveUserInvitePredicate(prefix string) string {
	return prefix + "redeemed_ts IS NULL AND " + prefix +
		"revoked_ts IS NULL AND " + prefix + "delete_ts IS NULL"
}

// liveInviteJoin selects the live, unexpired invite whose hash is $1 at time
// $2, joined to its still-invited login ($3 is AccountStatusInvited).
const liveInviteJoin = `
	FROM user_invite ui
	JOIN app_user u
	  ON u.app_user_id = ui.app_user_id
	 AND u.security_context_id = ui.security_context_id
	WHERE ui.token_hash = $1 AND ui.expires_ts > $2
	  AND u.delete_user_id IS NULL AND u.account_status = $3`

// CreateInvitedUser inserts user as an invited login (no password, cannot
// sign in) and its first invite, in one transaction. The caller fills the
// user's create/update stamps; actorUserID is the admin.
func (r *UserInviteRepository) CreateInvitedUser(
	ctx context.Context,
	user *model.AppUserModel,
	securityContext int,
	tokenHash string,
	expiresTS float64,
	actorUserID int,
) (*model.AppUserModel, error) {
	var created *model.AppUserModel
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		user.AccountStatus = model.AccountStatusInvited
		// An empty hash never verifies (auth.CheckPasswordHashOrDummy), and
		// /auth refuses an invited login whatever the password.
		user.PwHash = ""
		var err error
		created, err = insertAppUser(ctx, tx, user, securityContext)
		if err != nil {
			return err
		}
		return insertUserInvite(ctx, tx, securityContext, created.AppUserID,
			tokenHash, expiresTS, user.CreateTS, actorUserID)
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// ReissueInvite revokes every live invite of one invited login and issues a
// new one: the answer to a lost or expired link. A login that is not invited
// (already active, deleted, or of another tenant) is ErrNotFound: an active
// login has no use for an invite, and the other two are not this caller's
// to know about.
func (r *UserInviteRepository) ReissueInvite(
	ctx context.Context,
	appUserID int,
	securityContext int,
	tokenHash string,
	expiresTS float64,
	actorUserID int,
) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		ts := CurrentTimestamp()
		if err := lockInvitedUser(
			ctx, tx, appUserID, securityContext,
		); err != nil {
			return err
		}
		if _, err := revokeLiveUserInviteList(
			ctx, tx, appUserID, securityContext, ts, actorUserID,
		); err != nil {
			return err
		}
		return insertUserInvite(ctx, tx, securityContext, appUserID,
			tokenHash, expiresTS, ts, actorUserID)
	})
}

// RevokeInvite revokes every live invite of one invited login; the login
// stays invited, so a later ReissueInvite (or a delete) is still open. A
// login with no live invite is ErrConflict: there is nothing to revoke, and
// the admin's screen is out of date.
func (r *UserInviteRepository) RevokeInvite(
	ctx context.Context,
	appUserID int,
	securityContext int,
	actorUserID int,
) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		ts := CurrentTimestamp()
		if err := lockInvitedUser(
			ctx, tx, appUserID, securityContext,
		); err != nil {
			return err
		}
		revokedCount, err := revokeLiveUserInviteList(
			ctx, tx, appUserID, securityContext, ts, actorUserID,
		)
		if err != nil {
			return err
		}
		if revokedCount == 0 {
			return clientMessage(
				ErrConflict, "This user has no live invite link to revoke",
			)
		}
		return nil
	})
}

// RetrieveLiveInvite answers POST /inviteCheck: who a live, unexpired invite
// is for. Like RetrieveUserByUserName it is not scoped by
// security_context_id: it runs before there is a caller, and the token is
// the only key. Every way a token can be unusable (unknown, expired,
// redeemed, revoked, its login deleted or no longer invited) is ErrNotFound,
// so the answer says nothing about which.
func (r *UserInviteRepository) RetrieveLiveInvite(
	ctx context.Context, tokenHash string, now float64,
) (*model.UserInvitePreviewModel, error) {
	var preview model.UserInvitePreviewModel
	err := r.db.QueryRow(ctx, `
		SELECT u.user_name, u.first_name, ui.expires_ts`+liveInviteJoin+`
		  AND `+liveUserInvitePredicate("ui."),
		tokenHash, now, model.AccountStatusInvited,
	).Scan(&preview.UserName, &preview.FirstName, &preview.ExpiresTS)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read invite: %w", err)
	}
	return &preview, nil
}

// RedeemInvite spends a live invite: the login gets pwHash and becomes
// active, and the invite is marked redeemed with the client address. It
// answers the login's user name so the web app can sign in with it.
// Unusable tokens are ErrNotFound on the same terms as RetrieveLiveInvite;
// the row locks make a second, concurrent redeem of the same token one of
// those.
func (r *UserInviteRepository) RedeemInvite(
	ctx context.Context, tokenHash, pwHash, sourceIP string,
) (string, error) {
	var userName string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		ts := CurrentTimestamp()
		var inviteID, securityContext, appUserID int
		err := tx.QueryRow(ctx, `
			SELECT ui.user_invite_id, ui.security_context_id,
			       ui.app_user_id, u.user_name`+liveInviteJoin+`
			  AND `+liveUserInvitePredicate("ui.")+`
			FOR UPDATE OF ui, u`,
			tokenHash, ts, model.AccountStatusInvited,
		).Scan(&inviteID, &securityContext, &appUserID, &userName)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("failed to lock invite: %w", err)
		}

		// The invited person is the author of both writes: nobody else is
		// signed in, and the token proves they hold the link.
		if _, err := tx.Exec(ctx, `
			UPDATE app_user
			SET pw_hash = $1, account_status = $2, update_ts = $3,
			    update_user_id = $4
			WHERE app_user_id = $4 AND security_context_id = $5`,
			pwHash, model.AccountStatusActive, ts, appUserID,
			securityContext,
		); err != nil {
			return fmt.Errorf("failed to activate invited user: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE user_invite
			SET redeemed_ts = $1, redeemed_source_ip = $2, update_ts = $1,
			    update_user_id = $3
			WHERE user_invite_id = $4 AND security_context_id = $5`,
			ts, sourceIP, appUserID, inviteID, securityContext,
		); err != nil {
			return fmt.Errorf("failed to mark invite redeemed: %w", err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return userName, nil
}

// lockInvitedUser locks one invited login of this tenant FOR UPDATE, so a
// reissue and a redeem of the same login serialise. Anything else is
// ErrNotFound.
func lockInvitedUser(
	ctx context.Context, tx pgx.Tx, appUserID, securityContext int,
) error {
	var lockedID int
	err := tx.QueryRow(ctx, `
		SELECT app_user_id
		FROM app_user
		WHERE app_user_id = $1 AND security_context_id = $2
		  AND delete_user_id IS NULL AND account_status = $3
		FOR UPDATE`,
		appUserID, securityContext, model.AccountStatusInvited,
	).Scan(&lockedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return clientMessage(ErrNotFound, "No invited user with that id")
	}
	if err != nil {
		return fmt.Errorf("failed to lock invited user: %w", err)
	}
	return nil
}

func insertUserInvite(
	ctx context.Context,
	tx pgx.Tx,
	securityContext, appUserID int,
	tokenHash string,
	expiresTS, ts float64,
	userID int,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_invite (security_context_id, app_user_id,
		       token_hash, expires_ts, create_ts, create_user_id, update_ts,
		       update_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $5, $6)`,
		securityContext, appUserID, tokenHash, expiresTS, ts, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to create invite: %w", err)
	}
	return nil
}

// revokeLiveUserInviteList revokes the login's live invites, expired or
// not, and answers how many it revoked.
func revokeLiveUserInviteList(
	ctx context.Context,
	tx pgx.Tx,
	appUserID, securityContext int,
	ts float64,
	userID int,
) (int64, error) {
	result, err := tx.Exec(ctx, `
		UPDATE user_invite
		SET revoked_ts = $1, update_ts = $1, update_user_id = $2
		WHERE app_user_id = $3 AND security_context_id = $4
		  AND `+liveUserInvitePredicate(""),
		ts, userID, appUserID, securityContext,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to revoke invites: %w", err)
	}
	return result.RowsAffected(), nil
}

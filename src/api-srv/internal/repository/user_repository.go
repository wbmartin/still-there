package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wbmartin/starter-api/internal/model"
)

// CurrentTimestamp returns the current timestamp as float64
func CurrentTimestamp() float64 {
	return float64(time.Now().Unix()) + float64(time.Now().Nanosecond())/1e9
}

// UserRepository handles database operations for users
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// RetrieveUserByUserName retrieves a user by username
func (r *UserRepository) RetrieveUserByUserName(
	ctx context.Context, username string,
) (*model.AppUserModel, error) {
	query := `
		SELECT app_user_id, user_name, pw_hash, first_name, last_name, email,
		       security_context_id, app_user_group_membership_list, mfa_exempt, totp_seed, mfa_setup,
		       preference_jsonb,
		       create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM app_user
		WHERE user_name = $1 AND delete_user_id IS NULL`

	var user model.AppUserModel
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		row := r.db.QueryRow(ctx, query, username)
		return row.Scan(
			&user.AppUserID, &user.UserName, &user.PwHash, &user.FirstName,
			&user.LastName, &user.Email, &user.SecurityContext,
			&user.AppUserGroupMembershipList, &user.MFAExempt, &user.TOTPSeed,
			&user.MFASetup,
			&user.PreferenceJSONB,
			&user.CreateTS, &user.CreateUserID, &user.UpdateTS,
			&user.UpdateUserID, &user.DeleteTS, &user.DeleteUserID,
		)
	})

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// RetrieveUserGrantList retrieves user grants by security context

// GetGrantsForGroups retrieves all grants for a list of groups organized by security context
func (r *UserRepository) GetGrantsForGroups(
	ctx context.Context, groupIDs []int,
) (map[int][]int, error) {
	if len(groupIDs) == 0 {
		return make(map[int][]int), nil
	}

	query := `
		SELECT ug.security_context_id, ug.user_grant_list
		FROM app_user_group ug
		WHERE ug.app_user_group_id = ANY($1) AND ug.delete_user_id IS NULL`

	var grants map[int][]int
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		rowList, err := r.db.Query(ctx, query, groupIDs)
		if err != nil {
			return fmt.Errorf("failed to get grants for groups: %w", err)
		}
		defer rowList.Close()

		grants = make(map[int][]int)
		for rowList.Next() {
			var securityContext int
			var grantList []int
			err := rowList.Scan(&securityContext, &grantList)
			if err != nil {
				return fmt.Errorf("failed to scan grant list: %w", err)
			}

			if existing, exists := grants[securityContext]; exists {
				// Merge grant lists
				grants[securityContext] = append(existing, grantList...)
			} else {
				grants[securityContext] = grantList
			}
		}

		// Remove duplicates from grant lists
		for context, grantList := range grants {
			grants[context] = removeDuplicates(grantList)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return grants, nil
}

// CreateUser creates a new user within a security context
func (r *UserRepository) CreateUser(
	ctx context.Context, user *model.AppUserModel, securityContext int,
) (*model.AppUserModel, error) {
	// Validate that the user can only be created in the user's security context
	if user.SecurityContext != securityContext {
		return nil, fmt.Errorf(
			"cannot create user in different security context",
		)
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(user.UserName) {
		return nil, fmt.Errorf(
			"username must contain only alphanumeric characters, hyphens, and underscores",
		)
	}

	query := `
		INSERT INTO app_user (user_name, pw_hash, first_name, last_name, email,
		       security_context_id, app_user_group_membership_list, mfa_exempt,
		       totp_seed, mfa_setup,
		                     create_ts, create_user_id, update_ts, update_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING app_user_id`

	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		return r.db.QueryRow(
			ctx,
			query,
			user.UserName,
			user.PwHash,
			user.FirstName,
			user.LastName,
			user.Email,
			user.SecurityContext,
			user.AppUserGroupMembershipList,
			user.MFAExempt,
			// totp_seed/mfa_setup are NOT NULL with no default; a new user has no TOTP yet, so
			// seed empty + mfa_setup false (populated when the user completes MFA setup). Without
			// this, CreateUser fails: null value in column "totp_seed" violates not-null constraint.
			"",
			false,
			user.CreateTS,
			user.CreateUserID,
			user.UpdateTS,
			user.UpdateUserID,
		).Scan(&user.AppUserID)
	})

	if err != nil {
		if isUniqueViolation(err) {
			// app_user.user_name is UNIQUE.
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// UpdateUserPassword updates a user's password
func (r *UserRepository) UpdateUserPassword(
	ctx context.Context, userID int, newPasswordHash string,
	updateTS float64, updateUserID int,
) error {
	query := `
		UPDATE app_user
		SET pw_hash = $1, update_ts = $2, update_user_id = $3
		WHERE app_user_id = $4 AND delete_user_id IS NULL`

	var result pgconn.CommandTag
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		var err error
		result, err = r.db.Exec(ctx, query,
			newPasswordHash, updateTS, updateUserID, userID)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdateUserTOTPSeed updates a user's TOTP seed with optimistic locking
func (r *UserRepository) UpdateUserTOTPSeed(
	ctx context.Context, userID int, totpSeed string,
	expectedUpdateTS float64, updateUserID int,
) (newUpdateTS float64, err error) {
	newUpdateTS = CurrentTimestamp()
	query := `
		UPDATE app_user
		SET totp_seed = $1, mfa_setup = false, update_ts = $2, update_user_id = $3
		WHERE app_user_id = $4 AND update_ts = $5 AND delete_user_id IS NULL`

	var result pgconn.CommandTag
	err = ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		var execErr error
		result, execErr = r.db.Exec(ctx, query,
			totpSeed, newUpdateTS, updateUserID, userID, expectedUpdateTS)
		return execErr
	})
	if err != nil {
		return 0, fmt.Errorf("failed to update user TOTP seed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return 0, fmt.Errorf("user not found or concurrent modification")
	}

	return newUpdateTS, nil
}

// UpdateUserMFASetup updates a user's MFA setup status with optimistic locking
func (r *UserRepository) UpdateUserMFASetup(
	ctx context.Context, userID int, mfaSetup bool,
	expectedUpdateTS float64, updateUserID int,
) (newUpdateTS float64, err error) {
	newUpdateTS = CurrentTimestamp()
	query := `
		UPDATE app_user
		SET mfa_setup = $1, update_ts = $2, update_user_id = $3
		WHERE app_user_id = $4 AND update_ts = $5 AND delete_user_id IS NULL`

	var result pgconn.CommandTag
	err = ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		var execErr error
		result, execErr = r.db.Exec(ctx, query,
			mfaSetup, newUpdateTS, updateUserID, userID, expectedUpdateTS)
		return execErr
	})
	if err != nil {
		return 0, fmt.Errorf("failed to update user MFA setup: %w", err)
	}

	if result.RowsAffected() == 0 {
		return 0, fmt.Errorf("user not found or concurrent modification")
	}

	return newUpdateTS, nil
}

// UpdateUserMFAExempt updates a user's MFA exempt status with optimistic locking
func (r *UserRepository) UpdateUserMFAExempt(
	ctx context.Context, userID int, mfaExempt bool,
	expectedUpdateTS float64, updateUserID int,
) (newUpdateTS float64, err error) {
	newUpdateTS = CurrentTimestamp()
	query := `
		UPDATE app_user
		SET mfa_exempt = $1, update_ts = $2, update_user_id = $3
		WHERE app_user_id = $4 AND update_ts = $5 AND delete_user_id IS NULL`

	var result pgconn.CommandTag
	err = ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		var execErr error
		result, execErr = r.db.Exec(ctx, query,
			mfaExempt, newUpdateTS, updateUserID, userID, expectedUpdateTS)
		return execErr
	})
	if err != nil {
		return 0, fmt.Errorf("failed to update user MFA exempt: %w", err)
	}

	if result.RowsAffected() == 0 {
		return 0, fmt.Errorf("user not found or concurrent modification")
	}

	return newUpdateTS, nil
}

// RetrieveUserList retrieves users for a security context,
// optionally filtered by idList
func (r *UserRepository) RetrieveUserList(
	ctx context.Context, securityContext int, idList []int, updateTs float64,
) ([]*model.AppUserModel, error) {
	query := `
		SELECT app_user_id, user_name, first_name, last_name, email,
		       security_context_id, app_user_group_membership_list, mfa_exempt, totp_seed, mfa_setup,
		       preference_jsonb,
		       create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM app_user
		WHERE security_context_id = $1 AND delete_user_id IS NULL`

	args := []interface{}{securityContext}

	if len(idList) > 0 {
		query += ` AND app_user_id = ANY($2)`
		args = append(args, idList)
	}

	if updateTs > 0 {
		paramNum := len(args) + 1
		query += ` AND update_ts > $` + strconv.Itoa(paramNum)
		args = append(args, updateTs)
	}

	query += ` ORDER BY user_name`

	var users []*model.AppUserModel
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		rows, err := r.db.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to get users: %w", err)
		}
		defer rows.Close()

		users = []*model.AppUserModel{}
		for rows.Next() {
			var user model.AppUserModel
			err := rows.Scan(
				&user.AppUserID,
				&user.UserName,
				&user.FirstName,
				&user.LastName,
				&user.Email,
				&user.SecurityContext,
				&user.AppUserGroupMembershipList,
				&user.MFAExempt,
				&user.TOTPSeed,
				&user.MFASetup,
				&user.PreferenceJSONB,
				&user.CreateTS,
				&user.CreateUserID,
				&user.UpdateTS,
				&user.UpdateUserID,
				&user.DeleteTS,
				&user.DeleteUserID,
			)
			if err != nil {
				return fmt.Errorf("failed to scan user: %w", err)
			}
			users = append(users, &user)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return users, nil
}

// UpdateUser updates a user with optimistic locking within a security context
func (r *UserRepository) UpdateUser(
	ctx context.Context,
	user *model.AppUserModel,
	clientUpdateTS float64,
	securityContext int,
) error {
	query := `
		UPDATE app_user
		SET user_name = $1, first_name = $2, last_name = $3, email = $4,
		    app_user_group_membership_list = $5, mfa_exempt = $6,
		    update_ts = $7, update_user_id = $8
		WHERE app_user_id = $9 AND security_context_id = $10 AND update_ts = $11 AND delete_user_id IS NULL`

	var result pgconn.CommandTag
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		var execErr error
		result, execErr = r.db.Exec(ctx, query,
			user.UserName, user.FirstName, user.LastName, user.Email,
			user.AppUserGroupMembershipList, user.MFAExempt,
			user.UpdateTS, user.UpdateUserID,
			user.AppUserID, securityContext, clientUpdateTS)
		return execErr
	})
	if err != nil {
		if isUniqueViolation(err) {
			// app_user.user_name is UNIQUE.
			return ErrDuplicate
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if user exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM app_user
			WHERE app_user_id = $1 AND delete_user_id IS NULL)`
		err = ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
			return r.db.QueryRow(ctx, checkQuery, user.AppUserID).Scan(&exists)
		})
		if err != nil {
			return fmt.Errorf("failed to check user existence: %w", err)
		}
		if !exists {
			return ErrNotFound
		}
		return ErrConflict
	}

	return nil
}

// DeleteUser soft deletes a user with optimistic locking
func (r *UserRepository) DeleteUser(
	ctx context.Context,
	userID int,
	securityContext int,
	clientUpdateTS float64,
	deleteTS float64,
	deleteUserID int,
) (int, error) {
	query := `
		UPDATE app_user
		SET delete_ts = $1, delete_user_id = $2
		WHERE app_user_id = $3 AND security_context_id = $4 AND update_ts = $5 AND delete_user_id IS NULL`

	var result pgconn.CommandTag
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		var execErr error
		result, execErr = r.db.Exec(
			ctx,
			query,
			deleteTS,
			deleteUserID,
			userID,
			securityContext,
			clientUpdateTS,
		)
		return execErr
	})
	if err != nil {
		return 0, fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if user exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM app_user WHERE app_user_id = $1 AND delete_user_id IS NULL)`
		err = ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
			return r.db.QueryRow(ctx, checkQuery, userID).Scan(&exists)
		})
		if err != nil {
			return 0, fmt.Errorf("failed to check user existence: %w", err)
		}
		if !exists {
			return 0, ErrNotFound
		}
		return 0, ErrConflict
	}

	return userID, nil
}

// AddUserListToGroup adds multiple users to a group with optimistic locking
func (r *UserRepository) AddUserListToGroup(
	ctx context.Context,
	userIds []int,
	groupId int,
	updateTsList []float64,
	updateUserId int,
	securityContext int,
) ([]*model.AppUserModel, error) {
	if len(userIds) != len(updateTsList) {
		return nil, fmt.Errorf(
			"userIds and updateTsList must have the same length",
		)
	}

	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Update each user (only if group not already in membership list)
	for i, userId := range userIds {
		result, err := tx.Exec(
			ctx,
			`
			UPDATE app_user
			SET app_user_group_membership_list = array_append(app_user_group_membership_list, $2),
				update_ts = $3,
				update_user_id = $4
			WHERE app_user_id = $1 AND update_ts = $5 AND delete_user_id IS NULL AND $2 != ALL(app_user_group_membership_list)`,
			userId,
			groupId,
			float64(time.Now().UnixNano())/1e9,
			updateUserId,
			updateTsList[i],
		)

		if err != nil {
			return nil, fmt.Errorf("failed to update user %d: %w", userId, err)
		}
		if result.RowsAffected() == 0 {
			return nil, fmt.Errorf(
				"user %d not found, already has group, or concurrent update",
				userId,
			)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return updated users; rescan required for updated group membership
	return r.RetrieveUserList(ctx, securityContext, userIds, 0)
}

// RemoveUserListFromGroup removes multiple users from a group with optimistic locking
func (r *UserRepository) RemoveUserListFromGroup(
	ctx context.Context,
	userIds []int,
	groupId int,
	updateTsList []float64,
	updateUserId int,
	securityContext int,
) ([]*model.AppUserModel, error) {
	if len(userIds) != len(updateTsList) {
		return nil, fmt.Errorf(
			"userIds and updateTsList must have the same length",
		)
	}

	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Update each user
	for i, userId := range userIds {
		result, err := tx.Exec(
			ctx,
			`
			UPDATE app_user
			SET app_user_group_membership_list = array_remove(app_user_group_membership_list, $2),
				update_ts = $3,
				update_user_id = $4
			WHERE app_user_id = $1 AND update_ts = $5 AND delete_user_id IS NULL`,
			userId,
			groupId,
			float64(time.Now().UnixNano())/1e9,
			updateUserId,
			updateTsList[i],
		)

		if err != nil {
			return nil, fmt.Errorf("failed to update user %d: %w", userId, err)
		}
		if result.RowsAffected() == 0 {
			return nil, fmt.Errorf(
				"user %d not found or concurrent update",
				userId,
			)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return updated users; rescan required for updated group membership
	return r.RetrieveUserList(ctx, securityContext, userIds, 0)
}

// removeDuplicates removes duplicate integers from a slice
func removeDuplicates(slice []int) []int {
	if len(slice) == 0 {
		return []int{}
	}

	keys := make(map[int]bool)
	var resultList []int
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			resultList = append(resultList, item)
		}
	}
	return resultList
}

// RetrieveUserByID reads one live user in the caller's security context. It
// is the read behind GET /me; the id comes from the JWT, never the client.
func (r *UserRepository) RetrieveUserByID(
	ctx context.Context,
	userID int,
	securityContext int,
) (*model.AppUserModel, error) {
	query := `
		SELECT app_user_id, user_name, pw_hash, first_name, last_name, email,
		       security_context_id, app_user_group_membership_list, mfa_exempt, totp_seed, mfa_setup,
		       preference_jsonb,
		       create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM app_user
		WHERE app_user_id = $1 AND security_context_id = $2 AND delete_user_id IS NULL`

	var user model.AppUserModel
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		return r.db.QueryRow(ctx, query, userID, securityContext).Scan(
			&user.AppUserID,
			&user.UserName,
			&user.PwHash,
			&user.FirstName,
			&user.LastName,
			&user.Email,
			&user.SecurityContext,
			&user.AppUserGroupMembershipList,
			&user.MFAExempt,
			&user.TOTPSeed,
			&user.MFASetup,
			&user.PreferenceJSONB,
			&user.CreateTS,
			&user.CreateUserID,
			&user.UpdateTS,
			&user.UpdateUserID,
			&user.DeleteTS,
			&user.DeleteUserID,
		)
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to retrieve user by id: %w", err)
	}
	return &user, nil
}

// UpdateOwnProfile is the narrow write behind PUT /me: the columns a user
// may change about themselves, nothing else (no user name, groups, MFA
// flags). Optimistic lock on update_ts; the row's new update_ts is returned.
func (r *UserRepository) UpdateOwnProfile(
	ctx context.Context,
	userID int,
	securityContext int,
	firstName string,
	lastName string,
	email string,
	preference json.RawMessage,
	expectedUpdateTS float64,
) (newUpdateTS float64, err error) {
	newUpdateTS = CurrentTimestamp()
	query := `
		UPDATE app_user
		SET first_name = $1, last_name = $2, email = $3, preference_jsonb = $4,
		    update_ts = $5, update_user_id = $6
		WHERE app_user_id = $6 AND security_context_id = $7 AND update_ts = $8
		      AND delete_user_id IS NULL`

	var result pgconn.CommandTag
	err = ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		var execErr error
		result, execErr = r.db.Exec(ctx, query,
			firstName, lastName, email, jsonbParam(preference),
			newUpdateTS, userID, securityContext, expectedUpdateTS)
		return execErr
	})
	if err != nil {
		return 0, fmt.Errorf("failed to update own profile: %w", err)
	}
	if result.RowsAffected() == 0 {
		var exists bool
		checkErr := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
			return r.db.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM app_user
				 WHERE app_user_id = $1 AND security_context_id = $2 AND delete_user_id IS NULL)`,
				userID, securityContext).Scan(&exists)
		})
		if checkErr != nil {
			return 0, fmt.Errorf("failed to assess user state: %w", checkErr)
		}
		if !exists {
			return 0, ErrNotFound
		}
		return 0, ErrConflict
	}
	return newUpdateTS, nil
}

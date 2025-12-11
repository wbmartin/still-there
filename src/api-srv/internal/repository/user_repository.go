package repository

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wbmartin/starter-api/internal/model"
)

// UserRepository handles database operations for users
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// ReadUserByUserName retrieves a user by username
func (r *UserRepository) ReadUserByUserName(ctx context.Context, username string) (*model.AppUserModel, error) {
	query := `
		SELECT app_user_id, user_name, pw_hash, first_name, last_name, email,
		       security_context, user_group_membership_list, mfa_exempt,
		       create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM app_user
		WHERE user_name = $1 AND delete_user_id IS NULL`

	var user model.AppUserModel
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		return r.db.QueryRow(ctx, query, username).Scan(
			&user.AppUserID, &user.UserName, &user.PwHash, &user.FirstName, &user.LastName, &user.Email,
			&user.SecurityContext, &user.UserGroupMembershipList, &user.MFAExempt,
			&user.CreateTS, &user.CreateUserID, &user.UpdateTS, &user.UpdateUserID,
			&user.DeleteTS, &user.DeleteUserID,
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

// ReadUserGrantList retrieves user grants by security context
func (r *UserRepository) ReadUserGrantList(ctx context.Context, userGroupIDs []int) (map[int][]int, error) {
	if len(userGroupIDs) == 0 {
		return make(map[int][]int), nil
	}

	query := `
		SELECT ug.security_context, ug.user_grant_list
		FROM user_group ug
		WHERE ug.user_group_id = ANY($1) AND ug.delete_user_id IS NULL`

	rowList, err := r.db.Query(ctx, query, userGroupIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get user grants: %w", err)
	}
	defer rowList.Close()

	grants := make(map[int][]int)
	for rowList.Next() {
		var securityContext int
		var grantList []int
		err := rowList.Scan(&securityContext, &grantList)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user grants: %w", err)
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

	return grants, nil
}

// CreateUser creates a new user
func (r *UserRepository) CreateUser(ctx context.Context, user *model.AppUserModel) (*model.AppUserModel, error) {
	if !regexp.MustCompile(`^[a-zA-Z0-9-]+$`).MatchString(user.UserName) {
		return nil, fmt.Errorf("username must contain only alphanumeric characters")
	}

	query := `
		INSERT INTO app_user (user_name, pw_hash, first_name, last_name, email,
		                     security_context, user_group_membership_list, mfa_exempt,
		                     create_ts, create_user_id, update_ts, update_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING app_user_id`

	err := r.db.QueryRow(ctx, query,
		user.UserName, user.PwHash, user.FirstName, user.LastName, user.Email,
		user.SecurityContext, user.UserGroupMembershipList, user.MFAExempt,
		user.CreateTS, user.CreateUserID, user.UpdateTS, user.UpdateUserID,
	).Scan(&user.AppUserID)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// UpdateUserPassword updates a user's password
func (r *UserRepository) UpdateUserPassword(ctx context.Context, userID int, newPasswordHash string, updateTS float64, updateUserID int) error {
	query := `
		UPDATE app_user
		SET pw_hash = $1, update_ts = $2, update_user_id = $3
		WHERE app_user_id = $4 AND delete_user_id IS NULL`

	result, err := r.db.Exec(ctx, query, newPasswordHash, updateTS, updateUserID, userID)
	if err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// ReadUserList retrieves users for a security context, optionally filtered by idList
func (r *UserRepository) ReadUserList(ctx context.Context, securityContext int, idList []int) ([]*model.AppUserModel, error) {
	query := `
		SELECT app_user_id, user_name, first_name, last_name, email,
		       security_context, user_group_membership_list, mfa_exempt,
		       create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM app_user
		WHERE security_context = $1 AND delete_user_id IS NULL`

	args := []interface{}{securityContext}

	if len(idList) > 0 {
		query += ` AND app_user_id = ANY($2)`
		args = append(args, idList)
	}

	query += ` ORDER BY user_name`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	defer rows.Close()

	var users []*model.AppUserModel
	for rows.Next() {
		var user model.AppUserModel
		err := rows.Scan(
			&user.AppUserID, &user.UserName, &user.FirstName, &user.LastName, &user.Email,
			&user.SecurityContext, &user.UserGroupMembershipList, &user.MFAExempt,
			&user.CreateTS, &user.CreateUserID, &user.UpdateTS, &user.UpdateUserID,
			&user.DeleteTS, &user.DeleteUserID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	return users, nil
}

// UpdateUser updates a user with optimistic locking
func (r *UserRepository) UpdateUser(ctx context.Context, user *model.AppUserModel, clientUpdateTS float64) error {
	query := `
		UPDATE app_user
		SET user_name = $1, first_name = $2, last_name = $3, email = $4,
		    user_group_membership_list = $5, mfa_exempt = $6,
		    update_ts = $7, update_user_id = $8
		WHERE app_user_id = $9 AND update_ts = $10 AND delete_user_id IS NULL`

	result, err := r.db.Exec(ctx, query,
		user.UserName, user.FirstName, user.LastName, user.Email,
		user.UserGroupMembershipList, user.MFAExempt,
		user.UpdateTS, user.UpdateUserID,
		user.AppUserID, clientUpdateTS)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if user exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM app_user WHERE app_user_id = $1 AND delete_user_id IS NULL)`
		err = r.db.QueryRow(ctx, checkQuery, user.AppUserID).Scan(&exists)
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
func (r *UserRepository) DeleteUser(ctx context.Context, userID int, securityContext int, clientUpdateTS float64, deleteTS float64, deleteUserID int) (int, error) {
	query := `
		UPDATE app_user
		SET delete_ts = $1, delete_user_id = $2
		WHERE app_user_id = $3 AND security_context = $4 AND update_ts = $5 AND delete_user_id IS NULL`

	result, err := r.db.Exec(ctx, query, deleteTS, deleteUserID, userID, securityContext, clientUpdateTS)
	if err != nil {
		return 0, fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if user exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM app_user WHERE app_user_id = $1 AND delete_user_id IS NULL)`
		err = r.db.QueryRow(ctx, checkQuery, userID).Scan(&exists)
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
func (r *UserRepository) AddUserListToGroup(ctx context.Context, userIds []int, groupId int, updateTsList []float64, updateUserId int, securityContext int) ([]*model.AppUserModel, error) {
	if len(userIds) != len(updateTsList) {
		return nil, fmt.Errorf("userIds and updateTsList must have the same length")
	}

	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update each user (only if group not already in membership list)
	for i, userId := range userIds {
		result, err := tx.Exec(ctx, `
			UPDATE app_user
			SET user_group_membership_list = array_append(user_group_membership_list, $2),
				update_ts = $3,
				update_user_id = $4
			WHERE app_user_id = $1 AND update_ts = $5 AND delete_user_id IS NULL AND $2 != ALL(user_group_membership_list)`,
			userId, groupId, float64(time.Now().UnixNano())/1e9, updateUserId, updateTsList[i])

		if err != nil {
			return nil, fmt.Errorf("failed to update user %d: %w", userId, err)
		}
		if result.RowsAffected() == 0 {
			return nil, fmt.Errorf("user %d not found, already has group, or concurrent update", userId)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return updated users
	return r.ReadUserList(ctx, securityContext, userIds)
}

// RemoveUserListFromGroup removes multiple users from a group with optimistic locking
func (r *UserRepository) RemoveUserListFromGroup(ctx context.Context, userIds []int, groupId int, updateTsList []float64, updateUserId int, securityContext int) ([]*model.AppUserModel, error) {
	if len(userIds) != len(updateTsList) {
		return nil, fmt.Errorf("userIds and updateTsList must have the same length")
	}

	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update each user
	for i, userId := range userIds {
		result, err := tx.Exec(ctx, `
			UPDATE app_user
			SET user_group_membership_list = array_remove(user_group_membership_list, $2),
				update_ts = $3,
				update_user_id = $4
			WHERE app_user_id = $1 AND update_ts = $5 AND delete_user_id IS NULL`,
			userId, groupId, float64(time.Now().UnixNano())/1e9, updateUserId, updateTsList[i])

		if err != nil {
			return nil, fmt.Errorf("failed to update user %d: %w", userId, err)
		}
		if result.RowsAffected() == 0 {
			return nil, fmt.Errorf("user %d not found or concurrent update", userId)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return updated users
	return r.ReadUserList(ctx, securityContext, userIds)
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

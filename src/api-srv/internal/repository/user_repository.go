package repository

import (
	"context"
	"fmt"

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

// GetUserByUserName retrieves a user by username
func (r *UserRepository) GetUserByUserName(ctx context.Context, username string) (*model.AppUserModel, error) {
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

// GetUserGrantList retrieves user grants by security context
func (r *UserRepository) GetUserGrantList(ctx context.Context, userGroupIDs []int) (map[int][]int, error) {
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

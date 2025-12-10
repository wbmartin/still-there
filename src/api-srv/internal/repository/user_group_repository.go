package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wbmartin/starter-api/internal/model"
)

// UserGroupRepository handles database operations for user groups
type UserGroupRepository struct {
	db *pgxpool.Pool
}

// NewUserGroupRepository creates a new user group repository
func NewUserGroupRepository(db *pgxpool.Pool) *UserGroupRepository {
	return &UserGroupRepository{db: db}
}

// GetUserGroupListByIDList retrieves user groups by IDs
func (r *UserGroupRepository) GetUserGroupListByIDList(ctx context.Context, groupIDs []int) ([]*model.UserGroupModel, error) {
	if len(groupIDs) == 0 {
		return []*model.UserGroupModel{}, nil
	}

	query := `
		SELECT user_group_id, group_name, user_grant_list, security_context,
		       create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM user_group
		WHERE user_group_id = ANY($1) AND delete_user_id IS NULL`

	rowList, err := r.db.Query(ctx, query, groupIDs)
	if err != nil {
		return nil, err
	}
	defer rowList.Close()

	var groupList []*model.UserGroupModel
	for rowList.Next() {
		var group model.UserGroupModel
		err := rowList.Scan(
			&group.UserGroupID, &group.GroupName, &group.UserGrantList, &group.SecurityContext,
			&group.CreateTS, &group.CreateUserID, &group.UpdateTS, &group.UpdateUserID,
			&group.DeleteTS, &group.DeleteUserID,
		)
		if err != nil {
			return nil, err
		}
		groupList = append(groupList, &group)
	}

	return groupList, nil
}

// ReadUserGroupList retrieves all user groups for a security context
func (r *UserGroupRepository) ReadUserGroupList(ctx context.Context, securityContext int) ([]*model.UserGroupModel, error) {
	query := `
		SELECT user_group_id, group_name, user_grant_list, security_context,
		       create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM user_group
		WHERE security_context = $1 AND delete_user_id IS NULL
		ORDER BY group_name`

	rows, err := r.db.Query(ctx, query, securityContext)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*model.UserGroupModel
	for rows.Next() {
		var group model.UserGroupModel
		err := rows.Scan(
			&group.UserGroupID, &group.GroupName, &group.UserGrantList, &group.SecurityContext,
			&group.CreateTS, &group.CreateUserID, &group.UpdateTS, &group.UpdateUserID,
			&group.DeleteTS, &group.DeleteUserID,
		)
		if err != nil {
			return nil, err
		}
		groups = append(groups, &group)
	}

	return groups, nil
}

// CreateUserGroup creates a new user group
func (r *UserGroupRepository) CreateUserGroup(ctx context.Context, group *model.UserGroupModel) (*model.UserGroupModel, error) {
	query := `
		INSERT INTO user_group (group_name, user_grant_list, security_context,
		                        create_ts, create_user_id, update_ts, update_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING user_group_id`

	var newID int
	err := r.db.QueryRow(ctx, query,
		group.GroupName, group.UserGrantList, group.SecurityContext,
		group.CreateTS, group.CreateUserID, group.UpdateTS, group.UpdateUserID).Scan(&newID)
	if err != nil {
		return nil, fmt.Errorf("failed to create user group: %w", err)
	}

	group.UserGroupID = newID
	return group, nil
}

// UpdateUserGroup updates a user group with optimistic locking
func (r *UserGroupRepository) UpdateUserGroup(ctx context.Context, group *model.UserGroupModel, clientUpdateTS float64) error {
	query := `
		UPDATE user_group
		SET group_name = $1, user_grant_list = $2,
		    update_ts = $3, update_user_id = $4
		WHERE user_group_id = $5 AND update_ts = $6 AND delete_user_id IS NULL`

	result, err := r.db.Exec(ctx, query,
		group.GroupName, group.UserGrantList,
		group.UpdateTS, group.UpdateUserID,
		group.UserGroupID, clientUpdateTS)
	if err != nil {
		return fmt.Errorf("failed to update user group: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if group exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM user_group WHERE user_group_id = $1 AND delete_user_id IS NULL)`
		err = r.db.QueryRow(ctx, checkQuery, group.UserGroupID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check user group existence: %w", err)
		}
		if !exists {
			return ErrNotFound
		}
		return ErrConflict
	}

	return nil
}

// DeleteUserGroup soft deletes a user group if no users are assigned
func (r *UserGroupRepository) DeleteUserGroup(ctx context.Context, groupID int, securityContext int, clientUpdateTS float64, deleteTS float64, deleteUserID int) (int, error) {
	// Check if any users are assigned to this group
	checkQuery := `
		SELECT COUNT(*)
		FROM app_user
		WHERE $1 = ANY(user_group_membership_list) AND delete_user_id IS NULL`
	var count int
	err := r.db.QueryRow(ctx, checkQuery, groupID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to check user assignments: %w", err)
	}
	if count > 0 {
		return 0, fmt.Errorf("cannot delete group: %d users are assigned to it", count)
	}

	// Delete the group
	query := `
		UPDATE user_group
		SET delete_ts = $1, delete_user_id = $2
		WHERE user_group_id = $3 AND security_context = $4 AND update_ts = $5 AND delete_user_id IS NULL`

	result, err := r.db.Exec(ctx, query, deleteTS, deleteUserID, groupID, securityContext, clientUpdateTS)
	if err != nil {
		return 0, fmt.Errorf("failed to delete user group: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if group exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM user_group WHERE user_group_id = $1 AND delete_user_id IS NULL)`
		err = r.db.QueryRow(ctx, checkQuery, groupID).Scan(&exists)
		if err != nil {
			return 0, fmt.Errorf("failed to check user group existence: %w", err)
		}
		if !exists {
			return 0, ErrNotFound
		}
		return 0, ErrConflict
	}

	return groupID, nil
}

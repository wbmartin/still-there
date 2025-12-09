package repository

import (
	"context"

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

// GetUserGroupList retrieves all user groups for a security context
func (r *UserGroupRepository) GetUserGroupList(ctx context.Context, securityContext int) ([]*model.UserGroupModel, error) {
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

package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wbmartin/starter-api/internal/model"
)

// AppUserGroupRepository handles database operations for user groups
type AppUserGroupRepository struct {
	db *pgxpool.Pool
}

// NewAppUserGroupRepository creates a new user group repository
func NewAppUserGroupRepository(db *pgxpool.Pool) *AppUserGroupRepository {
	return &AppUserGroupRepository{db: db}
}

// RetrieveAppUserGroupListByIDList retrieves user groups by IDs within a security context
func (r *AppUserGroupRepository) RetrieveAppUserGroupListByIDList(
	ctx context.Context, groupIDs []int, securityContext int,
) ([]*model.AppUserGroupModel, error) {
	if len(groupIDs) == 0 {
		return []*model.AppUserGroupModel{}, nil
	}

	query := `
		SELECT app_user_group_id, group_name, user_grant_list, security_context_id,
		       create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM app_user_group
		WHERE app_user_group_id = ANY($1) AND security_context_id = $2 AND delete_user_id IS NULL`

	var groupList []*model.AppUserGroupModel
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		rowList, err := r.db.Query(ctx, query, groupIDs, securityContext)
		if err != nil {
			return err
		}
		defer rowList.Close()

		groupList = []*model.AppUserGroupModel{}
		for rowList.Next() {
			var group model.AppUserGroupModel
			err := rowList.Scan(
				&group.AppUserGroupID, &group.GroupName, &group.UserGrantList,
				&group.SecurityContext, &group.CreateTS, &group.CreateUserID,
				&group.UpdateTS, &group.UpdateUserID, &group.DeleteTS,
				&group.DeleteUserID,
			)
			if err != nil {
				return err
			}
			groupList = append(groupList, &group)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return groupList, nil
}

// RetrieveAppUserGroupList retrieves all user groups for a security context
func (r *AppUserGroupRepository) RetrieveAppUserGroupList(
	ctx context.Context,
	idList []int,
	securityContext int,
) ([]*model.AppUserGroupModel, error) {
	selectClause := `
		SELECT app_user_group_id, group_name, user_grant_list, security_context_id,
		       create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM app_user_group`
	orderBy := ` ORDER BY group_name`

	// Scoping comes from buildScopedWhere; this list has no incremental fetch.
	whereClause, argList := buildScopedWhere(
		securityContext, 0, "app_user_group_id", idList)

	query := selectClause + whereClause + orderBy

	var groups []*model.AppUserGroupModel
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		rows, err := r.db.Query(ctx, query, argList...)
		if err != nil {
			return err
		}
		defer rows.Close()

		groups = []*model.AppUserGroupModel{}
		for rows.Next() {
			var group model.AppUserGroupModel
			err := rows.Scan(
				&group.AppUserGroupID,
				&group.GroupName,
				&group.UserGrantList,
				&group.SecurityContext,
				&group.CreateTS,
				&group.CreateUserID,
				&group.UpdateTS,
				&group.UpdateUserID,
				&group.DeleteTS,
				&group.DeleteUserID,
			)
			if err != nil {
				return err
			}
			groups = append(groups, &group)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return groups, nil
}

// CreateAppUserGroup creates a new user group within a security context
func (r *AppUserGroupRepository) CreateAppUserGroup(
	ctx context.Context, group *model.AppUserGroupModel, securityContext int,
) (*model.AppUserGroupModel, error) {
	// Validate that the group can only be created in the user's security context
	if group.SecurityContext != securityContext {
		return nil, fmt.Errorf(
			"cannot create user group in different security context",
		)
	}

	query := `
		INSERT INTO app_user_group (group_name, user_grant_list, security_context_id,
		                        create_ts, create_user_id, update_ts, update_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING app_user_group_id`

	var newID int
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		return r.db.QueryRow(ctx, query,
			group.GroupName, group.UserGrantList, group.SecurityContext,
			group.CreateTS, group.CreateUserID, group.UpdateTS,
			group.UpdateUserID).Scan(&newID)
	})
	if err != nil {
		if isUniqueViolation(err) {
			// (group_name, security_context_id) is UNIQUE.
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("failed to create user group: %w", err)
	}

	group.AppUserGroupID = newID
	return group, nil
}

// UpdateAppUserGroup updates a user group with optimistic locking within a security context
func (r *AppUserGroupRepository) UpdateAppUserGroup(
	ctx context.Context,
	group *model.AppUserGroupModel,
	clientUpdateTS float64,
	securityContext int,
) error {
	query := `
		UPDATE app_user_group
		SET group_name = $1, user_grant_list = $2,
		    update_ts = $3, update_user_id = $4
		WHERE app_user_group_id = $5 AND security_context_id = $6 AND update_ts = $7 AND delete_user_id IS NULL`

	var result pgconn.CommandTag
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		var execErr error
		result, execErr = r.db.Exec(ctx, query,
			group.GroupName, group.UserGrantList,
			group.UpdateTS, group.UpdateUserID,
			group.AppUserGroupID, securityContext, clientUpdateTS)
		return execErr
	})
	if err != nil {
		if isUniqueViolation(err) {
			// (group_name, security_context_id) is UNIQUE.
			return ErrDuplicate
		}
		return fmt.Errorf("failed to update user group: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if group exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM app_user_group
			WHERE app_user_group_id = $1 AND delete_user_id IS NULL)`
		err = ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
			return r.db.QueryRow(ctx, checkQuery, group.AppUserGroupID).
				Scan(&exists)
		})
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

// RetrieveMemberUserNameList returns the user_name of every live login that
// holds groupID, a group of securityContext. A login of any tenant can hold
// it (GetGrantsForGroups keys grants by the group's tenant), so the tenant
// filter is on the group, not on app_user.
func (r *AppUserGroupRepository) RetrieveMemberUserNameList(
	ctx context.Context,
	groupID int,
	securityContext int,
) ([]string, error) {
	query := `
		SELECT u.user_name
		FROM app_user u
		WHERE $1 = ANY(u.app_user_group_membership_list)
		  AND u.delete_user_id IS NULL
		  AND EXISTS (SELECT 1 FROM app_user_group g
		              WHERE g.app_user_group_id = $1
		                AND g.security_context_id = $2)`

	var userNameList []string
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		userNameList = nil
		rowList, err := r.db.Query(ctx, query, groupID, securityContext)
		if err != nil {
			return fmt.Errorf("failed to read group members: %w", err)
		}
		defer rowList.Close()
		for rowList.Next() {
			var userName string
			if err := rowList.Scan(&userName); err != nil {
				return fmt.Errorf("failed to scan group member: %w", err)
			}
			userNameList = append(userNameList, userName)
		}
		return rowList.Err()
	})
	if err != nil {
		return nil, err
	}
	return userNameList, nil
}

// DeleteAppUserGroup soft deletes a user group if no users are assigned
func (r *AppUserGroupRepository) DeleteAppUserGroup(
	ctx context.Context,
	groupID int,
	securityContext int,
	clientUpdateTS float64,
	deleteTS float64,
	deleteUserID int,
) (int, error) {
	// Check if any users are assigned to this group
	checkQuery := `
		SELECT COUNT(*)
		FROM app_user
		WHERE $1 = ANY(app_user_group_membership_list) AND delete_user_id IS NULL`
	var count int
	err := ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		return r.db.QueryRow(ctx, checkQuery, groupID).Scan(&count)
	})
	if err != nil {
		return 0, fmt.Errorf("failed to check user assignments: %w", err)
	}
	if count > 0 {
		return 0, fmt.Errorf(
			"cannot delete group: %d users are assigned to it",
			count,
		)
	}

	// Delete the group
	query := `
		UPDATE app_user_group
		SET delete_ts = $1, delete_user_id = $2
		WHERE app_user_group_id = $3 AND security_context_id = $4 AND update_ts = $5 AND delete_user_id IS NULL`

	var result pgconn.CommandTag
	err = ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
		var execErr error
		result, execErr = r.db.Exec(
			ctx,
			query,
			deleteTS,
			deleteUserID,
			groupID,
			securityContext,
			clientUpdateTS,
		)
		return execErr
	})
	if err != nil {
		return 0, fmt.Errorf("failed to delete user group: %w", err)
	}

	if result.RowsAffected() == 0 {
		// Check if group exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM app_user_group WHERE app_user_group_id = $1 AND delete_user_id IS NULL)`
		err = ExecuteWithRetry(ctx, DefaultRetryConfig(), func() error {
			return r.db.QueryRow(ctx, checkQuery, groupID).Scan(&exists)
		})
		if err != nil {
			return 0, fmt.Errorf(
				"failed to check user group existence: %w",
				err,
			)
		}
		if !exists {
			return 0, ErrNotFound
		}
		return 0, ErrConflict
	}

	return groupID, nil
}

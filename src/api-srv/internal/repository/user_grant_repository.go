package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wbmartin/starter-api/internal/model"
)

// UserGrantRepository handles database operations for user grants
type UserGrantRepository struct {
	db *pgxpool.Pool
}

// NewUserGrantRepository creates a new user grant repository
func NewUserGrantRepository(db *pgxpool.Pool) *UserGrantRepository {
	return &UserGrantRepository{db: db}
}

// GetUserGrants retrieves user grants by IDs
func (r *UserGrantRepository) GetUserGrants(ctx context.Context, grantIDs []int) ([]*model.UserGrantModel, error) {
	if len(grantIDs) == 0 {
		return []*model.UserGrantModel{}, nil
	}

	query := `
		SELECT user_grant_id, grant_short_code, grant_description, security_context,
		       create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id
		FROM user_grant
		WHERE user_grant_id = ANY($1) AND delete_user_id IS NULL`

	rowList, err := r.db.Query(ctx, query, grantIDs)
	if err != nil {
		return nil, err
	}
	defer rowList.Close()

	var grantList []*model.UserGrantModel
	for rowList.Next() {
		var grant model.UserGrantModel
		err := rowList.Scan(
			&grant.UserGrantID, &grant.GrantShortCode, &grant.GrantDescription, &grant.SecurityContext,
			&grant.CreateTS, &grant.CreateUserID, &grant.UpdateTS, &grant.UpdateUserID,
			&grant.DeleteTS, &grant.DeleteUserID,
		)
		if err != nil {
			return nil, err
		}
		grantList = append(grantList, &grant)
	}

	return grantList, nil
}

package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Table and primary-key names for the tables zeroRowOutcome explains.
// Constants, so it is never handed caller input to format into SQL.
const (
	appUserTable      = "app_user"
	appUserPK         = "app_user_id"
	appUserGroupTable = "app_user_group"
	appUserGroupPK    = "app_user_group_id"
	deviceTable       = "device"
	devicePK          = "device_id"
	beaconTable       = "beacon"
	beaconPK          = "beacon_id"
	contactTable      = "contact"
	contactPK         = "contact_id"
)

// zeroRowOutcome explains a scoped write that matched no row: absent from
// this tenant (ErrNotFound) or present with a stale update_ts (ErrConflict).
// The check carries the same tenant predicate as the write, so an id that
// lives in another tenant reads as absent and never answers 409.
func zeroRowOutcome(
	ctx context.Context,
	db *pgxpool.Pool,
	table string,
	pk string,
	id int,
	securityContext int,
) error {
	query := fmt.Sprintf(`
		SELECT EXISTS(
			SELECT 1 FROM %s
			WHERE %s = $1 AND security_context_id = $2
			  AND delete_user_id IS NULL)`, table, pk)

	var exists bool
	err := db.QueryRow(ctx, query, id, securityContext).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check %s existence: %w", table, err)
	}
	if !exists {
		return ErrNotFound
	}
	return ErrConflict
}

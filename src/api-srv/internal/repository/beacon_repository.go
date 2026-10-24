package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"stillthere-api/internal/model"
)

// BeaconRepository owns beacon. The bands are part of the Pi's config, so
// every write here bumps the parent device's config_version in the same
// transaction. The bump leaves the device's update_ts alone: an admin
// editing the device while another edits its bands does not 409.
type BeaconRepository struct {
	db *pgxpool.Pool
}

// NewBeaconRepository creates the beacon repository.
func NewBeaconRepository(db *pgxpool.Pool) *BeaconRepository {
	return &BeaconRepository{db: db}
}

const beaconColumnList = `
	beacon_id, device_id, address, label, present, last_seen_ts,
	last_rssi, last_report_ts, security_context_id, create_ts,
	create_user_id, update_ts, update_user_id, delete_ts, delete_user_id`

func scanBeacon(row pgx.Row, beacon *model.BeaconModel) error {
	return row.Scan(
		&beacon.BeaconID, &beacon.DeviceID, &beacon.Address, &beacon.Label,
		&beacon.Present, &beacon.LastSeenTS, &beacon.LastRSSI,
		&beacon.LastReportTS, &beacon.SecurityContext, &beacon.CreateTS,
		&beacon.CreateUserID, &beacon.UpdateTS, &beacon.UpdateUserID,
		&beacon.DeleteTS, &beacon.DeleteUserID,
	)
}

// bumpDeviceConfigVersion moves a live device of the tenant to its next
// config version. No such device is ErrNotFound: a band cannot hang off a
// device the caller cannot see.
func bumpDeviceConfigVersion(
	ctx context.Context, tx pgx.Tx, deviceID int, securityContext int,
) error {
	result, err := tx.Exec(ctx, `
		UPDATE device SET config_version = config_version + 1
		WHERE device_id = $1 AND security_context_id = $2
		  AND delete_user_id IS NULL`, deviceID, securityContext)
	if err != nil {
		return fmt.Errorf("failed to bump device config version: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RetrieveBeaconList returns the tenant's live beacons of live devices:
// all, by id, of one device (deviceID > 0), or changed since afterUpdateTs.
func (r *BeaconRepository) RetrieveBeaconList(
	ctx context.Context,
	idList []int,
	deviceID int,
	securityContext int,
	afterUpdateTs float64,
) ([]*model.BeaconModel, error) {
	whereClause, argList := buildParentScopedWhere(
		securityContext, afterUpdateTs, "beacon_id", idList,
		"device_id", deviceID)
	whereClause = withLiveParent(
		whereClause, "beacon", "device_id", "device", "device_id")
	rowList, err := r.db.Query(ctx, `SELECT `+beaconColumnList+`
		FROM beacon`+whereClause+` ORDER BY beacon_id`, argList...)
	if err != nil {
		return nil, fmt.Errorf("failed to query beacons: %w", err)
	}
	defer rowList.Close()

	beaconList := make([]*model.BeaconModel, 0)
	for rowList.Next() {
		var beacon model.BeaconModel
		if err := scanBeacon(rowList, &beacon); err != nil {
			return nil, fmt.Errorf("failed to scan beacon: %w", err)
		}
		beaconList = append(beaconList, &beacon)
	}
	if err := rowList.Err(); err != nil {
		return nil, fmt.Errorf("error iterating beacons: %w", err)
	}
	return beaconList, nil
}

// CreateBeacon adds a band to a live device of the tenant and answers the
// stored row. An address the device already watches is ErrDuplicate.
func (r *BeaconRepository) CreateBeacon(
	ctx context.Context, beacon *model.BeaconModel, securityContext int,
) (*model.BeaconModel, error) {
	var created model.BeaconModel
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := bumpDeviceConfigVersion(
			ctx, tx, beacon.DeviceID, securityContext,
		); err != nil {
			return err
		}
		return scanBeacon(tx.QueryRow(ctx, `
			INSERT INTO beacon (
				device_id, address, label, security_context_id,
				create_ts, create_user_id, update_ts, update_user_id
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING `+beaconColumnList,
			beacon.DeviceID, beacon.Address, beacon.Label, securityContext,
			beacon.CreateTS, beacon.CreateUserID, beacon.UpdateTS,
			beacon.UpdateUserID,
		), &created)
	})
	if isUniqueViolation(err) {
		return nil, clientMessage(ErrDuplicate,
			"This device already watches that address")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create beacon: %w", err)
	}
	return &created, nil
}

// UpdateBeacon writes the address and label under the optimistic lock and
// answers the stored row. The device is fixed at create. A new address is
// a different band, so the last report is cleared with it.
func (r *BeaconRepository) UpdateBeacon(
	ctx context.Context, beacon *model.BeaconModel, originalUpdateTS float64,
) (*model.BeaconModel, error) {
	var updated model.BeaconModel
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		err := scanBeacon(tx.QueryRow(ctx, `
			UPDATE beacon
			SET present = CASE WHEN address = $1 THEN present END,
			    last_seen_ts = CASE WHEN address = $1 THEN last_seen_ts END,
			    last_rssi = CASE WHEN address = $1 THEN last_rssi END,
			    last_report_ts =
			        CASE WHEN address = $1 THEN last_report_ts END,
			    address = $1, label = $2, update_ts = $3,
			    update_user_id = $4
			WHERE beacon_id = $5 AND security_context_id = $6
			  AND update_ts = $7 AND delete_user_id IS NULL
			RETURNING `+beaconColumnList,
			beacon.Address, beacon.Label, beacon.UpdateTS,
			beacon.UpdateUserID, beacon.BeaconID, beacon.SecurityContext,
			originalUpdateTS,
		), &updated)
		if errors.Is(err, pgx.ErrNoRows) {
			return zeroRowOutcome(ctx, r.db, beaconTable, beaconPK,
				beacon.BeaconID, beacon.SecurityContext)
		}
		if err != nil {
			return err
		}
		return bumpDeviceConfigVersion(
			ctx, tx, updated.DeviceID, beacon.SecurityContext)
	})
	if isUniqueViolation(err) {
		return nil, clientMessage(ErrDuplicate,
			"This device already watches that address")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update beacon: %w", err)
	}
	return &updated, nil
}

// DeleteBeacon soft deletes a band under the optimistic lock.
func (r *BeaconRepository) DeleteBeacon(
	ctx context.Context,
	beaconID int,
	securityContext int,
	originalUpdateTS float64,
	deleteTS float64,
	deleteUserID int,
) error {
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var deviceID int
		err := tx.QueryRow(ctx, `
			UPDATE beacon SET delete_ts = $1, delete_user_id = $2
			WHERE beacon_id = $3 AND security_context_id = $4
			  AND update_ts = $5 AND delete_user_id IS NULL
			RETURNING device_id`,
			deleteTS, deleteUserID, beaconID, securityContext,
			originalUpdateTS,
		).Scan(&deviceID)
		if errors.Is(err, pgx.ErrNoRows) {
			return zeroRowOutcome(ctx, r.db, beaconTable, beaconPK,
				beaconID, securityContext)
		}
		if err != nil {
			return err
		}
		return bumpDeviceConfigVersion(ctx, tx, deviceID, securityContext)
	})
	if err != nil {
		return fmt.Errorf("failed to delete beacon: %w", err)
	}
	return nil
}

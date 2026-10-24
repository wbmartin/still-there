package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"stillthere-api/internal/model"
)

// DeviceRepository owns device and device_checkin. Tokens never reach this
// package: the handler hashes the token (auth.HashDeviceToken) and every
// method here takes or stores the hash, which no read returns.
type DeviceRepository struct {
	db *pgxpool.Pool
}

// NewDeviceRepository creates the device repository.
func NewDeviceRepository(db *pgxpool.Pool) *DeviceRepository {
	return &DeviceRepository{db: db}
}

const deviceColumnList = `
	device_id, device_name, checkin_interval_sec,
	missed_checkin_before_alarm, absent_after_sec, config_version,
	token_issued_ts, last_checkin_ts, last_checkin_json, last_agent_version,
	security_context_id, create_ts, create_user_id, update_ts,
	update_user_id, delete_ts, delete_user_id`

func scanDevice(row pgx.Row, device *model.DeviceModel) error {
	return row.Scan(
		&device.DeviceID, &device.DeviceName, &device.CheckinIntervalSec,
		&device.MissedCheckinBeforeAlarm, &device.AbsentAfterSec,
		&device.ConfigVersion, &device.TokenIssuedTS, &device.LastCheckinTS,
		&device.LastCheckinJSON, &device.LastAgentVersion,
		&device.SecurityContext, &device.CreateTS, &device.CreateUserID,
		&device.UpdateTS, &device.UpdateUserID, &device.DeleteTS,
		&device.DeleteUserID,
	)
}

// RetrieveDeviceList returns the tenant's live devices: all, by id, or
// changed since afterUpdateTs (buildScopedWhere).
func (r *DeviceRepository) RetrieveDeviceList(
	ctx context.Context,
	idList []int,
	securityContext int,
	afterUpdateTs float64,
) ([]*model.DeviceModel, error) {
	whereClause, argList := buildScopedWhere(
		securityContext, afterUpdateTs, "device_id", idList)
	rowList, err := r.db.Query(ctx, `SELECT `+deviceColumnList+`
		FROM device`+whereClause+` ORDER BY device_id`, argList...)
	if err != nil {
		return nil, fmt.Errorf("failed to query devices: %w", err)
	}
	defer rowList.Close()

	deviceList := make([]*model.DeviceModel, 0)
	for rowList.Next() {
		var device model.DeviceModel
		if err := scanDevice(rowList, &device); err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}
		deviceList = append(deviceList, &device)
	}
	if err := rowList.Err(); err != nil {
		return nil, fmt.Errorf("error iterating devices: %w", err)
	}
	return deviceList, nil
}

// CreateDevice inserts device at config version 1 with no token and
// answers the stored row, so nothing else a client sent comes back. The
// caller fills the create/update stamps.
func (r *DeviceRepository) CreateDevice(
	ctx context.Context, device *model.DeviceModel, securityContext int,
) (*model.DeviceModel, error) {
	var created model.DeviceModel
	err := scanDevice(r.db.QueryRow(ctx, `
		INSERT INTO device (
			device_name, checkin_interval_sec, missed_checkin_before_alarm,
			absent_after_sec, config_version, security_context_id,
			create_ts, create_user_id, update_ts, update_user_id
		)
		VALUES ($1, $2, $3, $4, 1, $5, $6, $7, $8, $9)
		RETURNING `+deviceColumnList,
		device.DeviceName, device.CheckinIntervalSec,
		device.MissedCheckinBeforeAlarm, device.AbsentAfterSec,
		securityContext, device.CreateTS, device.CreateUserID,
		device.UpdateTS, device.UpdateUserID,
	), &created)
	if err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}
	return &created, nil
}

// UpdateDevice writes the name and alarm settings under the optimistic
// lock and answers the stored row. A change to any alarm setting bumps
// config_version, which the Pi reads from its next check-in reply; a
// rename alone does not (the Pi does not hold the name).
func (r *DeviceRepository) UpdateDevice(
	ctx context.Context, device *model.DeviceModel, originalUpdateTS float64,
) (*model.DeviceModel, error) {
	var updated model.DeviceModel
	err := scanDevice(r.db.QueryRow(ctx, `
		UPDATE device
		SET device_name = $1, checkin_interval_sec = $2,
		    missed_checkin_before_alarm = $3, absent_after_sec = $4,
		    config_version = config_version + CASE
		        WHEN (checkin_interval_sec, missed_checkin_before_alarm,
		              absent_after_sec) = ($2, $3, $4) THEN 0 ELSE 1 END,
		    update_ts = $5, update_user_id = $6
		WHERE device_id = $7 AND security_context_id = $8
		  AND update_ts = $9 AND delete_user_id IS NULL
		RETURNING `+deviceColumnList,
		device.DeviceName, device.CheckinIntervalSec,
		device.MissedCheckinBeforeAlarm, device.AbsentAfterSec,
		device.UpdateTS, device.UpdateUserID, device.DeviceID,
		device.SecurityContext, originalUpdateTS,
	), &updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, zeroRowOutcome(ctx, r.db, deviceTable, devicePK,
			device.DeviceID, device.SecurityContext)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update device: %w", err)
	}
	return &updated, nil
}

// DeleteDevice soft deletes a device under the optimistic lock and clears
// its token, so a check-in with it is refused at once. Its bands and
// contacts are hidden with it (withLiveParent), not stamped.
func (r *DeviceRepository) DeleteDevice(
	ctx context.Context,
	deviceID int,
	securityContext int,
	originalUpdateTS float64,
	deleteTS float64,
	deleteUserID int,
) error {
	result, err := r.db.Exec(ctx, `
		UPDATE device
		SET delete_ts = $1, delete_user_id = $2, token_hash = NULL
		WHERE device_id = $3 AND security_context_id = $4
		  AND update_ts = $5 AND delete_user_id IS NULL`,
		deleteTS, deleteUserID, deviceID, securityContext, originalUpdateTS)
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}
	if result.RowsAffected() == 0 {
		return zeroRowOutcome(ctx, r.db, deviceTable, devicePK, deviceID,
			securityContext)
	}
	return nil
}

// IssueDeviceToken stores a new token's hash under the optimistic lock;
// the previous token stops working in the same write.
func (r *DeviceRepository) IssueDeviceToken(
	ctx context.Context,
	deviceID int,
	securityContext int,
	originalUpdateTS float64,
	tokenHash string,
	issuedTS float64,
	actorUserID int,
) error {
	result, err := r.db.Exec(ctx, `
		UPDATE device
		SET token_hash = $1, token_issued_ts = $2, update_ts = $2,
		    update_user_id = $3
		WHERE device_id = $4 AND security_context_id = $5
		  AND update_ts = $6 AND delete_user_id IS NULL`,
		tokenHash, issuedTS, actorUserID, deviceID, securityContext,
		originalUpdateTS)
	if err != nil {
		return fmt.Errorf("failed to store device token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return zeroRowOutcome(ctx, r.db, deviceTable, devicePK, deviceID,
			securityContext)
	}
	return nil
}

// RetrieveDeviceByTokenHash answers POST /deviceCheckin: the live device
// holding the token. Like RetrieveLiveInvite it is not scoped by
// security_context_id: it runs before there is a tenant, and the token is
// the only key. The device's own tenant scopes everything after it. An
// unknown hash or a deleted device is ErrNotFound.
func (r *DeviceRepository) RetrieveDeviceByTokenHash(
	ctx context.Context, tokenHash string,
) (*model.DeviceModel, error) {
	var device model.DeviceModel
	err := scanDevice(r.db.QueryRow(ctx, `SELECT `+deviceColumnList+`
		FROM device
		WHERE token_hash = $1 AND delete_user_id IS NULL`, tokenHash),
		&device)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read device by token: %w", err)
	}
	return &device, nil
}

// RecordDeviceCheckin writes one check-in in one transaction: the
// device_checkin row, the device's last_* columns and, for each reported
// band that is a live beacon of this device, its present / last_* columns.
// None of the device or beacon writes touches update_ts, so an admin
// editing either does not 409 on every check-in. It answers the device's
// config_version and the reported addresses that matched no beacon. A
// device deleted since the token lookup is ErrNotFound.
func (r *DeviceRepository) RecordDeviceCheckin(
	ctx context.Context,
	deviceID int,
	securityContext int,
	record *model.DeviceCheckinRecord,
	beaconList []model.DeviceCheckinBeacon,
) (configVersion int, unknownAddressList []string, err error) {
	err = pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			UPDATE device
			SET last_checkin_ts = $1, last_checkin_json = $2,
			    last_agent_version = $3
			WHERE device_id = $4 AND security_context_id = $5
			  AND delete_user_id IS NULL
			RETURNING config_version`,
			record.ReceivedTS, jsonbParam(record.BodyJSON),
			record.AgentVersion, deviceID, securityContext,
		).Scan(&configVersion)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("failed to update device check-in: %w", err)
		}

		// Insert-only: update_* repeat the create pair; the system is
		// user 0.
		if _, err := tx.Exec(ctx, `
			INSERT INTO device_checkin (
				security_context_id, device_id, received_ts, sent_ts,
				reason, body_json, source_ip, create_ts, create_user_id,
				update_ts, update_user_id
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $3, 0, $3, 0)`,
			securityContext, deviceID, record.ReceivedTS, record.SentTS,
			record.Reason, jsonbParam(record.BodyJSON), record.SourceIP,
		); err != nil {
			return fmt.Errorf("failed to insert device check-in: %w", err)
		}

		for _, beacon := range beaconList {
			address, _ := model.NormaliseBeaconAddress(beacon.Address)
			var lastSeenTS *float64
			if beacon.LastSeen != nil {
				seen := model.EpochSeconds(*beacon.LastSeen)
				lastSeenTS = &seen
			}
			// A Pi that restarted has not seen the band yet and sends no
			// lastSeen or rssi; the earlier sighting stays.
			result, err := tx.Exec(ctx, `
				UPDATE beacon
				SET present = $1,
				    last_seen_ts = COALESCE($2, last_seen_ts),
				    last_rssi = COALESCE($3, last_rssi),
				    last_report_ts = $4
				WHERE security_context_id = $5 AND device_id = $6
				  AND address = $7 AND delete_user_id IS NULL`,
				beacon.Present, lastSeenTS, beacon.RSSI, record.ReceivedTS,
				securityContext, deviceID, address)
			if err != nil {
				return fmt.Errorf("failed to update beacon %s: %w",
					address, err)
			}
			if result.RowsAffected() == 0 {
				unknownAddressList = append(unknownAddressList, address)
			}
		}
		return nil
	})
	if err != nil {
		return 0, nil, err
	}
	return configVersion, unknownAddressList, nil
}

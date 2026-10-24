package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"stillthere-api/internal/model"
)

// TenantSettingRepository owns tenant_setting: at most one live row per
// tenant (idx_tenant_setting_sc), created by the first save.
type TenantSettingRepository struct {
	db *pgxpool.Pool
}

// NewTenantSettingRepository creates the tenant setting repository.
func NewTenantSettingRepository(db *pgxpool.Pool) *TenantSettingRepository {
	return &TenantSettingRepository{db: db}
}

const tenantSettingColumnList = `
	tenant_setting_id, time_zone, remind_interval_min, daily_reminder_time,
	security_context_id, create_ts, create_user_id, update_ts,
	update_user_id, delete_ts, delete_user_id`

func scanTenantSetting(row pgx.Row, setting *model.TenantSettingModel) error {
	return row.Scan(
		&setting.TenantSettingID, &setting.TimeZone,
		&setting.RemindIntervalMin, &setting.DailyReminderTime,
		&setting.SecurityContext, &setting.CreateTS, &setting.CreateUserID,
		&setting.UpdateTS, &setting.UpdateUserID, &setting.DeleteTS,
		&setting.DeleteUserID,
	)
}

// RetrieveTenantSetting returns the tenant's row; none yet is ErrNotFound.
func (r *TenantSettingRepository) RetrieveTenantSetting(
	ctx context.Context, securityContext int,
) (*model.TenantSettingModel, error) {
	var setting model.TenantSettingModel
	err := scanTenantSetting(r.db.QueryRow(ctx, `
		SELECT `+tenantSettingColumnList+`
		FROM tenant_setting
		WHERE security_context_id = $1 AND delete_user_id IS NULL`,
		securityContext), &setting)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read tenant setting: %w", err)
	}
	return &setting, nil
}

// SaveTenantSetting updates the tenant's row under the optimistic lock,
// or creates it when the tenant has none (originalUpdateTS is then
// ignored). A row that exists with another update_ts, or a second first
// save racing this one, is ErrConflict. The caller fills the stamps; a
// create takes UpdateTS / UpdateUserID as its create pair too.
func (r *TenantSettingRepository) SaveTenantSetting(
	ctx context.Context,
	setting *model.TenantSettingModel,
	originalUpdateTS float64,
) (*model.TenantSettingModel, error) {
	var saved model.TenantSettingModel
	err := scanTenantSetting(r.db.QueryRow(ctx, `
		UPDATE tenant_setting
		SET time_zone = $1, remind_interval_min = $2,
		    daily_reminder_time = $3, update_ts = $4, update_user_id = $5
		WHERE security_context_id = $6 AND update_ts = $7
		  AND delete_user_id IS NULL
		RETURNING `+tenantSettingColumnList,
		setting.TimeZone, setting.RemindIntervalMin,
		setting.DailyReminderTime, setting.UpdateTS, setting.UpdateUserID,
		setting.SecurityContext, originalUpdateTS,
	), &saved)
	if err == nil {
		return &saved, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to update tenant setting: %w", err)
	}

	// No row matched: stale, or none yet. The insert tells them apart, as
	// the unique index refuses a second live row.
	err = scanTenantSetting(r.db.QueryRow(ctx, `
		INSERT INTO tenant_setting (
			time_zone, remind_interval_min, daily_reminder_time,
			security_context_id, create_ts, create_user_id, update_ts,
			update_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $5, $6)
		RETURNING `+tenantSettingColumnList,
		setting.TimeZone, setting.RemindIntervalMin,
		setting.DailyReminderTime, setting.SecurityContext,
		setting.UpdateTS, setting.UpdateUserID,
	), &saved)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant setting: %w", err)
	}
	return &saved, nil
}

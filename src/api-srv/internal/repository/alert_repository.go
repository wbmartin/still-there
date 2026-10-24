package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"stillthere-api/internal/model"
)

// AlertRepository is the API's side of alert and alert_event: read the
// tenant's alarms, and acknowledge or turn one off from the web app. The
// watchdog raises, reminds and resolves (internal/watchdog, its own
// repository).
type AlertRepository struct {
	db *pgxpool.Pool
}

// NewAlertRepository creates the alert repository.
func NewAlertRepository(db *pgxpool.Pool) *AlertRepository {
	return &AlertRepository{db: db}
}

const alertColumnList = `
	alert_id, device_id, beacon_id, alert_kind, alert_state, raised_ts,
	last_notified_ts, notify_count, ack_ts, ack_by, ack_via, off_ts,
	off_by, off_via, resolved_ts, security_context_id, create_ts,
	create_user_id, update_ts, update_user_id`

func scanAlert(row pgx.Row, alert *model.AlertModel) error {
	return row.Scan(
		&alert.AlertID, &alert.DeviceID, &alert.BeaconID, &alert.AlertKind,
		&alert.AlertState, &alert.RaisedTS, &alert.LastNotifiedTS,
		&alert.NotifyCount, &alert.AckTS, &alert.AckBy, &alert.AckVia,
		&alert.OffTS, &alert.OffBy, &alert.OffVia, &alert.ResolvedTS,
		&alert.SecurityContext, &alert.CreateTS, &alert.CreateUserID,
		&alert.UpdateTS, &alert.UpdateUserID,
	)
}

// RetrieveAlertList returns the tenant's alarms on live devices, of one
// device when deviceID > 0: every open one, then those resolved since
// resolvedSince, newest raised first within each.
func (r *AlertRepository) RetrieveAlertList(
	ctx context.Context,
	deviceID int,
	securityContext int,
	resolvedSince float64,
) ([]*model.AlertModel, error) {
	whereClause, argList := buildParentScopedWhere(
		securityContext, 0, "alert_id", nil, "device_id", deviceID)
	whereClause = withLiveParent(
		whereClause, "alert", "device_id", "device", "device_id")
	argList = append(argList, resolvedSince)
	rowList, err := r.db.Query(ctx, `SELECT `+alertColumnList+`
		FROM alert`+whereClause+fmt.Sprintf(`
		  AND (resolved_ts IS NULL OR resolved_ts > $%d)
		ORDER BY resolved_ts IS NOT NULL, raised_ts DESC, alert_id DESC`,
		len(argList)), argList...)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rowList.Close()

	alertList := make([]*model.AlertModel, 0)
	for rowList.Next() {
		var alert model.AlertModel
		if err := scanAlert(rowList, &alert); err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alertList = append(alertList, &alert)
	}
	if err := rowList.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alerts: %w", err)
	}
	return alertList, nil
}

// alertTransition is one thing a person does to an alarm from the web
// app: the states it starts from, the state it leaves, and the columns
// that record who and when. setWho is a constant, never caller input.
type alertTransition struct {
	fromStateList []string
	toState       string
	setWho        string // $1 the time, $2 the person, $3 the channel
	eventType     string
	verb          string
}

var (
	alertAcknowledge = alertTransition{
		fromStateList: []string{model.AlertStateRaised},
		toState:       model.AlertStateAcknowledged,
		setWho:        `ack_ts = $1, ack_by = $2, ack_via = $3`,
		eventType:     "acknowledged",
		verb:          "acknowledged",
	}
	alertTurnOff = alertTransition{
		fromStateList: []string{
			model.AlertStateRaised, model.AlertStateAcknowledged},
		toState:   model.AlertStateTurnedOff,
		setWho:    `off_ts = $1, off_by = $2, off_via = $3`,
		eventType: "turned_off",
		verb:      "turned off",
	}
)

// AcknowledgeAlert moves a raised alarm to acknowledged by actorName in
// the web app and queues the "acknowledged" text (alert_event), in one
// transaction. It answers the stored row.
func (r *AlertRepository) AcknowledgeAlert(
	ctx context.Context,
	alertID int,
	securityContext int,
	originalUpdateTS float64,
	actorName string,
	now float64,
	actorUserID int,
) (*model.AlertModel, error) {
	return r.changeAlertState(ctx, alertAcknowledge, alertID,
		securityContext, originalUpdateTS, actorName, now, actorUserID)
}

// TurnOffAlert moves a raised or acknowledged alarm to turned_off by
// actorName in the web app (no more reminders) and queues the
// "turned_off" text, in one transaction. It answers the stored row.
func (r *AlertRepository) TurnOffAlert(
	ctx context.Context,
	alertID int,
	securityContext int,
	originalUpdateTS float64,
	actorName string,
	now float64,
	actorUserID int,
) (*model.AlertModel, error) {
	return r.changeAlertState(ctx, alertTurnOff, alertID,
		securityContext, originalUpdateTS, actorName, now, actorUserID)
}

// changeAlertState applies transition under the optimistic lock. No live
// alarm of the tenant is ErrNotFound, a stale update_ts ErrConflict, and
// a state the transition does not start from (resolved included)
// ErrConflict naming the state.
func (r *AlertRepository) changeAlertState(
	ctx context.Context,
	transition alertTransition,
	alertID int,
	securityContext int,
	originalUpdateTS float64,
	actorName string,
	now float64,
	actorUserID int,
) (*model.AlertModel, error) {
	var changed model.AlertModel
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		err := scanAlert(tx.QueryRow(ctx, `
			UPDATE alert
			SET alert_state = $4, `+transition.setWho+`,
			    update_ts = $1, update_user_id = $5
			WHERE alert_id = $6 AND security_context_id = $7
			  AND update_ts = $8 AND delete_user_id IS NULL
			  AND resolved_ts IS NULL AND alert_state = ANY($9)
			RETURNING `+alertColumnList,
			now, actorName, model.AlertViaUI, transition.toState,
			actorUserID, alertID, securityContext, originalUpdateTS,
			transition.fromStateList,
		), &changed)
		if errors.Is(err, pgx.ErrNoRows) {
			return r.alertChangeRefusal(ctx, tx, transition, alertID,
				securityContext, originalUpdateTS)
		}
		if err != nil {
			return err
		}
		// The outbox row: the watchdog texts it and sets sent_ts.
		// Insert-only, so update_* repeat the create pair.
		_, err = tx.Exec(ctx, `
			INSERT INTO alert_event (
				security_context_id, alert_id, event_type, actor_name,
				actor_via, create_ts, create_user_id, update_ts,
				update_user_id
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $6, $7)`,
			securityContext, alertID, transition.eventType, actorName,
			model.AlertViaUI, now, actorUserID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to %s alert: %w",
			transition.eventType, err)
	}
	return &changed, nil
}

// alertChangeRefusal explains a transition that matched no row, with the
// write's tenant predicate.
func (r *AlertRepository) alertChangeRefusal(
	ctx context.Context,
	tx pgx.Tx,
	transition alertTransition,
	alertID int,
	securityContext int,
	originalUpdateTS float64,
) error {
	var updateTS float64
	var alertState string
	err := tx.QueryRow(ctx, `
		SELECT update_ts, alert_state FROM alert
		WHERE alert_id = $1 AND security_context_id = $2
		  AND delete_user_id IS NULL`,
		alertID, securityContext).Scan(&updateTS, &alertState)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to check alert: %w", err)
	}
	if updateTS != originalUpdateTS {
		return ErrConflict
	}
	return clientMessage(ErrConflict, "This alert is %s and cannot be %s",
		strings.ReplaceAll(alertState, "_", " "), transition.verb)
}

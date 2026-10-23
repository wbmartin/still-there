package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"stillthere-api/internal/model"
)

// HelpEventRepository stores and counts the help module's usage events
// (doc/151-help-module.md). Every read and write names its tenant.
type HelpEventRepository struct {
	db *pgxpool.Pool
}

// NewHelpEventRepository creates the repository.
func NewHelpEventRepository(db *pgxpool.Pool) *HelpEventRepository {
	return &HelpEventRepository{db: db}
}

// CreateHelpEvent inserts one event in securityContext and returns its id.
// The row's tenant is the argument, never the model's: the caller's
// session decides it.
func (r *HelpEventRepository) CreateHelpEvent(
	ctx context.Context, event *model.HelpEventModel, securityContext int,
) (int, error) {
	var helpEventID int
	err := r.db.QueryRow(ctx, `
		INSERT INTO help_event (
			security_context_id, app_user_id, event, entry_id, route,
			query, create_ts, create_user_id, update_ts, update_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING help_event_id`,
		securityContext, event.AppUserID, event.Event, event.EntryID,
		event.Route, event.Query, event.CreateTS, event.CreateUserID,
		event.UpdateTS, event.UpdateUserID,
	).Scan(&helpEventID)
	if err != nil {
		return 0, fmt.Errorf("failed to create help event: %w", err)
	}
	return helpEventID, nil
}

// RetrieveHelpEventCountSince counts one user's events in securityContext
// created at or after since (the per-user daily cap).
func (r *HelpEventRepository) RetrieveHelpEventCountSince(
	ctx context.Context, appUserID int, since float64, securityContext int,
) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM help_event
		WHERE security_context_id = $1 AND app_user_id = $2
		  AND create_ts >= $3 AND delete_ts IS NULL`,
		securityContext, appUserID, since,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count help events: %w", err)
	}
	return count, nil
}

// RetrieveHelpInsightCountList counts securityContext's events of one
// kind created at or after since, grouped by key: the search text for
// search_no_result, the entry or tour id for the others. Most frequent
// first, then by key; at most limit rows.
func (r *HelpEventRepository) RetrieveHelpInsightCountList(
	ctx context.Context, event string, since float64, limit int,
	securityContext int,
) ([]model.HelpInsightCount, error) {
	// The column is chosen here from the event, never formatted from
	// caller input; an event the table does not take is an error.
	var query string
	switch event {
	case model.HelpEventSearchNoResult:
		query = helpInsightCountQuery("query")
	case model.HelpEventEntryView, model.HelpEventTourComplete,
		model.HelpEventTourSkip:
		query = helpInsightCountQuery("entry_id")
	default:
		return nil, fmt.Errorf("unknown help event %q", event)
	}
	rows, err := r.db.Query(ctx, query, securityContext, event, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to count help events: %w", err)
	}
	defer rows.Close()
	countList := []model.HelpInsightCount{}
	for rows.Next() {
		var count model.HelpInsightCount
		if err := rows.Scan(&count.Key, &count.Count); err != nil {
			return nil, fmt.Errorf("failed to read help event count: %w", err)
		}
		countList = append(countList, count)
	}
	return countList, rows.Err()
}

// helpInsightCountQuery groups by keyColumn, one of two constants above.
func helpInsightCountQuery(keyColumn string) string {
	return fmt.Sprintf(`
		SELECT %[1]s, COUNT(*) AS event_count
		FROM help_event
		WHERE security_context_id = $1 AND event = $2 AND create_ts >= $3
		  AND delete_ts IS NULL AND %[1]s IS NOT NULL
		GROUP BY %[1]s
		ORDER BY event_count DESC, %[1]s
		LIMIT $4`, keyColumn)
}

package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wbmartin/starter-api/internal/model"
)

// IssueReportRepository stores in-app issue reports, their conversation
// and their files (migration 0006_issue_report.sql). The handler-facing
// methods take the caller's tenant and filter by it on every statement;
// a row of another tenant is ErrNotFound. The Relay* methods are the
// issue relay's view (internal/issuerelay): it is a system job that serves
// every tenant, so its listings cross tenants by design, and each of its
// writes still names the row's own tenant beside the id.
type IssueReportRepository struct {
	db *pgxpool.Pool
}

// NewIssueReportRepository creates a new issue report repository
func NewIssueReportRepository(db *pgxpool.Pool) *IssueReportRepository {
	return &IssueReportRepository{db: db}
}

// maxRelayErrorLen is relay_error's column width.
const maxRelayErrorLen = 500

const issueReportColumnList = `
	r.issue_report_id,
	r.security_context_id,
	r.kind,
	r.summary,
	r.detail,
	r.page_url,
	r.context_jsonb,
	r.state,
	r.status,
	r.status_note,
	r.reporter_name,
	r.reporter_email,
	r.remote_id,
	r.relay_state,
	r.relay_error,
	r.relay_attempts,
	r.last_synced_ts,
	r.last_activity_ts,
	(SELECT COUNT(*)
	 FROM issue_report_message m
	 WHERE m.issue_report_id = r.issue_report_id
	   AND m.security_context_id = r.security_context_id
	   AND m.delete_user_id IS NULL) AS message_count,
	r.create_ts,
	r.create_user_id,
	r.update_ts,
	r.update_user_id`

func scanIssueReport(row pgx.Row) (*model.IssueReportModel, error) {
	var report model.IssueReportModel
	err := row.Scan(
		&report.IssueReportID, &report.SecurityContext, &report.Kind,
		&report.Summary, &report.Detail, &report.PageURL,
		&report.ContextJSONB, &report.State, &report.Status,
		&report.StatusNote, &report.ReporterName, &report.ReporterEmail,
		&report.RemoteID, &report.RelayState, &report.RelayError,
		&report.RelayAttempts, &report.LastSyncedTS,
		&report.LastActivityTS, &report.MessageCount,
		&report.CreateTS, &report.CreateUserID,
		&report.UpdateTS, &report.UpdateUserID,
	)
	if err != nil {
		return nil, err
	}
	return &report, nil
}

// insertIssueReportFileList stores files on a report (messageID nil) or on
// one of its messages, inside the caller's transaction.
func insertIssueReportFileList(
	ctx context.Context,
	tx pgx.Tx,
	securityContext int,
	reportID int,
	messageID *int,
	fileList []model.IssueReportFile,
	now float64,
	userID int,
) error {
	for _, file := range fileList {
		_, err := tx.Exec(ctx, `
			INSERT INTO issue_report_attachment (
				security_context_id, issue_report_id,
				issue_report_message_id, filename, content_type,
				size_bytes, content, create_ts, create_user_id,
				update_ts, update_user_id
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $8, $9)`,
			securityContext, reportID, messageID, file.Filename,
			file.ContentType, len(file.Content), file.Content, now, userID,
		)
		if err != nil {
			return fmt.Errorf("failed to store attachment %q: %w",
				file.Filename, err)
		}
	}
	return nil
}

// CreateIssueReport stores a new report and its files in one transaction
// and returns its id. The report's tenant must be the caller's.
func (r *IssueReportRepository) CreateIssueReport(
	ctx context.Context,
	report *model.IssueReportModel,
	fileList []model.IssueReportFile,
	securityContext int,
) (int, error) {
	if report.SecurityContext != securityContext {
		return 0, fmt.Errorf(
			"cannot create issue report in different security context")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var reportID int
	err = tx.QueryRow(ctx, `
		INSERT INTO issue_report (
			security_context_id, kind, summary, detail, page_url,
			context_jsonb, state, status, reporter_name, reporter_email,
			relay_state, relay_attempts, last_activity_ts,
			create_ts, create_user_id, update_ts, update_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 0,
		        $12, $12, $13, $12, $13)
		RETURNING issue_report_id`,
		report.SecurityContext, report.Kind, report.Summary, report.Detail,
		report.PageURL, jsonbParam(report.ContextJSONB), report.State,
		report.Status, report.ReporterName, report.ReporterEmail,
		report.RelayState, report.CreateTS, report.CreateUserID,
	).Scan(&reportID)
	if err != nil {
		return 0, fmt.Errorf("failed to create issue report: %w", err)
	}
	err = insertIssueReportFileList(ctx, tx, securityContext, reportID,
		nil, fileList, report.CreateTS, report.CreateUserID)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit issue report: %w", err)
	}
	return reportID, nil
}

// RetrieveIssueReportList returns the tenant's reports, most recently
// active first, capped at model.IssueMaxListedReports.
func (r *IssueReportRepository) RetrieveIssueReportList(
	ctx context.Context, securityContext int,
) ([]*model.IssueReportModel, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+issueReportColumnList+`
		FROM issue_report r
		WHERE r.security_context_id = $1 AND r.delete_user_id IS NULL
		ORDER BY r.last_activity_ts DESC, r.issue_report_id DESC
		LIMIT $2`, securityContext, model.IssueMaxListedReports)
	if err != nil {
		return nil, fmt.Errorf("failed to list issue reports: %w", err)
	}
	defer rows.Close()
	reportList := []*model.IssueReportModel{}
	for rows.Next() {
		report, err := scanIssueReport(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to read issue report: %w", err)
		}
		reportList = append(reportList, report)
	}
	return reportList, rows.Err()
}

// RetrieveIssueReport returns one of the tenant's reports; ErrNotFound
// when there is no live one with that id in the tenant.
func (r *IssueReportRepository) RetrieveIssueReport(
	ctx context.Context, reportID int, securityContext int,
) (*model.IssueReportModel, error) {
	report, err := scanIssueReport(r.db.QueryRow(ctx, `
		SELECT `+issueReportColumnList+`
		FROM issue_report r
		WHERE r.issue_report_id = $1 AND r.security_context_id = $2
		  AND r.delete_user_id IS NULL`, reportID, securityContext))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read issue report: %w", err)
	}
	return report, nil
}

// RetrieveIssueReportDetail returns a report with its files and its
// conversation, oldest message first.
func (r *IssueReportRepository) RetrieveIssueReportDetail(
	ctx context.Context, reportID int, securityContext int,
) (*model.IssueReportDetailModel, error) {
	report, err := r.RetrieveIssueReport(ctx, reportID, securityContext)
	if err != nil {
		return nil, err
	}
	detail := &model.IssueReportDetailModel{
		IssueReportModel: *report,
		AttachmentList:   []model.IssueReportAttachmentModel{},
		ThreadList:       []model.IssueReportMessageModel{},
	}

	attachmentByMessage, err := r.retrieveAttachmentListByMessage(
		ctx, reportID, securityContext, detail)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT issue_report_message_id,
		       issue_report_id,
		       author_kind,
		       author_name,
		       body,
		       action,
		       relay_state,
		       create_ts,
		       create_user_id
		FROM issue_report_message
		WHERE issue_report_id = $1 AND security_context_id = $2
		  AND delete_user_id IS NULL
		ORDER BY create_ts, issue_report_message_id`,
		reportID, securityContext)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var message model.IssueReportMessageModel
		err := rows.Scan(
			&message.IssueReportMessageID, &message.IssueReportID,
			&message.AuthorKind, &message.AuthorName, &message.Body,
			&message.Action, &message.RelayState, &message.CreateTS,
			&message.CreateUserID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to read message: %w", err)
		}
		message.AttachmentList =
			attachmentByMessage[message.IssueReportMessageID]
		if message.AttachmentList == nil {
			message.AttachmentList = []model.IssueReportAttachmentModel{}
		}
		detail.ThreadList = append(detail.ThreadList, message)
	}
	return detail, rows.Err()
}

// retrieveAttachmentListByMessage puts the report's own files on detail
// and returns its messages' files by message id.
func (r *IssueReportRepository) retrieveAttachmentListByMessage(
	ctx context.Context,
	reportID int,
	securityContext int,
	detail *model.IssueReportDetailModel,
) (map[int][]model.IssueReportAttachmentModel, error) {
	rows, err := r.db.Query(ctx, `
		SELECT issue_report_attachment_id,
		       issue_report_id,
		       issue_report_message_id,
		       filename,
		       content_type,
		       size_bytes,
		       create_ts
		FROM issue_report_attachment
		WHERE issue_report_id = $1 AND security_context_id = $2
		  AND delete_user_id IS NULL
		ORDER BY issue_report_attachment_id`, reportID, securityContext)
	if err != nil {
		return nil, fmt.Errorf("failed to list attachments: %w", err)
	}
	defer rows.Close()
	byMessage := map[int][]model.IssueReportAttachmentModel{}
	for rows.Next() {
		var attachment model.IssueReportAttachmentModel
		err := rows.Scan(
			&attachment.IssueReportAttachmentID, &attachment.IssueReportID,
			&attachment.IssueReportMessageID, &attachment.Filename,
			&attachment.ContentType, &attachment.SizeBytes,
			&attachment.CreateTS,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to read attachment: %w", err)
		}
		if attachment.IssueReportMessageID == nil {
			detail.AttachmentList =
				append(detail.AttachmentList, attachment)
			continue
		}
		messageID := *attachment.IssueReportMessageID
		byMessage[messageID] = append(byMessage[messageID], attachment)
	}
	return byMessage, rows.Err()
}

// CreateIssueReportMessage adds a reporter message (with its files) or a
// reporter action to a report, in one transaction. newState and newStatus,
// when set, move the report too; every message counts as activity.
// ErrNotFound when the report is not a live one of the tenant.
func (r *IssueReportRepository) CreateIssueReportMessage(
	ctx context.Context,
	message *model.IssueReportMessageModel,
	fileList []model.IssueReportFile,
	newState *string,
	newStatus *string,
	securityContext int,
) (int, error) {
	if message.SecurityContext != securityContext {
		return 0, fmt.Errorf(
			"cannot create issue report message in different " +
				"security context")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// The report first, scoped: a message never lands on another
	// tenant's report.
	tag, err := tx.Exec(ctx, `
		UPDATE issue_report
		SET state = COALESCE($1, state),
		    status = COALESCE($2, status),
		    last_activity_ts = $3, update_ts = $3, update_user_id = $4
		WHERE issue_report_id = $5 AND security_context_id = $6
		  AND delete_user_id IS NULL`,
		newState, newStatus, message.CreateTS, message.CreateUserID,
		message.IssueReportID, securityContext)
	if err != nil {
		return 0, fmt.Errorf("failed to touch issue report: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return 0, ErrNotFound
	}

	var messageID int
	err = tx.QueryRow(ctx, `
		INSERT INTO issue_report_message (
			security_context_id, issue_report_id, author_kind,
			author_name, body, action, relay_state, relay_attempts,
			create_ts, create_user_id, update_ts, update_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 0, $8, $9, $8, $9)
		RETURNING issue_report_message_id`,
		securityContext, message.IssueReportID, message.AuthorKind,
		message.AuthorName, message.Body, message.Action,
		message.RelayState, message.CreateTS, message.CreateUserID,
	).Scan(&messageID)
	if err != nil {
		return 0, fmt.Errorf("failed to add message: %w", err)
	}
	err = insertIssueReportFileList(ctx, tx, securityContext,
		message.IssueReportID, &messageID, fileList, message.CreateTS,
		message.CreateUserID)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit message: %w", err)
	}
	return messageID, nil
}

// IssueReportAttachmentContent is one stored file with what a download
// needs: its bytes (nil for a staff file kept by the tracker, which then
// has RemoteID set) and its report's tracker id.
type IssueReportAttachmentContent struct {
	model.IssueReportAttachmentModel
	Content        []byte
	ReportRemoteID *string
}

// RetrieveIssueReportAttachment returns one of the tenant's files on a
// live report; ErrNotFound otherwise.
func (r *IssueReportRepository) RetrieveIssueReportAttachment(
	ctx context.Context, attachmentID int, securityContext int,
) (*IssueReportAttachmentContent, error) {
	var file IssueReportAttachmentContent
	err := r.db.QueryRow(ctx, `
		SELECT a.issue_report_attachment_id,
		       a.issue_report_id,
		       a.issue_report_message_id,
		       a.filename,
		       a.content_type,
		       a.size_bytes,
		       a.content,
		       a.remote_id,
		       a.create_ts,
		       r.remote_id
		FROM issue_report_attachment a
		JOIN issue_report r
		  ON r.issue_report_id = a.issue_report_id
		 AND r.security_context_id = a.security_context_id
		WHERE a.issue_report_attachment_id = $1
		  AND a.security_context_id = $2
		  AND a.delete_user_id IS NULL AND r.delete_user_id IS NULL`,
		attachmentID, securityContext,
	).Scan(
		&file.IssueReportAttachmentID, &file.IssueReportID,
		&file.IssueReportMessageID, &file.Filename, &file.ContentType,
		&file.SizeBytes, &file.Content, &file.RemoteID, &file.CreateTS,
		&file.ReportRemoteID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read attachment: %w", err)
	}
	return &file, nil
}

// ---- the relay's view ----

// IssueRelayFile is a stored file to send to the tracker.
type IssueRelayFile struct {
	Filename    string
	ContentType string
	Content     []byte
}

// IssueRelayReport is a report waiting to be sent, with its files.
type IssueRelayReport struct {
	model.IssueReportModel
	FileList []IssueRelayFile
}

// truncateRelayError fits an error into relay_error.
func truncateRelayError(message string) string {
	if len(message) > maxRelayErrorLen {
		return message[:maxRelayErrorLen]
	}
	return message
}

// RelayReportListToSend lists reports not yet in the tracker (pending, or
// failed fewer than maxAttempts times), oldest first, with their files.
func (r *IssueReportRepository) RelayReportListToSend(
	ctx context.Context, maxAttempts int, limit int,
) ([]*IssueRelayReport, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+issueReportColumnList+`
		FROM issue_report r
		WHERE r.delete_user_id IS NULL AND r.remote_id IS NULL
		  AND r.relay_state IN ('pending', 'failed')
		  AND r.relay_attempts < $1
		ORDER BY r.create_ts, r.issue_report_id
		LIMIT $2`, maxAttempts, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list reports to send: %w", err)
	}
	var relayReportList []*IssueRelayReport
	for rows.Next() {
		report, err := scanIssueReport(rows)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to read report to send: %w", err)
		}
		relayReportList = append(relayReportList,
			&IssueRelayReport{IssueReportModel: *report})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, relayReport := range relayReportList {
		relayReport.FileList, err = r.relayFileList(ctx,
			relayReport.IssueReportID, relayReport.SecurityContext, nil)
		if err != nil {
			return nil, err
		}
	}
	return relayReportList, nil
}

// relayFileList is the stored files of a report (messageID nil) or of one
// of its messages.
func (r *IssueReportRepository) relayFileList(
	ctx context.Context, reportID int, securityContext int, messageID *int,
) ([]IssueRelayFile, error) {
	rows, err := r.db.Query(ctx, `
		SELECT filename,
		       content_type,
		       content
		FROM issue_report_attachment
		WHERE issue_report_id = $1 AND security_context_id = $2
		  AND delete_user_id IS NULL AND content IS NOT NULL
		  AND issue_report_message_id IS NOT DISTINCT FROM $3
		ORDER BY issue_report_attachment_id`,
		reportID, securityContext, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to list files to send: %w", err)
	}
	defer rows.Close()
	var fileList []IssueRelayFile
	for rows.Next() {
		var file IssueRelayFile
		err := rows.Scan(&file.Filename, &file.ContentType, &file.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to read file to send: %w", err)
		}
		fileList = append(fileList, file)
	}
	return fileList, rows.Err()
}

// RelayMarkReportSent records the tracker's id and status after a send.
func (r *IssueReportRepository) RelayMarkReportSent(
	ctx context.Context,
	reportID int,
	securityContext int,
	remoteID string,
	state string,
	status string,
	statusNote *string,
	now float64,
) error {
	_, err := r.db.Exec(ctx, `
		UPDATE issue_report
		SET remote_id = $1, state = $2, status = $3, status_note = $4,
		    relay_state = 'sent', relay_error = NULL,
		    relay_attempts = relay_attempts + 1, last_synced_ts = $5
		WHERE issue_report_id = $6 AND security_context_id = $7`,
		remoteID, state, status, statusNote, now, reportID,
		securityContext)
	if err != nil {
		return fmt.Errorf("failed to record sent report: %w", err)
	}
	return nil
}

// RelayMarkReportFailed records a failed send; it is retried until
// relay_attempts reaches the relay's cap.
func (r *IssueReportRepository) RelayMarkReportFailed(
	ctx context.Context, reportID int, securityContext int, message string,
) error {
	_, err := r.db.Exec(ctx, `
		UPDATE issue_report
		SET relay_state = 'failed', relay_error = $1,
		    relay_attempts = relay_attempts + 1
		WHERE issue_report_id = $2 AND security_context_id = $3`,
		truncateRelayError(message), reportID, securityContext)
	if err != nil {
		return fmt.Errorf("failed to record failed report: %w", err)
	}
	return nil
}

// IssueRelayMessage is a reporter message or action waiting to be sent.
type IssueRelayMessage struct {
	model.IssueReportMessageModel
	ReportRemoteID string
	FileList       []IssueRelayFile
}

// RelayMessageListToSend lists reporter messages and actions not yet sent
// (pending, or failed fewer than maxAttempts times) whose report is in the
// tracker, oldest first, with their files.
func (r *IssueReportRepository) RelayMessageListToSend(
	ctx context.Context, maxAttempts int, limit int,
) ([]*IssueRelayMessage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.issue_report_message_id,
		       m.issue_report_id,
		       m.security_context_id,
		       m.author_kind,
		       m.author_name,
		       m.body,
		       m.action,
		       m.relay_state,
		       m.create_ts,
		       m.create_user_id,
		       r.remote_id
		FROM issue_report_message m
		JOIN issue_report r
		  ON r.issue_report_id = m.issue_report_id
		 AND r.security_context_id = m.security_context_id
		WHERE m.delete_user_id IS NULL AND r.delete_user_id IS NULL
		  AND r.remote_id IS NOT NULL
		  AND m.relay_state IN ('pending', 'failed')
		  AND m.relay_attempts < $1
		ORDER BY m.create_ts, m.issue_report_message_id
		LIMIT $2`, maxAttempts, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages to send: %w", err)
	}
	var relayMessageList []*IssueRelayMessage
	for rows.Next() {
		var message IssueRelayMessage
		err := rows.Scan(
			&message.IssueReportMessageID, &message.IssueReportID,
			&message.SecurityContext, &message.AuthorKind,
			&message.AuthorName, &message.Body, &message.Action,
			&message.RelayState, &message.CreateTS,
			&message.CreateUserID, &message.ReportRemoteID,
		)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf(
				"failed to read message to send: %w", err)
		}
		relayMessageList = append(relayMessageList, &message)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, message := range relayMessageList {
		messageID := message.IssueReportMessageID
		message.FileList, err = r.relayFileList(ctx,
			message.IssueReportID, message.SecurityContext, &messageID)
		if err != nil {
			return nil, err
		}
	}
	return relayMessageList, nil
}

// RelayMarkMessageSent records a message the tracker has.
func (r *IssueReportRepository) RelayMarkMessageSent(
	ctx context.Context, messageID int, securityContext int,
) error {
	_, err := r.db.Exec(ctx, `
		UPDATE issue_report_message
		SET relay_state = 'sent', relay_error = NULL,
		    relay_attempts = relay_attempts + 1
		WHERE issue_report_message_id = $1 AND security_context_id = $2`,
		messageID, securityContext)
	if err != nil {
		return fmt.Errorf("failed to record sent message: %w", err)
	}
	return nil
}

// RelayMarkMessageFailed records a failed send of a message.
func (r *IssueReportRepository) RelayMarkMessageFailed(
	ctx context.Context, messageID int, securityContext int, message string,
) error {
	_, err := r.db.Exec(ctx, `
		UPDATE issue_report_message
		SET relay_state = 'failed', relay_error = $1,
		    relay_attempts = relay_attempts + 1
		WHERE issue_report_message_id = $2 AND security_context_id = $3`,
		truncateRelayError(message), messageID, securityContext)
	if err != nil {
		return fmt.Errorf("failed to record failed message: %w", err)
	}
	return nil
}

// IssueRelaySyncTarget is a report to refresh from the tracker.
type IssueRelaySyncTarget struct {
	IssueReportID   int
	SecurityContext int
	RemoteID        string
	State           string
	Status          string
	StatusNote      *string
}

// RelaySyncTargetList lists reports in the tracker that are still open,
// or were active since `since` (so a just-closed report picks up its last
// replies), most recently active first.
func (r *IssueReportRepository) RelaySyncTargetList(
	ctx context.Context, since float64, limit int,
) ([]IssueRelaySyncTarget, error) {
	rows, err := r.db.Query(ctx, `
		SELECT issue_report_id,
		       security_context_id,
		       remote_id,
		       state,
		       status,
		       status_note
		FROM issue_report
		WHERE delete_user_id IS NULL AND remote_id IS NOT NULL
		  AND (state NOT IN ('closed', 'wont_fix')
		       OR last_activity_ts >= $1)
		ORDER BY last_activity_ts DESC
		LIMIT $2`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list reports to sync: %w", err)
	}
	defer rows.Close()
	var targetList []IssueRelaySyncTarget
	for rows.Next() {
		var target IssueRelaySyncTarget
		err := rows.Scan(&target.IssueReportID, &target.SecurityContext,
			&target.RemoteID, &target.State, &target.Status,
			&target.StatusNote)
		if err != nil {
			return nil, fmt.Errorf("failed to read report to sync: %w", err)
		}
		targetList = append(targetList, target)
	}
	return targetList, rows.Err()
}

// RelayApplyStatus stores the tracker's state and status for a report; a
// change counts as activity.
func (r *IssueReportRepository) RelayApplyStatus(
	ctx context.Context,
	target IssueRelaySyncTarget,
	state string,
	status string,
	statusNote *string,
	now float64,
) error {
	changed := state != target.State || status != target.Status ||
		!sameOptionalString(statusNote, target.StatusNote)
	var err error
	if changed {
		_, err = r.db.Exec(ctx, `
			UPDATE issue_report
			SET state = $1, status = $2, status_note = $3,
			    last_synced_ts = $4, last_activity_ts = $4
			WHERE issue_report_id = $5 AND security_context_id = $6`,
			state, status, statusNote, now, target.IssueReportID,
			target.SecurityContext)
	} else {
		_, err = r.db.Exec(ctx, `
			UPDATE issue_report
			SET last_synced_ts = $1
			WHERE issue_report_id = $2 AND security_context_id = $3`,
			now, target.IssueReportID, target.SecurityContext)
	}
	if err != nil {
		return fmt.Errorf("failed to store report status: %w", err)
	}
	return nil
}

func sameOptionalString(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

// IssueRemoteFile is a file the tracker keeps, fetched through the relay
// when downloaded.
type IssueRemoteFile struct {
	RemoteID    string
	Filename    string
	ContentType string
	SizeBytes   int
}

// RelayAddStaffMessage stores a tracker reply once (deduplicated on its
// remote id) with references to its files. It reports whether the reply
// was new.
func (r *IssueReportRepository) RelayAddStaffMessage(
	ctx context.Context,
	target IssueRelaySyncTarget,
	remoteID string,
	authorName string,
	body string,
	createTS float64,
	fileList []IssueRemoteFile,
) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var messageID int
	err = tx.QueryRow(ctx, `
		INSERT INTO issue_report_message (
			security_context_id, issue_report_id, author_kind,
			author_name, body, remote_id, relay_state, relay_attempts,
			create_ts, create_user_id, update_ts, update_user_id
		)
		VALUES ($1, $2, 'staff', $3, $4, $5, 'none', 0, $6, 0, $6, 0)
		ON CONFLICT (issue_report_id, remote_id)
		   WHERE remote_id IS NOT NULL DO NOTHING
		RETURNING issue_report_message_id`,
		target.SecurityContext, target.IssueReportID, authorName, body,
		remoteID, createTS,
	).Scan(&messageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil // stored on an earlier sync
	}
	if err != nil {
		return false, fmt.Errorf("failed to store a reply: %w", err)
	}
	for _, file := range fileList {
		_, err := tx.Exec(ctx, `
			INSERT INTO issue_report_attachment (
				security_context_id, issue_report_id,
				issue_report_message_id, filename, content_type,
				size_bytes, content, remote_id, create_ts,
				create_user_id, update_ts, update_user_id
			)
			VALUES ($1, $2, $3, $4, $5, $6, NULL, $7, $8, 0, $8, 0)
			ON CONFLICT (issue_report_id, remote_id)
			   WHERE remote_id IS NOT NULL DO NOTHING`,
			target.SecurityContext, target.IssueReportID, messageID,
			file.Filename, file.ContentType, file.SizeBytes,
			file.RemoteID, createTS)
		if err != nil {
			return false, fmt.Errorf("failed to store a reply file: %w",
				err)
		}
	}
	_, err = tx.Exec(ctx, `
		UPDATE issue_report
		SET last_activity_ts = GREATEST(last_activity_ts, $1)
		WHERE issue_report_id = $2 AND security_context_id = $3`,
		createTS, target.IssueReportID, target.SecurityContext)
	if err != nil {
		return false, fmt.Errorf("failed to touch issue report: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("failed to commit a reply: %w", err)
	}
	return true, nil
}

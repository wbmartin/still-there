package issuerelay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
)

// Store is the slice of *repository.IssueReportRepository the relay uses.
type Store interface {
	RelayQueueLocalBacklog(
		ctx context.Context,
	) (reportCount int, messageCount int, err error)
	RelayReportListToSend(
		ctx context.Context, maxAttempts int, limit int,
	) ([]*repository.IssueRelayReport, error)
	RelayMarkReportSent(
		ctx context.Context, reportID int, securityContext int,
		remoteID string, state string, status string, statusNote *string,
		now float64,
	) error
	RelayMarkReportFailed(
		ctx context.Context, reportID int, securityContext int,
		message string,
	) error
	RelayMessageListToSend(
		ctx context.Context, maxAttempts int, limit int,
	) ([]*repository.IssueRelayMessage, error)
	RelayMarkMessageSent(
		ctx context.Context, messageID int, securityContext int,
	) error
	RelayMarkMessageFailed(
		ctx context.Context, messageID int, securityContext int,
		message string,
	) error
	RelaySyncTargetList(
		ctx context.Context, since float64, limit int,
	) ([]repository.IssueRelaySyncTarget, error)
	RelayApplyStatus(
		ctx context.Context, target repository.IssueRelaySyncTarget,
		state string, status string, statusNote *string, now float64,
	) error
	RelayAddStaffMessage(
		ctx context.Context, target repository.IssueRelaySyncTarget,
		remoteID string, authorName string, body string, createTS float64,
		fileList []repository.IssueRemoteFile,
	) (bool, error)
}

// Logger is the part of the app logger the relay uses
// (*logging.Logger satisfies it).
type Logger interface {
	Info(code string, message string, keyvals ...any)
	Error(code string, message string, keyvals ...any)
}

// Defaults for the zero values of Relay's tuning fields.
const (
	defaultMaxAttempts = 20
	defaultSyncWindow  = 14 * 24 * time.Hour
	sendBatchSize      = 25
	messageBatchSize   = 50
	syncTargetLimit    = 500
	// The tracker answers at most 200 ids per GET.
	syncBatchSize = 100
)

// Relay moves reports between the app and the tracker, one Tick at a
// time: send new reports, send reporter messages and actions, then pull
// status and staff replies for the reports still open.
type Relay struct {
	Client *Client
	Store  Store
	Log    Logger
	// Now is the clock (tests set it); nil is the wall clock.
	Now func() float64
	// MaxAttempts is how many times a report or message is tried before
	// it is left failed (visible in the app). 0 is 20.
	MaxAttempts int
	// SyncWindow is how long a closed report is still synced after its
	// last activity. 0 is 14 days.
	SyncWindow time.Duration
	// WorkspaceLabel names a report's tenant for the tracker. nil sends
	// "tenant <id>"; a derived app with tenant names supplies its own.
	WorkspaceLabel func(securityContext int) string
}

func (r *Relay) now() float64 {
	if r.Now != nil {
		return r.Now()
	}
	return model.Now()
}

func (r *Relay) maxAttempts() int {
	if r.MaxAttempts > 0 {
		return r.MaxAttempts
	}
	return defaultMaxAttempts
}

func (r *Relay) workspaceLabel(securityContext int) string {
	if r.WorkspaceLabel != nil {
		return r.WorkspaceLabel(securityContext)
	}
	return "tenant " + strconv.Itoa(securityContext)
}

// Run queues what was filed while the relay was off, ticks at once, and
// then every interval until ctx is done.
func (r *Relay) Run(ctx context.Context, interval time.Duration) {
	r.QueueBacklog(ctx)
	r.Tick(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.Tick(ctx)
		}
	}
}

// QueueBacklog makes the reports and messages filed while the relay was
// off (relay_state local) pending, so the ticks send them. A failure is
// logged and the relay runs on; the next start tries again.
func (r *Relay) QueueBacklog(ctx context.Context) {
	reportCount, messageCount, err := r.Store.RelayQueueLocalBacklog(ctx)
	if err != nil {
		r.logError("RELAY_BACKLOG", "queueing local reports", err)
		return
	}
	if r.Log != nil {
		r.Log.Info("RELAY_BACKLOG", fmt.Sprintf(
			"Issue relay: %d local reports and %d local messages queued",
			reportCount, messageCount), "username", "issue-relay")
	}
}

// Tick sends what is waiting, then refreshes the open reports.
func (r *Relay) Tick(ctx context.Context) {
	r.sendReportList(ctx)
	r.sendMessageList(ctx)
	r.syncReportList(ctx)
}

func (r *Relay) logError(code string, message string, err error) {
	if r.Log != nil {
		r.Log.Error(code, message+": "+err.Error(), "username",
			"issue-relay")
	}
}

func toFileList(storedList []repository.IssueRelayFile) []File {
	fileList := make([]File, 0, len(storedList))
	for _, stored := range storedList {
		fileList = append(fileList, File{
			Filename:    stored.Filename,
			ContentType: stored.ContentType,
			Content:     stored.Content,
		})
	}
	return fileList
}

func valueOf(text *string) string {
	if text == nil {
		return ""
	}
	return *text
}

func (r *Relay) sendReportList(ctx context.Context) {
	reportList, err := r.Store.RelayReportListToSend(ctx, r.maxAttempts(),
		sendBatchSize)
	if err != nil {
		r.logError("RELAY_LIST", "listing reports to send", err)
		return
	}
	for _, report := range reportList {
		contextJSON := contextWithPage(report.ContextJSONB,
			valueOf(report.PageURL))
		remote, err := r.Client.Create(ctx, NewReport{
			Kind:          report.Kind,
			Summary:       report.Summary,
			Detail:        valueOf(report.Detail),
			ReporterName:  report.ReporterName,
			ReporterEmail: report.ReporterEmail,
			ReporterRef:   strconv.Itoa(report.CreateUserID),
			Workspace:     r.workspaceLabel(report.SecurityContext),
			ContextJSON:   contextJSON,
			FileList:      toFileList(report.FileList),
		})
		if err != nil {
			r.logError("RELAY_SEND", "sending report "+
				strconv.Itoa(report.IssueReportID), err)
			err = r.Store.RelayMarkReportFailed(ctx, report.IssueReportID,
				report.SecurityContext, err.Error())
			if err != nil {
				r.logError("RELAY_MARK", "recording a failed send", err)
			}
			continue
		}
		err = r.Store.RelayMarkReportSent(ctx, report.IssueReportID,
			report.SecurityContext, remote.ID, remote.State, remote.Status,
			remote.StatusNote, r.now())
		if err != nil {
			r.logError("RELAY_MARK", "recording a sent report", err)
		}
	}
}

func (r *Relay) sendMessageList(ctx context.Context) {
	messageList, err := r.Store.RelayMessageListToSend(ctx,
		r.maxAttempts(), messageBatchSize)
	if err != nil {
		r.logError("RELAY_LIST_MSG", "listing messages to send", err)
		return
	}
	for _, message := range messageList {
		action := valueOf(message.Action)
		if action == model.IssueActionWithdraw ||
			action == model.IssueActionReopen {
			_, err = r.Client.Action(ctx, message.ReportRemoteID, action,
				model.IssueActionNote(message.Body))
		} else {
			_, err = r.Client.Message(ctx, message.ReportRemoteID,
				message.Body, valueOf(message.AuthorName),
				strconv.Itoa(message.CreateUserID),
				toFileList(message.FileList))
		}
		// A 409 on an action means the tracker is already there (staff
		// closed the report first): done, not retried.
		var statusError *StatusError
		if err != nil && action != "" && errors.As(err, &statusError) &&
			statusError.Code == http.StatusConflict {
			err = nil
		}
		if err != nil {
			r.logError("RELAY_SEND_MSG", "sending message "+
				strconv.Itoa(message.IssueReportMessageID), err)
			err = r.Store.RelayMarkMessageFailed(ctx,
				message.IssueReportMessageID, message.SecurityContext,
				err.Error())
			if err != nil {
				r.logError("RELAY_MARK", "recording a failed send", err)
			}
			continue
		}
		err = r.Store.RelayMarkMessageSent(ctx, message.IssueReportMessageID,
			message.SecurityContext)
		if err != nil {
			r.logError("RELAY_MARK", "recording a sent message", err)
		}
	}
}

func (r *Relay) syncReportList(ctx context.Context) {
	window := r.SyncWindow
	if window <= 0 {
		window = defaultSyncWindow
	}
	targetList, err := r.Store.RelaySyncTargetList(ctx,
		r.now()-window.Seconds(), syncTargetLimit)
	if err != nil {
		r.logError("RELAY_LIST_SYNC", "listing reports to sync", err)
		return
	}
	targetByRemoteID := map[string]repository.IssueRelaySyncTarget{}
	remoteIDList := make([]string, 0, len(targetList))
	for _, target := range targetList {
		targetByRemoteID[target.RemoteID] = target
		remoteIDList = append(remoteIDList, target.RemoteID)
	}
	for start := 0; start < len(remoteIDList); start += syncBatchSize {
		end := min(start+syncBatchSize, len(remoteIDList))
		remoteList, err := r.Client.RetrieveList(ctx,
			remoteIDList[start:end])
		if err != nil {
			r.logError("RELAY_SYNC", "fetching report status", err)
			return
		}
		for _, remote := range remoteList {
			target, ok := targetByRemoteID[remote.ID]
			if !ok {
				continue
			}
			r.applyRemote(ctx, target, remote)
		}
	}
}

// applyRemote stores one report's status and its new staff replies.
func (r *Relay) applyRemote(
	ctx context.Context,
	target repository.IssueRelaySyncTarget,
	remote RemoteReport,
) {
	err := r.Store.RelayApplyStatus(ctx, target, remote.State,
		remote.Status, remote.StatusNote, r.now())
	if err != nil {
		r.logError("RELAY_APPLY", "storing report status", err)
	}
	for _, message := range remote.ThreadList {
		if message.AuthorKind != model.IssueAuthorStaff {
			continue // ours, echoed back by the tracker
		}
		fileList := make([]repository.IssueRemoteFile, 0,
			len(message.AttachmentList))
		for _, attachment := range message.AttachmentList {
			fileList = append(fileList, repository.IssueRemoteFile{
				RemoteID:    attachment.ID,
				Filename:    attachment.Filename,
				ContentType: attachment.ContentType,
				SizeBytes:   attachment.Size,
			})
		}
		authorName := valueOf(message.AuthorName)
		if authorName == "" {
			authorName = "Support"
		}
		_, err := r.Store.RelayAddStaffMessage(ctx, target, message.ID,
			authorName, message.Body,
			parseTime(message.CreatedAt, r.now()), fileList)
		if err != nil {
			r.logError("RELAY_REPLY", "storing a reply", err)
		}
	}
}

// contextWithPage is the context sent with a report: the browser's own
// object as stored, or, when there is none, one naming the page.
func contextWithPage(contextJSONB []byte, pageURL string) string {
	if len(contextJSONB) > 0 {
		return string(contextJSONB)
	}
	if pageURL == "" {
		return ""
	}
	encoded, err := json.Marshal(map[string]string{"page_url": pageURL})
	if err != nil {
		return ""
	}
	return string(encoded)
}

// parseTime reads the tracker's ISO timestamp as epoch seconds, fallback
// when it does not parse.
func parseTime(text string, fallback float64) float64 {
	for _, layout := range []string{
		time.RFC3339Nano, "2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05",
	} {
		if parsed, err := time.Parse(layout, text); err == nil {
			return model.EpochSeconds(parsed)
		}
	}
	return fallback
}

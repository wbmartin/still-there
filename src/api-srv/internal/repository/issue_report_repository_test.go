package repository

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"

	"github.com/wbmartin/starter-api/internal/model"
)

func TestIssueReportRepository(t *testing.T) {
	suite.Run(t, &IssueReportRepositoryTestSuite{})
}

type IssueReportRepositoryTestSuite struct {
	suite.Suite
	db   *pgxpool.Pool
	repo *IssueReportRepository
}

const issueTestSummaryPrefix = "RepoTest issue "

func (s *IssueReportRepositoryTestSuite) SetupSuite() {
	s.db = connectTestDB(s.T())
	s.repo = NewIssueReportRepository(s.db)
}

func (s *IssueReportRepositoryTestSuite) TearDownSuite() {
	ctx := context.Background()
	// Test data only, in the scratch database: the rows this suite made.
	for _, statement := range []string{
		`DELETE FROM issue_report_attachment WHERE issue_report_id IN (
			SELECT issue_report_id FROM issue_report
			WHERE summary LIKE 'RepoTest issue %')`,
		`DELETE FROM issue_report_message WHERE issue_report_id IN (
			SELECT issue_report_id FROM issue_report
			WHERE summary LIKE 'RepoTest issue %')`,
		`DELETE FROM issue_report WHERE summary LIKE 'RepoTest issue %'`,
	} {
		_, _ = s.db.Exec(ctx, statement)
	}
	s.db.Close()
}

var issueTestPNG = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

// createIn files a report with one file in securityContext.
func (s *IssueReportRepositoryTestSuite) createIn(
	summary string, securityContext int,
) int {
	now := model.Now()
	detail := "Steps"
	report := &model.IssueReportModel{
		SecurityContext: securityContext,
		Kind:            model.IssueKindBug,
		Summary:         issueTestSummaryPrefix + summary,
		Detail:          &detail,
		ContextJSONB:    json.RawMessage(`{"browser":"test"}`),
		State:           model.IssueStateOpen,
		Status:          "Sent",
		ReporterName:    "Repo Tester",
		ReporterEmail:   "tester@example.com",
		RelayState:      model.IssueRelayPending,
		CreateTS:        now,
		CreateUserID:    1,
		UpdateTS:        now,
		UpdateUserID:    1,
	}
	reportID, err := s.repo.CreateIssueReport(context.Background(), report,
		[]model.IssueReportFile{{Filename: "shot.png",
			ContentType: "image/png", Content: issueTestPNG}},
		securityContext)
	s.Require().NoError(err)
	return reportID
}

func (s *IssueReportRepositoryTestSuite) message(
	reportID int, securityContext int, body string,
) *model.IssueReportMessageModel {
	authorName := "Repo Tester"
	return &model.IssueReportMessageModel{
		IssueReportID:   reportID,
		SecurityContext: securityContext,
		AuthorKind:      model.IssueAuthorReporter,
		AuthorName:      &authorName,
		Body:            body,
		RelayState:      model.IssueRelayPending,
		CreateTS:        model.Now(),
		CreateUserID:    1,
	}
}

func (s *IssueReportRepositoryTestSuite) TestCreateAndRetrieveDetail() {
	ctx := context.Background()
	reportID := s.createIn("detail", testTenant)

	_, err := s.repo.CreateIssueReportMessage(ctx,
		s.message(reportID, testTenant, "More"),
		[]model.IssueReportFile{{Filename: "more.txt",
			ContentType: "text/plain", Content: []byte("words")}},
		nil, nil, testTenant)
	s.Require().NoError(err)

	detail, err := s.repo.RetrieveIssueReportDetail(ctx, reportID,
		testTenant)
	s.Require().NoError(err)
	s.Equal(issueTestSummaryPrefix+"detail", detail.Summary)
	s.Equal("Repo Tester", detail.ReporterName)
	s.JSONEq(`{"browser":"test"}`, string(detail.ContextJSONB))
	s.Equal(1, detail.MessageCount)
	s.Require().Len(detail.AttachmentList, 1)
	s.Equal("shot.png", detail.AttachmentList[0].Filename)
	s.Require().Len(detail.ThreadList, 1)
	s.Equal("More", detail.ThreadList[0].Body)
	s.Require().Len(detail.ThreadList[0].AttachmentList, 1)
	s.Equal("more.txt", detail.ThreadList[0].AttachmentList[0].Filename)

	file, err := s.repo.RetrieveIssueReportAttachment(ctx,
		detail.AttachmentList[0].IssueReportAttachmentID, testTenant)
	s.Require().NoError(err)
	s.Equal(issueTestPNG, file.Content)
}

func (s *IssueReportRepositoryTestSuite) TestActionMovesTheReport() {
	ctx := context.Background()
	reportID := s.createIn("action", testTenant)
	action := model.IssueActionWithdraw
	withdraw := s.message(reportID, testTenant, "Withdrew")
	withdraw.Action = &action
	state, status := model.IssueStateClosed, "Withdrawn"
	_, err := s.repo.CreateIssueReportMessage(ctx, withdraw, nil, &state,
		&status, testTenant)
	s.Require().NoError(err)

	report, err := s.repo.RetrieveIssueReport(ctx, reportID, testTenant)
	s.Require().NoError(err)
	s.Equal(model.IssueStateClosed, report.State)
	s.Equal("Withdrawn", report.Status)
}

func (s *IssueReportRepositoryTestSuite) TestListIsTheTenantsNewestFirst() {
	ctx := context.Background()
	olderID := s.createIn("older", testTenant)
	newerID := s.createIn("newer", testTenant)
	reportList, err := s.repo.RetrieveIssueReportList(ctx, testTenant)
	s.Require().NoError(err)
	position := map[int]int{}
	for i, report := range reportList {
		s.Equal(testTenant, report.SecurityContext)
		position[report.IssueReportID] = i
	}
	s.Contains(position, olderID)
	s.Contains(position, newerID)
	s.Less(position[newerID], position[olderID])
}

// The cross-tenant negative test (doc/110, Tenant isolation): a report of
// otherTenant (3) is not found from testTenant (2) by every read and
// write, nothing is written to it, and its owner still sees it whole.
func (s *IssueReportRepositoryTestSuite) TestCrossTenantIsolation() {
	ctx := context.Background()
	theirsID := s.createIn("theirs", otherTenant)
	theirs, err := s.repo.RetrieveIssueReportDetail(ctx, theirsID,
		otherTenant)
	s.Require().NoError(err)
	theirFileID := theirs.AttachmentList[0].IssueReportAttachmentID

	_, err = s.repo.RetrieveIssueReport(ctx, theirsID, testTenant)
	s.True(errors.Is(err, ErrNotFound), "retrieve: %v", err)
	_, err = s.repo.RetrieveIssueReportDetail(ctx, theirsID, testTenant)
	s.True(errors.Is(err, ErrNotFound), "detail: %v", err)
	_, err = s.repo.RetrieveIssueReportAttachment(ctx, theirFileID,
		testTenant)
	s.True(errors.Is(err, ErrNotFound), "attachment: %v", err)

	reportList, err := s.repo.RetrieveIssueReportList(ctx, testTenant)
	s.Require().NoError(err)
	for _, report := range reportList {
		s.NotEqual(theirsID, report.IssueReportID)
	}

	// A write naming their report from our tenant: not found, nothing
	// stored, the report not touched.
	state, status := model.IssueStateClosed, "Withdrawn"
	_, err = s.repo.CreateIssueReportMessage(ctx,
		s.message(theirsID, testTenant, "Sneaky"),
		[]model.IssueReportFile{{Filename: "x.txt",
			ContentType: "text/plain", Content: []byte("x")}},
		&state, &status, testTenant)
	s.True(errors.Is(err, ErrNotFound), "message: %v", err)

	// The model's tenant must be the caller's.
	_, err = s.repo.CreateIssueReportMessage(ctx,
		s.message(theirsID, otherTenant, "Mismatch"), nil, nil, nil,
		testTenant)
	s.Error(err)

	after, err := s.repo.RetrieveIssueReportDetail(ctx, theirsID,
		otherTenant)
	s.Require().NoError(err)
	s.Empty(after.ThreadList)
	s.Len(after.AttachmentList, 1)
	s.Equal(model.IssueStateOpen, after.State)
	s.Equal(theirs.UpdateTS, after.UpdateTS)

	var strayCount int
	err = s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM issue_report_message
		WHERE issue_report_id = $1`, theirsID).Scan(&strayCount)
	s.Require().NoError(err)
	s.Zero(strayCount)
}

func (s *IssueReportRepositoryTestSuite) relayReport(
	reportID int,
) *IssueRelayReport {
	reportList, err := s.repo.RelayReportListToSend(context.Background(),
		20, 1000)
	s.Require().NoError(err)
	for _, report := range reportList {
		if report.IssueReportID == reportID {
			return report
		}
	}
	return nil
}

// The relay's round trip: send, record, sync status, store a reply once.
func (s *IssueReportRepositoryTestSuite) TestRelayRoundTrip() {
	ctx := context.Background()
	reportID := s.createIn("relay", otherTenant)

	pending := s.relayReport(reportID)
	s.Require().NotNil(pending)
	s.Equal(otherTenant, pending.SecurityContext)
	s.Equal("tester@example.com", pending.ReporterEmail)
	s.Require().Len(pending.FileList, 1)
	s.Equal(issueTestPNG, pending.FileList[0].Content)

	// A failed send is retried until the cap.
	s.Require().NoError(s.repo.RelayMarkReportFailed(ctx, reportID,
		otherTenant, "tracker down"))
	s.NotNil(s.relayReport(reportID))
	capped, err := s.repo.RelayReportListToSend(ctx, 1, 1000)
	s.Require().NoError(err)
	for _, report := range capped {
		s.NotEqual(reportID, report.IssueReportID)
	}

	remoteID := "R-" + model.NewUUID()
	s.Require().NoError(s.repo.RelayMarkReportSent(ctx, reportID,
		otherTenant, remoteID, "open", "Under review", nil, model.Now()))
	s.Nil(s.relayReport(reportID))

	// A reporter message goes out once the report is in the tracker.
	messageID, err := s.repo.CreateIssueReportMessage(ctx,
		s.message(reportID, otherTenant, "More"), nil, nil, nil,
		otherTenant)
	s.Require().NoError(err)
	messageList, err := s.repo.RelayMessageListToSend(ctx, 20, 1000)
	s.Require().NoError(err)
	var found *IssueRelayMessage
	for _, message := range messageList {
		if message.IssueReportMessageID == messageID {
			found = message
		}
	}
	s.Require().NotNil(found)
	s.Equal(remoteID, found.ReportRemoteID)
	s.Require().NoError(s.repo.RelayMarkMessageSent(ctx, messageID,
		otherTenant))
	messageList, err = s.repo.RelayMessageListToSend(ctx, 20, 1000)
	s.Require().NoError(err)
	for _, message := range messageList {
		s.NotEqual(messageID, message.IssueReportMessageID)
	}

	// Sync: the status lands, a staff reply is stored once with its file.
	targetList, err := s.repo.RelaySyncTargetList(ctx, 0, 1000)
	s.Require().NoError(err)
	var target *IssueRelaySyncTarget
	for i := range targetList {
		if targetList[i].IssueReportID == reportID {
			target = &targetList[i]
		}
	}
	s.Require().NotNil(target)
	s.Equal(otherTenant, target.SecurityContext)
	s.Require().NoError(s.repo.RelayApplyStatus(ctx, *target, "fixing",
		"Being fixed", nil, model.Now()))
	fileList := []IssueRemoteFile{{RemoteID: "F-1", Filename: "fix.png",
		ContentType: "image/png", SizeBytes: 10}}
	isNew, err := s.repo.RelayAddStaffMessage(ctx, *target, "M-1",
		"Support", "On it", model.Now(), fileList)
	s.Require().NoError(err)
	s.True(isNew)
	isNew, err = s.repo.RelayAddStaffMessage(ctx, *target, "M-1",
		"Support", "On it", model.Now(), fileList)
	s.Require().NoError(err)
	s.False(isNew)

	detail, err := s.repo.RetrieveIssueReportDetail(ctx, reportID,
		otherTenant)
	s.Require().NoError(err)
	s.Equal("fixing", detail.State)
	s.Equal("Being fixed", detail.Status)
	s.Require().Len(detail.ThreadList, 2)
	reply := detail.ThreadList[1]
	s.Equal(model.IssueAuthorStaff, reply.AuthorKind)
	s.Require().Len(reply.AttachmentList, 1)
	staffFile, err := s.repo.RetrieveIssueReportAttachment(ctx,
		reply.AttachmentList[0].IssueReportAttachmentID, otherTenant)
	s.Require().NoError(err)
	s.Nil(staffFile.Content)
	s.Equal("F-1", *staffFile.RemoteID)
	s.Equal(remoteID, *staffFile.ReportRemoteID)

	// The staff file is the other tenant's too.
	_, err = s.repo.RetrieveIssueReportAttachment(ctx,
		reply.AttachmentList[0].IssueReportAttachmentID, testTenant)
	s.True(errors.Is(err, ErrNotFound))
}

// relayStateOf reads a row's relay_state straight from the table.
func (s *IssueReportRepositoryTestSuite) relayStateOf(
	table string, idColumn string, id int,
) string {
	var state string
	err := s.db.QueryRow(context.Background(),
		"SELECT relay_state FROM "+table+" WHERE "+idColumn+" = $1", id).
		Scan(&state)
	s.Require().NoError(err)
	return state
}

// createLocalIn files a report as the handler does with the relay off.
func (s *IssueReportRepositoryTestSuite) createLocalIn(
	summary string, securityContext int,
) int {
	reportID := s.createIn(summary, securityContext)
	_, err := s.db.Exec(context.Background(), `
		UPDATE issue_report SET relay_state = 'local'
		WHERE issue_report_id = $1`, reportID)
	s.Require().NoError(err)
	return reportID
}

func (s *IssueReportRepositoryTestSuite) localMessage(
	reportID int, securityContext int, body string,
) int {
	message := s.message(reportID, securityContext, body)
	message.RelayState = model.IssueRelayLocal
	messageID, err := s.repo.CreateIssueReportMessage(context.Background(),
		message, nil, nil, nil, securityContext)
	s.Require().NoError(err)
	return messageID
}

// What was filed while the relay was off is queued at relay start: live,
// open local reports in every tenant and their local messages; a closed
// (withdrawn) or deleted local report stays local with its messages; a
// report already in the tracker is untouched, and a local message on it
// is queued.
func (s *IssueReportRepositoryTestSuite) TestRelayQueueLocalBacklog() {
	ctx := context.Background()
	openID := s.createLocalIn("backlog open", testTenant)
	openMessageID := s.localMessage(openID, testTenant, "More")
	otherID := s.createLocalIn("backlog other tenant", otherTenant)

	withdrawnID := s.createLocalIn("backlog withdrawn", testTenant)
	action := model.IssueActionWithdraw
	withdraw := s.message(withdrawnID, testTenant, model.IssueWithdrawText)
	withdraw.Action = &action
	withdraw.RelayState = model.IssueRelayLocal
	state, status := model.IssueStateClosed, "Withdrawn"
	withdrawMessageID, err := s.repo.CreateIssueReportMessage(ctx,
		withdraw, nil, &state, &status, testTenant)
	s.Require().NoError(err)

	deletedID := s.createLocalIn("backlog deleted", testTenant)
	_, err = s.db.Exec(ctx, `
		UPDATE issue_report SET delete_ts = 1, delete_user_id = 1
		WHERE issue_report_id = $1`, deletedID)
	s.Require().NoError(err)

	sentID := s.createIn("backlog sent", testTenant)
	remoteID := "R-" + model.NewUUID()
	s.Require().NoError(s.repo.RelayMarkReportSent(ctx, sentID,
		testTenant, remoteID, "open", "Under review", nil, model.Now()))
	sentMessageID := s.localMessage(sentID, testTenant, "While off")

	reportCount, messageCount, err := s.repo.RelayQueueLocalBacklog(ctx)
	s.Require().NoError(err)
	s.GreaterOrEqual(reportCount, 2)
	s.GreaterOrEqual(messageCount, 2)

	report := func(id int) string {
		return s.relayStateOf("issue_report", "issue_report_id", id)
	}
	message := func(id int) string {
		return s.relayStateOf("issue_report_message",
			"issue_report_message_id", id)
	}
	s.Equal(model.IssueRelayPending, report(openID))
	s.Equal(model.IssueRelayPending, message(openMessageID))
	s.Equal(model.IssueRelayPending, report(otherID))
	s.Equal(model.IssueRelayLocal, report(withdrawnID))
	s.Equal(model.IssueRelayLocal, message(withdrawMessageID))
	s.Equal(model.IssueRelayLocal, report(deletedID))
	s.Equal(model.IssueRelaySent, report(sentID))
	s.Equal(model.IssueRelayPending, message(sentMessageID))
	sent, err := s.repo.RetrieveIssueReport(ctx, sentID, testTenant)
	s.Require().NoError(err)
	s.Equal(remoteID, *sent.RemoteID)

	// The queued report is sent like any other; its message waits for it.
	s.NotNil(s.relayReport(openID))
	messageList, err := s.repo.RelayMessageListToSend(ctx, 20, 1000)
	s.Require().NoError(err)
	queuedIDList := map[int]bool{}
	for _, queued := range messageList {
		queuedIDList[queued.IssueReportMessageID] = true
	}
	s.False(queuedIDList[openMessageID], "sent before its report")
	s.True(queuedIDList[sentMessageID])

	// A second start finds nothing new.
	reportCount, messageCount, err = s.repo.RelayQueueLocalBacklog(ctx)
	s.Require().NoError(err)
	s.Zero(reportCount)
	s.Zero(messageCount)
}

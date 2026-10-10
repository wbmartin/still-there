package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
	"github.com/wbmartin/starter-api/internal/security"
)

// The issue-report module (doc/150-issue-report-module.md). Anyone signed
// in (LOGON) files a bug, polish or feature report and sees every report
// of the tenant; reporters add to a report's conversation, withdraw an
// open report and reopen a closed one. Staff of the tracker own every
// other status. The module is off unless ISSUE_REPORTS_ENABLED=true: each
// route then answers 404 after its security checks and touches nothing.

// IssueReportRepositoryInterface is what IssueReportHandler needs from the
// repository; *repository.IssueReportRepository satisfies it.
type IssueReportRepositoryInterface interface {
	CreateIssueReport(
		ctx context.Context, report *model.IssueReportModel,
		fileList []model.IssueReportFile, securityContext int,
	) (int, error)
	RetrieveIssueReportList(
		ctx context.Context, securityContext int,
	) ([]*model.IssueReportModel, error)
	RetrieveIssueReport(
		ctx context.Context, reportID int, securityContext int,
	) (*model.IssueReportModel, error)
	RetrieveIssueReportDetail(
		ctx context.Context, reportID int, securityContext int,
	) (*model.IssueReportDetailModel, error)
	CreateIssueReportMessage(
		ctx context.Context, message *model.IssueReportMessageModel,
		fileList []model.IssueReportFile, newState *string,
		newStatus *string, securityContext int,
	) (int, error)
	RetrieveIssueReportAttachment(
		ctx context.Context, attachmentID int, securityContext int,
	) (*repository.IssueReportAttachmentContent, error)
}

// IssueReporterRepositoryInterface reads the caller's own login, for the
// name and email a report carries.
type IssueReporterRepositoryInterface interface {
	RetrieveUserByUserName(
		ctx context.Context, userName string,
	) (*model.AppUserModel, error)
}

// IssueRelayDownloader fetches a file the tracker keeps (a staff reply's
// attachment); *issuerelay.Client satisfies it. nil: no relay.
type IssueRelayDownloader interface {
	Download(
		ctx context.Context, remoteReportID string,
		remoteAttachmentID string, maxBytes int64,
	) ([]byte, error)
}

// IssueReportHandler serves the issue-report routes.
type IssueReportHandler struct {
	appsrv                *app.Services
	issueReportRepository IssueReportRepositoryInterface
	reporterRepository    IssueReporterRepositoryInterface
	relayDownloader       IssueRelayDownloader
}

// NewIssueReportHandler creates the handler. relayDownloader is nil when
// the relay is off.
func NewIssueReportHandler(
	appServices *app.Services,
	issueReportRepository IssueReportRepositoryInterface,
	reporterRepository IssueReporterRepositoryInterface,
	relayDownloader IssueRelayDownloader,
) *IssueReportHandler {
	return &IssueReportHandler{
		appsrv:                appServices,
		issueReportRepository: issueReportRepository,
		reporterRepository:    reporterRepository,
		relayDownloader:       relayDownloader,
	}
}

// moduleOn answers 404 when the module is off. Called after the checks,
// so an off module still refuses a caller the checks refuse.
func (h *IssueReportHandler) moduleOn(
	appHelper *security.AppHelper, w http.ResponseWriter,
) bool {
	if h.appsrv.Config.IssueReportsEnabled {
		return true
	}
	appHelper.WriteError(w, http.StatusNotFound,
		"Issue reports are not enabled", "ISSUE_MODULE_OFF", nil)
	return false
}

// newRelayState is pending when the relay will send the row, local when
// no relay is configured.
func (h *IssueReportHandler) newRelayState() string {
	if h.appsrv.Config.IssueRelayEnabled() {
		return model.IssueRelayPending
	}
	return model.IssueRelayLocal
}

// reporterOf is the caller's display name and email, read from their own
// login by the JWT's subject (the row stays in its home tenant across a
// switch; doc/105 Multi-tenancy). A row that is not the JWT's uid is 401.
func (h *IssueReportHandler) reporterOf(
	appHelper *security.AppHelper, w http.ResponseWriter, r *http.Request,
) (name string, email string, ok bool) {
	jwtClaims := appHelper.GetClaims()
	user, err := h.reporterRepository.RetrieveUserByUserName(
		r.Context(), jwtClaims.Subject)
	if errors.Is(err, repository.ErrNotFound) ||
		(err == nil && user.AppUserID != jwtClaims.UserID) {
		appHelper.WriteError(w, http.StatusUnauthorized,
			"Your session no longer matches a login", "ISSUE_REPORTER",
			err)
		return "", "", false
	}
	if err != nil {
		appHelper.WriteError(w, http.StatusInternalServerError,
			"Failed to read your login", "ISSUE_REPORTER", err)
		return "", "", false
	}
	name = strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		name = user.UserName
	}
	return truncateRunes(name, model.IssueMaxNameLen),
		truncateRunes(user.Email, model.IssueMaxNameLen), true
}

// truncateRunes keeps the first max runes of text.
func truncateRunes(text string, max int) string {
	if utf8.RuneCountInString(text) <= max {
		return text
	}
	return string([]rune(text)[:max])
}

// cleanIssueFilename is the base name with control characters and quotes
// removed (doc/105, Serving user files).
func cleanIssueFilename(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	return strings.Map(func(char rune) rune {
		if unicode.IsControl(char) || char == '"' || char == '\'' {
			return -1
		}
		return char
	}, name)
}

// parseIssueForm reads a multipart (or plain) form under the request cap:
// a declared length over it is 413 before reading, and the read itself is
// capped. On failure it has answered.
func parseIssueForm(
	appHelper *security.AppHelper, w http.ResponseWriter, r *http.Request,
) bool {
	if r.ContentLength > model.IssueMaxRequestBytes {
		appHelper.WriteError(w, http.StatusRequestEntityTooLarge,
			fmt.Sprintf("Keep the report and its files under %d MB",
				model.IssueMaxRequestBytes>>20),
			"BODY_TOO_LARGE", nil)
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, model.IssueMaxRequestBytes)
	err := r.ParseMultipartForm(model.IssueMaxRequestBytes)
	if errors.Is(err, http.ErrNotMultipart) {
		err = r.ParseForm() // no files: a plain form is fine
	}
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			appHelper.WriteError(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("Keep the report and its files under %d MB",
					model.IssueMaxRequestBytes>>20),
				"BODY_TOO_LARGE", err)
			return false
		}
		appHelper.WriteError(w, http.StatusBadRequest, "Invalid form",
			"INV_REQ_BODY", err)
		return false
	}
	return true
}

// cleanupIssueForm removes any temporary files the form parser wrote.
func cleanupIssueForm(r *http.Request) {
	if r.MultipartForm != nil {
		_ = r.MultipartForm.RemoveAll()
	}
}

// fileProblem is why an upload was refused: the status and the message.
type fileProblem struct {
	status  int
	message string
}

// readIssueFileList checks and reads the form's "files" parts for a report
// of kind: at most five, an allowed extension, at most 10 MB each, and
// bytes that are what the extension promises (the stored type is read
// from them; doc/105, Serving user files).
func readIssueFileList(
	r *http.Request, kind string,
) ([]model.IssueReportFile, *fileProblem) {
	if r.MultipartForm == nil {
		return nil, nil
	}
	headerList := r.MultipartForm.File["files"]
	if len(headerList) > model.IssueMaxFileCount {
		return nil, &fileProblem{http.StatusBadRequest, fmt.Sprintf(
			"Attach at most %d files", model.IssueMaxFileCount)}
	}
	fileList := make([]model.IssueReportFile, 0, len(headerList))
	for _, header := range headerList {
		name := cleanIssueFilename(header.Filename)
		if name == "" || name == "." || name == "/" ||
			utf8.RuneCountInString(name) > model.IssueMaxFilenameLen {
			return nil, &fileProblem{http.StatusBadRequest,
				"A file name is missing or too long"}
		}
		if !model.IssueFileExtensionAllowed(name, kind) {
			message := name + ": attach images (PNG, JPG, GIF, WebP), " +
				"PDF, TXT, LOG or CSV files, or an HTML page on a " +
				"polish or feature report"
			if model.IssueFileIsHTML(name) {
				message = name + ": an HTML page may go on a polish or " +
					"feature report, not on a bug"
			}
			return nil, &fileProblem{http.StatusUnsupportedMediaType,
				message}
		}
		tooLarge := &fileProblem{http.StatusRequestEntityTooLarge,
			fmt.Sprintf("%s is larger than %d MB", name,
				model.IssueMaxFileBytes>>20)}
		if header.Size > model.IssueMaxFileBytes {
			return nil, tooLarge
		}
		opened, err := header.Open()
		if err != nil {
			return nil, &fileProblem{http.StatusBadRequest,
				"Could not read " + name}
		}
		content, err := io.ReadAll(
			io.LimitReader(opened, model.IssueMaxFileBytes+1))
		opened.Close()
		if err != nil {
			return nil, &fileProblem{http.StatusBadRequest,
				"Could not read " + name}
		}
		if len(content) > model.IssueMaxFileBytes {
			return nil, tooLarge
		}
		contentType, ok := model.IssueFileType(name, content)
		if !ok {
			return nil, &fileProblem{http.StatusUnsupportedMediaType,
				name + ": the file's contents are not what its name says"}
		}
		fileList = append(fileList, model.IssueReportFile{
			Filename: name, ContentType: contentType, Content: content,
		})
	}
	return fileList, nil
}

// optionalText is nil for "".
func optionalText(text string) *string {
	if text == "" {
		return nil
	}
	return &text
}

// CreateIssueReport handles POST /issueReport (multipart: kind, summary,
// detail, pageUrl, context, files).
func (h *IssueReportHandler) CreateIssueReport(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "CreateIssueReport")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantLogon)

	if !appHelper.CheckAllChecksComplete() {
		return
	}
	if !h.moduleOn(appHelper, w) {
		return
	}

	jwtClaims := appHelper.GetClaims()
	if !parseIssueForm(appHelper, w, r) {
		return
	}
	defer cleanupIssueForm(r)

	kind := strings.TrimSpace(r.FormValue("kind"))
	summary := strings.TrimSpace(r.FormValue("summary"))
	detail := strings.TrimSpace(r.FormValue("detail"))
	pageURL := strings.TrimSpace(r.FormValue("pageUrl"))
	contextJSON := strings.TrimSpace(r.FormValue("context"))
	badRequest := func(message string) {
		appHelper.WriteError(w, http.StatusBadRequest, message,
			"BAD_REQUEST", nil)
	}
	switch {
	case !model.ValidIssueKind(kind):
		badRequest("Choose Bug, Polish or Feature")
		return
	case summary == "":
		badRequest("A summary is required")
		return
	case utf8.RuneCountInString(summary) > model.IssueMaxSummaryLen:
		badRequest(fmt.Sprintf("Keep the summary under %d characters",
			model.IssueMaxSummaryLen))
		return
	case utf8.RuneCountInString(detail) > model.IssueMaxDetailLen:
		badRequest(fmt.Sprintf("Keep the description under %d characters",
			model.IssueMaxDetailLen))
		return
	}
	pageURL = truncateRunes(pageURL, model.IssueMaxPageURLLen)
	// The context helps reproduce; a missing or malformed one never
	// blocks a report. Only a JSON object is kept.
	var contextJSONB json.RawMessage
	if len(contextJSON) <= model.IssueMaxContextBytes &&
		strings.HasPrefix(contextJSON, "{") &&
		json.Valid([]byte(contextJSON)) {
		contextJSONB = json.RawMessage(contextJSON)
	}
	fileList, problem := readIssueFileList(r, kind)
	if problem != nil {
		appHelper.WriteError(w, problem.status, problem.message,
			"ISSUE_BAD_FILE", nil)
		return
	}
	reporterName, reporterEmail, ok := h.reporterOf(appHelper, w, r)
	if !ok {
		return
	}

	now := model.Now()
	report := &model.IssueReportModel{
		SecurityContext: jwtClaims.SecurityContext,
		Kind:            kind,
		Summary:         summary,
		Detail:          optionalText(detail),
		PageURL:         optionalText(pageURL),
		ContextJSONB:    contextJSONB,
		State:           model.IssueStateOpen,
		Status:          "Sent",
		ReporterName:    reporterName,
		ReporterEmail:   reporterEmail,
		RelayState:      h.newRelayState(),
		CreateTS:        now,
		CreateUserID:    jwtClaims.UserID,
		UpdateTS:        now,
		UpdateUserID:    jwtClaims.UserID,
	}
	if report.RelayState == model.IssueRelayLocal {
		report.Status = "Received"
	}
	reportID, err := h.issueReportRepository.CreateIssueReport(
		r.Context(), report, fileList, jwtClaims.SecurityContext)
	if err != nil {
		mapRepoError(appHelper, w, err, "Failed to save the report",
			"ISSUE_CREATE_FAIL")
		return
	}
	h.writeDetail(appHelper, w, r, reportID, http.StatusCreated)
	appHelper.Info("ISSUE_CREATED", fmt.Sprintf(
		"Issue report %d (%s) with %d files", reportID, kind,
		len(fileList)))
}

// RetrieveIssueReportList handles GET /issueReportList: the tenant's
// reports, most recently active first.
func (h *IssueReportHandler) RetrieveIssueReportList(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv,
		"RetrieveIssueReportList")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantLogon)

	if !appHelper.CheckAllChecksComplete() {
		return
	}
	if !h.moduleOn(appHelper, w) {
		return
	}

	reportList, err := h.issueReportRepository.RetrieveIssueReportList(
		r.Context(), appHelper.GetClaims().SecurityContext)
	if err != nil {
		mapRepoError(appHelper, w, err, "Failed to list reports",
			"ISSUE_LIST_FAIL")
		return
	}
	appHelper.WriteJSON(w, http.StatusOK,
		model.DataWrapper{Data: reportList})
}

// RetrieveIssueReport handles GET /issueReport?issueReportId=N: one report
// with its files and conversation.
func (h *IssueReportHandler) RetrieveIssueReport(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv,
		"RetrieveIssueReport")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantLogon)

	if !appHelper.CheckAllChecksComplete() {
		return
	}
	if !h.moduleOn(appHelper, w) {
		return
	}

	reportID, err := GetIntFromQuery(r, "issueReportId")
	if err != nil {
		appHelper.WriteError(w, http.StatusBadRequest, err.Error(),
			"PARSE_QUERY_FAIL", nil)
		return
	}
	h.writeDetail(appHelper, w, r, reportID, http.StatusOK)
}

// CreateIssueReportMessage handles POST /issueReportMessage (multipart:
// issueReportId, body, files): the reporter adds to the conversation.
func (h *IssueReportHandler) CreateIssueReportMessage(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv,
		"CreateIssueReportMessage")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantLogon)

	if !appHelper.CheckAllChecksComplete() {
		return
	}
	if !h.moduleOn(appHelper, w) {
		return
	}

	jwtClaims := appHelper.GetClaims()
	if !parseIssueForm(appHelper, w, r) {
		return
	}
	defer cleanupIssueForm(r)

	reportID, err := strconv.Atoi(r.FormValue("issueReportId"))
	if err != nil {
		appHelper.WriteError(w, http.StatusBadRequest,
			"issueReportId is required", "BAD_REQUEST", nil)
		return
	}
	body := strings.TrimSpace(r.FormValue("body"))
	report, err := h.issueReportRepository.RetrieveIssueReport(
		r.Context(), reportID, jwtClaims.SecurityContext)
	if err != nil {
		mapRepoError(appHelper, w, err, "Failed to load the report",
			"ISSUE_GET_FAIL")
		return
	}
	fileList, problem := readIssueFileList(r, report.Kind)
	if problem != nil {
		appHelper.WriteError(w, problem.status, problem.message,
			"ISSUE_BAD_FILE", nil)
		return
	}
	if body == "" && len(fileList) == 0 {
		appHelper.WriteError(w, http.StatusBadRequest,
			"Write a message or attach a file", "BAD_REQUEST", nil)
		return
	}
	if utf8.RuneCountInString(body) > model.IssueMaxDetailLen {
		appHelper.WriteError(w, http.StatusBadRequest, fmt.Sprintf(
			"Keep messages under %d characters", model.IssueMaxDetailLen),
			"BAD_REQUEST", nil)
		return
	}
	if body == "" {
		body = "(attachment)" // the tracker refuses an empty message
	}
	authorName, _, ok := h.reporterOf(appHelper, w, r)
	if !ok {
		return
	}
	message := &model.IssueReportMessageModel{
		IssueReportID:   reportID,
		SecurityContext: jwtClaims.SecurityContext,
		AuthorKind:      model.IssueAuthorReporter,
		AuthorName:      &authorName,
		Body:            body,
		RelayState:      h.newRelayState(),
		CreateTS:        model.Now(),
		CreateUserID:    jwtClaims.UserID,
	}
	_, err = h.issueReportRepository.CreateIssueReportMessage(r.Context(),
		message, fileList, nil, nil, jwtClaims.SecurityContext)
	if err != nil {
		mapRepoError(appHelper, w, err, "Failed to send the message",
			"ISSUE_MSG_FAIL")
		return
	}
	h.writeDetail(appHelper, w, r, reportID, http.StatusOK)
}

// CreateIssueReportAction handles POST /issueReportAction {issueReportId,
// action: withdraw|reopen, note}. A reporter may withdraw a report that
// is not closed and reopen one that is (409 otherwise).
func (h *IssueReportHandler) CreateIssueReportAction(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv,
		"CreateIssueReportAction")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantLogon)

	if !appHelper.CheckAllChecksComplete() {
		return
	}
	if !h.moduleOn(appHelper, w) {
		return
	}

	jwtClaims := appHelper.GetClaims()
	var request model.IssueReportActionRequest
	if !decodeJSONBody(appHelper, w, r, model.IssueMaxActionBodyBytes,
		&request) {
		return
	}
	request.Note = strings.TrimSpace(request.Note)
	if utf8.RuneCountInString(request.Note) > model.IssueMaxNoteLen {
		appHelper.WriteError(w, http.StatusBadRequest, fmt.Sprintf(
			"Keep the note under %d characters", model.IssueMaxNoteLen),
			"BAD_REQUEST", nil)
		return
	}
	if request.Action != model.IssueActionWithdraw &&
		request.Action != model.IssueActionReopen {
		appHelper.WriteError(w, http.StatusBadRequest,
			"action must be withdraw or reopen", "BAD_REQUEST", nil)
		return
	}
	report, err := h.issueReportRepository.RetrieveIssueReport(
		r.Context(), request.IssueReportID, jwtClaims.SecurityContext)
	if err != nil {
		mapRepoError(appHelper, w, err, "Failed to load the report",
			"ISSUE_GET_FAIL")
		return
	}
	closed := model.IssueStateIsClosed(report.State)
	var newState, newStatus, body string
	switch {
	case request.Action == model.IssueActionWithdraw && closed:
		appHelper.WriteError(w, http.StatusConflict,
			"This report is already closed", "ISSUE_CLOSED", nil)
		return
	case request.Action == model.IssueActionReopen && !closed:
		appHelper.WriteError(w, http.StatusConflict,
			"Only a closed report can be reopened", "ISSUE_OPEN", nil)
		return
	case request.Action == model.IssueActionWithdraw:
		newState, newStatus = model.IssueStateClosed, "Withdrawn"
		body = model.IssueWithdrawText
	default:
		newState, newStatus = model.IssueStateReopened, "Reopened"
		body = model.IssueReopenText
	}
	if request.Note != "" {
		body += "\n\n" + request.Note
	}
	authorName, _, ok := h.reporterOf(appHelper, w, r)
	if !ok {
		return
	}
	action := request.Action
	message := &model.IssueReportMessageModel{
		IssueReportID:   report.IssueReportID,
		SecurityContext: jwtClaims.SecurityContext,
		AuthorKind:      model.IssueAuthorReporter,
		AuthorName:      &authorName,
		Body:            body,
		Action:          &action,
		RelayState:      h.newRelayState(),
		CreateTS:        model.Now(),
		CreateUserID:    jwtClaims.UserID,
	}
	_, err = h.issueReportRepository.CreateIssueReportMessage(r.Context(),
		message, nil, &newState, &newStatus, jwtClaims.SecurityContext)
	if err != nil {
		mapRepoError(appHelper, w, err, "Failed to update the report",
			"ISSUE_ACTION_FAIL")
		return
	}
	h.writeDetail(appHelper, w, r, report.IssueReportID, http.StatusOK)
	appHelper.Info("ISSUE_ACTION", fmt.Sprintf("Issue report %d: %s",
		report.IssueReportID, request.Action))
}

// RetrieveIssueReportAttachment handles GET
// /issueReportAttachment?issueReportAttachmentId=N: one file's bytes, by
// doc/105 Serving user files. The type is sniffed from the bytes on the
// way out; only an image is inline; nothing is cached; a staff file is
// fetched from the tracker; every miss is the same 404.
func (h *IssueReportHandler) RetrieveIssueReportAttachment(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv,
		"RetrieveIssueReportAttachment")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantLogon)

	if !appHelper.CheckAllChecksComplete() {
		return
	}
	if !h.moduleOn(appHelper, w) {
		return
	}

	attachmentID, err := GetIntFromQuery(r, "issueReportAttachmentId")
	if err != nil {
		appHelper.WriteError(w, http.StatusBadRequest, err.Error(),
			"PARSE_QUERY_FAIL", nil)
		return
	}
	file, err := h.issueReportRepository.RetrieveIssueReportAttachment(
		r.Context(), attachmentID, appHelper.GetClaims().SecurityContext)
	if err != nil {
		mapRepoError(appHelper, w, err, "Failed to load the file",
			"ISSUE_FILE_FAIL")
		return
	}
	content := file.Content
	if content == nil {
		if h.relayDownloader == nil || file.RemoteID == nil ||
			file.ReportRemoteID == nil {
			appHelper.WriteError(w, http.StatusNotFound, "Not found",
				"NOT_FOUND_ERROR", nil)
			return
		}
		content, err = h.relayDownloader.Download(r.Context(),
			*file.ReportRemoteID, *file.RemoteID, model.IssueMaxFileBytes)
		if err != nil {
			appHelper.WriteError(w, http.StatusBadGateway,
				"Could not fetch this file right now; try again shortly",
				"RELAY_DOWNLOAD_FAIL", err)
			return
		}
	}
	contentType, inline := model.IssueFileServeType(content)
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(
		disposition,
		map[string]string{"filename": cleanIssueFilename(file.Filename)}))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

// writeDetail answers with a report's detail.
func (h *IssueReportHandler) writeDetail(
	appHelper *security.AppHelper,
	w http.ResponseWriter,
	r *http.Request,
	reportID int,
	status int,
) {
	detail, err := h.issueReportRepository.RetrieveIssueReportDetail(
		r.Context(), reportID, appHelper.GetClaims().SecurityContext)
	if err != nil {
		mapRepoError(appHelper, w, err, "Failed to load the report",
			"ISSUE_GET_FAIL")
		return
	}
	appHelper.WriteJSON(w, status, model.DataWrapper{Data: detail})
}

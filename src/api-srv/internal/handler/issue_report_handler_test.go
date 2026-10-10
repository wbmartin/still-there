package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
	"github.com/wbmartin/starter-api/internal/security"
)

// MockIssueReportRepository satisfies IssueReportRepositoryInterface. A
// call with no expectation panics: a refused request never reaches it.
type MockIssueReportRepository struct {
	mock.Mock
}

func (m *MockIssueReportRepository) CreateIssueReport(
	ctx context.Context, report *model.IssueReportModel,
	fileList []model.IssueReportFile, securityContext int,
) (int, error) {
	args := m.Called(ctx, report, fileList, securityContext)
	return args.Int(0), args.Error(1)
}

func (m *MockIssueReportRepository) RetrieveIssueReportList(
	ctx context.Context, securityContext int,
) ([]*model.IssueReportModel, error) {
	args := m.Called(ctx, securityContext)
	return args.Get(0).([]*model.IssueReportModel), args.Error(1)
}

func (m *MockIssueReportRepository) RetrieveIssueReport(
	ctx context.Context, reportID int, securityContext int,
) (*model.IssueReportModel, error) {
	args := m.Called(ctx, reportID, securityContext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.IssueReportModel), args.Error(1)
}

func (m *MockIssueReportRepository) RetrieveIssueReportDetail(
	ctx context.Context, reportID int, securityContext int,
) (*model.IssueReportDetailModel, error) {
	args := m.Called(ctx, reportID, securityContext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.IssueReportDetailModel), args.Error(1)
}

func (m *MockIssueReportRepository) CreateIssueReportMessage(
	ctx context.Context, message *model.IssueReportMessageModel,
	fileList []model.IssueReportFile, newState *string,
	newStatus *string, securityContext int,
) (int, error) {
	args := m.Called(ctx, message, fileList, newState, newStatus,
		securityContext)
	return args.Int(0), args.Error(1)
}

func (m *MockIssueReportRepository) RetrieveIssueReportAttachment(
	ctx context.Context, attachmentID int, securityContext int,
) (*repository.IssueReportAttachmentContent, error) {
	args := m.Called(ctx, attachmentID, securityContext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.IssueReportAttachmentContent),
		args.Error(1)
}

// fakeDownloader stands in for the relay client.
type fakeDownloader struct {
	content  []byte
	err      error
	calledID string
}

func (f *fakeDownloader) Download(
	_ context.Context, remoteReportID string, remoteAttachmentID string,
	_ int64,
) ([]byte, error) {
	f.calledID = remoteReportID + "/" + remoteAttachmentID
	return f.content, f.err
}

const (
	issueTestTenant = 2
	issueTestUserID = 7
)

var (
	pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR" +
		"\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00")
	htmlBytes = []byte("<!doctype html><html><body>" +
		"<script>alert(1)</script></body></html>")
)

// issueServices is testServices with the module on.
func issueServices(t *testing.T) *app.Services {
	t.Helper()
	appServices := testServices(t)
	appServices.Config.IssueReportsEnabled = true
	return appServices
}

// issueRequest is method target as alice (login 7) of tenant 2 holding
// LOGON, with the auth cookie and the CSRF header.
func issueRequest(
	t *testing.T, appServices *app.Services, method string, target string,
	body io.Reader, contentType string,
) *http.Request {
	t.Helper()
	const sessionID = "session-issue"
	token, err := auth.GenerateJWT(
		"Test", "alice", issueTestUserID, sessionID, issueTestTenant,
		[]int{model.GrantLogon}, appServices.Config.JWTSecret,
		appServices.Config.JWTSessionExpireHours,
	)
	require.NoError(t, err)
	r := httptest.NewRequest(method, target, body)
	r.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	r.Header.Set("X-CSRF-Token", sessionID[len(sessionID)-3:])
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	return r
}

type formFile struct {
	name    string
	content []byte
}

// issueForm is a multipart body of fields and "files" parts.
func issueForm(
	t *testing.T, fieldList map[string]string, fileList []formFile,
) (*bytes.Buffer, string) {
	t.Helper()
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	for key, value := range fieldList {
		require.NoError(t, writer.WriteField(key, value))
	}
	for _, file := range fileList {
		part, err := writer.CreateFormFile("files", file.name)
		require.NoError(t, err)
		_, err = part.Write(file.content)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return &buffer, writer.FormDataContentType()
}

func aliceRepository() *MockProfileRepository {
	reporterRepository := new(MockProfileRepository)
	reporterRepository.On("RetrieveUserByUserName", mock.Anything,
		"alice").Return(&model.AppUserModel{
		AppUserID: issueTestUserID, UserName: "alice",
		FirstName: "Alice", LastName: "Example",
		Email: "alice@example.com", SecurityContext: issueTestTenant,
	}, nil)
	return reporterRepository
}

func validReportFields() map[string]string {
	return map[string]string{
		"kind":            "bug",
		"summary":         "Save does nothing",
		"detail":          "1. Open\n2. **Save**",
		"pageUrl":         "/exampleList",
		"context":         `{"browser":"test"}`,
		"securityContext": "3", // ignored: the tenant is the JWT's
	}
}

func TestIssueReportHandler_ModuleOffAnswers404AfterTheChecks(t *testing.T) {
	appServices := testServices(t) // IssueReportsEnabled false
	issueReportRepository := new(MockIssueReportRepository)
	issueHandler := NewIssueReportHandler(appServices,
		issueReportRepository, new(MockProfileRepository), nil)

	w := httptest.NewRecorder()
	issueHandler.RetrieveIssueReportList(w, issueRequest(t, appServices,
		http.MethodGet, "/issueReportList", nil, ""))
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Issue reports are not enabled")

	// Off, but the checks still run first: no session is 401, not 404.
	w = httptest.NewRecorder()
	issueHandler.RetrieveIssueReportList(w,
		httptest.NewRequest(http.MethodGet, "/issueReportList", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	issueReportRepository.AssertExpectations(t)
}

func TestIssueReportHandler_CreateStoresInTheCallersTenant(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		relayOn    bool
		wantRelay  string
		wantStatus string
	}{
		{"relay off", false, model.IssueRelayLocal, "Received"},
		{"relay on", true, model.IssueRelayPending, "Sent"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			appServices := issueServices(t)
			if testCase.relayOn {
				appServices.Config.IssueRelayURL = "https://t.example.com"
				appServices.Config.IssueRelayProjectID = "p"
				appServices.Config.IssueRelayToken = "token"
			}
			issueReportRepository := new(MockIssueReportRepository)
			var stored *model.IssueReportModel
			var storedFileList []model.IssueReportFile
			issueReportRepository.On("CreateIssueReport", mock.Anything,
				mock.Anything, mock.Anything, issueTestTenant).
				Run(func(args mock.Arguments) {
					stored = args.Get(1).(*model.IssueReportModel)
					storedFileList = args.Get(2).([]model.IssueReportFile)
				}).Return(41, nil)
			issueReportRepository.On("RetrieveIssueReportDetail",
				mock.Anything, 41, issueTestTenant).Return(
				&model.IssueReportDetailModel{
					IssueReportModel: model.IssueReportModel{
						IssueReportID: 41}}, nil)
			issueHandler := NewIssueReportHandler(appServices,
				issueReportRepository, aliceRepository(), nil)

			body, contentType := issueForm(t, validReportFields(),
				[]formFile{{"shot.png", pngBytes}})
			w := httptest.NewRecorder()
			issueHandler.CreateIssueReport(w, issueRequest(t, appServices,
				http.MethodPost, "/issueReport", body, contentType))

			require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
			require.NotNil(t, stored)
			assert.Equal(t, issueTestTenant, stored.SecurityContext)
			assert.Equal(t, issueTestUserID, stored.CreateUserID)
			assert.Equal(t, "Alice Example", stored.ReporterName)
			assert.Equal(t, "alice@example.com", stored.ReporterEmail)
			assert.Equal(t, model.IssueStateOpen, stored.State)
			assert.Equal(t, testCase.wantStatus, stored.Status)
			assert.Equal(t, testCase.wantRelay, stored.RelayState)
			assert.JSONEq(t, `{"browser":"test"}`,
				string(stored.ContextJSONB))
			require.Len(t, storedFileList, 1)
			assert.Equal(t, "image/png", storedFileList[0].ContentType)
			assert.Contains(t, w.Body.String(), `"issueReportId":41`)
			issueReportRepository.AssertExpectations(t)
		})
	}
}

// Every refusal of a bad upload answers before the repository.
func TestIssueReportHandler_CreateRefusesBadInput(t *testing.T) {
	sixFileList := make([]formFile, 6)
	for i := range sixFileList {
		sixFileList[i] = formFile{fmt.Sprintf("f%d.txt", i),
			[]byte("hi")}
	}
	withField := func(key string, value string) map[string]string {
		fieldList := validReportFields()
		fieldList[key] = value
		return fieldList
	}
	for _, testCase := range []struct {
		name       string
		fieldList  map[string]string
		fileList   []formFile
		wantStatus int
		wantText   string
	}{
		{"unknown kind", withField("kind", "idea"), nil,
			http.StatusBadRequest, "Choose Bug, Polish or Feature"},
		{"no summary", withField("summary", "  "), nil,
			http.StatusBadRequest, "A summary is required"},
		{"long summary", withField("summary", strings.Repeat("x", 501)),
			nil, http.StatusBadRequest, "Keep the summary under"},
		{"six files", validReportFields(), sixFileList,
			http.StatusBadRequest, "Attach at most 5 files"},
		{"executable", validReportFields(),
			[]formFile{{"run.exe", []byte("MZ")}},
			http.StatusUnsupportedMediaType, "attach images"},
		{"HTML named .png", validReportFields(),
			[]formFile{{"shot.png", htmlBytes}},
			http.StatusUnsupportedMediaType, "not what its name says"},
		{"HTML named .pdf", validReportFields(),
			[]formFile{{"doc.pdf", htmlBytes}},
			http.StatusUnsupportedMediaType, "not what its name says"},
		{"HTML in a .txt", validReportFields(),
			[]formFile{{"notes.txt", htmlBytes}},
			http.StatusUnsupportedMediaType, "not what its name says"},
		{"HTML page on a bug", validReportFields(),
			[]formFile{{"mockup.html", htmlBytes}},
			http.StatusUnsupportedMediaType, "not on a bug"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			appServices := issueServices(t)
			issueReportRepository := new(MockIssueReportRepository)
			issueHandler := NewIssueReportHandler(appServices,
				issueReportRepository, aliceRepository(), nil)
			body, contentType := issueForm(t, testCase.fieldList,
				testCase.fileList)
			w := httptest.NewRecorder()
			issueHandler.CreateIssueReport(w, issueRequest(t, appServices,
				http.MethodPost, "/issueReport", body, contentType))
			assert.Equal(t, testCase.wantStatus, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), testCase.wantText)
			issueReportRepository.AssertExpectations(t)
		})
	}
}

// An HTML page may go on a polish or feature report (a mock-up).
func TestIssueReportHandler_HTMLPageOnAFeature(t *testing.T) {
	appServices := issueServices(t)
	issueReportRepository := new(MockIssueReportRepository)
	var storedFileList []model.IssueReportFile
	issueReportRepository.On("CreateIssueReport", mock.Anything,
		mock.Anything, mock.Anything, issueTestTenant).
		Run(func(args mock.Arguments) {
			storedFileList = args.Get(2).([]model.IssueReportFile)
		}).Return(5, nil)
	issueReportRepository.On("RetrieveIssueReportDetail", mock.Anything, 5,
		issueTestTenant).Return(&model.IssueReportDetailModel{}, nil)
	issueHandler := NewIssueReportHandler(appServices,
		issueReportRepository, aliceRepository(), nil)
	fieldList := validReportFields()
	fieldList["kind"] = model.IssueKindFeature
	body, contentType := issueForm(t, fieldList,
		[]formFile{{"mockup.html", htmlBytes}})
	w := httptest.NewRecorder()
	issueHandler.CreateIssueReport(w, issueRequest(t, appServices,
		http.MethodPost, "/issueReport", body, contentType))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Len(t, storedFileList, 1)
	assert.Equal(t, "text/html", storedFileList[0].ContentType)
}

// Size, twice: a declared length over the cap is 413 before reading, and
// so is a body that runs over it with no declared length.
func TestIssueReportHandler_CreateOverTheCapIs413(t *testing.T) {
	over := model.IssueMaxRequestBytes + 1
	for _, testCase := range []struct {
		name          string
		contentLength int64
	}{
		{"declared", int64(over)},
		{"actual", -1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			appServices := issueServices(t)
			issueReportRepository := new(MockIssueReportRepository)
			issueHandler := NewIssueReportHandler(appServices,
				issueReportRepository, aliceRepository(), nil)
			body, contentType := issueForm(t, validReportFields(),
				[]formFile{{"big.txt",
					bytes.Repeat([]byte("a"), over)}})
			r := issueRequest(t, appServices, http.MethodPost,
				"/issueReport", body, contentType)
			r.ContentLength = testCase.contentLength
			w := httptest.NewRecorder()
			issueHandler.CreateIssueReport(w, r)
			assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code,
				w.Body.String())
			issueReportRepository.AssertExpectations(t)
		})
	}
}

// Another tenant's report is not found: the lookup carries the caller's
// tenant from the JWT, so nothing is written.
func TestIssueReportHandler_MessageOnAnotherTenantsReportIs404(
	t *testing.T,
) {
	appServices := issueServices(t)
	issueReportRepository := new(MockIssueReportRepository)
	issueReportRepository.On("RetrieveIssueReport", mock.Anything, 99,
		issueTestTenant).Return(nil, repository.ErrNotFound)
	issueHandler := NewIssueReportHandler(appServices,
		issueReportRepository, aliceRepository(), nil)
	body, contentType := issueForm(t, map[string]string{
		"issueReportId": "99", "body": "hello"}, nil)
	w := httptest.NewRecorder()
	issueHandler.CreateIssueReportMessage(w, issueRequest(t, appServices,
		http.MethodPost, "/issueReportMessage", body, contentType))
	assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	issueReportRepository.AssertExpectations(t)
}

func TestIssueReportHandler_Action(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		state      string
		action     string
		wantStatus int
		wantState  string
	}{
		{"withdraw open", model.IssueStateOpen, "withdraw",
			http.StatusOK, model.IssueStateClosed},
		{"withdraw closed", model.IssueStateClosed, "withdraw",
			http.StatusConflict, ""},
		{"reopen closed", model.IssueStateWontFix, "reopen",
			http.StatusOK, model.IssueStateReopened},
		{"reopen open", "fixing", "reopen", http.StatusConflict, ""},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			appServices := issueServices(t)
			issueReportRepository := new(MockIssueReportRepository)
			issueReportRepository.On("RetrieveIssueReport", mock.Anything,
				3, issueTestTenant).Return(&model.IssueReportModel{
				IssueReportID: 3, State: testCase.state}, nil)
			var newState *string
			var message *model.IssueReportMessageModel
			if testCase.wantStatus == http.StatusOK {
				issueReportRepository.On("CreateIssueReportMessage",
					mock.Anything, mock.Anything, mock.Anything,
					mock.Anything, mock.Anything, issueTestTenant).
					Run(func(args mock.Arguments) {
						message =
							args.Get(1).(*model.IssueReportMessageModel)
						newState = args.Get(3).(*string)
					}).Return(1, nil)
				issueReportRepository.On("RetrieveIssueReportDetail",
					mock.Anything, 3, issueTestTenant).Return(
					&model.IssueReportDetailModel{}, nil)
			}
			issueHandler := NewIssueReportHandler(appServices,
				issueReportRepository, aliceRepository(), nil)
			payload, _ := json.Marshal(model.IssueReportActionRequest{
				IssueReportID: 3, Action: testCase.action,
				Note: "a note"})
			w := httptest.NewRecorder()
			issueHandler.CreateIssueReportAction(w, issueRequest(t,
				appServices, http.MethodPost, "/issueReportAction",
				bytes.NewReader(payload), "application/json"))
			require.Equal(t, testCase.wantStatus, w.Code, w.Body.String())
			if testCase.wantStatus == http.StatusOK {
				require.NotNil(t, newState)
				assert.Equal(t, testCase.wantState, *newState)
				assert.Equal(t, testCase.action, *message.Action)
				assert.Contains(t, message.Body, "a note")
				assert.Equal(t, issueTestTenant, message.SecurityContext)
			}
			issueReportRepository.AssertExpectations(t)
		})
	}
}

// download runs GET /issueReportAttachment through the security-headers
// middleware, as main.go mounts it.
func download(
	t *testing.T, appServices *app.Services, issueHandler *IssueReportHandler,
) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	security.HeadersMiddleware(
		http.HandlerFunc(issueHandler.RetrieveIssueReportAttachment),
	).ServeHTTP(w, issueRequest(t, appServices, http.MethodGet,
		"/issueReportAttachment?issueReportAttachmentId=8", nil, ""))
	return w
}

func storedFile(
	name string, content []byte,
) *repository.IssueReportAttachmentContent {
	return &repository.IssueReportAttachmentContent{
		IssueReportAttachmentModel: model.IssueReportAttachmentModel{
			IssueReportAttachmentID: 8, Filename: name,
			ContentType: "image/png"},
		Content: content,
	}
}

// doc/105 Serving user files: the type comes from the bytes (never the
// stored column), only an image is inline, nothing is cached, nosniff and
// DENY stay, and HTML never goes out as HTML.
func TestIssueReportHandler_AttachmentServing(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		file            *repository.IssueReportAttachmentContent
		wantType        string
		wantDisposition string
	}{
		{"image inline", storedFile("shot.png", pngBytes), "image/png",
			`inline; filename=shot.png`},
		{"HTML stored as png is a download",
			storedFile("mockup.html", htmlBytes),
			"application/octet-stream", `attachment; filename=mockup.html`},
		{"text", storedFile("notes.txt", []byte("plain words")),
			"text/plain; charset=utf-8", `attachment; filename=notes.txt`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			appServices := issueServices(t)
			issueReportRepository := new(MockIssueReportRepository)
			issueReportRepository.On("RetrieveIssueReportAttachment",
				mock.Anything, 8, issueTestTenant).Return(testCase.file, nil)
			w := download(t, appServices, NewIssueReportHandler(
				appServices, issueReportRepository, aliceRepository(),
				nil))
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assert.Equal(t, testCase.file.Content, w.Body.Bytes())
			assert.Equal(t, testCase.wantType, w.Header().Get("Content-Type"))
			assert.Equal(t, testCase.wantDisposition,
				w.Header().Get("Content-Disposition"))
			assert.Equal(t, "private, no-store",
				w.Header().Get("Cache-Control"))
			assert.Equal(t, "nosniff",
				w.Header().Get("X-Content-Type-Options"))
			assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		})
	}
}

func TestIssueReportHandler_AttachmentMissesAre404(t *testing.T) {
	appServices := issueServices(t)

	// Another tenant's id: the repository, scoped by the JWT's tenant,
	// finds nothing.
	issueReportRepository := new(MockIssueReportRepository)
	issueReportRepository.On("RetrieveIssueReportAttachment", mock.Anything,
		8, issueTestTenant).Return(nil, repository.ErrNotFound)
	w := download(t, appServices, NewIssueReportHandler(appServices,
		issueReportRepository, aliceRepository(), nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	// A tracker-kept file with no relay to fetch it through.
	remoteID, reportRemoteID := "A-1", "R-1"
	trackerFile := storedFile("reply.png", nil)
	trackerFile.RemoteID = &remoteID
	trackerFile.ReportRemoteID = &reportRemoteID
	issueReportRepository = new(MockIssueReportRepository)
	issueReportRepository.On("RetrieveIssueReportAttachment", mock.Anything,
		8, issueTestTenant).Return(trackerFile, nil)
	w = download(t, appServices, NewIssueReportHandler(appServices,
		issueReportRepository, aliceRepository(), nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	// With the relay it is fetched, and sniffed like any other.
	downloader := &fakeDownloader{content: pngBytes}
	w = download(t, appServices, NewIssueReportHandler(appServices,
		issueReportRepository, aliceRepository(), downloader))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "R-1/A-1", downloader.calledID)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))

	// The tracker failing is a 502, not a missing file.
	downloader = &fakeDownloader{err: errors.New("tracker down")}
	w = download(t, appServices, NewIssueReportHandler(appServices,
		issueReportRepository, aliceRepository(), downloader))
	assert.Equal(t, http.StatusBadGateway, w.Code)
}

// The reporter's name comes from their own login, read by the JWT's
// subject; a row that is not the JWT's uid is refused.
func TestIssueReportHandler_ReporterMustBeTheSessionsLogin(t *testing.T) {
	appServices := issueServices(t)
	issueReportRepository := new(MockIssueReportRepository)
	reporterRepository := new(MockProfileRepository)
	reporterRepository.On("RetrieveUserByUserName", mock.Anything,
		"alice").Return(&model.AppUserModel{AppUserID: 8,
		UserName: "alice"}, nil)
	issueHandler := NewIssueReportHandler(appServices,
		issueReportRepository, reporterRepository, nil)
	body, contentType := issueForm(t, validReportFields(), nil)
	w := httptest.NewRecorder()
	issueHandler.CreateIssueReport(w, issueRequest(t, appServices,
		http.MethodPost, "/issueReport", body, contentType))
	assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	issueReportRepository.AssertExpectations(t)
}

func TestIssueFileType(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		content  []byte
		wantType string
		wantOK   bool
	}{
		{"shot.PNG", pngBytes, "image/png", true},
		{"photo.jpg", []byte("\xFF\xD8\xFF\xE0rest"), "image/jpeg", true},
		{"doc.pdf", []byte("%PDF-1.7\n"), "application/pdf", true},
		{"rows.csv", []byte("a,b\n1,2\n"), "text/csv", true},
		{"server.log", []byte("line one\n"), "text/plain", true},
		{"mockup.htm", htmlBytes, "text/html", true},
		{"photo.jpg", pngBytes, "image/png", true}, // an image either way
		{"photo.jpg", []byte("%PDF-1.7\n"), "", false},
		{"doc.pdf", pngBytes, "", false},
		{"notes.txt", htmlBytes, "", false},
		{"notes.txt", []byte("\x00\x01\x02binary"), "", false},
		{"run.exe", []byte("MZ"), "", false},
	} {
		gotType, gotOK := model.IssueFileType(testCase.name,
			testCase.content)
		assert.Equal(t, testCase.wantOK, gotOK, testCase.name)
		assert.Equal(t, testCase.wantType, gotType, testCase.name)
	}
}

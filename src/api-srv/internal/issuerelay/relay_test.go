package issuerelay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
)

// fakeStore records what the relay asks of the repository.
type fakeStore struct {
	mutex sync.Mutex
	// localList is the backlog: reports filed while the relay was off,
	// handed to reportList by RelayQueueLocalBacklog.
	localList   []*repository.IssueRelayReport
	queueCount  int
	reportList  []*repository.IssueRelayReport
	messageList []*repository.IssueRelayMessage
	targetList  []repository.IssueRelaySyncTarget
	sentByID    map[int]string
	failedByID  map[int]string
	messageByID map[int]string
	statusByID  map[int]string
	replyList   []string
	tenantList  []int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		sentByID:    map[int]string{},
		failedByID:  map[int]string{},
		messageByID: map[int]string{},
		statusByID:  map[int]string{},
	}
}

func (f *fakeStore) RelayQueueLocalBacklog(
	context.Context,
) (int, int, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.queueCount++
	queued := len(f.localList)
	f.reportList = append(f.reportList, f.localList...)
	f.localList = nil
	return queued, 0, nil
}

// RelayReportListToSend lists what is not yet sent, as the repository
// does: a sent report leaves the list.
func (f *fakeStore) RelayReportListToSend(
	context.Context, int, int,
) ([]*repository.IssueRelayReport, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	var waitingList []*repository.IssueRelayReport
	for _, report := range f.reportList {
		if _, sent := f.sentByID[report.IssueReportID]; !sent {
			waitingList = append(waitingList, report)
		}
	}
	return waitingList, nil
}

func (f *fakeStore) RelayMarkReportSent(
	_ context.Context, reportID int, securityContext int, remoteID string,
	_ string, status string, _ *string, _ float64,
) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.sentByID[reportID] = remoteID + "|" + status
	f.tenantList = append(f.tenantList, securityContext)
	return nil
}

func (f *fakeStore) RelayMarkReportFailed(
	_ context.Context, reportID int, _ int, message string,
) error {
	f.failedByID[reportID] = message
	return nil
}

func (f *fakeStore) RelayMessageListToSend(
	context.Context, int, int,
) ([]*repository.IssueRelayMessage, error) {
	return f.messageList, nil
}

func (f *fakeStore) RelayMarkMessageSent(
	_ context.Context, messageID int, _ int,
) error {
	f.messageByID[messageID] = model.IssueRelaySent
	return nil
}

func (f *fakeStore) RelayMarkMessageFailed(
	_ context.Context, messageID int, _ int, message string,
) error {
	f.messageByID[messageID] = model.IssueRelayFailed + "|" + message
	return nil
}

func (f *fakeStore) RelaySyncTargetList(
	context.Context, float64, int,
) ([]repository.IssueRelaySyncTarget, error) {
	return f.targetList, nil
}

func (f *fakeStore) RelayApplyStatus(
	_ context.Context, target repository.IssueRelaySyncTarget,
	state string, status string, _ *string, _ float64,
) error {
	f.statusByID[target.IssueReportID] = state + "|" + status
	return nil
}

func (f *fakeStore) RelayAddStaffMessage(
	_ context.Context, _ repository.IssueRelaySyncTarget, remoteID string,
	authorName string, body string, _ float64,
	fileList []repository.IssueRemoteFile,
) (bool, error) {
	entry := remoteID + "|" + authorName + "|" + body
	for _, file := range fileList {
		entry += "|" + file.RemoteID + ":" + file.Filename
	}
	f.replyList = append(f.replyList, entry)
	return true, nil
}

func textPointer(text string) *string { return &text }

const fakeReportPath = "/api/projects/P1/external/reports"

// fakeHouse is the tracker's report API as the relay uses it: it checks
// the token and the fields, and records each call.
type fakeHouse struct {
	t         *testing.T
	token     string
	mutex     sync.Mutex
	callList  []string
	formByKey map[string]string
	fileNames []string
}

func (h *fakeHouse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Report-Token") != h.token {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	h.mutex.Lock()
	h.callList = append(h.callList, r.Method+" "+
		strings.TrimPrefix(r.URL.Path, fakeReportPath))
	h.mutex.Unlock()
	switch {
	case r.Method == http.MethodPost && r.URL.Path == fakeReportPath:
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			h.t.Errorf("create: not multipart: %v", err)
			return
		}
		h.formByKey = map[string]string{}
		for key, valueList := range r.MultipartForm.Value {
			h.formByKey[key] = valueList[0]
		}
		for _, header := range r.MultipartForm.File["files"] {
			h.fileNames = append(h.fileNames, header.Filename)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(RemoteReport{
			ID: "R-1", State: "open", Status: "Under review"})
	case r.Method == http.MethodPost &&
		r.URL.Path == fakeReportPath+"/R-1/messages":
		_ = r.ParseMultipartForm(1 << 20)
		if r.FormValue("body") != "More detail" ||
			r.FormValue("author_name") != "Ben" {
			h.t.Errorf("message: body %q author %q",
				r.FormValue("body"), r.FormValue("author_name"))
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(RemoteReport{ID: "R-1"})
	case r.Method == http.MethodPost &&
		r.URL.Path == fakeReportPath+"/R-1/withdraw":
		var note map[string]string
		_ = json.NewDecoder(r.Body).Decode(&note)
		if note["note"] != "Found it" {
			h.t.Errorf("withdraw sends only the note, got %q", note["note"])
		}
		w.WriteHeader(http.StatusConflict) // staff closed it first
	case r.Method == http.MethodPost &&
		r.URL.Path == fakeReportPath+"/R-2/messages":
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"detail":"empty"}`))
	case r.Method == http.MethodGet && r.URL.Path == fakeReportPath:
		if r.URL.Query().Get("ids") != "R-1" {
			h.t.Errorf("sync ids = %q", r.URL.Query().Get("ids"))
		}
		_ = json.NewEncoder(w).Encode([]RemoteReport{{
			ID: "R-1", State: "fixing", Status: "Being fixed",
			ThreadList: []RemoteMessage{
				{ID: "M-1", AuthorKind: "app", Body: "ours"},
				{ID: "M-2", AuthorKind: "staff",
					AuthorName: textPointer("Sam"), Body: "On it",
					CreatedAt: "2026-10-01T12:00:00Z",
					AttachmentList: []RemoteAttachment{{ID: "F-1",
						Filename: "fix.png", Size: 3,
						ContentType: "image/png"}}},
				{ID: "M-3", AuthorKind: "staff", Body: "No name"},
			},
		}})
	case r.Method == http.MethodGet &&
		r.URL.Path == fakeReportPath+"/R-1/attachments/F-1":
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("png"))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func newFakeHouse(t *testing.T, token string) (*fakeHouse, *httptest.Server) {
	house := &fakeHouse{t: t, token: token}
	server := httptest.NewServer(house)
	t.Cleanup(server.Close)
	return house, server
}

func TestRelayTick_SendsSyncsAndHandlesConflicts(t *testing.T) {
	house, server := newFakeHouse(t, "token")
	store := newFakeStore()
	pageURL := "/exampleList"
	store.reportList = []*repository.IssueRelayReport{{
		IssueReportModel: model.IssueReportModel{
			IssueReportID: 7, SecurityContext: 3, Kind: "bug",
			Summary: "Save fails", ReporterName: "Ben",
			ReporterEmail: "ben@example.com", CreateUserID: 42,
			PageURL: &pageURL,
		},
		FileList: []repository.IssueRelayFile{{Filename: "shot.png",
			ContentType: "image/png", Content: []byte("png")}},
	}}
	withdraw := model.IssueActionWithdraw
	store.messageList = []*repository.IssueRelayMessage{
		{IssueReportMessageModel: model.IssueReportMessageModel{
			IssueReportMessageID: 1, Body: "More detail",
			AuthorName: textPointer("Ben")}, ReportRemoteID: "R-1"},
		{IssueReportMessageModel: model.IssueReportMessageModel{
			IssueReportMessageID: 2,
			Body:                 "Withdrew this report.\n\nFound it",
			Action:               &withdraw}, ReportRemoteID: "R-1"},
		{IssueReportMessageModel: model.IssueReportMessageModel{
			IssueReportMessageID: 3, Body: "refused"},
			ReportRemoteID: "R-2"},
	}
	store.targetList = []repository.IssueRelaySyncTarget{{
		IssueReportID: 7, RemoteID: "R-1", State: "open",
		Status: "Under review"}}

	relay := &Relay{
		Client: NewClient(Config{BaseURL: server.URL + "/",
			ProjectID: "P1", Token: "token"}, nil),
		Store: store,
	}
	relay.Tick(context.Background())

	if store.sentByID[7] != "R-1|Under review" {
		t.Errorf("report not recorded as sent: %v (failed: %v)",
			store.sentByID, store.failedByID)
	}
	if len(store.tenantList) != 1 || store.tenantList[0] != 3 {
		t.Errorf("the sent mark names tenant %v, want the row's 3",
			store.tenantList)
	}
	for key, want := range map[string]string{
		"kind": "bug", "summary": "Save fails", "reporter_name": "Ben",
		"reporter_email": "ben@example.com", "reporter_ref": "42",
		"workspace": "tenant 3",
		"context":   `{"page_url":"/exampleList"}`,
	} {
		if house.formByKey[key] != want {
			t.Errorf("create field %s = %q, want %q", key,
				house.formByKey[key], want)
		}
	}
	if len(house.fileNames) != 1 || house.fileNames[0] != "shot.png" {
		t.Errorf("create files = %v, want [shot.png]", house.fileNames)
	}
	if store.messageByID[1] != model.IssueRelaySent {
		t.Errorf("message 1 = %q, want sent", store.messageByID[1])
	}
	if store.messageByID[2] != model.IssueRelaySent {
		t.Errorf("a withdraw answered 409 is done, got %q",
			store.messageByID[2])
	}
	if !strings.HasPrefix(store.messageByID[3], "failed|") ||
		!strings.Contains(store.messageByID[3], "422") {
		t.Errorf("a refused message is recorded failed, got %q",
			store.messageByID[3])
	}
	if store.statusByID[7] != "fixing|Being fixed" {
		t.Errorf("status not applied: %v", store.statusByID)
	}
	wantReplyList := []string{
		"M-2|Sam|On it|F-1:fix.png",
		"M-3|Support|No name",
	}
	if strings.Join(store.replyList, ";") !=
		strings.Join(wantReplyList, ";") {
		t.Errorf("only the staff replies are stored: %v",
			store.replyList)
	}
	if len(house.callList) != 5 {
		t.Errorf("calls = %v, want create, 3 messages, sync",
			house.callList)
	}
}

func TestRelayTick_FailedSendIsRecorded(t *testing.T) {
	_, server := newFakeHouse(t, "right")
	store := newFakeStore()
	store.reportList = []*repository.IssueRelayReport{{
		IssueReportModel: model.IssueReportModel{IssueReportID: 3,
			Kind: "bug", Summary: "x"}}}
	relay := &Relay{
		Client: NewClient(Config{BaseURL: server.URL, ProjectID: "P1",
			Token: "wrong"}, nil),
		Store: store,
	}
	relay.Tick(context.Background())
	if !strings.Contains(store.failedByID[3], "401") {
		t.Errorf("want a recorded 401, got %q", store.failedByID[3])
	}
	if _, sent := store.sentByID[3]; sent {
		t.Error("a failed send must not be marked sent")
	}
}

func TestRelay_WorkspaceLabelHook(t *testing.T) {
	house, server := newFakeHouse(t, "token")
	store := newFakeStore()
	store.reportList = []*repository.IssueRelayReport{{
		IssueReportModel: model.IssueReportModel{IssueReportID: 1,
			SecurityContext: 5, Kind: "feature", Summary: "Export"}}}
	relay := &Relay{
		Client: NewClient(Config{BaseURL: server.URL, ProjectID: "P1",
			Token: "token"}, nil),
		Store:          store,
		WorkspaceLabel: func(int) string { return "Northwind" },
	}
	relay.Tick(context.Background())
	if house.formByKey["workspace"] != "Northwind" {
		t.Errorf("workspace = %q", house.formByKey["workspace"])
	}
	if _, sent := house.formByKey["context"]; sent {
		t.Error("an empty context is left out, not sent blank")
	}
}

func TestClientDownload(t *testing.T) {
	_, server := newFakeHouse(t, "token")
	client := NewClient(Config{BaseURL: server.URL, ProjectID: "P1",
		Token: "token"}, nil)
	content, err := client.Download(context.Background(), "R-1", "F-1",
		10)
	if err != nil || string(content) != "png" {
		t.Fatalf("Download = %q, %v", content, err)
	}
	if _, err := client.Download(context.Background(), "R-1", "F-1",
		2); err == nil {
		t.Error("a file over maxBytes must be an error")
	}
	if _, err := client.Download(context.Background(), "R-1", "F-9",
		10); err == nil {
		t.Error("a 404 must be an error")
	}
}

func TestConfigEnabled(t *testing.T) {
	if (Config{BaseURL: "u", ProjectID: "p"}).Enabled() {
		t.Error("a missing token must leave the relay off")
	}
	if !(Config{BaseURL: "u", ProjectID: "p", Token: "t"}).Enabled() {
		t.Error("all three set turns it on")
	}
}

func TestParseTime(t *testing.T) {
	if got := parseTime("2026-10-01T12:00:00Z", 1); got != 1790856000 {
		t.Errorf("parseTime = %v", got)
	}
	if got := parseTime("2026-10-01T12:00:00.5", 1); got != 1790856000.5 {
		t.Errorf("parseTime without a zone = %v", got)
	}
	if got := parseTime("yesterday", 9); got != 9 {
		t.Errorf("an unreadable time falls back, got %v", got)
	}
}

// A report filed while the relay was off is queued when the relay starts
// and sent by its first tick, once; the backlog is queued only at start.
func TestRelayRun_SendsTheBacklogAtStart(t *testing.T) {
	_, server := newFakeHouse(t, "token")
	store := newFakeStore()
	store.localList = []*repository.IssueRelayReport{{
		IssueReportModel: model.IssueReportModel{IssueReportID: 9,
			SecurityContext: 3, Kind: "bug", Summary: "Save fails",
			ReporterName: "Ben"},
		FileList: []repository.IssueRelayFile{{Filename: "shot.png",
			ContentType: "image/png", Content: []byte("png")}},
	}}
	relay := &Relay{
		Client: NewClient(Config{BaseURL: server.URL, ProjectID: "P1",
			Token: "token"}, nil),
		Store: store,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		relay.Run(ctx, time.Hour)
		close(done)
	}()
	deadline := time.After(5 * time.Second)
	for {
		store.mutex.Lock()
		sent := store.sentByID[9]
		store.mutex.Unlock()
		if sent != "" {
			break
		}
		select {
		case <-deadline:
			t.Fatal("the backlog report was not sent after start")
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	<-done
	if store.sentByID[9] != "R-1|Under review" {
		t.Errorf("sent = %q", store.sentByID[9])
	}
	if store.queueCount != 1 {
		t.Errorf("backlog queued %d times, want once at start",
			store.queueCount)
	}
}

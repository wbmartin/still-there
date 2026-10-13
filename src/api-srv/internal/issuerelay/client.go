// Package issuerelay sends the app's in-app issue reports to an external
// issue tracker and pulls back their status and the tracker's replies
// (doc/150-issue-report-module.md). It is configured only by the API's
// config file (ISSUE_RELAY_URL, ISSUE_RELAY_PROJECT_ID, ISSUE_RELAY_TOKEN),
// so the repository names no tracker URL or token. With any of the three
// unset the relay is off and reports stay in the app.
//
// The tracker's report API, under
// {URL}/api/projects/{PROJECT_ID}/external/reports, with the token in the
// X-Report-Token header:
//
//	POST   ""                          file a report (multipart)
//	GET    "?ids=a,b"                  status and thread of those reports
//	POST   "/{id}/messages"            the reporter adds to the thread
//	POST   "/{id}/withdraw", "/reopen" the reporter's two actions
//	GET    "/{id}/attachments/{aid}"   download one file of the report
package issuerelay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"
)

// Config is the relay's target. The relay is on only when all three are
// set; config.LoadConfig refuses a file that sets some but not all.
type Config struct {
	BaseURL   string
	ProjectID string
	Token     string
}

// Enabled reports whether all three values are set.
func (c Config) Enabled() bool {
	return c.BaseURL != "" && c.ProjectID != "" && c.Token != ""
}

// clientTimeout bounds one call to the tracker, uploads included.
const clientTimeout = 60 * time.Second

// maxResponseBytes bounds a JSON answer from the tracker.
const maxResponseBytes = 32 << 20

// maxErrorBodyBytes is how much of a refusal's body an error keeps.
const maxErrorBodyBytes = 500

// Client talks to the tracker's report API.
type Client struct {
	config     Config
	httpClient *http.Client
}

// NewClient returns a client for config. httpClient nil means a default
// one with a 60-second timeout.
func NewClient(config Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: clientTimeout}
	}
	config.BaseURL = strings.TrimRight(config.BaseURL, "/")
	return &Client{config: config, httpClient: httpClient}
}

// RemoteAttachment, RemoteMessage and RemoteReport are the tracker's
// report JSON, the fields the relay reads.
type RemoteAttachment struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	Size        int    `json:"size"`
	ContentType string `json:"content_type"`
}

type RemoteMessage struct {
	ID             string             `json:"id"`
	AuthorKind     string             `json:"author_kind"`
	AuthorName     *string            `json:"author_name"`
	Body           string             `json:"body"`
	CreatedAt      string             `json:"created_at"`
	AttachmentList []RemoteAttachment `json:"attachments"`
}

type RemoteReport struct {
	ID         string          `json:"id"`
	Kind       string          `json:"kind"`
	State      string          `json:"state"`
	Status     string          `json:"status"`
	StatusNote *string         `json:"status_note"`
	ThreadList []RemoteMessage `json:"thread"`
}

// File is an attachment to upload.
type File struct {
	Filename    string
	ContentType string
	Content     []byte
}

// NewReport is what the relay sends for a new report.
type NewReport struct {
	Kind          string
	Summary       string
	Detail        string
	ReporterName  string
	ReporterEmail string
	ReporterRef   string
	Workspace     string
	ContextJSON   string
	FileList      []File
}

// StatusError is a non-2xx answer from the tracker.
type StatusError struct {
	Code int
	Body string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("tracker answered %d: %s", e.Code, e.Body)
}

func (c *Client) reportURL(pathList ...string) string {
	target := c.config.BaseURL + "/api/projects/" +
		url.PathEscape(c.config.ProjectID) + "/external/reports"
	for _, part := range pathList {
		target += "/" + url.PathEscape(part)
	}
	return target
}

func (c *Client) newRequest(
	ctx context.Context, method string, target string, body io.Reader,
) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Report-Token", c.config.Token)
	return request, nil
}

// send makes one call and decodes a 2xx JSON answer into out.
func (c *Client) send(
	ctx context.Context,
	method string,
	target string,
	contentType string,
	body io.Reader,
	out any,
) error {
	request, err := c.newRequest(ctx, method, target, body)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		text, _ := io.ReadAll(io.LimitReader(response.Body,
			maxErrorBodyBytes))
		return &StatusError{
			Code: response.StatusCode,
			Body: strings.TrimSpace(string(text)),
		}
	}
	return json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).
		Decode(out)
}

// multipartBody writes the non-empty fields and the files, each as a
// "files" part.
func multipartBody(
	fieldList [][2]string, fileList []File,
) (*bytes.Buffer, string, error) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	for _, field := range fieldList {
		if field[1] == "" {
			continue
		}
		if err := writer.WriteField(field[0], field[1]); err != nil {
			return nil, "", err
		}
	}
	for _, file := range fileList {
		part, err := writer.CreatePart(filePartHeader(file))
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write(file.Content); err != nil {
			return nil, "", err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return &buffer, writer.FormDataContentType(), nil
}

// Create files a new report and returns the tracker's view of it.
func (c *Client) Create(
	ctx context.Context, report NewReport,
) (*RemoteReport, error) {
	body, contentType, err := multipartBody([][2]string{
		{"kind", report.Kind},
		{"summary", report.Summary},
		{"detail", report.Detail},
		{"reporter_name", report.ReporterName},
		{"reporter_email", report.ReporterEmail},
		{"reporter_ref", report.ReporterRef},
		{"workspace", report.Workspace},
		{"context", report.ContextJSON},
	}, report.FileList)
	if err != nil {
		return nil, err
	}
	var remote RemoteReport
	err = c.send(ctx, http.MethodPost, c.reportURL(), contentType, body,
		&remote)
	if err != nil {
		return nil, err
	}
	return &remote, nil
}

// Message adds a reporter message, with its files, to a report.
func (c *Client) Message(
	ctx context.Context,
	remoteID string,
	body string,
	authorName string,
	authorRef string,
	fileList []File,
) (*RemoteReport, error) {
	payload, contentType, err := multipartBody([][2]string{
		{"body", body},
		{"author_name", authorName},
		{"author_ref", authorRef},
	}, fileList)
	if err != nil {
		return nil, err
	}
	var remote RemoteReport
	err = c.send(ctx, http.MethodPost, c.reportURL(remoteID, "messages"),
		contentType, payload, &remote)
	if err != nil {
		return nil, err
	}
	return &remote, nil
}

// Action withdraws or reopens a report (action "withdraw" or "reopen").
func (c *Client) Action(
	ctx context.Context, remoteID string, action string, note string,
) (*RemoteReport, error) {
	payload, err := json.Marshal(map[string]string{"note": note})
	if err != nil {
		return nil, err
	}
	var remote RemoteReport
	err = c.send(ctx, http.MethodPost, c.reportURL(remoteID, action),
		"application/json", bytes.NewReader(payload), &remote)
	if err != nil {
		return nil, err
	}
	return &remote, nil
}

// RetrieveList returns the tracker's view of the given reports; an id
// the tracker does not know is absent from the answer.
func (c *Client) RetrieveList(
	ctx context.Context, remoteIDList []string,
) ([]RemoteReport, error) {
	query := url.Values{"ids": {strings.Join(remoteIDList, ",")}}
	var remoteList []RemoteReport
	err := c.send(ctx, http.MethodGet, c.reportURL()+"?"+query.Encode(),
		"", nil, &remoteList)
	if err != nil {
		return nil, err
	}
	return remoteList, nil
}

// Download reads one of a report's files from the tracker, at most
// maxBytes of it; more is an error.
func (c *Client) Download(
	ctx context.Context,
	remoteReportID string,
	remoteAttachmentID string,
	maxBytes int64,
) ([]byte, error) {
	request, err := c.newRequest(ctx, http.MethodGet,
		c.reportURL(remoteReportID, "attachments", remoteAttachmentID),
		nil)
	if err != nil {
		return nil, err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, &StatusError{Code: response.StatusCode}
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maxBytes {
		return nil, fmt.Errorf("tracker file is over %d bytes", maxBytes)
	}
	return content, nil
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

// filePartHeader names a file part with its real type. CreateFormFile would
// label every file application/octet-stream, so the tracker could not tell
// a screenshot from any other file.
func filePartHeader(file File) textproto.MIMEHeader {
	contentType := file.ContentType
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(file.Content)
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(
		`form-data; name="files"; filename="%s"`,
		quoteEscaper.Replace(file.Filename)))
	header.Set("Content-Type", contentType)
	return header
}

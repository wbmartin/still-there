package model

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
)

// The issue-report module (migration 0006_issue_report.sql,
// doc/150-issue-report-module.md).

// Report kinds.
const (
	IssueKindBug     = "bug"
	IssueKindPolish  = "polish"
	IssueKindFeature = "feature"
)

// ValidIssueKind reports whether kind is a known report kind.
func ValidIssueKind(kind string) bool {
	return kind == IssueKindBug || kind == IssueKindPolish ||
		kind == IssueKindFeature
}

// Report states: the tracker's issue states. The app sets only open (on
// create), closed (withdraw) and reopened (reopen); the relay copies the
// rest from the tracker.
const (
	IssueStateOpen     = "open"
	IssueStateClosed   = "closed"
	IssueStateReopened = "reopened"
	IssueStateWontFix  = "wont_fix"
)

// IssueStateIsClosed is true for closed and won't fix: the states a
// reporter may reopen, and may not withdraw.
func IssueStateIsClosed(state string) bool {
	return state == IssueStateClosed || state == IssueStateWontFix
}

// Relay states of a report or message.
const (
	IssueRelayPending = "pending" // waiting to be sent
	IssueRelaySent    = "sent"
	IssueRelayFailed  = "failed" // the last try failed; retried
	IssueRelayLocal   = "local"  // no relay configured when it was filed
	IssueRelayNone    = "none"   // a staff reply: nothing to send
)

// Message authors and reporter actions.
const (
	IssueAuthorReporter = "reporter"
	IssueAuthorStaff    = "staff"

	IssueActionWithdraw = "withdraw"
	IssueActionReopen   = "reopen"

	// The opening line of an action's message; the reporter's note, if
	// any, follows after a blank line. The relay sends the tracker only
	// the note: the tracker writes its own line for the action.
	IssueWithdrawText = "Withdrew this report."
	IssueReopenText   = "Reopened this report."
)

// IssueActionNote is the reporter's note in an action message's body.
func IssueActionNote(body string) string {
	for _, prefix := range []string{IssueWithdrawText, IssueReopenText} {
		if strings.HasPrefix(body, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(body, prefix))
		}
	}
	return strings.TrimSpace(body)
}

// Limits. Files: the tracker's own limits (5 files, 10 MB each). The
// request cap keeps a whole report under nginx's client_max_body_size
// (20M); doc/105 Request Body Limits.
const (
	IssueMaxSummaryLen      = 500
	IssueMaxDetailLen       = 20000
	IssueMaxPageURLLen      = 1000
	IssueMaxContextBytes    = 8 << 10
	IssueMaxFileCount       = 5
	IssueMaxFileBytes       = 10 << 20
	IssueMaxRequestBytes    = 18 << 20
	IssueMaxFilenameLen     = 255
	IssueMaxNoteLen         = 2000
	IssueMaxActionBodyBytes = 16 << 10
	IssueMaxListedReports   = 500
	IssueMaxNameLen         = 200
)

// IssueReportModel is one report as listed and shown. The relay columns
// the web has no use for stay off the wire.
type IssueReportModel struct {
	IssueReportID   int             `json:"issueReportId"`
	SecurityContext int             `json:"securityContext"`
	Kind            string          `json:"kind"`
	Summary         string          `json:"summary"`
	Detail          *string         `json:"detail,omitempty"`
	PageURL         *string         `json:"pageUrl,omitempty"`
	ContextJSONB    json.RawMessage `json:"-"`
	State           string          `json:"state"`
	Status          string          `json:"status"`
	StatusNote      *string         `json:"statusNote,omitempty"`
	ReporterName    string          `json:"reporterName"`
	ReporterEmail   string          `json:"-"`
	RemoteID        *string         `json:"-"`
	RelayState      string          `json:"relayState"`
	RelayError      *string         `json:"-"`
	RelayAttempts   int             `json:"-"`
	LastSyncedTS    *float64        `json:"lastSyncedTs,omitempty"`
	LastActivityTS  float64         `json:"lastActivityTs"`
	// Reporter and staff messages on the report (computed, not a column).
	MessageCount int     `json:"messageCount"`
	CreateTS     float64 `json:"createTs"`
	CreateUserID int     `json:"createUserId"`
	UpdateTS     float64 `json:"updateTs"`
	UpdateUserID int     `json:"updateUserId"`
}

// IssueReportMessageModel is one entry in a report's conversation.
type IssueReportMessageModel struct {
	IssueReportMessageID int                          `json:"issueReportMessageId"`
	IssueReportID        int                          `json:"issueReportId"`
	SecurityContext      int                          `json:"-"`
	AuthorKind           string                       `json:"authorKind"`
	AuthorName           *string                      `json:"authorName,omitempty"`
	Body                 string                       `json:"body"`
	Action               *string                      `json:"action,omitempty"`
	RelayState           string                       `json:"relayState"`
	CreateTS             float64                      `json:"createTs"`
	CreateUserID         int                          `json:"-"`
	AttachmentList       []IssueReportAttachmentModel `json:"attachmentList"`
}

// IssueReportAttachmentModel is a file's metadata; the bytes are served by
// GET /issueReportAttachment.
type IssueReportAttachmentModel struct {
	IssueReportAttachmentID int     `json:"issueReportAttachmentId"`
	IssueReportID           int     `json:"issueReportId"`
	IssueReportMessageID    *int    `json:"issueReportMessageId,omitempty"`
	Filename                string  `json:"filename"`
	ContentType             string  `json:"contentType"`
	SizeBytes               int     `json:"sizeBytes"`
	RemoteID                *string `json:"-"`
	CreateTS                float64 `json:"createTs"`
}

// IssueReportDetailModel is a report with its files and conversation.
type IssueReportDetailModel struct {
	IssueReportModel
	AttachmentList []IssueReportAttachmentModel `json:"attachmentList"`
	ThreadList     []IssueReportMessageModel    `json:"threadList"`
}

// IssueReportActionRequest is the body of POST /issueReportAction.
type IssueReportActionRequest struct {
	IssueReportID int    `json:"issueReportId"`
	Action        string `json:"action"`
	Note          string `json:"note"`
}

// IssueReportFile is an uploaded file the handler has checked, waiting to
// be stored. ContentType is the type read from the bytes.
type IssueReportFile struct {
	Filename    string
	ContentType string
	Content     []byte
}

// Types a stored file may have, read from its bytes. Images may be shown
// inline (none runs script); everything else is a download.
const (
	issueTypePNG  = "image/png"
	issueTypeJPEG = "image/jpeg"
	issueTypeGIF  = "image/gif"
	issueTypeWebP = "image/webp"
	issueTypePDF  = "application/pdf"
	issueTypeText = "text/plain"
	issueTypeCSV  = "text/csv"
	issueTypeHTML = "text/html"
)

// issueExtensionFamily is the allowed file extensions (the tracker refuses
// any other) and the family of byte types each may hold.
var issueExtensionFamily = map[string]string{
	".png": "image", ".jpg": "image", ".jpeg": "image", ".gif": "image",
	".webp": "image", ".pdf": "pdf", ".txt": "text", ".log": "text",
	".csv": "text", ".html": "html", ".htm": "html",
}

// IssueFileExtensionAllowed reports whether name has an extension that may
// be attached to a report of kind. An HTML page (a mock-up of the change)
// may go on a polish or feature report, never on a bug.
func IssueFileExtensionAllowed(name string, kind string) bool {
	family, ok := issueExtensionFamily[strings.ToLower(filepath.Ext(name))]
	if !ok {
		return false
	}
	return family != "html" || kind != IssueKindBug
}

// IssueFileIsHTML reports whether name has an HTML extension.
func IssueFileIsHTML(name string) bool {
	return issueExtensionFamily[strings.ToLower(filepath.Ext(name))] ==
		"html"
}

// sniffIssueFile is the type the bytes hold, from the allowlist above, or
// "" for anything else. http.DetectContentType is the sniffer; its answer
// is mapped through the list, never used as it comes.
func sniffIssueFile(content []byte) string {
	detected := http.DetectContentType(content)
	mediaType, _, _ := strings.Cut(detected, ";")
	switch mediaType {
	case issueTypePNG, issueTypeJPEG, issueTypeGIF, issueTypeWebP,
		issueTypePDF, issueTypeText, issueTypeHTML:
		return mediaType
	}
	return ""
}

// IssueFileType is the type to store for an upload named name: read from
// the bytes, and only when they are what the extension promises (an image
// extension holds an image, .pdf a PDF, .txt/.log/.csv plain text, .html
// HTML or plain text). ok is false otherwise: the handler answers 415.
func IssueFileType(name string, content []byte) (string, bool) {
	sniffed := sniffIssueFile(content)
	extension := strings.ToLower(filepath.Ext(name))
	switch issueExtensionFamily[extension] {
	case "image":
		if strings.HasPrefix(sniffed, "image/") {
			return sniffed, true
		}
	case "pdf":
		if sniffed == issueTypePDF {
			return sniffed, true
		}
	case "text":
		if sniffed == issueTypeText {
			if extension == ".csv" {
				return issueTypeCSV, true
			}
			return issueTypeText, true
		}
	case "html":
		if sniffed == issueTypeHTML || sniffed == issueTypeText {
			return issueTypeHTML, true
		}
	}
	return "", false
}

// IssueFileServeType is how a stored file goes out: its type sniffed again
// from the bytes (never a stored column), and inline only for an image.
// Plain text is a download with its type; anything else, HTML included,
// is a download as application/octet-stream.
func IssueFileServeType(content []byte) (contentType string, inline bool) {
	switch sniffed := sniffIssueFile(content); sniffed {
	case issueTypePNG, issueTypeJPEG, issueTypeGIF, issueTypeWebP:
		return sniffed, true
	case issueTypePDF:
		return sniffed, false
	case issueTypeText:
		return "text/plain; charset=utf-8", false
	}
	return "application/octet-stream", false
}

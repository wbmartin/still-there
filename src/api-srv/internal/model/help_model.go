package model

import "regexp"

// The help module's usage events (doc/151-help-module.md): what the web
// app's Help panel and tours report so an admin can see where the help
// falls short. Usage data is best-effort and anonymous in the insights
// view: counts per entry or per search text, never per person.

// The help_event.event values.
const (
	// A search in the Help panel that found nothing; query keeps the text.
	HelpEventSearchNoResult = "search_no_result"
	// A help entry (glossary term or topic) opened; entry_id names it.
	HelpEventEntryView = "entry_view"
	// A page tour finished or skipped; entry_id is the tour's id.
	HelpEventTourComplete = "tour_complete"
	HelpEventTourSkip     = "tour_skip"
)

// Bounds on what POST /helpEvent stores.
const (
	// The longest search text kept; a longer one is cut, not refused.
	HelpMaxQueryRunes = 200
	// Each user's events per rolling day; past it a call is answered 204
	// and dropped, so a runaway client cannot fill the table.
	HelpMaxEventsPerUserPerDay = 500
	// The request body: a few short fields.
	HelpMaxEventBodyBytes = 4 << 10
	// GET /helpInsights: the window in days (default, most) and how many
	// rows each list returns.
	HelpInsightsDefaultDays = 30
	HelpInsightsMaxDays     = 365
	HelpInsightsListLimit   = 25
)

// The ids the web app's help content uses (help/content/*.ts) and the
// routes it is shown on (constants/routes.ts): checked so the table holds
// what the panel sends and nothing a client makes up beyond that shape.
var (
	helpEntryIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	helpRoutePattern   = regexp.MustCompile(`^/[A-Za-z0-9/_-]{0,63}$`)
)

// IsHelpEvent reports whether event is one of the four the table takes.
func IsHelpEvent(event string) bool {
	switch event {
	case HelpEventSearchNoResult, HelpEventEntryView, HelpEventTourComplete,
		HelpEventTourSkip:
		return true
	}
	return false
}

// IsHelpEntryID reports whether id has the shape of a help entry or tour
// id: lower-case letters, digits and hyphens, at most 64 characters.
func IsHelpEntryID(id string) bool {
	return helpEntryIDPattern.MatchString(id)
}

// IsHelpRoute reports whether route has the shape of a web app route: a
// leading slash, then letters, digits, '/', '_' or '-', at most 64
// characters. No query string or fragment.
func IsHelpRoute(route string) bool {
	return helpRoutePattern.MatchString(route)
}

// HelpEventRequest is the body of POST /helpEvent. entryId is required
// for every event but search_no_result, which takes query instead.
type HelpEventRequest struct {
	Event   string `json:"event"`
	EntryID string `json:"entryId"`
	Route   string `json:"route"`
	Query   string `json:"query"`
}

// HelpEventModel is one help_event row.
type HelpEventModel struct {
	HelpEventID     int
	SecurityContext int
	AppUserID       int
	Event           string
	EntryID         *string
	Route           string
	Query           *string
	CreateTS        float64
	CreateUserID    int
	UpdateTS        float64
	UpdateUserID    int
}

// HelpInsightCount is one key (an entry id, tour id or search text) and
// how many events carried it.
type HelpInsightCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// HelpInsightsView is GET /helpInsights: the tenant's help usage over the
// last Days days, each list most frequent first.
type HelpInsightsView struct {
	Days               int                `json:"days"`
	NoResultSearchList []HelpInsightCount `json:"noResultSearchList"`
	EntryViewList      []HelpInsightCount `json:"entryViewList"`
	TourCompleteList   []HelpInsightCount `json:"tourCompleteList"`
	TourSkipList       []HelpInsightCount `json:"tourSkipList"`
}

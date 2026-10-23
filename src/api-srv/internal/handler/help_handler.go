package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"stillthere-api/internal/app"
	"stillthere-api/internal/model"
	"stillthere-api/internal/security"
)

// The help module (doc/151-help-module.md). The help content itself lives
// in the web app (src/web-app/help/content/) and needs no route; the
// server keeps only usage: POST /helpEvent records what the Help panel
// and page tours report, GET /helpInsights counts it for an admin. The
// module is off unless HELP_ENABLED=true: each route then answers 404
// after its security checks and touches nothing.

// HelpEventRepositoryInterface is what HelpHandler needs from the
// repository; *repository.HelpEventRepository satisfies it.
type HelpEventRepositoryInterface interface {
	CreateHelpEvent(
		ctx context.Context, event *model.HelpEventModel,
		securityContext int,
	) (int, error)
	RetrieveHelpEventCountSince(
		ctx context.Context, appUserID int, since float64,
		securityContext int,
	) (int, error)
	RetrieveHelpInsightCountList(
		ctx context.Context, event string, since float64, limit int,
		securityContext int,
	) ([]model.HelpInsightCount, error)
}

// HelpHandler serves the help module's routes.
type HelpHandler struct {
	appsrv              *app.Services
	helpEventRepository HelpEventRepositoryInterface
}

// NewHelpHandler creates the handler.
func NewHelpHandler(
	appServices *app.Services,
	helpEventRepository HelpEventRepositoryInterface,
) *HelpHandler {
	return &HelpHandler{
		appsrv:              appServices,
		helpEventRepository: helpEventRepository,
	}
}

// helpEventRecorded is POST /helpEvent's answer: false when the caller's
// daily cap dropped the event, so the client is never told it was kept.
type helpEventRecorded struct {
	Recorded bool `json:"recorded"`
}

// moduleOn answers 404 when the module is off. Called after the checks,
// so an off module still refuses a caller the checks refuse.
func (h *HelpHandler) moduleOn(
	appHelper *security.AppHelper, w http.ResponseWriter,
) bool {
	if h.appsrv.Config.HelpEnabled {
		return true
	}
	appHelper.WriteError(w, http.StatusNotFound,
		"Help is not enabled", "HELP_MODULE_OFF", nil)
	return false
}

// CreateHelpEvent handles POST /helpEvent {event, entryId, route, query}:
// one usage event from the caller, in the caller's session tenant. Anyone
// signed in. Only search_no_result keeps what was typed (cut to 200
// characters); the other events keep the entry or tour id. Past the
// per-user daily cap the event is dropped and the answer says so.
func (h *HelpHandler) CreateHelpEvent(
	w http.ResponseWriter, r *http.Request,
) {
	// Security checks
	appHelper := security.NewAppHelper(w, r, h.appsrv, "CreateHelpEvent")
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

	var req model.HelpEventRequest
	if !decodeJSONBody(
		appHelper, w, r, model.HelpMaxEventBodyBytes, &req,
	) {
		return
	}

	// Validate
	req.Query = strings.TrimSpace(req.Query)
	switch {
	case !model.IsHelpEvent(req.Event):
		appHelper.WriteError(w, http.StatusBadRequest,
			"event is not a help event", "BAD_REQUEST", nil)
		return
	case !model.IsHelpRoute(req.Route):
		appHelper.WriteError(w, http.StatusBadRequest,
			"route must be an app route of at most 64 characters",
			"BAD_REQUEST", nil)
		return
	case req.Event == model.HelpEventSearchNoResult && req.Query == "":
		appHelper.WriteError(w, http.StatusBadRequest,
			"query is required for search_no_result", "BAD_REQUEST", nil)
		return
	case req.Event != model.HelpEventSearchNoResult &&
		!model.IsHelpEntryID(req.EntryID):
		appHelper.WriteError(w, http.StatusBadRequest,
			"entryId must be a help id: lower-case letters, digits "+
				"and hyphens, at most 64 characters",
			"BAD_REQUEST", nil)
		return
	}

	jwtClaims := appHelper.GetClaims()
	now := model.Now()
	countToday, err := h.helpEventRepository.RetrieveHelpEventCountSince(
		r.Context(), jwtClaims.UserID, now-86400,
		jwtClaims.SecurityContext,
	)
	if err != nil {
		appHelper.WriteError(w, http.StatusInternalServerError,
			"Failed to record help usage", "HELP_EVENT_FAIL", err)
		return
	}
	if countToday >= model.HelpMaxEventsPerUserPerDay {
		appHelper.WriteJSON(w, http.StatusOK,
			model.DataWrapper{Data: helpEventRecorded{Recorded: false}})
		return
	}

	event := &model.HelpEventModel{
		AppUserID:    jwtClaims.UserID,
		Event:        req.Event,
		Route:        req.Route,
		CreateTS:     now,
		CreateUserID: jwtClaims.UserID,
		UpdateTS:     now,
		UpdateUserID: jwtClaims.UserID,
	}
	if req.Event == model.HelpEventSearchNoResult {
		query := truncateHelpRunes(req.Query, model.HelpMaxQueryRunes)
		event.Query = &query
	} else {
		entryID := req.EntryID
		event.EntryID = &entryID
	}
	if _, err := h.helpEventRepository.CreateHelpEvent(
		r.Context(), event, jwtClaims.SecurityContext,
	); err != nil {
		appHelper.WriteError(w, http.StatusInternalServerError,
			"Failed to record help usage", "HELP_EVENT_FAIL", err)
		return
	}
	appHelper.WriteJSON(w, http.StatusOK,
		model.DataWrapper{Data: helpEventRecorded{Recorded: true}})
}

// RetrieveHelpInsights handles GET /helpInsights?days=N (default 30, at
// most 365): the session tenant's help usage, counted by search text,
// entry and tour, never by person. Searches are what users typed and may
// hold anything, so it takes the user-admin read grant (READ_USER).
func (h *HelpHandler) RetrieveHelpInsights(
	w http.ResponseWriter, r *http.Request,
) {
	// Security checks
	appHelper := security.NewAppHelper(w, r, h.appsrv,
		"RetrieveHelpInsights")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantUserRead)

	if !appHelper.CheckAllChecksComplete() {
		return
	}
	if !h.moduleOn(appHelper, w) {
		return
	}

	days := model.HelpInsightsDefaultDays
	if raw := r.URL.Query().Get("days"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > model.HelpInsightsMaxDays {
			appHelper.WriteError(w, http.StatusBadRequest,
				"days must be a whole number from 1 to 365",
				"BAD_REQUEST", nil)
			return
		}
		days = parsed
	}

	securityContext := appHelper.GetClaims().SecurityContext
	since := model.Now() - float64(days)*86400
	view := model.HelpInsightsView{Days: days}
	for _, part := range []struct {
		event string
		into  *[]model.HelpInsightCount
	}{
		{model.HelpEventSearchNoResult, &view.NoResultSearchList},
		{model.HelpEventEntryView, &view.EntryViewList},
		{model.HelpEventTourComplete, &view.TourCompleteList},
		{model.HelpEventTourSkip, &view.TourSkipList},
	} {
		countList, err := h.helpEventRepository.RetrieveHelpInsightCountList(
			r.Context(), part.event, since, model.HelpInsightsListLimit,
			securityContext,
		)
		if err != nil {
			appHelper.WriteError(w, http.StatusInternalServerError,
				"Failed to read help insights", "HELP_INSIGHTS_FAIL", err)
			return
		}
		*part.into = countList
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: view})
}

// truncateHelpRunes keeps the first max runes of text.
func truncateHelpRunes(text string, max int) string {
	if utf8.RuneCountInString(text) <= max {
		return text
	}
	return string([]rune(text)[:max])
}

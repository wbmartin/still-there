package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
	"github.com/wbmartin/starter-api/internal/security"
)

// The profile a user may change about themselves. User name, groups and MFA
// flags stay admin-only through PUT /user.
type ProfileUpdateRequest struct {
	FirstName       string          `json:"firstName"`
	LastName        string          `json:"lastName"`
	Email           string          `json:"email"`
	PreferenceJSONB json.RawMessage `json:"preferenceJsonb,omitempty"`
	UpdateTS        float64         `json:"updateTs"`
}

// Bound on the preferences document; it is a handful of UI settings.
const maxPreferenceBytes = 8 << 10

// ProfileRepositoryInterface is the slice of the user repository /me needs.
type ProfileRepositoryInterface interface {
	RetrieveUserByID(
		ctx context.Context, userID int, securityContext int,
	) (*model.AppUserModel, error)
	UpdateOwnProfile(
		ctx context.Context, userID int, securityContext int,
		firstName string, lastName string, email string,
		preference json.RawMessage, expectedUpdateTS float64,
	) (float64, error)
}

type ProfileHandler struct {
	appsrv   *app.Services
	userRepo ProfileRepositoryInterface
}

func NewProfileHandler(
	appServices *app.Services, userRepo ProfileRepositoryInterface,
) *ProfileHandler {
	return &ProfileHandler{appsrv: appServices, userRepo: userRepo}
}

// RetrieveOwnProfile handles GET /me. The row is the caller's, by JWT claim.
func (h *ProfileHandler) RetrieveOwnProfile(
	w http.ResponseWriter, r *http.Request,
) {
	// Security checks
	app := security.NewAppHelper(w, r, h.appsrv, "RetrieveOwnProfile")
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantLogon)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("ME_RD_START", "RetrieveOwnProfile invoked")
	jwtClaims := app.GetClaims()

	user, err := h.userRepo.RetrieveUserByID(
		r.Context(), jwtClaims.UserID, jwtClaims.SecurityContext,
	)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			app.WriteError(w, http.StatusNotFound, "User not found",
				"NOT_FOUND", nil)
			return
		}
		app.WriteError(w, http.StatusInternalServerError,
			"Failed to retrieve profile", "ME_RD_FAIL", err)
		return
	}

	app.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: user})
	app.Info("ME_RD_DONE", "RetrieveOwnProfile completed")
}

// UpdateOwnProfile handles PUT /me: first/last name, email and the
// preferences document, with optimistic locking on updateTs.
func (h *ProfileHandler) UpdateOwnProfile(
	w http.ResponseWriter, r *http.Request,
) {
	// Security checks
	app := security.NewAppHelper(w, r, h.appsrv, "UpdateOwnProfile")
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantLogon)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("ME_UP_START", "UpdateOwnProfile invoked")
	jwtClaims := app.GetClaims()

	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodyBytes+maxPreferenceBytes)
	var req ProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body",
			"BAD_REQUEST", err)
		return
	}

	// Validate
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Email = strings.TrimSpace(req.Email)
	switch {
	case req.FirstName == "" || req.LastName == "":
		app.WriteError(w, http.StatusBadRequest,
			"firstName and lastName are required", "BAD_REQUEST", nil)
		return
	case len(req.FirstName) > 200 || len(req.LastName) > 200 || len(req.Email) > 200:
		app.WriteError(w, http.StatusBadRequest,
			"firstName, lastName and email are at most 200 characters",
			"BAD_REQUEST", nil)
		return
	case !isPlainEmailAddress(req.Email):
		app.WriteError(w, http.StatusBadRequest, "email is not valid",
			"BAD_REQUEST", nil)
		return
	case req.UpdateTS == 0:
		app.WriteError(w, http.StatusBadRequest, "updateTs is required",
			"BAD_REQUEST", nil)
		return
	case !isPreferenceDocument(req.PreferenceJSONB):
		app.WriteError(w, http.StatusBadRequest,
			"preferenceJsonb must be a JSON object of at most 8 KiB",
			"BAD_REQUEST", nil)
		return
	}

	_, err := h.userRepo.UpdateOwnProfile(
		r.Context(), jwtClaims.UserID, jwtClaims.SecurityContext,
		req.FirstName, req.LastName, req.Email, req.PreferenceJSONB,
		req.UpdateTS,
	)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			app.WriteError(w, http.StatusConflict,
				"Profile was modified by another request; reload and retry",
				"CONFLICT", nil)
		case errors.Is(err, repository.ErrNotFound):
			app.WriteError(w, http.StatusNotFound, "User not found",
				"NOT_FOUND", nil)
		default:
			app.WriteError(w, http.StatusInternalServerError,
				"Failed to update profile", "ME_UP_FAIL", err)
		}
		return
	}

	// Return the row as stored so the client's next PUT carries the lock.
	user, err := h.userRepo.RetrieveUserByID(
		r.Context(), jwtClaims.UserID, jwtClaims.SecurityContext,
	)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError,
			"Profile updated but could not be re-read", "ME_UP_FAIL", err)
		return
	}
	app.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: user})
	app.Info("ME_UP_DONE", "UpdateOwnProfile completed")
}

// isPlainEmailAddress accepts "user@host" forms only, not "Name <user@host>".
func isPlainEmailAddress(email string) bool {
	if email == "" {
		return false
	}
	parsed, err := mail.ParseAddress(email)
	return err == nil && parsed.Address == email
}

// isPreferenceDocument accepts absent, or a JSON object within the size bound.
func isPreferenceDocument(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	if len(raw) > maxPreferenceBytes || !json.Valid(raw) {
		return false
	}
	return bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{"))
}

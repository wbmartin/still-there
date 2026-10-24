package handler

import (
	"context"
	"fmt"
	"net/http"

	"stillthere-api/internal/app"
	"stillthere-api/internal/model"
	"stillthere-api/internal/security"
)

// The people texted about a device (contact). An admin adds them; each
// starts invited and becomes active only by replying Y to the invite text
// the watchdog sends. contactStatus and the invite / opt timestamps are
// never taken from a client.

// ContactRepositoryInterface is what ContactHandler needs from
// repository.ContactRepository.
type ContactRepositoryInterface interface {
	RetrieveContactList(
		ctx context.Context, idList []int, deviceID int, securityContext int,
		afterUpdateTs float64,
	) ([]*model.ContactModel, error)
	CreateContact(
		ctx context.Context, contact *model.ContactModel, securityContext int,
	) (*model.ContactModel, error)
	UpdateContact(
		ctx context.Context, contact *model.ContactModel,
		originalUpdateTS float64,
	) (*model.ContactModel, error)
	DeleteContact(
		ctx context.Context, contactID int, securityContext int,
		originalUpdateTS float64, deleteTS float64, deleteUserID int,
	) error
}

// ContactHandler serves the contact routes.
type ContactHandler struct {
	appsrv            *app.Services
	contactRepository ContactRepositoryInterface
}

// NewContactHandler creates the contact handler.
func NewContactHandler(
	appServices *app.Services, contactRepository ContactRepositoryInterface,
) *ContactHandler {
	return &ContactHandler{
		appsrv: appServices, contactRepository: contactRepository,
	}
}

// RetrieveContactList handles GET
// /contactList?deviceId=<id>&contactIdList=1,2&afterUpdateTs=<ts>; each
// filter is optional.
func (h *ContactHandler) RetrieveContactList(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "RetrieveContactList")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantContactRead)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	idList, afterUpdateTs, ok := listQueryOf(appHelper, w, r, "contactIdList")
	if !ok {
		return
	}
	deviceID, err := GetOptionalIntFromQuery(r, "deviceId")
	if err != nil {
		appHelper.WriteError(w, http.StatusBadRequest, err.Error(),
			"PARSE_QUERY_FAIL", nil)
		return
	}
	contactList, err := h.contactRepository.RetrieveContactList(r.Context(),
		idList, deviceID, appHelper.GetClaims().SecurityContext,
		afterUpdateTs)
	if err != nil {
		appHelper.WriteError(w, http.StatusInternalServerError,
			"Failed to retrieve contacts", "CON_LIST_FAIL", err)
		return
	}
	appHelper.WriteJSON(w, http.StatusOK,
		model.DataWrapper{Data: contactList})
}

// CreateContact handles POST /contact {deviceId, contactName, phoneE164,
// notifyEventList}: an invited contact; notifyEventList defaults to every
// event. The device must be a live one of the caller's tenant (404).
func (h *ContactHandler) CreateContact(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "CreateContact")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantContactCreate)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	jwtClaims := appHelper.GetClaims()
	contact, ok := decodeContact(appHelper, w, r)
	if !ok {
		return
	}
	if contact.DeviceID <= 0 {
		appHelper.WriteError(w, http.StatusBadRequest,
			"deviceId is required", "BAD_REQUEST", nil)
		return
	}
	if contact.NotifyEventList == nil {
		contact.NotifyEventList = append([]string{},
			model.ContactNotifyEventList...)
	}
	now := model.Now()
	contact.CreateTS, contact.CreateUserID = now, jwtClaims.UserID
	contact.UpdateTS, contact.UpdateUserID = now, jwtClaims.UserID
	created, err := h.contactRepository.CreateContact(
		r.Context(), contact, jwtClaims.SecurityContext)
	if err != nil {
		mapRepoError(appHelper, w, err, "Failed to create contact",
			"CON_CREATE_FAIL")
		return
	}
	appHelper.WriteJSON(w, http.StatusCreated,
		model.DataWrapper{Data: created})
}

// UpdateContact handles PUT /contact: name, number and notifyEventList
// (absent keeps the stored list), under the optimistic lock. A contact
// who has replied keeps their number (409); an invited contact's new
// number is invited afresh.
func (h *ContactHandler) UpdateContact(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "UpdateContact")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantContactUpdate)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	jwtClaims := appHelper.GetClaims()
	contact, ok := decodeContact(appHelper, w, r)
	if !ok {
		return
	}
	if contact.ContactID <= 0 {
		appHelper.WriteError(w, http.StatusBadRequest,
			"contactId is required", "BAD_REQUEST", nil)
		return
	}
	originalUpdateTS := contact.UpdateTS
	contact.SecurityContext = jwtClaims.SecurityContext
	contact.UpdateTS, contact.UpdateUserID = model.Now(), jwtClaims.UserID
	updated, err := h.contactRepository.UpdateContact(
		r.Context(), contact, originalUpdateTS)
	if err != nil {
		mapRepoError(appHelper, w, err,
			fmt.Sprintf("Failed to update contact: contactId=%d",
				contact.ContactID),
			"CON_UPDATE_FAIL")
		return
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: updated})
}

// DeleteContact handles DELETE /contact?contactId=<id>&updateTs=<ts>.
func (h *ContactHandler) DeleteContact(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "DeleteContact")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantContactDelete)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	contactID, updateTS, ok := deleteQueryOf(appHelper, w, r, "contactId")
	if !ok {
		return
	}
	jwtClaims := appHelper.GetClaims()
	err := h.contactRepository.DeleteContact(r.Context(), contactID,
		jwtClaims.SecurityContext, updateTS, model.Now(), jwtClaims.UserID)
	if err != nil {
		mapRepoError(appHelper, w, err,
			fmt.Sprintf("Failed to delete contact: contactId=%d", contactID),
			"CON_DELETE_FAIL")
		return
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{
		Data: map[string]any{"contactId": contactID},
	})
}

// decodeContact reads and checks a contact body.
func decodeContact(
	appHelper *security.AppHelper, w http.ResponseWriter, r *http.Request,
) (*model.ContactModel, bool) {
	var contact model.ContactModel
	if !decodeJSONBody(appHelper, w, r, maxDeviceBodyBytes, &contact) {
		return nil, false
	}
	if problem := model.ContactProblem(&contact); problem != "" {
		appHelper.WriteError(w, http.StatusBadRequest, problem,
			"BAD_REQUEST", nil)
		return nil, false
	}
	return &contact, true
}

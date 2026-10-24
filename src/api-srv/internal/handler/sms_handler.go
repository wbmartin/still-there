package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"stillthere-api/internal/app"
	"stillthere-api/internal/model"
	"stillthere-api/internal/security"
	"stillthere-api/internal/sms"
)

// InboundTextHandler is the slice of *watchdog.Inbound the routes use.
type InboundTextHandler interface {
	Handle(ctx context.Context, from string, body string) (string, error)
	HandleInTenant(ctx context.Context, from string, body string, securityContext int) (string, error)
}

// SMSHandler takes texts people send us: the Telnyx webhook when
// SMS_PROVIDER=telnyx, a simulated text from an admin when it is log.
// Each route answers 404 under the other provider.
type SMSHandler struct {
	appsrv  *app.Services
	inbound InboundTextHandler
}

func NewSMSHandler(appServices *app.Services, inbound InboundTextHandler) *SMSHandler {
	return &SMSHandler{appsrv: appServices, inbound: inbound}
}

// smsWebhookMaxBodyBytes is far above a message.received event (~2 KiB).
const smsWebhookMaxBodyBytes int64 = 64 << 10

// SMSInbound handles POST /smsInbound, Telnyx's webhook. A message.received
// event is acted on; every other event (delivery reports) is acknowledged
// and dropped.
func (h *SMSHandler) SMSInbound(w http.ResponseWriter, r *http.Request) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "SMSInbound")
	// The Telnyx signature is the credential. Each exception is stated here:
	// no JWT: Telnyx has no login
	appHelper.SkipJWT()
	// the per-address bucket (MAX_UNAUTH_REQ_PER_MIN)
	appHelper.CheckRateLimit()
	// the block list is keyed by JWT subject, and there is none
	appHelper.SkipUserBlock()
	// no cookie authorizes the call, so there is nothing to forge
	appHelper.SkipCSRF()
	// the Ed25519 signature over the body is the authorization
	appHelper.SkipAuthorization()
	if !appHelper.CheckAllChecksComplete() {
		return
	}
	cfg := h.appsrv.Config
	if cfg.SMSProvider != "telnyx" {
		appHelper.WriteError(w, http.StatusNotFound, "Not found", "SMS_INBOUND_OFF", nil)
		return
	}
	body, ok := readBody(appHelper, w, r, smsWebhookMaxBodyBytes)
	if !ok {
		return
	}
	err := sms.VerifyTelnyxWebhook(cfg.TelnyxPublicKey,
		r.Header.Get("telnyx-signature-ed25519"), r.Header.Get("telnyx-timestamp"),
		body, time.Now())
	if err != nil {
		appHelper.WriteError(w, http.StatusUnauthorized, "Invalid signature", "SMS_INBOUND_SIG", err)
		return
	}
	event, err := sms.ParseTelnyxWebhook(body)
	if err != nil {
		appHelper.WriteError(w, http.StatusBadRequest, "Invalid request body", "INV_REQ_BODY", err)
		return
	}
	if event.EventType == "message.received" && sms.E164.MatchString(event.From) {
		if _, err := h.inbound.Handle(r.Context(), event.From, event.Text); err != nil {
			// 500 makes Telnyx retry; every keyword is safe to repeat.
			appHelper.WriteError(w, http.StatusInternalServerError,
				"Text could not be handled", "SMS_INBOUND_FAIL", err)
			return
		}
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: map[string]string{}})
}

// SMSSimulateRequest is a text as if fromPhone had sent it.
type SMSSimulateRequest struct {
	FromPhone string `json:"fromPhone"`
	Body      string `json:"body"`
}

// SMSSimulateInbound handles POST /smsSimulateInbound {fromPhone, body}
// while SMS_PROVIDER=log, so the reply flow can be tried before Telnyx
// can reach the server. It acts only on the caller's tenant's contacts and
// answers the reply that would have been texted.
func (h *SMSHandler) SMSSimulateInbound(w http.ResponseWriter, r *http.Request) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "SMSSimulateInbound")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantAlertUpdate)
	if !appHelper.CheckAllChecksComplete() {
		return
	}
	if h.appsrv.Config.SMSProvider != "log" {
		appHelper.WriteError(w, http.StatusNotFound, "Not found", "SMS_SIMULATE_OFF", nil)
		return
	}
	var req SMSSimulateRequest
	if !decodeJSONBody(appHelper, w, r, smsWebhookMaxBodyBytes, &req) {
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if !sms.E164.MatchString(req.FromPhone) || req.Body == "" {
		appHelper.WriteError(w, http.StatusBadRequest,
			"fromPhone (E.164) and body are required", "BAD_REQUEST",
			errors.New("invalid simulate request"))
		return
	}
	reply, err := h.inbound.HandleInTenant(r.Context(), req.FromPhone, req.Body,
		appHelper.GetClaims().SecurityContext)
	if err != nil {
		appHelper.WriteError(w, http.StatusInternalServerError,
			"Text could not be handled", "SMS_SIMULATE_FAIL", err)
		return
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: map[string]string{"reply": reply}})
	appHelper.Info("SMS_SIMULATE_DONE", "simulated text from "+sms.MaskPhone(req.FromPhone))
}

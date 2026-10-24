package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// defaultQueue is the queue of POST /telnyx-inbound with none in the path,
// the URL Still There registered in Telnyx before the relay existed.
const defaultQueue = "stillthere"

// maxBodyBytes caps every request body; a message.received event is
// about 2 KiB.
const maxBodyBytes int64 = 64 << 10

const (
	defaultLimit = 10
	maxLimit     = 100
	minLeaseSec  = 5
	maxLeaseSec  = 600
)

// Server holds the routes. The webhook is authenticated by its Telnyx
// signature, the consumer API by a bearer token per queue. Neither has a
// rate limit: nginx exposes only the webhook, and the consumer API listens
// on loopback.
type Server struct {
	Store     *Store
	PublicKey ed25519.PublicKey
	TokenList []ConsumerToken
	Log       *slog.Logger
	// Now is the clock (tests set it); nil is the wall clock.
	Now func() time.Time
}

func (s *Server) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func epochOf(t time.Time) float64 {
	return float64(t.UnixNano()) / 1e9
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /telnyx-inbound", s.telnyxInbound)
	mux.HandleFunc("POST /telnyx-inbound/{queue}", s.telnyxInbound)
	mux.HandleFunc("GET /queue/{queue}/peek", s.peek)
	mux.HandleFunc("POST /queue/{queue}/receive", s.receive)
	mux.HandleFunc("POST /queue/{queue}/ack", s.ack)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.Store.db.Ping(ctx); err != nil {
		s.Log.Error("health: database unreachable", "err", err)
		writeError(w, http.StatusServiceUnavailable, "database unreachable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// queueOf is the path's queue, or defaultQueue for the bare webhook URL;
// ok is false for a name that is not a queue name.
func queueOf(r *http.Request) (string, bool) {
	queue := r.PathValue("queue")
	if queue == "" && r.URL.Path == "/telnyx-inbound" {
		return defaultQueue, true
	}
	return queue, queueNamePattern.MatchString(queue)
}

// telnyxInbound handles POST /telnyx-inbound[/{queue}]: a signed
// message.received event is stored for the queue; any other signed event
// (delivery reports) is answered and dropped. A 5xx makes Telnyx retry.
func (s *Server) telnyxInbound(w http.ResponseWriter, r *http.Request) {
	queue, ok := queueOf(r)
	if !ok {
		writeError(w, http.StatusNotFound, "no such queue")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "body too large")
		return
	}
	if err := verifyTelnyxWebhook(s.PublicKey,
		r.Header.Get("telnyx-signature-ed25519"),
		r.Header.Get("telnyx-timestamp"), body, s.now(),
	); err != nil {
		s.Log.Info("webhook: bad signature", "queue", queue)
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}
	event, err := parseTelnyxWebhook(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if event.EventType == "message.received" {
		id, err := s.Store.Insert(r.Context(), queue, event, body, epochOf(s.now()))
		if err != nil {
			s.Log.Error("webhook: storing text", "queue", queue, "err", err)
			writeError(w, http.StatusInternalServerError, "not stored")
			return
		}
		s.Log.Info("webhook: text stored", "queue", queue, "id", id)
	}
	writeJSON(w, http.StatusOK, map[string]string{})
}

// authorized checks the bearer token against the queue's tokens. Every
// entry is compared, matching or not, so the time taken says nothing
// about which queue or token came close.
func (s *Server) authorized(r *http.Request, queue string) bool {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || token == "" {
		return false
	}
	hash := sha256.Sum256([]byte(token))
	match := 0
	for _, entry := range s.TokenList {
		equal := subtle.ConstantTimeCompare(hash[:], entry.Hash[:])
		if entry.Queue == queue {
			match |= equal
		}
	}
	return match == 1
}

// consumerQueue is the path's queue when the caller may read it; it has
// answered 404 or 401 when ok is false.
func (s *Server) consumerQueue(w http.ResponseWriter, r *http.Request) (string, bool) {
	queue, ok := queueOf(r)
	if !ok {
		writeError(w, http.StatusNotFound, "no such queue")
		return "", false
	}
	if !s.authorized(r, queue) {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return "", false
	}
	return queue, true
}

func limitOf(limit int) (int, bool) {
	if limit == 0 {
		return defaultLimit, true
	}
	return limit, limit >= 1 && limit <= maxLimit
}

func decodeBody(w http.ResponseWriter, r *http.Request, into any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return false
	}
	return true
}

// peek handles GET /queue/{queue}/peek?limit=N.
func (s *Server) peek(w http.ResponseWriter, r *http.Request) {
	queue, ok := s.consumerQueue(w, r)
	if !ok {
		return
	}
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		var err error
		if limit, err = strconv.Atoi(raw); err != nil || limit == 0 {
			limit = -1
		}
	}
	limit, ok = limitOf(limit)
	if !ok {
		writeError(w, http.StatusBadRequest, "limit must be 1 to 100")
		return
	}
	messageList, err := s.Store.Peek(r.Context(), queue, limit, epochOf(s.now()))
	if err != nil {
		s.Log.Error("peek", "queue", queue, "err", err)
		writeError(w, http.StatusInternalServerError, "peek failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messageList": messageList})
}

type receiveRequest struct {
	Limit    int `json:"limit"`
	LeaseSec int `json:"leaseSec"`
}

// receive handles POST /queue/{queue}/receive {limit, leaseSec}.
func (s *Server) receive(w http.ResponseWriter, r *http.Request) {
	queue, ok := s.consumerQueue(w, r)
	if !ok {
		return
	}
	var req receiveRequest
	if !decodeBody(w, r, &req) {
		return
	}
	limit, ok := limitOf(req.Limit)
	if !ok {
		writeError(w, http.StatusBadRequest, "limit must be 1 to 100")
		return
	}
	if req.LeaseSec < minLeaseSec || req.LeaseSec > maxLeaseSec {
		writeError(w, http.StatusBadRequest, "leaseSec must be 5 to 600")
		return
	}
	leaseID, messageList, err := s.Store.Receive(r.Context(), queue, limit,
		req.LeaseSec, epochOf(s.now()))
	if err != nil {
		s.Log.Error("receive", "queue", queue, "err", err)
		writeError(w, http.StatusInternalServerError, "receive failed")
		return
	}
	if len(messageList) > 0 {
		s.Log.Info("receive: leased", "queue", queue, "count", len(messageList))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"leaseId": leaseID, "messageList": messageList,
	})
}

type ackRequest struct {
	LeaseID string  `json:"leaseId"`
	IDList  []int64 `json:"idList"`
}

// ack handles POST /queue/{queue}/ack {leaseId, idList}.
func (s *Server) ack(w http.ResponseWriter, r *http.Request) {
	queue, ok := s.consumerQueue(w, r)
	if !ok {
		return
	}
	var req ackRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.LeaseID == "" || len(req.IDList) == 0 || len(req.IDList) > maxLimit {
		writeError(w, http.StatusBadRequest,
			"leaseId and an idList of 1 to 100 ids are required")
		return
	}
	count, err := s.Store.Ack(r.Context(), queue, req.LeaseID, req.IDList,
		epochOf(s.now()))
	if err != nil {
		s.Log.Error("ack", "queue", queue, "err", err)
		writeError(w, http.StatusInternalServerError, "ack failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"ackCount": count})
}

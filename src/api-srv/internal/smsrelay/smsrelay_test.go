package smsrelay

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// relayStub plays the relay's consumer API for queue stillthere.
type relayStub struct {
	messageList  []Message
	receiveBody  map[string]int
	ackedLease   string
	ackedList    []int64
	receiveCalls int
	ackCalls     int
}

func (s *relayStub) server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	auth := func(w http.ResponseWriter, r *http.Request) bool {
		if r.Header.Get("Authorization") != "Bearer token-1" {
			w.WriteHeader(http.StatusUnauthorized)
			return false
		}
		return true
	}
	mux.HandleFunc("POST /queue/stillthere/receive", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		s.receiveCalls++
		json.NewDecoder(r.Body).Decode(&s.receiveBody)
		json.NewEncoder(w).Encode(map[string]any{"leaseId": "lease-1", "messageList": s.messageList})
	})
	mux.HandleFunc("POST /queue/stillthere/ack", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		s.ackCalls++
		var body struct {
			LeaseID string  `json:"leaseId"`
			IDList  []int64 `json:"idList"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		s.ackedLease, s.ackedList = body.LeaseID, body.IDList
		json.NewEncoder(w).Encode(map[string]int{"ackCount": len(body.IDList)})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

type fakeHandler struct {
	failID      int64
	handledList []int64
}

func (h *fakeHandler) HandleRelayed(_ context.Context, id int64, _ string, _ string) error {
	if id == h.failID {
		return errors.New("database down")
	}
	h.handledList = append(h.handledList, id)
	return nil
}

type recordLog struct{ codeList []string }

func (l *recordLog) Info(code string, _ string, _ ...any)  { l.codeList = append(l.codeList, code) }
func (l *recordLog) Error(code string, _ string, _ ...any) { l.codeList = append(l.codeList, code) }

func newPoller(t *testing.T, stub *relayStub, token string, handler Handler) (*Poller, *recordLog) {
	log := &recordLog{}
	return &Poller{
		Client:  &Client{BaseURL: stub.server(t).URL + "/", Token: token, Queue: "stillthere"},
		Handler: handler, Log: log,
	}, log
}

func TestPollAcksOnlyWhatWasHandled(t *testing.T) {
	stub := &relayStub{messageList: []Message{
		{ID: 1, From: "+15550000001", Text: "OK"},
		{ID: 2, From: "+15550000002", Text: "Y"},
		{ID: 3, From: "not-a-number", Text: "x"},
	}}
	handler := &fakeHandler{failID: 2}
	poller, log := newPoller(t, stub, "token-1", handler)
	poller.Poll(context.Background())

	if stub.receiveBody["limit"] != 25 || stub.receiveBody["leaseSec"] != 60 {
		t.Fatalf("receive asked %v, want limit 25, leaseSec 60", stub.receiveBody)
	}
	if !slices.Equal(handler.handledList, []int64{1}) {
		t.Fatalf("handled %v, want [1]", handler.handledList)
	}
	// 2 failed: not acked, it returns after the lease. 3 has no E.164
	// sender: dropped, so acked without being handled.
	if stub.ackedLease != "lease-1" || !slices.Equal(stub.ackedList, []int64{1, 3}) {
		t.Fatalf("acked %v under %q, want [1 3] under lease-1", stub.ackedList, stub.ackedLease)
	}
	if !slices.Contains(log.codeList, "SMS_RELAY_HANDLE") {
		t.Fatalf("the failed text was not logged: %v", log.codeList)
	}
}

func TestPollWithNothingDoesNotAck(t *testing.T) {
	stub := &relayStub{}
	poller, _ := newPoller(t, stub, "token-1", &fakeHandler{})
	poller.Poll(context.Background())
	if stub.receiveCalls != 1 || stub.ackCalls != 0 {
		t.Fatalf("receive %d, ack %d calls; want 1, 0", stub.receiveCalls, stub.ackCalls)
	}
}

func TestPollRefusedTokenHandlesNothing(t *testing.T) {
	stub := &relayStub{messageList: []Message{{ID: 1, From: "+15550000001", Text: "OK"}}}
	handler := &fakeHandler{}
	poller, log := newPoller(t, stub, "wrong", handler)
	poller.Poll(context.Background())
	if len(handler.handledList) != 0 || stub.ackCalls != 0 ||
		!slices.Contains(log.codeList, "SMS_RELAY_RECEIVE") {
		t.Fatalf("handled %v, acks %d, log %v", handler.handledList, stub.ackCalls, log.codeList)
	}
}

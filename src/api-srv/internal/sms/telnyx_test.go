package sms

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestTelnyxSend(t *testing.T) {
	var got map[string]string
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got)
		_, _ = w.Write([]byte(`{"data":{"id":"msg-123"}}`))
	}))
	defer srv.Close()

	s := TelnyxSender{APIKey: "KEY", FromNumber: "+18005550100", MessagingProfileID: "prof", URL: srv.URL}
	id, status, err := s.Send(context.Background(), "+15550000001", "hello")
	if err != nil || id != "msg-123" || status != "sent" {
		t.Fatalf("send: id %q status %q err %v", id, status, err)
	}
	if auth != "Bearer KEY" || got["to"] != "+15550000001" || got["from"] != "+18005550100" ||
		got["text"] != "hello" || got["messaging_profile_id"] != "prof" {
		t.Fatalf("request: auth %q body %v", auth, got)
	}
}

func TestTelnyxSendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"unverified"}]}`))
	}))
	defer srv.Close()
	_, _, err := TelnyxSender{URL: srv.URL}.Send(context.Background(), "+15550000001", "x")
	if err == nil {
		t.Fatal("want an error for HTTP 422")
	}
}

func TestVerifyTelnyxWebhook(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	pubB64 := base64.StdEncoding.EncodeToString(pub)
	now := time.Unix(1_800_000_000, 0)
	ts := strconv.FormatInt(now.Unix(), 10)
	body := []byte(`{"data":{"event_type":"message.received","payload":{"text":"OK","from":{"phone_number":"+15550000001"}}}}`)
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, append([]byte(ts+"|"), body...)))

	if err := VerifyTelnyxWebhook(pubB64, sig, ts, body, now); err != nil {
		t.Fatalf("good signature rejected: %v", err)
	}
	if err := VerifyTelnyxWebhook(pubB64, sig, ts, append(body, ' '), now); err == nil {
		t.Fatal("altered body accepted")
	}
	if err := VerifyTelnyxWebhook(pubB64, sig, ts, body, now.Add(10*time.Minute)); err == nil {
		t.Fatal("stale timestamp accepted")
	}
	in, err := ParseTelnyxWebhook(body)
	if err != nil || in.EventType != "message.received" || in.From != "+15550000001" || in.Text != "OK" {
		t.Fatalf("parse: %+v %v", in, err)
	}
}

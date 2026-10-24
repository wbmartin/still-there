package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The parts that need no database: signatures, queue names, the token
// list, the env file, and the routes' refusals before any SQL.

var testNow = time.Unix(1800000000, 0)

func newTestKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return publicKey, privateKey
}

// sign returns the two Telnyx headers for body at ts.
func sign(privateKey ed25519.PrivateKey, body []byte, ts time.Time) (string, string) {
	timestamp := strconv.FormatInt(ts.Unix(), 10)
	signed := append([]byte(timestamp+"|"), body...)
	return base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, signed)), timestamp
}

func TestVerifyTelnyxWebhook(t *testing.T) {
	publicKey, privateKey := newTestKey(t)
	body := []byte(`{"data":{"event_type":"message.received"}}`)
	signature, timestamp := sign(privateKey, body, testNow)
	if err := verifyTelnyxWebhook(publicKey, signature, timestamp, body, testNow.Add(time.Minute)); err != nil {
		t.Fatalf("a good signature was refused: %v", err)
	}
	_, otherKey := newTestKey(t)
	otherSignature, _ := sign(otherKey, body, testNow)
	for name, c := range map[string]struct {
		signature, timestamp string
		body                 []byte
		now                  time.Time
	}{
		"tampered body":  {signature, timestamp, []byte(`{"data":{}}`), testNow},
		"other key":      {otherSignature, timestamp, body, testNow},
		"stale":          {signature, timestamp, body, testNow.Add(6 * time.Minute)},
		"future":         {signature, timestamp, body, testNow.Add(-6 * time.Minute)},
		"bad timestamp":  {signature, "soon", body, testNow},
		"not base64":     {"!!", timestamp, body, testNow},
		"empty headers":  {"", "", body, testNow},
		"other ts value": {signature, strconv.FormatInt(testNow.Unix()+1, 10), body, testNow},
	} {
		if err := verifyTelnyxWebhook(publicKey, c.signature, c.timestamp, c.body, c.now); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestParseTelnyxWebhook(t *testing.T) {
	in, err := parseTelnyxWebhook([]byte(`{"data":{"event_type":"message.received",
		"payload":{"text":"OK","from":{"phone_number":"+15550000001"},
		"to":[{"phone_number":"+18005550100"}]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	want := telnyxInbound{EventType: "message.received", From: "+15550000001", To: "+18005550100", Text: "OK"}
	if in != want {
		t.Fatalf("got %+v, want %+v", in, want)
	}
	if _, err := parseTelnyxWebhook([]byte(`not json`)); err == nil {
		t.Fatal("parsed a body that is not JSON")
	}
}

func TestQueueNamePattern(t *testing.T) {
	for name, valid := range map[string]bool{
		"stillthere": true, "a": true, "gsd-notes2": true,
		strings.Repeat("a", 32): true, strings.Repeat("a", 33): false,
		"": false, "Still": false, "2app": false, "-app": false,
		"app_x": false, "app/x": false, "app.x": false,
	} {
		if queueNamePattern.MatchString(name) != valid {
			t.Errorf("queue name %q: want valid=%v", name, valid)
		}
	}
}

func hashHex(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func TestParseConsumerTokenList(t *testing.T) {
	tokenList, err := parseConsumerTokenList(
		"stillthere:" + hashHex("a") + ", gsd:" + hashHex("b"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tokenList) != 2 || tokenList[0].Queue != "stillthere" || tokenList[1].Queue != "gsd" ||
		tokenList[1].Hash != sha256.Sum256([]byte("b")) {
		t.Fatalf("parsed %+v", tokenList)
	}
	for _, bad := range []string{
		"", "stillthere", "stillthere:", "Still:" + hashHex("a"),
		"stillthere:" + hashHex("a")[:63], "stillthere:" + strings.Repeat("z", 64),
		"stillthere:" + hashHex("a") + ",",
	} {
		if _, err := parseConsumerTokenList(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func testServer(t *testing.T, store *Store) (*Server, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey := newTestKey(t)
	tokenList, err := parseConsumerTokenList(
		"stillthere:" + hashHex("st-token") + ",gsd:" + hashHex("gsd-token"))
	if err != nil {
		t.Fatal(err)
	}
	return &Server{
		Store: store, PublicKey: publicKey, TokenList: tokenList,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now: func() time.Time { return testNow },
	}, privateKey
}

func TestAuthorizedOnlyForItsQueue(t *testing.T) {
	server, _ := testServer(t, nil)
	for _, c := range []struct {
		header, queue string
		want          bool
	}{
		{"Bearer st-token", "stillthere", true},
		{"Bearer gsd-token", "gsd", true},
		{"Bearer gsd-token", "stillthere", false},
		{"Bearer st-token", "other", false},
		{"Bearer wrong", "stillthere", false},
		{"st-token", "stillthere", false},
		{"Bearer ", "stillthere", false},
		{"", "stillthere", false},
	} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if c.header != "" {
			r.Header.Set("Authorization", c.header)
		}
		if got := server.authorized(r, c.queue); got != c.want {
			t.Errorf("%q on %s: authorized=%v, want %v", c.header, c.queue, got, c.want)
		}
	}
}

// Every refusal here answers before the store is touched (it is nil).
func TestRoutesRefuseBeforeTheDatabase(t *testing.T) {
	server, privateKey := testServer(t, nil)
	handler := server.Handler()
	body := `{"data":{"event_type":"message.received"}}`
	signature, timestamp := sign(privateKey, []byte(body), testNow)
	for _, c := range []struct {
		name, method, path, auth, body string
		signed                         bool
		want                           int
	}{
		{"unsigned webhook", "POST", "/telnyx-inbound", "", body, false, 401},
		{"unsigned queue webhook", "POST", "/telnyx-inbound/gsd", "", body, false, 401},
		{"bad queue name", "POST", "/telnyx-inbound/Bad_Name", "", body, true, 404},
		{"webhook GET", "GET", "/telnyx-inbound", "", "", false, 405},
		{"oversized webhook", "POST", "/telnyx-inbound", "", strings.Repeat("x", 65<<10), true, 413},
		{"signed delivery report", "POST", "/telnyx-inbound", "", "", true, 200},
		{"peek no token", "GET", "/queue/stillthere/peek", "", "", false, 401},
		{"peek other queue's token", "GET", "/queue/stillthere/peek", "Bearer gsd-token", "", false, 401},
		{"receive bad lease", "POST", "/queue/stillthere/receive", "Bearer st-token", `{"limit":5,"leaseSec":4}`, false, 400},
		{"receive long lease", "POST", "/queue/stillthere/receive", "Bearer st-token", `{"limit":5,"leaseSec":601}`, false, 400},
		{"receive big limit", "POST", "/queue/stillthere/receive", "Bearer st-token", `{"limit":101,"leaseSec":60}`, false, 400},
		{"receive unknown field", "POST", "/queue/stillthere/receive", "Bearer st-token", `{"leaseSec":60,"x":1}`, false, 400},
		{"peek bad limit", "GET", "/queue/stillthere/peek?limit=abc", "Bearer st-token", "", false, 400},
		{"ack no ids", "POST", "/queue/stillthere/ack", "Bearer st-token", `{"leaseId":"x","idList":[]}`, false, 400},
		{"ack no lease", "POST", "/queue/stillthere/ack", "Bearer st-token", `{"idList":[1]}`, false, 400},
	} {
		requestBody := c.body
		r := httptest.NewRequest(c.method, c.path, strings.NewReader(requestBody))
		if c.signed {
			// A delivery report signed for its own (empty) body.
			if c.body == "" {
				requestBody = `{"data":{"event_type":"message.finalized"}}`
				r = httptest.NewRequest(c.method, c.path, strings.NewReader(requestBody))
				s, ts := sign(privateKey, []byte(requestBody), testNow)
				r.Header.Set("telnyx-signature-ed25519", s)
				r.Header.Set("telnyx-timestamp", ts)
			} else {
				r.Header.Set("telnyx-signature-ed25519", signature)
				r.Header.Set("telnyx-timestamp", timestamp)
			}
		}
		if c.auth != "" {
			r.Header.Set("Authorization", c.auth)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s: %d %s, want %d", c.name, w.Code, w.Body.String(), c.want)
		}
	}
}

func writeEnv(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "relay.env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfig(t *testing.T) {
	publicKey, _ := newTestKey(t)
	good := map[string]string{
		"DATABASE_URL":        "postgres://u:p@127.0.0.1:5432/prd_smsrelay_db",
		"SERVER_BIND_ADDRESS": "127.0.0.1",
		"SERVER_PORT":         "8095",
		"TELNYX_PUBLIC_KEY":   base64.StdEncoding.EncodeToString(publicKey),
		"CONSUMER_TOKEN_LIST": "stillthere:" + hashHex("a"),
		"RETAIN_DAYS":         "30",
		"LOG_LEVEL":           "info",
	}
	render := func(override map[string]string) string {
		var b strings.Builder
		b.WriteString("# comment\n\n")
		for key, value := range good {
			if v, ok := override[key]; ok {
				value = v
			}
			b.WriteString(key + "=" + value + "\n")
		}
		return b.String()
	}
	config, err := loadConfig(writeEnv(t, render(map[string]string{"LOG_LEVEL": `"debug"`})))
	if err != nil {
		t.Fatal(err)
	}
	if config.ServerPort != 8095 || config.RetainDays != 30 || config.LogLevel != slog.LevelDebug ||
		len(config.ConsumerTokenList) != 1 {
		t.Fatalf("loaded %+v", config)
	}
	for key := range good {
		if _, err := loadConfig(writeEnv(t, render(map[string]string{key: ""}))); err == nil ||
			!strings.Contains(err.Error(), key) {
			t.Errorf("without %s: err %v", key, err)
		}
	}
	for key, bad := range map[string]string{
		"SERVER_PORT": "0", "RETAIN_DAYS": "-1", "LOG_LEVEL": "loud",
		"TELNYX_PUBLIC_KEY": "SERVERONLY", "CONSUMER_TOKEN_LIST": "stillthere:SERVERONLY",
	} {
		if _, err := loadConfig(writeEnv(t, render(map[string]string{key: bad}))); err == nil {
			t.Errorf("%s=%s accepted", key, bad)
		}
	}
	if databaseURL, err := loadDatabaseURL(writeEnv(t, "DATABASE_URL=postgres://x/y\n")); err != nil ||
		databaseURL != "postgres://x/y" {
		t.Fatalf("loadDatabaseURL: %q %v", databaseURL, err)
	}
}

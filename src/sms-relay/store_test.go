package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The SQL against a scratch database of its own, loc_smsrelay_test_db,
// dropped and rebuilt once per run on the gate's Postgres
// (TEST_PG_URL_BASE, etc/loc-test.env). Without that variable these tests
// skip; bin/300-run-tests sets it and fails any skip.

const testDatabaseName = "loc_smsrelay_test_db"

var (
	testDBOnce sync.Once
	testDB     *pgxpool.Pool
	testDBErr  error
)

// checkScratchName refuses any database not named *_test_db.
func checkScratchName(name string) error {
	if !strings.HasSuffix(name, "_test_db") {
		return fmt.Errorf("%q is not a scratch database (*_test_db): refusing", name)
	}
	return nil
}

func TestCheckScratchName(t *testing.T) {
	for name, ok := range map[string]bool{
		testDatabaseName: true, "prd_smsrelay_db": false,
		"loc_smsrelay_test_db_old": false, "": false,
	} {
		if (checkScratchName(name) == nil) != ok {
			t.Errorf("%q: want accepted=%v", name, ok)
		}
	}
}

func buildTestDB(base string) (*pgxpool.Pool, error) {
	if err := checkScratchName(testDatabaseName); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, strings.TrimRight(base, "/")+"/postgres")
	if err != nil {
		return nil, err
	}
	defer admin.Close()
	for _, statement := range []string{
		"DROP DATABASE IF EXISTS " + testDatabaseName + " WITH (FORCE)",
		"CREATE DATABASE " + testDatabaseName,
	} {
		if _, err := admin.Exec(ctx, statement); err != nil {
			return nil, err
		}
	}
	databaseURL := strings.TrimRight(base, "/") + "/" + testDatabaseName
	parsed, err := url.Parse(databaseURL)
	if err != nil || checkScratchName(strings.TrimPrefix(parsed.Path, "/")) != nil {
		return nil, fmt.Errorf("test database URL does not name %s", testDatabaseName)
	}
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if _, err := migrateUp(ctx, db, epochNow()); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// testStore is the scratch database with inbox_message emptied.
func testStore(t *testing.T) *Store {
	t.Helper()
	base := os.Getenv("TEST_PG_URL_BASE")
	if base == "" {
		t.Skip("skipping integration tests: TEST_PG_URL_BASE unset (run bin/300-run-tests go)")
	}
	testDBOnce.Do(func() { testDB, testDBErr = buildTestDB(base) })
	if testDBErr != nil {
		t.Fatalf("building %s: %v", testDatabaseName, testDBErr)
	}
	if _, err := testDB.Exec(context.Background(),
		"TRUNCATE inbox_message RESTART IDENTITY"); err != nil {
		t.Fatal(err)
	}
	return &Store{db: testDB}
}

const t0 = 1800000000.0

func insertN(t *testing.T, store *Store, queue string, n int, now float64) {
	t.Helper()
	for i := range n {
		in := telnyxInbound{EventType: "message.received", From: "+15550000001",
			To: "+18005550100", Text: fmt.Sprintf("text %d", i)}
		if _, err := store.Insert(context.Background(), queue, in, []byte(`{}`), now); err != nil {
			t.Fatal(err)
		}
	}
}

func idsOf(messageList []Message) []int64 {
	var idList []int64
	for _, m := range messageList {
		idList = append(idList, m.ID)
	}
	return idList
}

func TestMigrateIsCurrentAndIdempotent(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	if count, err := migrateUp(ctx, store.db, t0); err != nil || count != 0 {
		t.Fatalf("second migrate: applied %d, err %v", count, err)
	}
	if err := checkSchema(ctx, store.db); err != nil {
		t.Fatal(err)
	}
}

func TestReceiveLeasesAndAckFinishes(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	insertN(t, store, "stillthere", 3, t0)
	insertN(t, store, "gsd", 1, t0)

	peeked, err := store.Peek(ctx, "stillthere", 10, t0)
	if err != nil || len(peeked) != 3 || peeked[0].Text != "text 0" || peeked[0].To != "+18005550100" {
		t.Fatalf("peek: %+v %v", peeked, err)
	}
	if again, _ := store.Peek(ctx, "stillthere", 10, t0); len(again) != 3 {
		t.Fatal("peek changed state")
	}

	leaseID, got, err := store.Receive(ctx, "stillthere", 2, 60, t0)
	if err != nil || len(got) != 2 || got[0].ID != peeked[0].ID || got[1].ID != peeked[1].ID {
		t.Fatalf("receive: %+v %v", got, err)
	}
	if left, _ := store.Peek(ctx, "stillthere", 10, t0+1); len(left) != 1 {
		t.Fatalf("peek under a lease: want 1 left, got %d", len(left))
	}
	if count, _ := store.Ack(ctx, "gsd", leaseID, idsOf(got), t0+1); count != 0 {
		t.Fatal("acked through another queue")
	}
	if count, _ := store.Ack(ctx, "stillthere", "other-lease", idsOf(got), t0+1); count != 0 {
		t.Fatal("acked under another lease id")
	}
	if count, err := store.Ack(ctx, "stillthere", leaseID, idsOf(got[:1]), t0+1); err != nil || count != 1 {
		t.Fatalf("ack: %d %v", count, err)
	}

	// The unacked one returns when its lease expires; the acked one never.
	leaseID2, back, _ := store.Receive(ctx, "stillthere", 10, 60, t0+61)
	if len(back) != 2 || back[0].ID != got[1].ID || back[1].ID != peeked[2].ID {
		t.Fatalf("after expiry: %+v", back)
	}
	if count, _ := store.Ack(ctx, "stillthere", leaseID, idsOf(got[1:]), t0+62); count != 0 {
		t.Fatal("an expired lease's ack took effect")
	}
	if count, _ := store.Ack(ctx, "stillthere", leaseID2, idsOf(back), t0+200); count != 0 {
		t.Fatal("acked after the lease ran out")
	}
	var receiveCount int
	if err := store.db.QueryRow(ctx, `SELECT receive_count FROM inbox_message
		WHERE inbox_message_id = $1`, got[1].ID).Scan(&receiveCount); err != nil || receiveCount != 2 {
		t.Fatalf("receive_count %d %v, want 2", receiveCount, err)
	}
}

func TestRetentionDeletesOnlyOldDone(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	insertN(t, store, "stillthere", 2, t0)
	leaseID, got, _ := store.Receive(ctx, "stillthere", 1, 60, t0)
	store.Ack(ctx, "stillthere", leaseID, idsOf(got), t0+1)
	if count, err := store.DeleteDoneBefore(ctx, t0+1); err != nil || count != 0 {
		t.Fatalf("deleted %d before the cutoff passed (%v)", count, err)
	}
	if count, err := store.DeleteDoneBefore(ctx, t0+86400); err != nil || count != 1 {
		t.Fatalf("deleted %d, want the one done (%v)", count, err)
	}
	if left, _ := store.Peek(ctx, "stillthere", 10, t0+86400); len(left) != 1 {
		t.Fatal("an undone message was deleted")
	}
}

// Concurrent receives each get disjoint rows: SKIP LOCKED passes over the
// rows another transaction holds instead of waiting for or doubling them.
func TestConcurrentReceiveNeverDoubles(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	insertN(t, store, "stillthere", 50, t0)
	var mu sync.Mutex
	seen := map[int64]int{}
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				_, got, err := store.Receive(ctx, "stillthere", 3, 60, t0)
				if err != nil {
					t.Error(err)
					return
				}
				if len(got) == 0 {
					return
				}
				mu.Lock()
				for _, m := range got {
					seen[m.ID]++
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if len(seen) != 50 {
		t.Fatalf("received %d distinct messages, want 50", len(seen))
	}
	for id, n := range seen {
		if n != 1 {
			t.Fatalf("message %d received %d times under live leases", id, n)
		}
	}
}

// The routes end to end: a signed webhook is stored for the URL's queue
// (the bare URL is stillthere), and the consumer API reads it back.
func TestWebhookToConsumer(t *testing.T) {
	store := testStore(t)
	server, privateKey := testServer(t, store)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	post := func(path string, body string, header map[string]string) (int, map[string]any) {
		req, _ := http.NewRequest(http.MethodPost, httpServer.URL+path, strings.NewReader(body))
		for k, v := range header {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		out := map[string]any{}
		json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}
	webhook := func(path, text string) int {
		body := `{"data":{"event_type":"message.received","payload":{"text":"` + text +
			`","from":{"phone_number":"+15550000001"},"to":[{"phone_number":"+18005550100"}]}}}`
		signature, timestamp := sign(privateKey, []byte(body), testNow)
		code, _ := post(path, body, map[string]string{
			"telnyx-signature-ed25519": signature, "telnyx-timestamp": timestamp,
		})
		return code
	}
	if code := webhook("/telnyx-inbound", "OK"); code != 200 {
		t.Fatalf("bare webhook: %d", code)
	}
	if code := webhook("/telnyx-inbound/gsd", "Y"); code != 200 {
		t.Fatalf("queue webhook: %d", code)
	}

	stAuth := map[string]string{"Authorization": "Bearer st-token"}
	code, out := post("/queue/stillthere/receive", `{"limit":10,"leaseSec":60}`, stAuth)
	messageList, _ := out["messageList"].([]any)
	if code != 200 || len(messageList) != 1 {
		t.Fatalf("receive: %d %v", code, out)
	}
	first := messageList[0].(map[string]any)
	if first["text"] != "OK" || first["from"] != "+15550000001" {
		t.Fatalf("message %v", first)
	}
	code, out = post("/queue/stillthere/ack", fmt.Sprintf(`{"leaseId":%q,"idList":[%v]}`,
		out["leaseId"], first["id"]), stAuth)
	if code != 200 || out["ackCount"] != 1.0 {
		t.Fatalf("ack: %d %v", code, out)
	}
	if code, _ := post("/queue/gsd/receive", `{"leaseSec":60}`, stAuth); code != 401 {
		t.Fatalf("stillthere's token read gsd: %d", code)
	}

	req, _ := http.NewRequest(http.MethodGet, httpServer.URL+"/queue/gsd/peek?limit=5", nil)
	req.Header.Set("Authorization", "Bearer gsd-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var peek struct{ MessageList []Message }
	json.NewDecoder(resp.Body).Decode(&peek)
	if resp.StatusCode != 200 || len(peek.MessageList) != 1 || peek.MessageList[0].Text != "Y" {
		t.Fatalf("gsd peek: %d %+v", resp.StatusCode, peek)
	}

	health, err := http.Get(httpServer.URL + "/health")
	if err != nil || health.StatusCode != 200 {
		t.Fatalf("health: %v %v", health, err)
	}
	health.Body.Close()
}

package sqlfiles_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/wbmartin/starter-api/internal/migrate"
	sqlfiles "github.com/wbmartin/starter-api/sql"
)

// export-scrub.sql is what bin/620-export-tenant-data runs on its copy of
// the local database before the dump leaves the laptop. These tests run
// the file as written against a database of their own: migrated from the
// embedded migrations, no fixtures, no tenant lock, so tenant 1 can hold
// rows the way a real export's source does. Tenant 3 stands in for a
// second real tenant; tenant 2 is the test tenant the scrub refuses.

// scrubDatabase creates and migrates <test db>_scrub_test_db beside the
// scratch database TEST_DATABASE_URL names, and drops it after the test.
func scrubDatabase(t *testing.T) *pgx.Conn {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("skipping integration tests: TEST_DATABASE_URL unset " +
			"(run bin/300-run-tests go)")
	}
	parsed, err := url.Parse(base)
	if err != nil {
		t.Skipf("skipping integration tests: bad DATABASE_URL (%v)", err)
	}
	testDatabaseName := strings.TrimPrefix(parsed.Path, "/")
	if !strings.HasSuffix(testDatabaseName, "_test_db") {
		t.Fatalf("refusing to run against %s: not a _test_db", parsed.Path)
	}
	name := strings.TrimSuffix(testDatabaseName, "_test_db") +
		"_scrub_test_db"
	ctx, cancel := context.WithTimeout(context.Background(),
		30*time.Second)
	defer cancel()

	admin := *parsed
	admin.Path = "/postgres"
	adminConn, err := pgx.Connect(ctx, admin.String())
	if err != nil {
		t.Skipf("skipping integration tests: no reachable test database "+
			"(%v)", err)
	}
	_, _ = adminConn.Exec(ctx, "DROP DATABASE IF EXISTS "+name)
	if _, err := adminConn.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	scratch := *parsed
	scratch.Path = "/" + name
	conn, err := pgx.Connect(ctx, scratch.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Close(context.Background())
		_, _ = adminConn.Exec(context.Background(),
			"DROP DATABASE IF EXISTS "+name)
		adminConn.Close(context.Background())
	})

	migrationList, err := migrate.Load(sqlfiles.Migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := migrate.New(conn, migrationList).Up(ctx); err != nil {
		t.Fatal(err)
	}
	return conn
}

func readScrub(t *testing.T) string {
	t.Helper()
	script, err := os.ReadFile("export-scrub.sql")
	if err != nil {
		t.Fatal(err)
	}
	return string(script)
}

// seedUser inserts one login carrying every credential the scrub clears.
func seedUser(t *testing.T, tx pgx.Tx, userName string, tenant int,
	accountStatus string, deleted bool) int {
	t.Helper()
	var appUserID int
	err := tx.QueryRow(context.Background(), `
		INSERT INTO app_user (user_name, pw_hash, first_name, last_name,
		       email, security_context_id, app_user_group_membership_list,
		       mfa_exempt, totp_seed, mfa_setup, account_status,
		       create_ts, create_user_id, update_ts, update_user_id,
		       delete_ts, delete_user_id)
		VALUES ($1, '$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$aGFzaA',
		       'First', 'Last', $5, $2, ARRAY[1], TRUE,
		       'encrypted-seed', TRUE, $3, 1, 0, 1, 0,
		       CASE WHEN $4 THEN 2 END, CASE WHEN $4 THEN 0 END)
		RETURNING app_user_id`,
		userName, tenant, accountStatus, deleted,
		userName+"@example.com").Scan(&appUserID)
	if err != nil {
		t.Fatal(err)
	}
	return appUserID
}

func seedInvite(t *testing.T, tx pgx.Tx, appUserID, tenant int,
	tokenHash string, redeemed bool) {
	t.Helper()
	_, err := tx.Exec(context.Background(), `
		INSERT INTO user_invite (security_context_id, app_user_id,
		       token_hash, expires_ts, redeemed_ts, create_ts,
		       create_user_id, update_ts, update_user_id)
		VALUES ($1, $2, $3, 9999999999, CASE WHEN $4 THEN 5 END,
		       1, 0, 1, 0)`,
		tenant, appUserID, tokenHash, redeemed)
	if err != nil {
		t.Fatal(err)
	}
}

func seedExample(t *testing.T, tx pgx.Tx, text string, tenant int) {
	t.Helper()
	_, err := tx.Exec(context.Background(), `
		INSERT INTO example (example_text, example_uuid,
		       security_context_id, create_ts, create_user_id, update_ts,
		       update_user_id)
		VALUES ($1, gen_random_uuid(), $2, 1, 0, 1, 0)`, text, tenant)
	if err != nil {
		t.Fatal(err)
	}
}

// seedIssueReport files a report with a message and a file, as if the
// relay had sent it to a tracker (remote ids set).
func seedIssueReport(t *testing.T, tx pgx.Tx, tenant int) {
	t.Helper()
	ctx := context.Background()
	var reportID int
	err := tx.QueryRow(ctx, `
		INSERT INTO issue_report (security_context_id, kind, summary,
			state, status, reporter_name, reporter_email, remote_id,
			relay_state, relay_attempts, last_activity_ts, create_ts,
			create_user_id, update_ts, update_user_id)
		VALUES ($1, 'bug', 'laptop report', 'open', 'Under review',
			'Alice', 'alice@example.com', $2, 'sent',
			1, 1, 1, 1, 1, 1)
		RETURNING issue_report_id`, tenant,
		fmt.Sprintf("R-laptop-%d", tenant)).Scan(&reportID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO issue_report_message (security_context_id,
			issue_report_id, author_kind, body, remote_id, relay_state,
			relay_attempts, create_ts, create_user_id, update_ts,
			update_user_id)
		VALUES ($1, $2, 'staff', 'On it', 'M-laptop', 'none', 0, 1, 0, 1,
			0)`, tenant, reportID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO issue_report_attachment (security_context_id,
			issue_report_id, filename, content_type, size_bytes, content,
			create_ts, create_user_id, update_ts, update_user_id)
		VALUES ($1, $2, 'shot.png', 'image/png', 3, 'png', 1, 1, 1, 1)`,
		tenant, reportID)
	if err != nil {
		t.Fatal(err)
	}
}

// seedHelpEvent records a help search with no result, as the laptop's
// Help panel would.
func seedHelpEvent(t *testing.T, tx pgx.Tx, tenant int) {
	t.Helper()
	_, err := tx.Exec(context.Background(), `
		INSERT INTO help_event (security_context_id, app_user_id, event,
			route, query, create_ts, create_user_id, update_ts,
			update_user_id)
		VALUES ($1, 1, 'search_no_result', '/dashboard', 'laptop search',
			1, 1, 1, 1)`, tenant)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExportScrub(t *testing.T) {
	conn := scrubDatabase(t)
	script := readScrub(t)
	ctx := context.Background()

	// Each case seeds and scrubs inside a transaction it rolls back, so
	// the cases share one migrated database and never see each other.
	inTx := func(t *testing.T, body func(tx pgx.Tx)) {
		tx, err := conn.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		body(tx)
	}
	count := func(t *testing.T, tx pgx.Tx, query string) int {
		t.Helper()
		var n int
		if err := tx.QueryRow(ctx, query).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	t.Run("empties every credential and invites live logins",
		func(t *testing.T) {
			inTx(t, func(tx pgx.Tx) {
				aliceID := seedUser(t, tx, "alice", 1, "active", false)
				seedUser(t, tx, "bob", 1, "blocked", false)
				seedUser(t, tx, "carol", 1, "active", true)
				daveID := seedUser(t, tx, "dave", 1, "invited", false)
				seedUser(t, tx, "erin", 3, "active", false)
				seedInvite(t, tx, daveID, 1, "live-hash", false)
				seedInvite(t, tx, aliceID, 1, "redeemed-hash", true)
				seedExample(t, tx, "customer row one", 1)
				seedExample(t, tx, "customer row two", 3)
				seedIssueReport(t, tx, 1)
				seedIssueReport(t, tx, 3)
				if n := count(t, tx, `SELECT count(*)
					FROM issue_report`); n != 2 {
					t.Fatalf("issue reports seeded: %d, want 2", n)
				}
				seedHelpEvent(t, tx, 1)
				seedHelpEvent(t, tx, 3)
				if n := count(t, tx, `SELECT count(*)
					FROM help_event`); n != 2 {
					t.Fatalf("help events seeded: %d, want 2", n)
				}

				if _, err := tx.Exec(ctx, script); err != nil {
					t.Fatalf("scrub refused clean data: %v", err)
				}

				if n := count(t, tx, `SELECT count(*) FROM app_user
					WHERE pw_hash <> '' OR totp_seed <> ''
					   OR mfa_setup OR mfa_exempt`); n != 0 {
					t.Errorf("%d login(s) kept a credential", n)
				}
				rows, err := tx.Query(ctx, `SELECT user_name,
					account_status FROM app_user ORDER BY user_name`)
				if err != nil {
					t.Fatal(err)
				}
				statusByUser, err := pgx.CollectRows(rows,
					func(row pgx.CollectableRow) ([2]string, error) {
						var pair [2]string
						return pair, row.Scan(&pair[0], &pair[1])
					})
				if err != nil {
					t.Fatal(err)
				}
				want := [][2]string{
					{"alice", "invited"}, {"bob", "blocked"},
					{"carol", "active"}, {"dave", "invited"},
					{"erin", "invited"},
				}
				if len(statusByUser) != len(want) {
					t.Fatalf("logins after scrub: %v", statusByUser)
				}
				for i := range want {
					if statusByUser[i] != want[i] {
						t.Errorf("login %s: status %s, want %s",
							want[i][0], statusByUser[i][1], want[i][1])
					}
				}
				liveInviteCount := count(t, tx, `SELECT count(*)
					FROM user_invite
					WHERE redeemed_ts IS NULL AND revoked_ts IS NULL`)
				if liveInviteCount != 0 {
					t.Errorf("%d invite link(s) still live",
						liveInviteCount)
				}
				if n := count(t, tx, `SELECT count(*) FROM user_invite
					WHERE token_hash = 'redeemed-hash'
					  AND revoked_ts IS NULL`); n != 1 {
					t.Error("a redeemed invite was rewritten")
				}
				if n := count(t, tx, `SELECT count(*) FROM example
					WHERE example_text LIKE 'customer row %'`); n != 2 {
					t.Errorf("example rows after scrub: %d, want 2", n)
				}
				// The laptop's issue reports and their tracker ids
				// do not travel.
				// Nor does the laptop's help usage.
				for _, table := range []string{"issue_report",
					"issue_report_message", "issue_report_attachment",
					"help_event"} {
					if n := count(t, tx,
						"SELECT count(*) FROM "+table); n != 0 {
						t.Errorf("%s rows after scrub: %d, want 0",
							table, n)
					}
				}
			})
		})

	t.Run("refuses rows in the test tenant", func(t *testing.T) {
		inTx(t, func(tx pgx.Tx) {
			seedUser(t, tx, "alice", 1, "active", false)
			seedExample(t, tx, "test tenant row", 2)
			_, err := tx.Exec(ctx, script)
			if err == nil || !strings.Contains(err.Error(),
				"rows in tenant 2, the test tenant: example(1)") {
				t.Fatalf("want a tenant-2 refusal naming example(1), "+
					"got %v", err)
			}
		})
	})

	t.Run("refuses live fixture logins", func(t *testing.T) {
		inTx(t, func(tx pgx.Tx) {
			seedUser(t, tx, "admin", 1, "active", false)
			seedUser(t, tx, "switcher", 3, "active", false)
			seedUser(t, tx, "basic", 1, "active", true)
			_, err := tx.Exec(ctx, script)
			if err == nil || !strings.Contains(err.Error(),
				"fixture logins present") ||
				!strings.HasSuffix(err.Error(),
					": admin, switcher (SQLSTATE P0001)") {
				t.Fatalf("want a refusal naming admin and switcher "+
					"(not the deleted basic), got %v", err)
			}
		})
	})
}

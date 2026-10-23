package repository

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The tenants the database-backed tests act in: never 1, which the scratch
// databases refuse (sql/fixtures/test-tenant-lock.sql). test-seed.sql puts
// the fixture logins and groups 1-3 in testTenant and groups 4-5 in
// otherTenant.
const (
	testTenant  = 2
	otherTenant = 3
)

// connectTestDB returns a pool for the integration database, or skips.
//
// The suites in this package are integration tests: they create and delete
// rows in a scratch database built by bin/510-reset-test-db. Where that
// database is comes ONLY from TEST_DATABASE_URL, which
// bin/300-run-tests exports from etc/loc-test.env. There is deliberately no
// fallback to etc/loc-api-srv.env: that is the developer's live database, and
// running a suite that inserts and deletes rows against it is not a test.
//
// A host without the variable skips rather than fails, which keeps
// `go test ./...` meaningful in both places: full coverage where the scratch
// database exists, and the hermetic packages still gating the build where it
// does not. The test runner greps for that skip so it cannot pass silently.
// A URL naming anything but a scratch database fails the test outright.
// pgxpool.New connects lazily, so the reachability check has to be an
// explicit Ping -- without it the failure surfaces later, inside a test, as
// a query error.
func connectTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("skipping integration tests: TEST_DATABASE_URL unset (run bin/300-run-tests go)")
	}
	if err := checkScratch(databaseURL); err != nil {
		t.Fatal(err)
	}
	db, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Skipf("skipping integration tests: bad DATABASE_URL (%v)", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Skipf("skipping integration tests: no reachable test database (%v)", err)
	}

	return db
}

// checkScratch refuses a URL whose database name does not end in _test_db
// or _e2e_db, the names bin/510-reset-test-db builds and locks.
func checkScratch(databaseURL string) error {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("TEST_DATABASE_URL does not parse: refusing to run")
	}
	databaseName := strings.TrimPrefix(parsed.Path, "/")
	if strings.HasSuffix(databaseName, "_test_db") ||
		strings.HasSuffix(databaseName, "_e2e_db") {
		return nil
	}
	return fmt.Errorf("TEST_DATABASE_URL names database %q, not a scratch "+
		"database (*_test_db / *_e2e_db): refusing to run", databaseName)
}

func TestCheckScratch(t *testing.T) {
	for databaseURL, accepted := range map[string]bool{
		"postgres://u:p@127.0.0.1:5432/loc_stillthere_test_db?sslmode=disable": true,
		"postgres://u:p@127.0.0.1:5432/loc_stillthere_e2e_db":                  true,
		"postgres://u:p@127.0.0.1:5432/loc_stillthere_db":                      false,
		"postgres://u:p@127.0.0.1:5432/uat_stillthere_db?sslmode=disable":      false,
		"postgres://u:p@127.0.0.1:5432/loc_stillthere_test_db_old":             false,
		"postgres://u:p@127.0.0.1:5432/":                                  false,
		"://not a url":                                                    false,
	} {
		if got := checkScratch(databaseURL) == nil; got != accepted {
			t.Errorf("checkScratch(%q) accepted=%v, want %v",
				databaseURL, got, accepted)
		}
	}
}

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
// pgxpool.New connects lazily, so the reachability check has to be an
// explicit Ping -- without it the failure surfaces later, inside a test, as
// a query error.
func connectTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("skipping integration tests: TEST_DATABASE_URL unset (run bin/300-run-tests go)")
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

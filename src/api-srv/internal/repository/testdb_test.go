package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wbmartin/starter-api/internal/app_lib"
)

// connectTestDB returns a pool for the integration database, or skips.
//
// The suites in this package are integration tests: they create and delete
// rows in a scratch database built by bin/510-reset-test-db. Where that
// database is comes from PROTO_TEST_DATABASE_URL when set -- bin/200-run-tests
// exports it from etc/loc-test.env -- and otherwise from etc/loc-api-srv.env,
// a DEVELOPER's config file whose DATABASE_URL arrives via the vault.
//
// A host with neither skips rather than fails, which keeps `go test ./...`
// meaningful in both places: full coverage where the database exists, and
// the hermetic packages still gating the build where it does not. The test
// runner greps for that skip so it cannot pass silently. pgxpool.New
// connects lazily, so the reachability check has to be an explicit Ping --
// without it the failure surfaces later, inside a test, as a query error.
func connectTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("PROTO_TEST_DATABASE_URL")
	if databaseURL == "" {
		config, err := app_lib.LoadConfig("../../../../etc/loc-api-srv.env")
		if err != nil {
			t.Skipf("skipping integration tests: no local config (%v)", err)
		}
		databaseURL = config.DatabaseURL
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

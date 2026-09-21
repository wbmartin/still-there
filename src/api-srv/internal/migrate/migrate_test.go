package migrate

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5"

	sqlfiles "github.com/wbmartin/starter-api/sql"
)

func fsWith(names ...string) fstest.MapFS {
	files := fstest.MapFS{}
	for i, name := range names {
		files["migrations/"+name] = &fstest.MapFile{Data: []byte(fmt.Sprintf("-- %d", i))}
	}
	return files
}

func TestLoadSortsByVersionNotName(t *testing.T) {
	list, err := Load(fsWith("0010_ten.sql", "0002_two.sql", "0001_one.sql"))
	if err != nil {
		t.Fatal(err)
	}
	got := []int{}
	for _, m := range list {
		got = append(got, m.Version)
	}
	if fmt.Sprint(got) != "[1 2 10]" {
		t.Errorf("expected [1 2 10], got %v", got)
	}
	if list[2].Name != "ten" {
		t.Errorf("expected name 'ten', got %q", list[2].Name)
	}
	if len(list[0].Checksum) != 64 {
		t.Errorf("expected a sha256 hex checksum, got %q", list[0].Checksum)
	}
}

func TestLoadRefusesDuplicateVersions(t *testing.T) {
	_, err := Load(fsWith("0001_a.sql", "0001_b.sql", "0002_c.sql"))
	if err == nil || !strings.Contains(err.Error(), "share version 1") {
		t.Errorf("expected a duplicate-version error, got %v", err)
	}
}

func TestLoadRefusesStrayFiles(t *testing.T) {
	for _, name := range []string{"notes.sql", "0001.sql", "0000_zero.sql", "0001_has space.sql"} {
		if _, err := Load(fsWith(name)); err == nil {
			t.Errorf("expected %q to be refused", name)
		}
	}
}

// The embedded set must always load: a bad filename committed to
// sql/migrations/ should fail here, not on the deploy.
func TestEmbeddedMigrationsLoad(t *testing.T) {
	list, err := Load(sqlfiles.Migrations)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 || list[0].Version != 1 || list[0].Name != "initial_schema" {
		t.Errorf("expected 0001_initial_schema first, got %+v", list)
	}
}

// ---------------------------------------------------------------------------
// Integration: a database of its own, created and dropped by the test, on
// the same server the repository suite uses. TEST_DATABASE_URL names a
// *_test_db on that server (bin/200-run-tests exports it); without it the
// tests skip. No fallback to etc/loc-api-srv.env -- that is the developer's
// live database.

func scratchDatabase(t *testing.T) *pgx.Conn {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("skipping integration tests: TEST_DATABASE_URL unset (run bin/200-run-tests go)")
	}
	parsed, err := url.Parse(base)
	if err != nil {
		t.Skipf("skipping integration tests: bad DATABASE_URL (%v)", err)
	}
	name := "loc_proto_migrate_test_db"
	if !strings.HasSuffix(strings.TrimPrefix(parsed.Path, "/"), "_test_db") {
		t.Fatalf("refusing to run against %s: not a _test_db", parsed.Path)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	admin := *parsed
	admin.Path = "/postgres"
	adminConn, err := pgx.Connect(ctx, admin.String())
	if err != nil {
		t.Skipf("skipping integration tests: no reachable test database (%v)", err)
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
		_, _ = adminConn.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name)
		adminConn.Close(context.Background())
	})
	return conn
}

func TestUpAppliesEmbeddedSchemaToAnEmptyDatabase(t *testing.T) {
	conn := scratchDatabase(t)
	ctx := context.Background()
	migrations, err := Load(sqlfiles.Migrations)
	if err != nil {
		t.Fatal(err)
	}
	m := New(conn, migrations)

	ran, err := m.Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ran) != len(migrations) {
		t.Errorf("expected %d applied, got %d", len(migrations), len(ran))
	}
	var tables int
	if err := conn.QueryRow(ctx, `
		SELECT count(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name <> 'schema_migration'`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	// app_user_group, app_user, example
	if tables < 3 {
		t.Errorf("expected the application schema, got %d tables", tables)
	}

	// Idempotent, and verify agrees.
	ran, err = m.Up(ctx)
	if err != nil || len(ran) != 0 {
		t.Errorf("second up: expected nothing, got %d, %v", len(ran), err)
	}
	if err := m.Verify(ctx); err != nil {
		t.Error(err)
	}
}

func TestStampRecordsWithoutRunning(t *testing.T) {
	conn := scratchDatabase(t)
	ctx := context.Background()
	migrations, _ := Load(sqlfiles.Migrations)
	m := New(conn, migrations)

	stamped, err := m.Stamp(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(stamped) != 1 {
		t.Errorf("expected 1 stamped, got %d", len(stamped))
	}
	var tables int
	_ = conn.QueryRow(ctx, `
		SELECT count(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name <> 'schema_migration'`).Scan(&tables)
	if tables != 0 {
		t.Errorf("stamp must not run SQL; found %d tables", tables)
	}
	if _, err := m.Stamp(ctx, 99); err == nil {
		t.Error("expected an unknown version to be refused")
	}
}

func TestVerifyCatchesAnEditedMigration(t *testing.T) {
	conn := scratchDatabase(t)
	ctx := context.Background()
	migrations, _ := Load(sqlfiles.Migrations)
	if _, err := New(conn, migrations).Up(ctx); err != nil {
		t.Fatal(err)
	}
	edited := append([]Migration{}, migrations...)
	edited[0].Checksum = "0000"
	m := New(conn, edited)
	if err := m.Verify(ctx); !errors.Is(err, ErrDrift) {
		t.Errorf("expected ErrDrift, got %v", err)
	}
	if _, err := m.Up(ctx); !errors.Is(err, ErrDrift) {
		t.Errorf("up must refuse on drift, got %v", err)
	}
	// A recorded version with no file behind it is drift too.
	if err := New(conn, nil).Verify(ctx); !errors.Is(err, ErrDrift) {
		t.Errorf("expected orphaned version to be ErrDrift, got %v", err)
	}
}

func TestUpRefusesOutOfOrderAndRollsBackAFailure(t *testing.T) {
	conn := scratchDatabase(t)
	ctx := context.Background()
	files := fstest.MapFS{
		"migrations/0001_a.sql": &fstest.MapFile{Data: []byte("CREATE TABLE a (id int);")},
		"migrations/0003_c.sql": &fstest.MapFile{Data: []byte("CREATE TABLE c (id int);")},
	}
	first, _ := Load(files)
	if _, err := New(conn, first).Up(ctx); err != nil {
		t.Fatal(err)
	}

	// 0002 turns up after 0003 is applied.
	files["migrations/0002_b.sql"] = &fstest.MapFile{Data: []byte("CREATE TABLE b (id int);")}
	later, _ := Load(files)
	if _, err := New(conn, later).Up(ctx); err == nil || !strings.Contains(err.Error(), "out of order") {
		t.Errorf("expected out-of-order refusal, got %v", err)
	}

	// A failing file leaves neither its SQL nor a record.
	delete(files, "migrations/0002_b.sql")
	files["migrations/0004_bad.sql"] = &fstest.MapFile{
		Data: []byte("CREATE TABLE d (id int); CREATE TABLE d (id int);")}
	bad, _ := Load(files)
	if _, err := New(conn, bad).Up(ctx); err == nil {
		t.Fatal("expected the duplicate CREATE to fail")
	}
	var exists bool
	_ = conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'd')`).Scan(&exists)
	if exists {
		t.Error("failed migration's table survived the rollback")
	}
	var recorded int
	_ = conn.QueryRow(ctx, `SELECT count(*) FROM schema_migration WHERE version = 4`).Scan(&recorded)
	if recorded != 0 {
		t.Error("failed migration was recorded")
	}
}

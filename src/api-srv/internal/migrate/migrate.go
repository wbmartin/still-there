// Package migrate applies versioned SQL files to a database and records what
// it applied -- the Alembic-shaped part of this application, without
// downgrade. See doc/900-database-guidelines.md for the design.
//
// Rules:
//   - A migration is migrations/NNNN_name.sql. NNNN is the version, applied
//     in ascending order. Two files with one version is an error.
//   - Each file runs in its own transaction and is recorded in
//     schema_migration in that same transaction, so a failure leaves neither
//     half-applied SQL nor a lying record.
//   - Forward only. A wrong migration is fixed by the next one.
//   - The SHA-256 of each file is stored. An applied file that has since
//     changed is a verify failure, never a silent divergence between hosts.
//   - A run holds a session advisory lock so two deploys cannot interleave.
package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// lockKey is arbitrary but fixed: every migrator run on a database contends
// for the same advisory lock.
const lockKey int64 = 0x50524f544f // "PROTO"

var filePattern = regexp.MustCompile(`^(\d+)_([A-Za-z0-9_-]+)\.sql$`)

// Migration is one embedded file.
type Migration struct {
	Version  int
	Name     string
	Checksum string
	SQL      string
}

// Applied is one schema_migration row.
type Applied struct {
	Version   int
	Name      string
	Checksum  string
	AppliedAt time.Time
	AppliedBy string
}

// Status pairs each known version with what the database says about it.
type Status struct {
	Version  int
	Name     string
	Applied  *Applied // nil when pending
	Drifted  bool     // applied, and the file's checksum no longer matches
	Orphaned bool     // applied, but no file of that version is embedded
}

// Load reads every migration from the file system's migrations/ directory,
// sorted by version. Duplicate versions and unparseable names are errors:
// the file set is the contract, and a stray file must not be silently
// ignored.
func Load(files fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(files, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations: %w", err)
	}
	seen := map[int]string{}
	list := []Migration{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		match := filePattern.FindStringSubmatch(entry.Name())
		if match == nil {
			return nil, fmt.Errorf(
				"migration %q does not match NNNN_name.sql", entry.Name())
		}
		version, err := strconv.Atoi(match[1])
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("migration %q has an invalid version", entry.Name())
		}
		if other, dup := seen[version]; dup {
			return nil, fmt.Errorf(
				"migrations %q and %q share version %d", other, entry.Name(), version)
		}
		seen[version] = entry.Name()
		body, err := fs.ReadFile(files, path.Join("migrations", entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", entry.Name(), err)
		}
		sum := sha256.Sum256(body)
		list = append(list, Migration{
			Version:  version,
			Name:     match[2],
			Checksum: hex.EncodeToString(sum[:]),
			SQL:      string(body),
		})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Version < list[j].Version })
	return list, nil
}

// Migrator runs against one connection. It is not safe for concurrent use;
// the advisory lock is what serialises runs across processes.
type Migrator struct {
	conn       *pgx.Conn
	migrations []Migration
}

func New(conn *pgx.Conn, migrations []Migration) *Migrator {
	return &Migrator{conn: conn, migrations: migrations}
}

// ensureTable creates schema_migration if it is missing. Called by every
// command so a database is brought under management by the first `up` or
// `stamp` rather than by a hand-written statement.
func (m *Migrator) ensureTable(ctx context.Context) error {
	_, err := m.conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migration (
			version     INTEGER PRIMARY KEY,
			name        TEXT NOT NULL,
			checksum    TEXT NOT NULL,
			applied_at  TIMESTAMPTZ NOT NULL,
			applied_by  TEXT NOT NULL
		)`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migration: %w", err)
	}
	return nil
}

func (m *Migrator) lock(ctx context.Context) (func(), error) {
	if _, err := m.conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockKey); err != nil {
		return nil, fmt.Errorf("failed to take the migration lock: %w", err)
	}
	return func() {
		_, _ = m.conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, lockKey)
	}, nil
}

func (m *Migrator) applied(ctx context.Context) (map[int]Applied, error) {
	rows, err := m.conn.Query(ctx, `
		SELECT version, name, checksum, applied_at, applied_by
		FROM schema_migration ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema_migration: %w", err)
	}
	defer rows.Close()
	result := map[int]Applied{}
	for rows.Next() {
		var row Applied
		if err := rows.Scan(&row.Version, &row.Name, &row.Checksum,
			&row.AppliedAt, &row.AppliedBy); err != nil {
			return nil, fmt.Errorf("failed to read schema_migration: %w", err)
		}
		result[row.Version] = row
	}
	return result, rows.Err()
}

// Status reports every embedded version and every applied version, merged
// and sorted, so a single listing shows pending, applied, drifted and
// orphaned rows.
func (m *Migrator) Status(ctx context.Context) ([]Status, error) {
	if err := m.ensureTable(ctx); err != nil {
		return nil, err
	}
	applied, err := m.applied(ctx)
	if err != nil {
		return nil, err
	}
	byVersion := map[int]*Status{}
	checksums := map[int]string{}
	for _, mig := range m.migrations {
		byVersion[mig.Version] = &Status{Version: mig.Version, Name: mig.Name}
		checksums[mig.Version] = mig.Checksum
	}
	for version, row := range applied {
		row := row
		status, known := byVersion[version]
		if !known {
			byVersion[version] = &Status{Version: version, Name: row.Name,
				Applied: &row, Orphaned: true}
			continue
		}
		status.Applied = &row
		status.Drifted = checksums[version] != row.Checksum
	}
	list := []Status{}
	for _, status := range byVersion {
		list = append(list, *status)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Version < list[j].Version })
	return list, nil
}

// ErrDrift is returned by Verify and Up when the database and the embedded
// files disagree about what has been applied.
var ErrDrift = errors.New("applied migration differs from the embedded file")

// Verify fails if any applied migration's checksum differs from the embedded
// file, or if the database records a version that no longer exists. It is
// the first thing a deploy runs: a mismatch means the SQL that built this
// database is not the SQL in this binary, and nothing should proceed on that.
func (m *Migrator) Verify(ctx context.Context) error {
	list, err := m.Status(ctx)
	if err != nil {
		return err
	}
	for _, status := range list {
		if status.Drifted {
			return fmt.Errorf("%w: version %d (%s)", ErrDrift, status.Version, status.Name)
		}
		if status.Orphaned {
			return fmt.Errorf("%w: version %d (%s) is applied but no file exists",
				ErrDrift, status.Version, status.Name)
		}
	}
	return nil
}

// Up applies every pending migration in order, each in its own transaction,
// after verifying what is already applied. A pending version LOWER than one
// already applied is refused: it means a file was added behind the database's
// back, and applying it out of order would produce a schema no other
// environment has.
func (m *Migrator) Up(ctx context.Context) ([]Migration, error) {
	unlock, err := m.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := m.Verify(ctx); err != nil {
		return nil, err
	}
	applied, err := m.applied(ctx)
	if err != nil {
		return nil, err
	}
	highest := 0
	for version := range applied {
		if version > highest {
			highest = version
		}
	}
	ran := []Migration{}
	for _, mig := range m.migrations {
		if _, done := applied[mig.Version]; done {
			continue
		}
		if mig.Version < highest {
			return ran, fmt.Errorf(
				"migration %d (%s) is pending but %d is already applied; out of order",
				mig.Version, mig.Name, highest)
		}
		if err := m.apply(ctx, mig); err != nil {
			return ran, err
		}
		ran = append(ran, mig)
		highest = mig.Version
	}
	return ran, nil
}

func (m *Migrator) apply(ctx context.Context, mig Migration) error {
	tx, err := m.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin migration %d: %w", mig.Version, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// No arguments, so pgx uses the simple protocol and the file may hold
	// many statements.
	if _, err := tx.Exec(ctx, mig.SQL); err != nil {
		return fmt.Errorf("migration %d (%s) failed: %w", mig.Version, mig.Name, err)
	}
	if err := m.record(ctx, tx, mig); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration %d: %w", mig.Version, err)
	}
	return nil
}

func (m *Migrator) record(ctx context.Context, tx pgx.Tx, mig Migration) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO schema_migration (version, name, checksum, applied_at, applied_by)
		VALUES ($1, $2, $3, now(), current_user)`,
		mig.Version, mig.Name, mig.Checksum)
	if err != nil {
		return fmt.Errorf("failed to record migration %d: %w", mig.Version, err)
	}
	return nil
}

// Stamp records every migration up to and including version as applied
// WITHOUT running it. This is how a database that predates the migrator --
// built by hand from the schema file -- is brought under management: its
// schema already matches 0001, so 0001 is stamped, not run.
func (m *Migrator) Stamp(ctx context.Context, version int) ([]Migration, error) {
	unlock, err := m.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := m.ensureTable(ctx); err != nil {
		return nil, err
	}
	applied, err := m.applied(ctx)
	if err != nil {
		return nil, err
	}
	found := false
	for _, mig := range m.migrations {
		if mig.Version == version {
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("no migration has version %d", version)
	}
	tx, err := m.conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin stamp: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	stamped := []Migration{}
	for _, mig := range m.migrations {
		if mig.Version > version {
			break
		}
		if _, done := applied[mig.Version]; done {
			continue
		}
		if err := m.record(ctx, tx, mig); err != nil {
			return nil, err
		}
		stamped = append(stamped, mig)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit stamp: %w", err)
	}
	return stamped, nil
}

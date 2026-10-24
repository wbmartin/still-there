package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The schema is migrations/NNNN_name.sql, applied in order, forward only.
// relay_schema_migration records each file's SHA-256; a file changed after
// it was applied is drift, and both migrate and serve refuse it. A wrong
// migration is fixed by the next one.

//go:embed migrations/*.sql
var migrationFS embed.FS

type migration struct {
	version  int
	name     string
	sql      string
	checksum string
}

func migrationList() ([]migration, error) {
	nameList, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(nameList)
	var list []migration
	for _, name := range nameList {
		base := strings.TrimPrefix(name, "migrations/")
		version, err := strconv.Atoi(strings.SplitN(base, "_", 2)[0])
		if err != nil {
			return nil, fmt.Errorf("migration %s: no NNNN_ prefix", base)
		}
		content, err := migrationFS.ReadFile(name)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(content)
		list = append(list, migration{
			version: version, name: base, sql: string(content),
			checksum: hex.EncodeToString(sum[:]),
		})
	}
	return list, nil
}

// appliedMap is version -> checksum of what the database has applied; an
// absent table is an empty map.
func appliedMap(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}) (map[int]string, error) {
	applied := map[int]string{}
	var exists bool
	if err := q.QueryRow(ctx,
		`SELECT to_regclass('relay_schema_migration') IS NOT NULL`,
	).Scan(&exists); err != nil || !exists {
		return applied, err
	}
	rows, err := q.Query(ctx, `
		SELECT version, checksum
		FROM relay_schema_migration`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var version int
		var checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, err
		}
		applied[version] = checksum
	}
	return applied, rows.Err()
}

// pendingList is the migrations not yet applied; drift is an error.
func pendingList(applied map[int]string, list []migration) ([]migration, error) {
	var pending []migration
	for _, m := range list {
		checksum, ok := applied[m.version]
		switch {
		case !ok:
			pending = append(pending, m)
		case checksum != m.checksum:
			return nil, fmt.Errorf(
				"migration %s changed after it was applied", m.name,
			)
		}
	}
	return pending, nil
}

// migrateUp applies every pending migration, each in its own transaction
// with its record, under an advisory lock so two runs cannot interleave.
// It returns how many it applied.
func migrateUp(ctx context.Context, db *pgxpool.Pool, now float64) (int, error) {
	list, err := migrationList()
	if err != nil {
		return 0, err
	}
	if _, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS relay_schema_migration (
		    version INTEGER PRIMARY KEY,
		    name TEXT NOT NULL,
		    checksum TEXT NOT NULL,
		    applied_ts NUMERIC(17,7) NOT NULL
		)`); err != nil {
		return 0, err
	}
	count := 0
	for {
		tx, err := db.Begin(ctx)
		if err != nil {
			return count, err
		}
		applied, err := lockedApplied(ctx, tx)
		if err != nil {
			tx.Rollback(ctx)
			return count, err
		}
		pending, err := pendingList(applied, list)
		if err != nil || len(pending) == 0 {
			tx.Rollback(ctx)
			return count, err
		}
		m := pending[0]
		if _, err := tx.Exec(ctx, m.sql); err != nil {
			tx.Rollback(ctx)
			return count, fmt.Errorf("migration %s: %w", m.name, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO relay_schema_migration (version, name, checksum, applied_ts)
			VALUES ($1, $2, $3, $4)`,
			m.version, m.name, m.checksum, now); err != nil {
			tx.Rollback(ctx)
			return count, err
		}
		if err := tx.Commit(ctx); err != nil {
			return count, err
		}
		count++
	}
}

func lockedApplied(ctx context.Context, tx pgx.Tx) (map[int]string, error) {
	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtext('relay_schema_migration'))`,
	); err != nil {
		return nil, err
	}
	return appliedMap(ctx, tx)
}

// checkSchema fails unless every embedded migration is applied unchanged:
// serve runs as a role that cannot change the schema, so it refuses to
// start on one that is behind.
func checkSchema(ctx context.Context, db *pgxpool.Pool) error {
	list, err := migrationList()
	if err != nil {
		return err
	}
	applied, err := appliedMap(ctx, db)
	if err != nil {
		return err
	}
	pending, err := pendingList(applied, list)
	if err != nil {
		return err
	}
	if len(pending) > 0 {
		return fmt.Errorf(
			"%d migration(s) pending, from %s: run sms-relay migrate first",
			len(pending), pending[0].name,
		)
	}
	return nil
}

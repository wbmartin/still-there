// stillthere-migrate applies the embedded schema migrations to a database.
//
//	stillthere-migrate -config <env file> status          what is applied, pending, drifted
//	stillthere-migrate -config <env file> pending         print the number of pending migrations (for scripts)
//	stillthere-migrate -config <env file> verify          fail if the database disagrees with the files
//	stillthere-migrate -config <env file> up              apply every pending migration, in order
//	stillthere-migrate -config <env file> stamp <version> record up to <version> as applied without running
//	stillthere-migrate -config <env file> schema-reference print the schema as CREATE TABLE/INDEX text (doc/003)
//	stillthere-migrate new <name>                        create sql/migrations/NNNN_<name>.sql (next free NNNN); no database
//	stillthere-migrate -config <env file> -base-url <origin> [-invite-ttl-hours N] invite <userName>
//	                                                print a first sign-in link for an invited login (invite.go)
//
// -config is a KEY=VALUE file of which only DATABASE_URL is read -- the
// migrator's own file, with the DDL role's credentials, never the API's.
// -url takes the URL directly, for the test harness.
//
// Exit status is the whole contract for the deploy scripts: 0 means the
// database is at the embedded version, anything else means stop.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"stillthere-api/internal/config"
	"stillthere-api/internal/migrate"
	sqlfiles "stillthere-api/sql"
)

func main() {
	configPath := flag.String("config", "", "env file holding DATABASE_URL")
	url := flag.String("url", "", "database URL (instead of -config)")
	dir := flag.String("dir", "sql/migrations", "source directory `new` writes into (relative to the cwd)")
	baseURL := flag.String("base-url", "",
		"site origin the `invite` link is built on, e.g. https://app.example.com")
	inviteTTLHours := flag.Int("invite-ttl-hours", 72,
		"hours the `invite` link lives (pass the API's INVITE_TTL_HOURS)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: stillthere-migrate (-config <file> | -url <url>) status|pending|verify|up|stamp <version>|schema-reference\n")
		fmt.Fprintf(os.Stderr, "       stillthere-migrate (-config <file> | -url <url>) -base-url <origin> [-invite-ttl-hours N] invite <userName>\n")
		fmt.Fprintf(os.Stderr, "       stillthere-migrate [-dir sql/migrations] new <name>\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if len(flag.Args()) > 0 && flag.Arg(0) == "new" {
		if err := newMigration(*dir, flag.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
			os.Exit(1)
		}
		return
	}
	invite := inviteOptions{baseURL: *baseURL, ttlHours: *inviteTTLHours}
	if err := run(*configPath, *url, invite, flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}
}

// inviteOptions carries the flags only `invite` reads.
type inviteOptions struct {
	baseURL  string
	ttlHours int
}

func run(
	configPath, url string, invite inviteOptions, args []string,
) error {
	if len(args) == 0 {
		return fmt.Errorf("a command is required: status, pending, verify, " +
			"up, stamp, schema-reference, invite, new")
	}
	if url == "" {
		if configPath == "" {
			return fmt.Errorf("-config or -url is required")
		}
		env, err := config.ReadEnvFile(configPath)
		if err != nil {
			return err
		}
		url = env["DATABASE_URL"]
		if url == "" {
			return fmt.Errorf("%s does not set DATABASE_URL", configPath)
		}
	}

	migrations, err := migrate.Load(sqlfiles.Migrations)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close(context.Background())
	m := migrate.New(conn, migrations)

	switch args[0] {
	case "status":
		list, err := m.Status(ctx)
		if err != nil {
			return err
		}
		for _, s := range list {
			state := "pending"
			switch {
			case s.Orphaned:
				state = "ORPHANED (applied, no file)"
			case s.Drifted:
				state = "DRIFTED (applied " + s.Applied.AppliedAt.UTC().Format(time.RFC3339) + ", file changed since)"
			case s.Applied != nil:
				state = "applied " + s.Applied.AppliedAt.UTC().Format(time.RFC3339) + " by " + s.Applied.AppliedBy
			}
			fmt.Printf("%04d  %-40s %s\n", s.Version, s.Name, state)
		}
		return nil
	case "pending":
		// A number on stdout and nothing else, so a deploy script can decide
		// whether to take a backup before `up`.
		list, err := m.Status(ctx)
		if err != nil {
			return err
		}
		pending := 0
		for _, s := range list {
			if s.Applied == nil {
				pending++
			}
		}
		fmt.Println(pending)
		return nil
	case "verify":
		if err := m.Verify(ctx); err != nil {
			return err
		}
		fmt.Println("[INFO] verify: database matches the embedded migrations")
		return nil
	case "up":
		ran, err := m.Up(ctx)
		for _, mig := range ran {
			fmt.Printf("[INFO] applied %04d_%s\n", mig.Version, mig.Name)
		}
		if err != nil {
			return err
		}
		if len(ran) == 0 {
			fmt.Println("[INFO] up: nothing pending")
		}
		return nil
	case "stamp":
		if len(args) != 2 {
			return fmt.Errorf("stamp needs a version")
		}
		version, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("stamp: %q is not a version", args[1])
		}
		stamped, err := m.Stamp(ctx, version)
		if err != nil {
			return err
		}
		for _, mig := range stamped {
			fmt.Printf("[INFO] stamped %04d_%s (not run)\n", mig.Version, mig.Name)
		}
		if len(stamped) == 0 {
			fmt.Println("[INFO] stamp: nothing to record")
		}
		return nil
	case "schema-reference":
		// stdout only, so bin/510 can redirect it into
		// doc/003-schema-reference.sql.
		return migrate.WriteSchemaReference(ctx, conn, os.Stdout)
	case "invite":
		if len(args) != 2 {
			return fmt.Errorf("invite needs exactly one user name")
		}
		// The writes below are this binary's SQL: refuse a database whose
		// schema is not the one it was built with.
		if err := m.Verify(ctx); err != nil {
			return err
		}
		pool, err := pgxpool.New(ctx, url)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer pool.Close()
		issued, err := issueInvite(ctx, pool, args[1], invite.baseURL,
			invite.ttlHours, time.Now())
		if err != nil {
			return err
		}
		// The link alone on stdout, so a script can capture it; it is the
		// only copy of the token (only its hash is stored).
		fmt.Fprintf(os.Stderr, "[INFO] invite: %s (tenant %d), expires "+
			"%s; any earlier link for it is revoked\n", issued.userName,
			issued.securityContext,
			issued.expires.UTC().Format(time.RFC3339))
		fmt.Println(issued.link)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

// newName is what a migration may be called: the part after NNNN_.
var newName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// newMigration writes the next NNNN_<name>.sql into dir with the header every
// migration carries. The number is one past the highest already there -- the
// files on disk, not the embedded set, so it works on an uncommitted tree.
func newMigration(dir string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("new needs exactly one name, e.g. new add_widgets")
	}
	name := args[0]
	if !newName.MatchString(name) {
		return fmt.Errorf("new: %q must be lower-case snake_case (a-z, 0-9, _)", name)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("new: %w (run from src/api-srv, or pass -dir)", err)
	}
	next := 1
	for _, e := range entries {
		if v, err := strconv.Atoi(strings.SplitN(e.Name(), "_", 2)[0]); err == nil && v >= next {
			next = v + 1
		}
	}
	file := filepath.Join(dir, fmt.Sprintf("%04d_%s.sql", next, name))
	body := fmt.Sprintf(`-- Migration %04d: %s
--
-- Conventions (doc/900-database-guidelines.md): singular table names,
-- <entity>_id SERIAL keys, security_context_id on every table, no foreign
-- keys, no triggers, no DEFAULTs on application data (create_ts/update_ts/
-- delete_ts NUMERIC(17,7) are set by the Go handlers), snake_case.
--
-- Applied by cmd/migrate inside one transaction. Never edit this file once
-- it has been applied anywhere: the migrator stores its checksum and
-- 'verify' will refuse a database whose %04d differs from this file.

`, next, name, next)
	if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
		return fmt.Errorf("new: %w", err)
	}
	fmt.Printf("[INFO] created %s\n", file)
	return nil
}

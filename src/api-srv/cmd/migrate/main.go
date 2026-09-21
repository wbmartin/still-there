// proto-migrate applies the embedded schema migrations to a database.
//
//	proto-migrate -config <env file> status          what is applied, pending, drifted
//	proto-migrate -config <env file> pending         print the number of pending migrations (for scripts)
//	proto-migrate -config <env file> verify          fail if the database disagrees with the files
//	proto-migrate -config <env file> up              apply every pending migration, in order
//	proto-migrate -config <env file> stamp <version> record up to <version> as applied without running
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
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/wbmartin/starter-api/internal/config"
	"github.com/wbmartin/starter-api/internal/migrate"
	sqlfiles "github.com/wbmartin/starter-api/sql"
)

func main() {
	configPath := flag.String("config", "", "env file holding DATABASE_URL")
	url := flag.String("url", "", "database URL (instead of -config)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: proto-migrate (-config <file> | -url <url>) status|pending|verify|up|stamp <version>\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if err := run(*configPath, *url, flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}
}

func run(configPath, url string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("a command is required: status, pending, verify, up, stamp")
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
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

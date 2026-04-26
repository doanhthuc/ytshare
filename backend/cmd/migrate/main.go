// Binary migrate is the standalone schema-migration CLI.
//
// Usage:
//
//	migrate up                 # apply all pending migrations
//	migrate down               # roll back the most recent migration
//	migrate status             # print current schema version
//	migrate force <version>    # mark version applied (recovery only)
//	migrate create <name>      # scaffold the next *.up.sql / *.down.sql pair
//
// Configuration is read from .env / environment exactly like cmd/server.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"backend/internal/config"
	"backend/internal/database"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("missing subcommand: up | down | status | force <v> | create <name>")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch args[0] {
	case "up":
		res, err := database.RunMigrations(cfg.DB.URL(), cfg.Migrations.Dir, database.MigrateUp)
		if err != nil {
			return err
		}
		printResult(res)
		return nil

	case "down":
		res, err := database.RunMigrations(cfg.DB.URL(), cfg.Migrations.Dir, database.MigrateDown)
		if err != nil {
			return err
		}
		printResult(res)
		return nil

	case "status":
		res, err := database.RunMigrations(cfg.DB.URL(), cfg.Migrations.Dir, database.MigrateUp)
		if err != nil {
			return err
		}
		printResult(res)
		return nil

	case "force":
		if len(args) < 2 {
			return errors.New("force: missing <version>")
		}
		v, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("force: %w", err)
		}
		if err := database.ForceMigrationVersion(cfg.DB.URL(), cfg.Migrations.Dir, v); err != nil {
			return err
		}
		fmt.Printf("forced version=%d\n", v)
		return nil

	case "create":
		if len(args) < 2 {
			return errors.New("create: missing <name>")
		}
		return scaffold(cfg.Migrations.Dir, args[1])

	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func printResult(r database.MigrationResult) {
	fmt.Printf("version=%d dirty=%t no_op=%t\n", r.Version, r.Dirty, r.NoOp)
}

// scaffold creates an empty *.up.sql / *.down.sql pair using a 6-digit
// timestamp-style sequence so collisions on multi-developer branches are
// unlikely. We deliberately keep this minimal — no boilerplate inside the
// new files; every migration is custom.
func scaffold(dir, name string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	seq := time.Now().UTC().Format("20060102150405")
	for _, suffix := range []string{"up.sql", "down.sql"} {
		path := filepath.Join(dir, fmt.Sprintf("%s_%s.%s", seq, name, suffix))
		// O_EXCL so we never silently overwrite an existing file.
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return fmt.Errorf("create %s: %w", path, err)
		}
		_ = f.Close()
		fmt.Println("created", path)
	}
	return nil
}

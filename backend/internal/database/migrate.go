package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // register pg driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // register file source
)

// MigrateAction enumerates the supported migration operations.
type MigrateAction string

const (
	// MigrateUp applies all pending migrations.
	MigrateUp MigrateAction = "up"
	// MigrateDown rolls back the most recent migration.
	MigrateDown MigrateAction = "down"
)

// MigrationResult reports the schema version after the operation. `Dirty`
// means a migration failed mid-way and the operator must intervene with
// `make migrate-force version=<n>` after fixing the broken file.
type MigrationResult struct {
	Version uint
	Dirty   bool
	NoOp    bool
}

// RunMigrations applies the requested action against the database at
// `databaseURL` (postgres URL form), reading versioned files from `dir`.
//
// `databaseURL` must be the URL form, e.g.
//
//	postgres://user:password@host:5432/dbname?sslmode=disable
//
// `dir` is a filesystem path; passing `./migrations` works.
func RunMigrations(databaseURL, dir string, action MigrateAction) (MigrationResult, error) {
	source := "file://" + dir
	m, err := migrate.New(source, databaseURL)
	if err != nil {
		return MigrationResult{}, fmt.Errorf("database: open migrator: %w", err)
	}
	defer func() {
		// Close returns two errors — one each from the source and database
		// — neither of which we can act on after the work is done.
		_, _ = m.Close()
	}()

	switch action {
	case MigrateUp:
		err = m.Up()
	case MigrateDown:
		err = m.Steps(-1)
	default:
		return MigrationResult{}, fmt.Errorf("database: unknown migrate action %q", action)
	}

	noOp := errors.Is(err, migrate.ErrNoChange)
	if err != nil && !noOp {
		return MigrationResult{}, fmt.Errorf("database: migrate %s: %w", action, err)
	}

	version, dirty, vErr := m.Version()
	if vErr != nil && !errors.Is(vErr, migrate.ErrNilVersion) {
		return MigrationResult{}, fmt.Errorf("database: read version: %w", vErr)
	}
	return MigrationResult{Version: version, Dirty: dirty, NoOp: noOp}, nil
}

// ForceMigrationVersion marks the database at the supplied version, dirty
// flag cleared. Use after manually resolving a failed migration.
func ForceMigrationVersion(databaseURL, dir string, version int) error {
	m, err := migrate.New("file://"+dir, databaseURL)
	if err != nil {
		return fmt.Errorf("database: open migrator: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Force(version); err != nil {
		return fmt.Errorf("database: force version %d: %w", version, err)
	}
	return nil
}

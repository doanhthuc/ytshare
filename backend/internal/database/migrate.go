package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // register pg driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // register file source
)

type MigrateAction string

const (
	MigrateUp   MigrateAction = "up"
	MigrateDown MigrateAction = "down"
)

// MigrationResult reports the schema version after the operation. Dirty
// means a migration failed mid-way and requires `migrate force <n>` after fixing.
type MigrationResult struct {
	Version uint
	Dirty   bool
	NoOp    bool
}

// RunMigrations applies action against databaseURL (postgres URL form),
// reading versioned files from dir.
func RunMigrations(databaseURL, dir string, action MigrateAction) (MigrationResult, error) {
	source := "file://" + dir
	m, err := migrate.New(source, databaseURL)
	if err != nil {
		return MigrationResult{}, fmt.Errorf("database: open migrator: %w", err)
	}
	defer func() { _, _ = m.Close() }()

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

// ForceMigrationVersion marks the database at version with dirty flag cleared.
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

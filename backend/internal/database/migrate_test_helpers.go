package database

import (
	"fmt"

	"gorm.io/gorm"

	"backend/internal/modules/users"
	"backend/internal/modules/videos"
)

// AutoMigrateForTests applies the GORM model schema to the supplied
// connection. It exists so that integration tests can spin up an
// in-memory SQLite database without running the production *.sql
// migrations (which target Postgres-only features such as pgcrypto and
// TIMESTAMPTZ). Production code MUST use RunMigrations instead.
func AutoMigrateForTests(db *gorm.DB) error {
	if err := db.AutoMigrate(&users.User{}, &videos.Video{}); err != nil {
		return fmt.Errorf("database: auto migrate (test): %w", err)
	}
	return nil
}

package database

import (
	"fmt"

	"gorm.io/gorm"

	"backend/internal/modules/users"
	"backend/internal/modules/videos"
)

// AutoMigrateForTests applies GORM schema for in-memory SQLite tests.
// Production code must use RunMigrations (the *.sql files target Postgres-only features).
func AutoMigrateForTests(db *gorm.DB) error {
	if err := db.AutoMigrate(&users.User{}, &videos.Video{}); err != nil {
		return fmt.Errorf("database: auto migrate (test): %w", err)
	}
	return nil
}

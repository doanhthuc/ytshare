// Package users owns the persistence model for application users.
package users

import (
	"time"

	"github.com/google/uuid"
)

// User is the domain entity. It does not include the password hash on
// purpose — that lives on the auth-side aggregate to keep concerns separate.
type User struct {
	ID                      uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Email                   string     `gorm:"uniqueIndex;size:320;not null"     json:"email"`
	Name                    string     `gorm:"size:120;not null"                 json:"name"`
	PasswordHash            string     `gorm:"size:255;not null"                 json:"-"`
	LastNotificationsSeenAt *time.Time `gorm:"column:last_notifications_seen_at" json:"-"`
	CreatedAt               time.Time  `json:"createdAt"`
	UpdatedAt               time.Time  `json:"updatedAt"`
}

// TableName lets GORM use a stable plural name.
func (User) TableName() string { return "users" }

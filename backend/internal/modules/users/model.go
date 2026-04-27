// Package users owns the persistence model for application users.
package users

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                      uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Email                   string     `gorm:"uniqueIndex;size:320;not null"     json:"email"`
	Name                    string     `gorm:"size:120;not null"                 json:"name"`
	PasswordHash            string     `gorm:"size:255;not null"                 json:"-"`
	LastNotificationsSeenAt *time.Time `gorm:"column:last_notifications_seen_at" json:"-"`
	CreatedAt               time.Time  `json:"createdAt"`
	UpdatedAt               time.Time  `json:"updatedAt"`
}

func (User) TableName() string { return "users" }

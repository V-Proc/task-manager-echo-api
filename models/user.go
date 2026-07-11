package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents an application account. Password is stored as a bcrypt hash
// and never returned over the API (see JSON tags).
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:100;not null" json:"name"`
	Email     string         `gorm:"size:150;uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"size:255;not null" json:"-"` // "-" => never serialized
	Tasks     []Task         `gorm:"foreignKey:UserID" json:"tasks,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName is set explicitly so future renames don't silently break the schema.
func (User) TableName() string { return "users" }

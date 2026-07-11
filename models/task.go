package models

import (
	"time"

	"gorm.io/gorm"
)

// TaskStatus enumerates the lifecycle states a task may be in.
type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

// Task is a unit of work owned by a single user.
type Task struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Title       string         `gorm:"size:200;not null" json:"title"`
	Description string         `gorm:"type:text" json:"description"`
	Status      TaskStatus     `gorm:"size:20;default:pending;not null" json:"status"`
	UserID      uint           `gorm:"index;not null" json:"user_id"`
	User        User           `gorm:"foreignKey:UserID" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Task) TableName() string { return "tasks" }

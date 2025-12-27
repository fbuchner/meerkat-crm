package models

import (
	"time"

	"gorm.io/gorm"
)

// JobExecution tracks the execution of scheduled jobs to prevent duplicates
// during rapid restarts or crash loops.
type JobExecution struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	JobName   string         `gorm:"uniqueIndex;not null" json:"job_name"`
	LastRunAt time.Time      `gorm:"not null" json:"last_run_at"`
	LockedAt  *time.Time     `json:"locked_at"`
	LockedBy  string         `json:"locked_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

const (
	// JobNameDailyReminders is the job name for the daily reminder email job
	JobNameDailyReminders = "daily_reminders"
)

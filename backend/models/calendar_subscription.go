package models

import (
	"time"

	"gorm.io/gorm"
)

// Calendar sync status values stored on CalendarSubscription.LastSyncStatus.
const (
	CalendarSyncStatusSuccess = "success"
	CalendarSyncStatusError   = "error"
)

// CalendarSubscription is a saved CalDAV/iCS calendar a user imports activities from.
// Credentials are optional (public calendars) and the password is stored encrypted.
type CalendarSubscription struct {
	gorm.Model
	UserID            uint       `gorm:"not null;index" json:"-"`
	Name              string     `gorm:"not null" json:"name"`
	URL               string     `gorm:"not null" json:"url"`
	Username          string     `json:"username"`
	PasswordEncrypted string     `json:"-"`
	SyncEnabled       bool       `gorm:"default:true" json:"sync_enabled"`
	PastDays          int        `gorm:"default:5" json:"past_days"`
	FutureDays        int        `gorm:"default:10" json:"future_days"`
	LastSyncedAt      *time.Time `json:"last_synced_at"`
	LastSyncStatus    string     `json:"last_sync_status"`
	LastSyncError     string     `json:"last_sync_error"`
}

// CalendarEventLink maps an imported iCalendar event (by UID) to the activity it
// created, so re-syncs update or skip instead of duplicating. ContentHash tracks
// the imported fields to detect upstream changes.
type CalendarEventLink struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	SubscriptionID uint      `gorm:"not null;index;uniqueIndex:idx_calendar_event_links_sub_uid,priority:1" json:"subscription_id"`
	UserID         uint      `gorm:"not null;index" json:"-"`
	UID            string    `gorm:"not null;uniqueIndex:idx_calendar_event_links_sub_uid,priority:2" json:"uid"`
	ActivityID     uint      `gorm:"not null;index" json:"activity_id"`
	ContentHash    string    `gorm:"not null" json:"-"`
}

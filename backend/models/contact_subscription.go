package models

import (
	"time"

	"gorm.io/gorm"
)

// Contact sync status values stored on ContactSubscription.LastSyncStatus.
const (
	ContactSyncStatusSuccess = "success"
	ContactSyncStatusError   = "error"
)

// ContactSubscription is a saved CardDAV address book a user syncs contacts
// in from (WP-73b: opposite direction of the CardDAV *server*, WP-73).
// Mirrors CalendarSubscription's shape exactly (see
// docs/fork-plan/50-integration-and-rebrand.md WP-73b), minus the
// PastDays/FutureDays window fields, which have no contacts analog.
//
// SyncToken is the RFC 6578 sync-collection token returned by the remote
// server, persisted so the next sync only fetches the delta. It is empty
// until the first successful sync, and is deliberately not exposed in JSON
// (internal bookkeeping, not user-facing, like PasswordEncrypted).
type ContactSubscription struct {
	gorm.Model
	UserID            uint       `gorm:"not null;index" json:"-"`
	Name              string     `gorm:"not null" json:"name"`
	URL               string     `gorm:"not null" json:"url"`
	Username          string     `json:"username"`
	PasswordEncrypted string     `json:"-"`
	SyncEnabled       bool       `gorm:"default:true" json:"sync_enabled"`
	SyncToken         string     `json:"-"`
	LastSyncedAt      *time.Time `json:"last_synced_at"`
	LastSyncStatus    string     `json:"last_sync_status"`
	LastSyncError     string     `json:"last_sync_error"`
}

// ContactSyncLink maps a synced CardDAV address object (by resource path,
// i.e. Href) to the real Contact row it created, so re-syncs update or
// archive instead of duplicating. Unlike CalendarEventLink (keyed by
// iCalendar UID), contacts don't have a stable UID supplied by the remote
// server the way iCalendar events do, so the CardDAV resource path (Href)
// is the key instead. ContentHash tracks the imported vCard bytes to detect
// upstream changes, same idea as CalendarEventLink.ContentHash.
type ContactSyncLink struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	SubscriptionID uint      `gorm:"not null;index;uniqueIndex:idx_contact_sync_links_sub_href,priority:1" json:"subscription_id"`
	UserID         uint      `gorm:"not null;index" json:"-"`
	Href           string    `gorm:"not null;uniqueIndex:idx_contact_sync_links_sub_href,priority:2" json:"href"`
	ContactID      uint      `gorm:"not null;index" json:"contact_id"`
	ETag           string    `json:"-"`
	ContentHash    string    `gorm:"not null" json:"-"`
}

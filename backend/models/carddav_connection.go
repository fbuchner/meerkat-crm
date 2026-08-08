package models

import "time"

// sync direction
const (
	CardDAVDirectionTwoWay = "two_way"
	CardDAVDirectionPull   = "pull"
	CardDAVDirectionPush   = "push"
)

const (
	CardDAVSyncStatusSuccess = "success"
	CardDAVSyncStatusPartial = "partial"
	CardDAVSyncStatusError   = "error"
)

// CardDAVConnection connects a user to one external CardDAV address book
// (Nextcloud, Radicale, sabre/dav) that their contacts sync with. One
// connection per user. Rows are hard-deleted: a soft-deleted row would block
// reconnecting under the unique user_id index.
type CardDAVConnection struct {
	ID                uint       `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	UserID            uint       `gorm:"not null;uniqueIndex" json:"-"`
	BaseURL           string     `gorm:"not null" json:"base_url"`
	AddressBookPath   string     `gorm:"not null" json:"address_book_path"`
	AddressBookName   string     `json:"address_book_name"`
	Username          string     `json:"username"`
	PasswordEncrypted string     `json:"-"`
	Direction         string     `gorm:"not null;default:two_way" json:"direction"`
	SyncEnabled       bool       `gorm:"default:true" json:"sync_enabled"`
	SyncToken         string     `json:"-"`
	LastSyncedAt      *time.Time `json:"last_synced_at"`
	LastSyncStatus    string     `json:"last_sync_status"`
	LastSyncError     string     `json:"last_sync_error"`
	// LastSyncStats is the JSON-encoded ContactSyncStats of the last run. Syncs
	// run in the background and are polled for, so the counts must outlive the
	// request that started them.
	LastSyncStats string `json:"-"`
}

func (CardDAVConnection) TableName() string {
	return "carddav_connections"
}

// CardDAVContactLink records the state of one contact on the remote address
// book as of the last sync — the merge base that lets the sync engine tell
// which side changed. RemoteETag detects remote changes, LocalHash (hash of
// the vCard generated from the local contact) detects local changes.
type CardDAVContactLink struct {
	ID           uint       `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ConnectionID uint       `gorm:"not null;index;uniqueIndex:idx_carddav_links_conn_contact,priority:1;uniqueIndex:idx_carddav_links_conn_uid,priority:1" json:"connection_id"`
	UserID       uint       `gorm:"not null;index" json:"-"`
	ContactID    uint       `gorm:"not null;index;uniqueIndex:idx_carddav_links_conn_contact,priority:2" json:"contact_id"`
	RemoteUID    string     `gorm:"not null;uniqueIndex:idx_carddav_links_conn_uid,priority:2" json:"remote_uid"`
	RemotePath   string     `gorm:"not null" json:"remote_path"`
	RemoteETag   string     `gorm:"column:remote_etag" json:"remote_etag"`
	LocalHash    string     `json:"-"`
	SyncedAt     *time.Time `json:"synced_at"`
}

// TableName overrides GORM's default (card_dav_contact_links) to match the migration.
func (CardDAVContactLink) TableName() string {
	return "carddav_contact_links"
}

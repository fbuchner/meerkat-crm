package models

import "time"

// CardDAVSync tracks sync tokens for CardDAV clients
type CardDAVSync struct {
	ID           uint      `gorm:"primaryKey"`
	UserID       uint      `gorm:"not null;uniqueIndex"`
	SyncToken    string    `gorm:"not null"`
	LastModified time.Time `gorm:"not null"`
}

func (CardDAVSync) TableName() string {
	return "carddav_sync"
}

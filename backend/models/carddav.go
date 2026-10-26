package models

import (
	"slices"
	"strings"
	"time"
)

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

// Modes of the per-user circle filter on the built-in CardDAV server
const (
	CardDAVCircleModeAll     = "all"
	CardDAVCircleModeInclude = "include"
	CardDAVCircleModeExclude = "exclude"
)

// CardDAVCircleFilterMatches reports whether a contact in contactCircles is
// visible to CardDAV clients under the given filter. "include" shows contacts in
// at least one of filterCircles, "exclude" hides them, any other mode shows all.
// Circle names compare case-insensitively, as they do in validateUniqueCircles.
func CardDAVCircleFilterMatches(mode string, filterCircles, contactCircles []string) bool {
	if mode != CardDAVCircleModeInclude && mode != CardDAVCircleModeExclude {
		return true
	}
	inFilter := slices.ContainsFunc(contactCircles, func(c string) bool {
		return slices.ContainsFunc(filterCircles, func(f string) bool {
			return strings.EqualFold(strings.TrimSpace(c), f)
		})
	})
	return inFilter == (mode == CardDAVCircleModeInclude)
}

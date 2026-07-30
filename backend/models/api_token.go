package models

import (
	"time"

	"gorm.io/gorm"
)

type ApiToken struct {
	gorm.Model
	UserID     uint   `gorm:"not null"`
	Name       string `gorm:"not null"`
	TokenHash  string `gorm:"not null;unique" json:"-"`
	LastUsedAt *time.Time
	RevokedAt  *time.Time

	// ExpiresAt bounds how long a leaked token stays useful. NULL means no
	// expiry and exists only for rows predating this column; tokens created
	// through the API always get a concrete value.
	ExpiresAt *time.Time

	// Scope is "full" (default, near-full REST access) or "carddav" (only
	// usable against the CardDAV Basic-Auth path). "full" still works for
	// CardDAV too, since it already grants broader access than CardDAV
	// exposes -- the restriction is one-directional.
	Scope string `gorm:"not null;default:'full'"`
}

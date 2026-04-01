package models

import (
	"time"

	"gorm.io/gorm"
)

type ApiToken struct {
	gorm.Model
	UserID     uint       `gorm:"not null"`
	Name       string     `gorm:"not null"`
	TokenHash  string     `gorm:"not null;unique" json:"-"`
	LastUsedAt *time.Time
	RevokedAt  *time.Time
}

func (t *ApiToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

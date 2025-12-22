package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username                 string     `gorm:"unique" validate:"required,min=3,max=50,safe_string"`
	Password                 string     `validate:"required,min=8,strong_password"`
	Email                    string     `gorm:"unique" validate:"required,email"`
	PasswordResetTokenHash   *string    `gorm:"column:password_reset_token_hash"`
	PasswordResetExpiresAt   *time.Time `gorm:"column:password_reset_expires_at"`
	PasswordResetRequestedAt *time.Time `gorm:"column:password_reset_requested_at"`
}

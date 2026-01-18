package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username                 string     `gorm:"unique" validate:"required,min=3,max=50,safe_string,no_at_sign"`
	Password                 string     `validate:"required,min=8,strong_password"`
	Email                    string     `gorm:"unique" validate:"required,email"`
	Language                 string     `gorm:"default:'en'" json:"language" validate:"omitempty,oneof=en de"`
	PasswordResetTokenHash   *string    `gorm:"column:password_reset_token_hash"`
	PasswordResetExpiresAt   *time.Time `gorm:"column:password_reset_expires_at"`
	PasswordResetRequestedAt *time.Time `gorm:"column:password_reset_requested_at"`
}

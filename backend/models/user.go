package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username                 string     `gorm:"unique" validate:"required,min=1,max=50,no_at_sign"`
	Password                 string     `validate:"required,min=8,strong_password"`
	Email                    string     `gorm:"uniqueIndex:idx_users_email,where:email <> ''" validate:"required,email"` // empty for OIDC accounts whose provider sends none
	Language                 string     `gorm:"default:'en'" json:"language" validate:"omitempty,oneof=en de it es fr zh"`
	DateFormat               string     `gorm:"default:'eu'" json:"date_format" validate:"omitempty,oneof=eu us iso"`
	IsAdmin                  bool       `gorm:"default:false" json:"is_admin"`
	PasswordResetTokenHash   *string    `gorm:"column:password_reset_token_hash"`
	PasswordResetExpiresAt   *time.Time `gorm:"column:password_reset_expires_at"`
	PasswordResetRequestedAt *time.Time `gorm:"column:password_reset_requested_at"`
	PasswordChangedAt        *time.Time `gorm:"column:password_changed_at" json:"-"` // sessions signed in before this are not renewed
	CustomFieldNames         []string   `gorm:"type:text;serializer:json" json:"custom_field_names"`
	EnabledContactFields     []string   `gorm:"type:text;serializer:json" json:"enabled_contact_fields"`
	// Limits which contacts the built-in CardDAV server shows this user's devices.
	// See CardDAVCircleFilterMatches.
	CardDAVCircleMode      string   `gorm:"column:carddav_circle_mode;default:'all'" json:"carddav_circle_mode"`
	CardDAVCircles         []string `gorm:"column:carddav_circles;type:text;serializer:json" json:"carddav_circles"`
	HideEmptyContactFields bool     `gorm:"default:false" json:"hide_empty_contact_fields"`
	OIDCSubject            *string  `gorm:"column:oidc_subject"`
	OIDCProvider           *string  `gorm:"column:oidc_provider"`
}

// dummyPasswordHash lets a failed login for an account without a password hash cost the same
// as a wrong password, so timing does not reveal which accounts exist.
var dummyPasswordHash = func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("meerkat-timing-equalizer"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return hash
}()

// BurnPasswordCheck spends the cost of a password comparison and discards the result.
func BurnPasswordCheck(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
}

// CheckPassword compares password to the stored hash. OIDC-only accounts have no hash, and
// comparing against an empty one returns immediately, so they burn the same cost and always fail.
func (u User) CheckPassword(password string) error {
	if u.Password == "" {
		BurnPasswordCheck(password)
		return bcrypt.ErrMismatchedHashAndPassword
	}
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}

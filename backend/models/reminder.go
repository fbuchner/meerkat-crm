package models

import (
	"time"

	"gorm.io/gorm"
)

type Reminder struct {
	gorm.Model
	UserID                uint       `gorm:"not null;index" json:"-"`
	Message               string     `gorm:"not null type:text" json:"message" validate:"required,min=1,max=500"`
	ByMail                *bool      `gorm:"default:false" json:"by_mail"`
	RemindAt              time.Time  `gorm:"not null" json:"remind_at" validate:"required"`
	Recurrence            string     `gorm:"not null" json:"recurrence" validate:"required,oneof=once weekly monthly quarterly six-months yearly"`
	ReoccurFromCompletion *bool      `gorm:"default:true" json:"reoccur_from_completion"`
	Completed             bool       `gorm:"default:false" json:"completed"`
	EmailSent             bool       `gorm:"default:false" json:"email_sent"`
	LastSent              *time.Time `gorm:"default:null" json:"last_sent"`
	ContactID             *uint      `gorm:"not null" json:"contact_id" validate:"required"`
	Contact               Contact    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"contact,omitempty" validate:"-"`
}

type ReminderCompletion struct {
	gorm.Model
	UserID      uint      `gorm:"not null;index" json:"-"`
	ReminderID  *uint     `gorm:"index" json:"reminder_id,omitempty"`
	ContactID   uint      `gorm:"not null;index" json:"contact_id"`
	Message     string    `gorm:"not null;type:text" json:"message"`
	CompletedAt time.Time `gorm:"not null" json:"completed_at"`
}

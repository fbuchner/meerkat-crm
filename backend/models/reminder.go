package models

import (
	"time"

	"gorm.io/gorm"
)

type Reminder struct {
	gorm.Model
	Message               string     `gorm:"not null type:text" json:"message"`
	ByMail                bool       `gorm:"default:false" json:"by_mail"`
	RemindAt              time.Time  `gorm:"not null" json:"remind_at"`
	Recurrence            string     `gorm:"not null" json:"recurrence"`
	ReocurrFromCompletion bool       `gorm:"default:true" json:"reoccur_from_completion"`
	LastSent              *time.Time `gorm:"default:null" json:"last_sent"`
	ContactID             *uint      `gorm:"not null" json:"contact_id"`
	Contact               Contact    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"contact,omitempty"`
}

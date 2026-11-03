package models

import (
	"time"

	"gorm.io/gorm"
)

type Webhook struct {
	gorm.Model
	UserID   uint     `gorm:"not null;index"`
	Name     string   `gorm:"not null"`
	URL      string   `gorm:"not null"`
	Events   []string `gorm:"type:text;serializer:json"`
	Secret   string   `gorm:"not null"`
	IsActive bool     // no gorm default tag: GORM would replace false with it on create
}

type WebhookDelivery struct {
	gorm.Model
	WebhookID   uint   `gorm:"not null;index"`
	EventType   string `gorm:"not null"`
	Payload     string `gorm:"not null"`
	StatusCode  *int
	Error       *string
	Attempts    int `gorm:"default:1"`
	NextRetryAt *time.Time
}

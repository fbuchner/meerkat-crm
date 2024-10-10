package models

import (
	"gorm.io/gorm"
)

// Activity struct to represent shared activities with one or more contacts
type Activity struct {
	gorm.Model
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	Date        Date      `json:"date"`
	Contacts    []Contact `gorm:"foreignKey:ContactID" json:"contact,omitempty"`
}

// Note struct to represent notes attached to a contact
type Note struct {
	gorm.Model
	Content string  `json:"content"`
	Date    Date    `json:"date"`
	Contact Contact `gorm:"foreignKey:ContactID" json:"contact,omitempty"`
}

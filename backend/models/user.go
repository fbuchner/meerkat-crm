package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username string `gorm:"unique" validate:"required,min=3,max=50,safe_string"`
	Password string `validate:"required,min=8"`
	Email    string `gorm:"unique" validate:"required,email"`
}

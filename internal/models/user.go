package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
	Name     string `gorm:"not null"`
	Phone    string `gorm:"unique;not null"`
}

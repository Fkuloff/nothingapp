package models

import "gorm.io/gorm"

type Message struct {
	gorm.Model
	ChatID uint
	UserID uint
	Text   string `gorm:"not null"`
}

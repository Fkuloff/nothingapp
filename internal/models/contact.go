package models

import "gorm.io/gorm"

type Contact struct {
	ContactUser *User `gorm:"foreignKey:ContactUserID"`
	gorm.Model
	UserID        uint `gorm:"index"`
	ContactUserID uint `gorm:"index"`
}

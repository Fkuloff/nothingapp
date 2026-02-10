package models

import "gorm.io/gorm"

type Contact struct {
	gorm.Model
	UserID        uint `gorm:"index"`
	ContactUserID uint `gorm:"index"`

	ContactUser *User `gorm:"foreignKey:ContactUserID"`
}

package models

import "gorm.io/gorm"

// Contact represents a user-to-user contact relationship.
type Contact struct {
	gorm.Model
	UserID        uint `gorm:"index"`
	ContactUserID uint `gorm:"index"`

	ContactUser *User `gorm:"foreignKey:ContactUserID"`
}

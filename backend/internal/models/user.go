package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username  string  `gorm:"unique;not null"`
	Password  string  `gorm:"not null"`
	Name      string  `gorm:"not null"`
	AvatarURL *string `gorm:"type:varchar(500)"`
}

// GetDisplayName returns the display name for the user (Name if set, otherwise Username)
func (u *User) GetDisplayName() string {
	if u.Name != "" {
		return u.Name
	}
	return u.Username
}

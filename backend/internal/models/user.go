package models

import "gorm.io/gorm"

type User struct {
	AvatarURL *string `gorm:"type:varchar(500)"`
	gorm.Model
	Username string  `gorm:"unique;not null"`
	Password string  `gorm:"not null"`
	Name     string  `gorm:"not null"`
	Phone    *string `gorm:"type:varchar(20)"`
}

// GetDisplayName returns the display name for the user (Name if set, otherwise Username)
func (u *User) GetDisplayName() string {
	if u.Name != "" {
		return u.Name
	}
	return u.Username
}

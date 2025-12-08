package models

import "gorm.io/gorm"

type Chat struct {
	gorm.Model
	User1ID  uint
	User1    User `gorm:"foreignKey:User1ID"`
	User2ID  uint
	User2    User      `gorm:"foreignKey:User2ID"`
	Messages []Message `gorm:"foreignKey:ChatID"`
}

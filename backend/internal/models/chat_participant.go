package models

import "gorm.io/gorm"

type ParticipantRole string

const (
	RoleCreator ParticipantRole = "creator"
	RoleAdmin   ParticipantRole = "admin"
	RoleMember  ParticipantRole = "member"
)

type ChatParticipant struct {
	gorm.Model
	ChatID uint            `gorm:"uniqueIndex:idx_chat_user;index:idx_participant_chat;not null"`
	UserID uint            `gorm:"uniqueIndex:idx_chat_user;index:idx_participant_user;not null"`
	Role   ParticipantRole `gorm:"type:varchar(20);not null;default:'member'"`
	User   User            `gorm:"foreignKey:UserID"`
	Chat   Chat            `gorm:"foreignKey:ChatID"`
}

func (p *ChatParticipant) IsCreator() bool {
	return p.Role == RoleCreator
}

// IsAdmin returns true for both admin and creator roles.
func (p *ChatParticipant) IsAdmin() bool {
	return p.Role == RoleAdmin || p.Role == RoleCreator
}

package models

import "gorm.io/gorm"

// ParticipantRole defines a member's permission level in a group chat.
type ParticipantRole string

const (
	// RoleCreator is the user who created the group (highest privilege).
	RoleCreator ParticipantRole = "creator"
	// RoleAdmin can manage members and settings.
	RoleAdmin ParticipantRole = "admin"
	// RoleMember is a regular group participant.
	RoleMember ParticipantRole = "member"
)

// ChatParticipant represents a user's membership in a group chat.
type ChatParticipant struct {
	gorm.Model
	ChatID uint            `gorm:"uniqueIndex:idx_chat_user;index:idx_participant_chat;not null"`
	UserID uint            `gorm:"uniqueIndex:idx_chat_user;index:idx_participant_user;not null"`
	Role   ParticipantRole `gorm:"type:varchar(20);not null;default:'member'"`
	User   User            `gorm:"foreignKey:UserID"`
	Chat   Chat            `gorm:"foreignKey:ChatID"`
}

// IsCreator returns true if the participant has the creator role.
func (p *ChatParticipant) IsCreator() bool {
	return p.Role == RoleCreator
}

// IsAdmin returns true for both admin and creator roles.
func (p *ChatParticipant) IsAdmin() bool {
	return p.Role == RoleAdmin || p.Role == RoleCreator
}

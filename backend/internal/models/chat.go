package models

import "gorm.io/gorm"

type Chat struct {
	gorm.Model
	User1ID uint `gorm:"index:idx_chat_users,unique;index:idx_user1"`
	User2ID uint `gorm:"index:idx_chat_users,unique;index:idx_user2"`

	// Group chat fields
	IsGroup   bool    `gorm:"default:false;index:idx_is_group"`
	GroupName *string `gorm:"type:varchar(100)"`
	AvatarURL *string `gorm:"type:varchar(500)"`
	CreatorID *uint   `gorm:"index:idx_group_creator"`

	User1        User              `gorm:"foreignKey:User1ID;constraint:false"`
	User2        User              `gorm:"foreignKey:User2ID;constraint:false"`
	Messages     []Message         `gorm:"foreignKey:ChatID"`
	Participants []ChatParticipant `gorm:"foreignKey:ChatID"`
}

// BeforeCreate normalizes user IDs to prevent duplicate 1-on-1 chats.
// Skipped for group chats which use the participants table instead.
func (chat *Chat) BeforeCreate(tx *gorm.DB) error {
	if chat.IsGroup {
		return nil
	}
	// Always store smaller ID as User1ID
	if chat.User1ID > chat.User2ID {
		chat.User1ID, chat.User2ID = chat.User2ID, chat.User1ID
	}
	return nil
}

// HasUser checks if a user is a participant in this 1-on-1 chat.
// For group chats, always returns false — use the participant repository instead.
func (chat *Chat) HasUser(userID uint) bool {
	if chat.IsGroup {
		return false
	}
	return chat.User1ID == userID || chat.User2ID == userID
}

// GetOtherUser returns the other user in this chat and their ID
func (chat *Chat) GetOtherUser(currentUserID uint) (*User, uint) {
	if chat.User1ID == currentUserID {
		return &chat.User2, chat.User2ID
	}
	return &chat.User1, chat.User1ID
}

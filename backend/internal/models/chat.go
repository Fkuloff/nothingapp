package models

import "gorm.io/gorm"

// Chat represents a 1-on-1 or group conversation.
type Chat struct {
	gorm.Model
	User1ID *uint `gorm:"index:idx_chat_users,unique;index:idx_user1"`
	User2ID *uint `gorm:"index:idx_chat_users,unique;index:idx_user2"`

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
func (chat *Chat) BeforeCreate(_ *gorm.DB) error {
	if chat.IsGroup {
		return nil
	}
	// Always store smaller ID as User1ID
	if chat.User1ID != nil && chat.User2ID != nil && *chat.User1ID > *chat.User2ID {
		chat.User1ID, chat.User2ID = chat.User2ID, chat.User1ID
	}
	return nil
}

// GetUser1ID returns the User1ID value or 0 if nil.
func (chat *Chat) GetUser1ID() uint {
	if chat.User1ID != nil {
		return *chat.User1ID
	}
	return 0
}

// GetUser2ID returns the User2ID value or 0 if nil.
func (chat *Chat) GetUser2ID() uint {
	if chat.User2ID != nil {
		return *chat.User2ID
	}
	return 0
}

// HasUser checks if a user is a participant in this 1-on-1 chat.
// For group chats, always returns false — use the participant repository instead.
func (chat *Chat) HasUser(userID uint) bool {
	if chat.IsGroup {
		return false
	}
	return chat.GetUser1ID() == userID || chat.GetUser2ID() == userID
}

// GetGroupName returns the group name or empty string if unset.
func (chat *Chat) GetGroupName() string {
	if chat.GroupName != nil {
		return *chat.GroupName
	}
	return ""
}

// GetOtherUser returns the other user in this chat and their ID
func (chat *Chat) GetOtherUser(currentUserID uint) (*User, uint) {
	if chat.GetUser1ID() == currentUserID {
		return &chat.User2, chat.GetUser2ID()
	}
	return &chat.User1, chat.GetUser1ID()
}

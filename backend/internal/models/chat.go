package models

import "gorm.io/gorm"

type Chat struct {
	User1 User `gorm:"foreignKey:User1ID"`
	User2 User `gorm:"foreignKey:User2ID"`
	gorm.Model
	Messages []Message `gorm:"foreignKey:ChatID"`
	User1ID  uint      `gorm:"index:idx_chat_users,unique;index:idx_user1"`
	User2ID  uint      `gorm:"index:idx_chat_users,unique;index:idx_user2"`
}

// BeforeCreate normalizes user IDs to prevent duplicate chats
func (chat *Chat) BeforeCreate(tx *gorm.DB) error {
	// Always store smaller ID as User1ID
	if chat.User1ID > chat.User2ID {
		chat.User1ID, chat.User2ID = chat.User2ID, chat.User1ID
	}
	return nil
}

// HasUser checks if a user is a participant in this chat
func (chat *Chat) HasUser(userID uint) bool {
	return chat.User1ID == userID || chat.User2ID == userID
}

// GetOtherUser returns the other user in this chat and their ID
func (chat *Chat) GetOtherUser(currentUserID uint) (*User, uint) {
	if chat.User1ID == currentUserID {
		return &chat.User2, chat.User2ID
	}
	return &chat.User1, chat.User1ID
}

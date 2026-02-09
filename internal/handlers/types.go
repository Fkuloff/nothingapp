package handlers

import "time"

// MessageAction represents a WebSocket message action from the client
type MessageAction struct {
	Action    string `json:"action"`
	Text      string `json:"text"`
	ChatID    uint   `json:"chat_id"`
	ReplyToID uint   `json:"reply_to_id"`
	MessageID uint   `json:"message_id"`
}

// UserListItem represents a user in a list (contacts, search results)
type UserListItem struct {
	AvatarURL *string `json:"avatar_url"`
	Username  string  `json:"username"`
	Name      string  `json:"name"`
	ID        uint    `json:"id"`
}

// ChatListItem represents a chat in the chat list
type ChatListItem struct {
	UpdatedAt     time.Time `json:"updated_at"`
	AvatarURL     *string   `json:"avatar_url"`
	OtherUserName string    `json:"other_user_name"`
	LastMessage   string    `json:"last_message"`
	ID            uint      `json:"id"`
	OtherUserID   uint      `json:"other_user_id"`
	UnreadCount   int       `json:"unread_count"`
}

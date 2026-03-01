package handlers

import "time"

// messageAction represents a WebSocket message action from the client.
type messageAction struct {
	Action    string `json:"action"`
	Text      string `json:"text"`
	ChatID    uint   `json:"chat_id"`
	ReplyToID uint   `json:"reply_to_id"`
	MessageID uint   `json:"message_id"`
}

// callAction represents a WebSocket call signaling message (offer, answer, ICE, hangup, reject).
type callAction struct {
	Action    string `json:"action"`
	ChatID    uint   `json:"chat_id"`
	CallID    string `json:"call_id"`
	SDP       string `json:"sdp,omitempty"`
	SDPType   string `json:"sdp_type,omitempty"`
	Candidate string `json:"candidate,omitempty"`
}

// userListItem represents a user in a list (contacts, search results).
type userListItem struct {
	ID        uint    `json:"id"`
	Username  string  `json:"username"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url"`
}

// chatListItem represents a chat in the chat list (1-on-1 or group).
type chatListItem struct {
	ID          uint      `json:"id"`
	IsGroup     bool      `json:"is_group"`
	LastMessage string    `json:"last_message"`
	UnreadCount int       `json:"unread_count"`
	UpdatedAt   time.Time `json:"updated_at"`
	AvatarURL   *string   `json:"avatar_url"`

	// 1-on-1 fields (omitted for groups)
	OtherUserID   uint   `json:"other_user_id,omitempty"`
	OtherUserName string `json:"other_user_name,omitempty"`

	// Group fields (omitted for 1-on-1)
	GroupName   string `json:"group_name,omitempty"`
	MemberCount int    `json:"member_count,omitempty"`
}

// groupMemberItem represents a group participant for API responses.
type groupMemberItem struct {
	UserID    uint    `json:"user_id"`
	Username  string  `json:"username"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url"`
	Role      string  `json:"role"`
	IsOnline  bool    `json:"is_online"`
}

package handlers

import "time"

// messageAction represents a WebSocket message action from the client.
//
// Scheme + IV are populated only for client-side encrypted (E2E, scheme=2) messages.
// For legacy server-side encryption (scheme=1 or unset) Text is plaintext, IV is
// empty, and the server encrypts before persisting.
//
// Envelopes is populated for group scheme=2 messages — one entry per current
// participant (including the sender). When Envelopes is non-empty, Text/IV on
// this struct are empty by convention; the real ciphertexts live inside each
// envelope, addressed to a specific RecipientID.
type messageAction struct {
	Action    string                  `json:"action"`
	Text      string                  `json:"text"`
	ChatID    uint                    `json:"chat_id"`
	ReplyToID uint                    `json:"reply_to_id"`
	MessageID uint                    `json:"message_id"`
	Scheme    uint8                   `json:"scheme,omitempty"`
	IV        string                  `json:"iv,omitempty"`
	Envelopes []messageEnvelopeAction `json:"envelopes,omitempty"`
}

// messageEnvelopeAction is the WS-layer mirror of services.MessageEnvelopeInput.
// One per recipient for group scheme=2 sends/edits.
type messageEnvelopeAction struct {
	RecipientID uint   `json:"recipient_id"`
	Ciphertext  string `json:"ciphertext"`
	IV          string `json:"iv"`
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

	// Last-message E2E metadata. LastMessage is the server's best-effort
	// preview (plaintext for system messages, "🔒 placeholder" for scheme=2)
	// — fine as a fallback. For scheme=2 the client overrides it by
	// decrypting LastMessageCiphertext + LastMessageIV under the chat_key
	// derived from LastMessageSenderID's public_key.
	LastMessageScheme     uint8  `json:"last_message_scheme,omitempty"`
	LastMessageCiphertext string `json:"last_message_ciphertext,omitempty"`
	LastMessageIV         string `json:"last_message_iv,omitempty"`
	LastMessageSenderID   uint   `json:"last_message_sender_id,omitempty"`

	// 1-on-1 fields (omitted for groups)
	OtherUserID   uint   `json:"other_user_id,omitempty"`
	OtherUserName string `json:"other_user_name,omitempty"`

	// IsFavorites is true for the user's "Saved Messages" self-chat
	// (a 1-on-1 chat where user1_id == user2_id). Client uses it to render
	// a special title + icon and to hide the delete button.
	IsFavorites bool `json:"is_favorites,omitempty"`

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

package handlers

import (
	"errors"
	"time"
)

// errAccessDenied is a sentinel error used internally to signal that access check already wrote the HTTP response.
var errAccessDenied = errors.New("access denied")

// Message and chat constants.
const (
	maxMessageSize           = 10000               // 10KB max message size
	maxRecentMessages        = 100                 // Max messages to load in recent history
	maxChatListPreview       = 100                 // Max characters for last message preview
	maxConnectionsPerUser    = 5                   // Max WebSocket connections per user
	maxMessagesPerSecond     = 10                  // Rate limit: messages per second per connection
	maxAttachmentsPerMessage = 10                  // Max file attachments per message
	writeWait                = 10 * time.Second    // Time allowed to write a message
	pongWait                 = 60 * time.Second    // Time allowed to read the next pong message
	pingPeriod               = (pongWait * 9) / 10 // Send pings to peer with this period
)

// Multipart form parse limits.
const (
	multipartFormSizeAttachment = 512 << 20 // 512 MB
	multipartFormSizeAvatar     = 10 << 20  // 10 MB
)

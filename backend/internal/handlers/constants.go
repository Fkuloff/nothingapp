package handlers

import (
	"errors"
	"time"
)

// errAccessDenied is a sentinel error used internally to signal that access check already wrote the HTTP response.
var errAccessDenied = errors.New("access denied")

// Message and chat constants
const (
	MaxMessageSize           = 10000               // 10KB max message size
	MaxRecentMessages        = 100                 // Max messages to load in recent history
	MaxChatListPreview       = 100                 // Max characters for last message preview
	MaxConnectionsPerUser    = 5                   // Max WebSocket connections per user
	MaxMessagesPerSecond     = 10                  // Rate limit: messages per second per connection
	MaxAttachmentsPerMessage = 10                  // Max file attachments per message
	WriteWait                = 10 * time.Second    // Time allowed to write a message
	PongWait                 = 60 * time.Second    // Time allowed to read the next pong message
	PingPeriod               = (PongWait * 9) / 10 // Send pings to peer with this period
)

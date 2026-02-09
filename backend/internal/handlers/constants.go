package handlers

import "time"

// Message and chat constants
const (
	MaxMessageSize        = 10000               // 10KB max message size
	MaxRecentMessages     = 100                 // Max messages to load in recent history
	MaxChatListPreview    = 100                 // Max characters for last message preview
	ThumbnailWidth        = 300                 // Thumbnail width in pixels
	ThumbnailHeight       = 300                 // Thumbnail height in pixels
	MaxConnectionsPerUser = 5                   // Max WebSocket connections per user
	MaxMessagesPerSecond  = 10                  // Rate limit: messages per second per connection
	WriteWait             = 10 * time.Second    // Time allowed to write a message
	PongWait              = 60 * time.Second    // Time allowed to read the next pong message
	PingPeriod            = (PongWait * 9) / 10 // Send pings to peer with this period
)

// internal/handlers/websocket.go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"messenger/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocket constants are now defined in constants.go

type wsClient struct {
	lastMessage  time.Time
	conn         *websocket.Conn
	userID       uint
	messageCount int
	mu           sync.Mutex
}

// WebSocketHandler manages global WebSocket connections
type WebSocketHandler struct {
	chatService     *services.ChatService
	presenceService *services.PresenceService
	logger          *zap.Logger
	clients         map[uint][]*wsClient // userID -> []clients
	mu              sync.RWMutex
}

// NewWebSocketHandler creates a new global WebSocket handler
func NewWebSocketHandler(
	chatService *services.ChatService,
	presenceService *services.PresenceService,
	logger *zap.Logger,
) *WebSocketHandler {
	return &WebSocketHandler{
		chatService:     chatService,
		presenceService: presenceService,
		logger:          logger,
		clients:         make(map[uint][]*wsClient),
	}
}

// HandleWebSocket handles the global WebSocket connection for a user
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	// Connection limit: Users are limited to MaxConnectionsPerUser (5) simultaneous connections
	// to prevent resource abuse and ensure fair usage across all users
	h.mu.RLock()
	userConns := len(h.clients[userID])
	h.mu.RUnlock()

	if userConns >= MaxConnectionsPerUser {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many connections"})
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin:     h.checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("websocket upgrade error", zap.Error(err))
		return
	}

	// Configure connection limits
	conn.SetReadLimit(MaxMessageSize)
	if err := conn.SetReadDeadline(time.Now().Add(PongWait)); err != nil {
		h.logger.Warn("failed to set initial read deadline", zap.Error(err))
	}
	conn.SetPongHandler(func(string) error {
		if err := conn.SetReadDeadline(time.Now().Add(PongWait)); err != nil {
			h.logger.Warn("failed to set read deadline in pong handler", zap.Error(err))
		}
		// Update activity in presence service
		h.presenceService.UpdateActivity(userID)
		return nil
	})

	client := &wsClient{
		conn:         conn,
		userID:       userID,
		lastMessage:  time.Now(),
		messageCount: 0,
	}

	// Add client to map
	h.mu.Lock()
	h.clients[userID] = append(h.clients[userID], client)
	h.mu.Unlock()

	// Mark user as online
	h.presenceService.UserConnected(userID)

	// Send pending unread messages
	go h.sendPendingMessages(client, userID)

	// Start ping ticker
	ticker := time.NewTicker(PingPeriod)
	defer ticker.Stop()

	// Channel for graceful shutdown
	done := make(chan struct{})

	// Start write pump (ping sender)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error("panic in write pump",
					zap.Any("panic", r),
					zap.Uint("user_id", userID),
				)
			}
		}()
		for {
			select {
			case <-ticker.C:
				if err := conn.SetWriteDeadline(time.Now().Add(WriteWait)); err != nil {
					h.logger.Warn("failed to set write deadline for ping", zap.Error(err))
					return
				}
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	defer func() {
		close(done)
		h.removeClient(userID, client)
		h.presenceService.UserDisconnected(userID)
		if closeErr := conn.Close(); closeErr != nil {
			h.logger.Warn("failed to close websocket connection", zap.Error(closeErr), zap.Uint("user_id", userID))
		}
	}()

	// Helper function to send error to client
	sendError := func(errorMsg string) {
		errData := map[string]interface{}{
			"error": errorMsg,
		}
		errJSON, err := json.Marshal(errData)
		if err != nil {
			h.logger.Error("failed to marshal error message", zap.Error(err))
			return
		}
		if err := conn.SetWriteDeadline(time.Now().Add(WriteWait)); err != nil {
			h.logger.Warn("failed to set write deadline for error message", zap.Error(err))
		}
		if err := conn.WriteMessage(websocket.TextMessage, errJSON); err != nil {
			h.logger.Error("failed to send error message", zap.Error(err))
		}
	}

	// Main read loop with panic recovery
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("panic in websocket handler",
				zap.Any("panic", r),
				zap.Uint("user_id", userID),
			)
		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("websocket error",
					zap.Error(err),
					zap.Uint("user_id", userID),
				)
			}
			break
		}

		// Update activity
		h.presenceService.UpdateActivity(userID)

		// Rate limiting check
		if !h.checkRateLimit(client) {
			h.logger.Warn("rate limit exceeded",
				zap.Uint("user_id", userID),
			)
			sendError("Rate limit exceeded. Please slow down.")
			continue
		}

		var msgData struct {
			Action    string `json:"action"`
			Text      string `json:"text"`
			ChatID    uint   `json:"chat_id"`
			ReplyToID uint   `json:"reply_to_id"`
			MessageID uint   `json:"message_id"`
		}
		if err := json.Unmarshal(msg, &msgData); err != nil {
			h.logger.Error("invalid message format",
				zap.Error(err),
				zap.Uint("user_id", userID),
			)
			sendError("Invalid message format")
			continue
		}

		// Validate chat_id is provided
		if msgData.ChatID == 0 && msgData.Action != "mark_read" {
			sendError("chat_id is required")
			continue
		}

		// Default action is "send" for backward compatibility
		if msgData.Action == "" {
			msgData.Action = "send"
		}

		var handlerErr error
		switch msgData.Action {
		case "send":
			handlerErr = h.handleSendMessage(userID, msgData)
		case "edit":
			handlerErr = h.handleEditMessage(userID, msgData)
		case "delete":
			handlerErr = h.handleDeleteMessage(userID, msgData)
		case "mark_read":
			handlerErr = h.handleMarkRead(userID, msgData)
		default:
			handlerErr = &wsError{message: "Unknown action: " + msgData.Action}
		}

		if handlerErr != nil {
			sendError(handlerErr.Error())
		}
	}
}

// checkOrigin validates the WebSocket origin
func (h *WebSocketHandler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	host := r.Host
	return origin == "http://"+host || origin == "https://"+host
}

// checkRateLimit enforces rate limiting per connection
func (h *WebSocketHandler) checkRateLimit(client *wsClient) bool {
	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()
	if now.Sub(client.lastMessage) > time.Second {
		client.messageCount = 0
		client.lastMessage = now
	}

	if client.messageCount >= MaxMessagesPerSecond {
		return false
	}

	client.messageCount++
	return true
}

// removeClient safely removes a client from the map
func (h *WebSocketHandler) removeClient(userID uint, client *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients := h.clients[userID]
	for i, c := range clients {
		if c == client {
			h.clients[userID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}

	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
	}
}

// broadcastToUser sends a message to all connections of a specific user
func (h *WebSocketHandler) broadcastToUser(userID uint, msgJSON []byte) {
	h.mu.RLock()
	clientsToSend := h.clients[userID]
	h.mu.RUnlock()

	for _, client := range clientsToSend {
		go func(c *wsClient) {
			defer func() {
				if r := recover(); r != nil {
					h.logger.Error("panic in broadcast",
						zap.Any("panic", r),
						zap.Uint("user_id", userID),
					)
				}
			}()

			if err := c.conn.SetWriteDeadline(time.Now().Add(WriteWait)); err != nil {
				h.logger.Warn("failed to set write deadline",
					zap.Error(err),
					zap.Uint("user_id", userID),
				)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msgJSON); err != nil {
				h.logger.Error("websocket write error",
					zap.Error(err),
					zap.Uint("user_id", userID),
				)
			}
		}(client)
	}
}

// broadcastToChat sends a message to all participants in a chat
func (h *WebSocketHandler) broadcastToChat(chatID uint, msgJSON []byte, excludeUserID ...uint) error {
	// Get chat participants
	chat, err := h.chatService.FindChatByIDLight(context.Background(), chatID)
	if err != nil {
		return err
	}

	participants := []uint{chat.User1ID, chat.User2ID}

	// Send to each participant
	for _, userID := range participants {
		// Skip excluded users: When broadcasting, we may need to exclude specific users
		// (e.g., the message sender or users who have already received the message via another channel)
		skip := false
		for _, excludeID := range excludeUserID {
			if userID == excludeID {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Check if user is online before broadcasting
		if h.presenceService.IsUserOnline(userID) {
			h.broadcastToUser(userID, msgJSON)
		} else {
			// User is offline, we should save to unread messages (handled in send logic)
			h.logger.Info("user is offline, message saved to unread queue",
				zap.Uint("user_id", userID),
			)
		}
	}

	return nil
}

// sendPendingMessages sends all unread messages to a newly connected user
func (h *WebSocketHandler) sendPendingMessages(client *wsClient, userID uint) {
	unreadMessages, err := h.chatService.GetUnreadMessagesForUser(context.Background(), userID)
	if err != nil {
		h.logger.Error("failed to get pending messages",
			zap.Error(err),
			zap.Uint("user_id", userID),
		)
		return
	}

	if len(unreadMessages) == 0 {
		h.logger.Info("no pending messages for user",
			zap.Uint("user_id", userID),
		)
		return
	}

	h.logger.Info("sending pending messages to user",
		zap.Uint("user_id", userID),
		zap.Int("count", len(unreadMessages)),
	)

	// Send each unread message to the client
	for _, unread := range unreadMessages {
		if unread.Message.ID == 0 {
			continue
		}

		replyToIDVal := uint(0)
		if unread.Message.ReplyToID != nil {
			replyToIDVal = *unread.Message.ReplyToID
		}

		broadcastData := map[string]interface{}{
			"action":    "new",
			"chat_id":   unread.ChatID,
			"userID":    unread.Message.UserID,
			"text":      unread.Message.Text,
			"replyToID": replyToIDVal,
			"id":        unread.Message.ID,
		}

		msgJSON, err := json.Marshal(broadcastData)
		if err != nil {
			h.logger.Error("failed to marshal pending message",
				zap.Error(err),
				zap.Uint("message_id", unread.MessageID),
			)
			continue
		}

		if err := client.conn.SetWriteDeadline(time.Now().Add(WriteWait)); err != nil {
			h.logger.Warn("failed to set write deadline for pending message",
				zap.Error(err),
				zap.Uint("user_id", userID),
			)
			continue
		}
		if err := client.conn.WriteMessage(websocket.TextMessage, msgJSON); err != nil {
			h.logger.Error("failed to send pending message",
				zap.Error(err),
				zap.Uint("user_id", userID),
				zap.Uint("message_id", unread.MessageID),
			)
			// Don't return on error, try to send remaining messages
		}
	}
}

// wsError represents a WebSocket error
type wsError struct {
	message string
}

func (e *wsError) Error() string {
	return e.message
}

// GetUnreadMessagesAPI returns all unread messages for the current user
func (h *WebSocketHandler) GetUnreadMessagesAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	unreadMessages, err := h.chatService.GetUnreadMessagesForUser(context.Background(), userID)
	if err != nil {
		h.logger.Error("failed to get unread messages",
			zap.Error(err),
			zap.Uint("user_id", userID),
		)
		sendInternalError(c, "Failed to load unread messages")
		return
	}

	sendSuccess(c, gin.H{
		"unread_messages": unreadMessages,
	})
}

// GetUnreadCountsAPI returns unread message counts per chat
func (h *WebSocketHandler) GetUnreadCountsAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	counts, err := h.chatService.GetUnreadCounts(context.Background(), userID)
	if err != nil {
		h.logger.Error("failed to get unread counts",
			zap.Error(err),
			zap.Uint("user_id", userID),
		)
		sendInternalError(c, "Failed to load unread counts")
		return
	}

	sendSuccess(c, gin.H{
		"unread_counts": counts,
	})
}

// GetUserPresenceAPI returns online status of a user
func (h *WebSocketHandler) GetUserPresenceAPI(c *gin.Context) {
	_, ok := requireUserID(c)
	if !ok {
		return
	}

	targetUserID, err := parseUintParam(c, "user_id")
	if err != nil {
		sendBadRequest(c, "Invalid user ID")
		return
	}

	status := h.presenceService.GetUserStatus(targetUserID)

	sendSuccess(c, gin.H{
		"user_id":   status.UserID,
		"is_online": status.IsOnline,
		"last_seen": status.LastSeen,
	})
}

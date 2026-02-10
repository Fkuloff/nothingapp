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
	ctx          context.Context
	cancel       context.CancelFunc
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
	h.logger.Info("WebSocket handler called",
		zap.String("path", c.Request.URL.Path),
		zap.String("query", c.Request.URL.RawQuery),
	)

	userID, ok := requireUserID(c)
	if !ok {
		h.logger.Warn("WebSocket: no user_id in context")
		return
	}
	h.logger.Info("WebSocket: user authenticated", zap.Uint("user_id", userID))

	// Check connection limit
	if !h.checkConnectionLimit(c, userID) {
		return
	}

	// Upgrade to WebSocket
	conn, err := h.upgradeConnection(c)
	if err != nil {
		return
	}

	// Create context that will be canceled when connection closes
	ctx, cancel := context.WithCancel(context.Background())

	// Setup connection and client
	client := h.setupConnection(ctx, cancel, conn, userID)

	// Start background tasks
	done := h.startBackgroundTasks(client, userID)

	// Cleanup on exit
	defer h.cleanup(done, client, userID, conn)

	// Handle incoming messages
	h.handleReadLoop(client, userID, conn)
}

// checkConnectionLimit verifies the user hasn't exceeded connection limits
func (h *WebSocketHandler) checkConnectionLimit(c *gin.Context, userID uint) bool {
	h.mu.RLock()
	userConns := len(h.clients[userID])
	h.mu.RUnlock()

	if userConns >= MaxConnectionsPerUser {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many connections"})
		return false
	}
	return true
}

// upgradeConnection upgrades HTTP connection to WebSocket
func (h *WebSocketHandler) upgradeConnection(c *gin.Context) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin:     h.checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	h.logger.Info("attempting WebSocket upgrade",
		zap.String("origin", c.GetHeader("Origin")),
		zap.String("host", c.Request.Host),
	)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("websocket upgrade error",
			zap.Error(err),
			zap.String("origin", c.GetHeader("Origin")),
		)
		return nil, err
	}

	h.logger.Info("WebSocket upgrade successful")
	return conn, nil
}

// setupConnection configures the WebSocket connection and creates client
func (h *WebSocketHandler) setupConnection(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, userID uint) *wsClient {
	// Configure connection limits
	conn.SetReadLimit(MaxMessageSize)
	if err := conn.SetReadDeadline(time.Now().Add(PongWait)); err != nil {
		h.logger.Warn("failed to set initial read deadline", zap.Error(err))
	}
	conn.SetPongHandler(func(string) error {
		if err := conn.SetReadDeadline(time.Now().Add(PongWait)); err != nil {
			h.logger.Warn("failed to set read deadline in pong handler", zap.Error(err))
		}
		h.presenceService.UpdateActivity(userID)
		return nil
	})

	client := &wsClient{
		ctx:         ctx,
		cancel:      cancel,
		conn:        conn,
		userID:      userID,
		lastMessage: time.Now(),
	}

	// Add client to map
	h.mu.Lock()
	h.clients[userID] = append(h.clients[userID], client)
	h.mu.Unlock()

	// Mark user as online
	h.presenceService.UserConnected(userID)

	return client
}

// startBackgroundTasks starts write pump and pending message delivery
func (h *WebSocketHandler) startBackgroundTasks(client *wsClient, userID uint) chan struct{} {
	// Send pending unread messages
	go h.sendPendingMessages(client, userID)

	// Start ping ticker
	ticker := time.NewTicker(PingPeriod)
	done := make(chan struct{})

	// Start write pump (ping sender)
	go func() {
		defer ticker.Stop()
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error("panic in write pump",
					zap.Any("panic", r),
					zap.Uint("user_id", userID),
				)
			}
		}()
		h.writePump(client.conn, ticker, done)
	}()

	return done
}

// writePump sends periodic ping messages to keep connection alive
func (h *WebSocketHandler) writePump(conn *websocket.Conn, ticker *time.Ticker, done chan struct{}) {
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
}

// cleanup performs cleanup when connection closes
func (h *WebSocketHandler) cleanup(done chan struct{}, client *wsClient, userID uint, conn *websocket.Conn) {
	close(done)
	client.cancel() // Cancel context to stop any pending operations
	h.removeClient(userID, client)
	h.presenceService.UserDisconnected(userID)
	if closeErr := conn.Close(); closeErr != nil {
		h.logger.Warn("failed to close websocket connection", zap.Error(closeErr), zap.Uint("user_id", userID))
	}
}

// sendError sends error message to WebSocket client
func (h *WebSocketHandler) sendError(conn *websocket.Conn, errorMsg string) {
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

// handleReadLoop processes incoming WebSocket messages
func (h *WebSocketHandler) handleReadLoop(client *wsClient, userID uint, conn *websocket.Conn) {
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
			h.logger.Warn("rate limit exceeded", zap.Uint("user_id", userID))
			h.sendError(conn, "Rate limit exceeded. Please slow down.")
			continue
		}

		// Parse and handle message
		if err := h.processMessage(client.ctx, userID, conn, msg); err != nil {
			h.sendError(conn, err.Error())
		}
	}
}

// processMessage parses and routes a WebSocket message to appropriate handler
func (h *WebSocketHandler) processMessage(ctx context.Context, userID uint, conn *websocket.Conn, msg []byte) error {
	var msgData MessageAction
	if err := json.Unmarshal(msg, &msgData); err != nil {
		h.logger.Error("invalid message format", zap.Error(err), zap.Uint("user_id", userID))
		return &wsError{message: "Invalid message format"}
	}

	// Validate chat_id is provided
	if msgData.ChatID == 0 && msgData.Action != "mark_read" {
		return &wsError{message: "chat_id is required"}
	}

	// Default action is "send" for backward compatibility
	if msgData.Action == "" {
		msgData.Action = "send"
	}

	// Route to appropriate handler
	switch msgData.Action {
	case "send":
		return h.handleSendMessage(ctx, userID, msgData)
	case "edit":
		return h.handleEditMessage(ctx, userID, msgData)
	case "delete":
		return h.handleDeleteMessage(ctx, userID, msgData)
	case "mark_read":
		return h.handleMarkRead(ctx, userID, msgData)
	default:
		return &wsError{message: "Unknown action: " + msgData.Action}
	}
}

// checkOrigin validates the WebSocket origin
func (h *WebSocketHandler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	h.logger.Debug("WebSocket origin check",
		zap.String("origin", origin),
		zap.String("host", r.Host),
	)
	// In development, allow all origins
	// TODO: In production, restrict to specific origins
	return true
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
func (h *WebSocketHandler) broadcastToChat(ctx context.Context, chatID uint, msgJSON []byte, excludeUserID ...uint) error {
	// Get chat participants
	chat, err := h.chatService.FindChatByIDLight(ctx, chatID)
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
	unreadMessages, err := h.chatService.GetUnreadMessagesForUser(client.ctx, userID)
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
			"action":      "new",
			"chat_id":     unread.ChatID,
			"user_id":     unread.Message.UserID,
			"text":        unread.Message.Text,
			"reply_to_id": replyToIDVal,
			"id":          unread.Message.ID,
			"created_at":  unread.Message.CreatedAt,
			"is_deleted":  unread.Message.IsDeleted,
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

// Verify interface compliance at compile time.
var _ error = (*wsError)(nil) //nolint:errcheck // compile-time interface check

func (e *wsError) Error() string {
	return e.message
}

// GetUnreadMessagesAPI returns all unread messages for the current user
func (h *WebSocketHandler) GetUnreadMessagesAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	unreadMessages, err := h.chatService.GetUnreadMessagesForUser(c.Request.Context(), userID)
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

	counts, err := h.chatService.GetUnreadCounts(c.Request.Context(), userID)
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

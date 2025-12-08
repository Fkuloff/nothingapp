// internal/handlers/chat.go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	"messenger/internal/repositories"
	"messenger/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

type ChatHandler struct {
	chatService *services.ChatService
	userRepo    *repositories.UserRepo
	db          *gorm.DB
	clients     map[uint]map[uint][]*websocket.Conn // chatID -> userID -> []conns
	mu          sync.Mutex
}

func NewChatHandler(chatService *services.ChatService, userRepo *repositories.UserRepo, db *gorm.DB) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
		userRepo:    userRepo,
		db:          db,
		clients:     make(map[uint]map[uint][]*websocket.Conn),
	}
}

func (h *ChatHandler) ShowApp(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	userID := userIDInterface.(uint)

	chatIDStr := c.Query("chat_id")
	var chatID uint
	var hasChat bool
	if chatIDStr != "" {
		chatID64, err := strconv.ParseUint(chatIDStr, 10, 32)
		if err == nil {
			chatID = uint(chatID64)
			hasChat = true
		}
	}

	chats, err := h.chatService.GetUserChats(userID)
	if err != nil {
		log.Println("Error getting user chats:", err)
		c.HTML(http.StatusInternalServerError, "base.html", gin.H{
			"Page":   "app",
			"Title":  "Messenger",
			"error":  "Failed to load chats",
			"UserID": userID,
		})
		return
	}

	data := gin.H{
		"Page":   "app",
		"Title":  "Messenger",
		"Chats":  chats,
		"UserID": userID,
		"ChatID": chatID,
	}

	if hasChat {
		chat, err := h.chatService.FindChatByID(chatID)
		if err != nil || (chat.User1ID != userID && chat.User2ID != userID) {
			c.HTML(http.StatusForbidden, "base.html", gin.H{
				"Page":  "app",
				"Title": "Messenger",
				"error": "Access denied to this chat",
			})
			return
		}

		messages, err := h.chatService.GetMessages(chatID)
		if err != nil {
			log.Println("Error getting messages:", err)
			c.HTML(http.StatusInternalServerError, "base.html", gin.H{
				"Page":   "app",
				"Title":  "Messenger",
				"error":  "Failed to load messages",
				"UserID": userID,
			})
			return
		}

		var otherUsername string
		if chat.User1ID == userID {
			otherUsername = chat.User2.Name
			if otherUsername == "" {
				otherUsername = chat.User2.Username
			}
		} else {
			otherUsername = chat.User1.Name
			if otherUsername == "" {
				otherUsername = chat.User1.Username
			}
		}
		data["OtherUsername"] = otherUsername

		data["Messages"] = messages
		data["ChatID"] = chatID
	}

	c.HTML(http.StatusOK, "base.html", data)
}

func (h *ChatHandler) CreateChat(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	userID := userIDInterface.(uint)

	var req struct {
		OtherUsername string `form:"other_username" binding:"required"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.HTML(http.StatusBadRequest, "base.html", gin.H{
			"Page":  "app",
			"Title": "Messenger",
			"error": "Invalid input: " + err.Error(),
		})
		return
	}

	otherUser, err := h.userRepo.FindByUsername(req.OtherUsername)
	if err != nil {
		c.HTML(http.StatusBadRequest, "base.html", gin.H{
			"Page":  "app",
			"Title": "Messenger",
			"error": "User not found",
		})
		return
	}
	if otherUser.ID == userID {
		c.HTML(http.StatusBadRequest, "base.html", gin.H{
			"Page":  "app",
			"Title": "Messenger",
			"error": "Cannot create chat with yourself",
		})
		return
	}

	chat, err := h.chatService.CreateChat(userID, otherUser.ID)
	if err != nil {
		log.Println("Error creating chat:", err)
		c.HTML(http.StatusInternalServerError, "base.html", gin.H{
			"Page":  "app",
			"Title": "Messenger",
			"error": "Failed to create chat",
		})
		return
	}

	c.Redirect(http.StatusFound, "/?chat_id="+strconv.Itoa(int(chat.ID)))
}

func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDInterface.(uint)

	chatIDStr := c.Param("id")
	chatID64, err := strconv.ParseUint(chatIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}
	chatID := uint(chatID64)

	// Check access to chat
	chat, err := h.chatService.FindChatByID(chatID)
	if err != nil || (chat.User1ID != userID && chat.User2ID != userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Adjust for prod
		},
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	h.mu.Lock()
	if h.clients[chatID] == nil {
		h.clients[chatID] = make(map[uint][]*websocket.Conn)
	}
	h.clients[chatID][userID] = append(h.clients[chatID][userID], conn)
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		conns := h.clients[chatID][userID]
		for i, connItem := range conns {
			if connItem == conn {
				h.clients[chatID][userID] = append(conns[:i], conns[i+1:]...)
				break
			}
		}
		if len(h.clients[chatID][userID]) == 0 {
			delete(h.clients[chatID], userID)
		}
		if len(h.clients[chatID]) == 0 {
			delete(h.clients, chatID)
		}
		h.mu.Unlock()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket read error:", err)
			break
		}

		var msgData struct {
			Text      string `json:"text"`
			ReplyToID uint   `json:"reply_to_id"`
		}
		if err := json.Unmarshal(msg, &msgData); err != nil {
			log.Println("Invalid message format:", err)
			continue
		}

		message, err := h.chatService.SendMessage(chatID, userID, msgData.Text, msgData.ReplyToID)
		if err != nil {
			log.Println("Error sending message:", err)
			continue
		}

		replyToIDVal := uint(0)
		if message.ReplyToID != nil {
			replyToIDVal = *message.ReplyToID
		}

		broadcastData := map[string]interface{}{
			"userID":    userID,
			"text":      msgData.Text,
			"replyToID": replyToIDVal,
			"id":        message.ID,
		}
		msgJSON, err := json.Marshal(broadcastData)
		if err != nil {
			log.Println("JSON marshal error:", err)
			continue
		}

		h.mu.Lock()
		for _, conns := range h.clients[chatID] {
			for _, client := range conns {
				err := client.WriteMessage(websocket.TextMessage, msgJSON)
				if err != nil {
					log.Println("WebSocket write error:", err)
				}
			}
		}
		h.mu.Unlock()
	}
}

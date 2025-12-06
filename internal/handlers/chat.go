// internal/handlers/chat.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"messenger/internal/repositories"
	"messenger/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ChatHandler struct {
	chatService *services.ChatService
	userRepo    *repositories.UserRepo
	clients     map[uint]map[*websocket.Conn]bool // chatID -> connections
	mu          sync.Mutex
}

func NewChatHandler(chatService *services.ChatService, userRepo *repositories.UserRepo) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
		userRepo:    userRepo,
		clients:     make(map[uint]map[*websocket.Conn]bool),
	}
}

func (h *ChatHandler) ShowChats(c *gin.Context) {
	userIDStr := c.Query("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	chats, err := h.chatService.GetUserChats(uint(userID))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "base.html", gin.H{
			"Page":   "chats",
			"Title":  "Your Chats",
			"error":  err.Error(),
			"UserID": userID,
		})
		return
	}

	c.HTML(http.StatusOK, "base.html", gin.H{
		"Page":   "chats",
		"Title":  "Your Chats",
		"Chats":  chats,
		"UserID": userID,
	})
}

func (h *ChatHandler) CreateChat(c *gin.Context) {
	userIDStr := c.PostForm("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	otherUsername := c.PostForm("other_username")
	if otherUsername == "" {
		c.HTML(http.StatusBadRequest, "base.html", gin.H{
			"Page":   "chats",
			"Title":  "Your Chats",
			"error":  "Other username is required",
			"UserID": userID,
		})
		return
	}

	otherUser, err := h.userRepo.FindByUsername(otherUsername)
	if err != nil {
		c.HTML(http.StatusBadRequest, "base.html", gin.H{
			"Page":   "chats",
			"Title":  "Your Chats",
			"error":  "User not found",
			"UserID": userID,
		})
		return
	}

	chat, err := h.chatService.CreateChat(uint(userID), otherUser.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "base.html", gin.H{
			"Page":   "chats",
			"Title":  "Your Chats",
			"error":  err.Error(),
			"UserID": userID,
		})
		return
	}

	c.Redirect(http.StatusFound, "/chat/"+strconv.Itoa(int(chat.ID))+"?user_id="+strconv.Itoa(int(userID)))
}

func (h *ChatHandler) ShowChat(c *gin.Context) {
	userIDStr := c.Query("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	chatIDStr := c.Param("id")
	chatID, err := strconv.ParseUint(chatIDStr, 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "base.html", gin.H{
			"Page":   "chats",
			"Title":  "Your Chats",
			"error":  "Invalid chat ID",
			"UserID": userID,
		})
		return
	}

	chat, err := h.chatService.FindChatByID(uint(chatID))
	if err != nil || (chat.User1ID != uint(userID) && chat.User2ID != uint(userID)) {
		c.HTML(http.StatusForbidden, "base.html", gin.H{
			"Page":   "chats",
			"Title":  "Your Chats",
			"error":  "Access denied",
			"UserID": userID,
		})
		return
	}

	messages, err := h.chatService.GetMessages(uint(chatID))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "base.html", gin.H{
			"Page":   "chat",
			"Title":  "Chat",
			"error":  err.Error(),
			"UserID": userID,
		})
		return
	}

	c.HTML(http.StatusOK, "base.html", gin.H{
		"Page":     "chat",
		"Title":    "Chat",
		"Messages": messages,
		"ChatID":   chatID,
		"UserID":   userID,
		"Chat":     chat,
	})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Для MVP, в проде проверьте origin
	},
}

func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	chatIDStr := c.Param("id")
	chatID, err := strconv.ParseUint(chatIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat ID"})
		return
	}

	userIDStr := c.Query("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	chat, err := h.chatService.FindChatByID(uint(chatID))
	if err != nil || (chat.User1ID != uint(userID) && chat.User2ID != uint(userID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	h.mu.Lock()
	if h.clients[uint(chatID)] == nil {
		h.clients[uint(chatID)] = make(map[*websocket.Conn]bool)
	}
	h.clients[uint(chatID)][conn] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients[uint(chatID)], conn)
		h.mu.Unlock()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		text := string(msg)
		err = h.chatService.SendMessage(uint(chatID), uint(userID), text)
		if err != nil {
			continue
		}

		// Бродкаст в формате JSON
		msgData := map[string]interface{}{
			"userID": userID,
			"text":   text,
		}
		msgJSON, err := json.Marshal(msgData)
		if err != nil {
			continue
		}

		h.mu.Lock()
		for client := range h.clients[uint(chatID)] {
			err := client.WriteMessage(websocket.TextMessage, msgJSON)
			if err != nil {
				delete(h.clients[uint(chatID)], client)
				client.Close()
			}
		}
		h.mu.Unlock()
	}
}

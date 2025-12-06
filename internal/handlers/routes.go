// internal/handlers/routes.go
package handlers

import (
	"messenger/internal/repositories"
	"messenger/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB) {
	userRepo := repositories.NewUserRepo(db)
	chatRepo := repositories.NewChatRepo(db)
	messageRepo := repositories.NewMessageRepo(db)

	authService := services.NewAuthService(userRepo)
	chatService := services.NewChatService(chatRepo, messageRepo)

	authHandler := NewAuthHandler(authService)
	chatHandler := NewChatHandler(chatService, userRepo)

	// HTML routes
	router.GET("/register", authHandler.ShowRegister)
	router.POST("/register", authHandler.Register)
	router.GET("/login", authHandler.ShowLogin)
	router.POST("/login", authHandler.Login)
	router.GET("/logout", authHandler.Logout)
	router.GET("/chats", chatHandler.ShowChats)
	router.POST("/chats", chatHandler.CreateChat)
	router.GET("/chat/:id", chatHandler.ShowChat)

	// WebSocket route
	router.GET("/ws/chat/:id", chatHandler.HandleWebSocket)
}

// internal/handlers/routes.go
package handlers

import (
	"messenger/internal/repositories"
	"messenger/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(router *gin.Engine, db *gorm.DB, secret []byte) {
	userRepo := repositories.NewUserRepo(db)
	chatRepo := repositories.NewChatRepo(db)
	messageRepo := repositories.NewMessageRepo(db)
	contactRepo := repositories.NewContactRepo(db)

	authService := services.NewAuthService(userRepo)
	chatService := services.NewChatService(chatRepo, messageRepo)
	contactService := services.NewContactService(contactRepo)

	authHandler := NewAuthHandler(authService, secret)
	chatHandler := NewChatHandler(chatService, userRepo, db)
	profileHandler := NewProfileHandler(userRepo, contactRepo, contactService, db)

	// HTML routes
	router.GET("/register", authHandler.ShowRegister)
	router.POST("/register", authHandler.Register)
	router.GET("/login", authHandler.ShowLogin)
	router.POST("/login", authHandler.Login)
	router.GET("/logout", authHandler.Logout)
	router.GET("/", chatHandler.ShowApp)
	router.POST("/chats", chatHandler.CreateChat)

	// Profile routes
	router.GET("/profile/:user_id", profileHandler.ShowProfile)
	router.GET("/profile", profileHandler.ShowProfile) // Own profile
	router.POST("/contacts/add/:user_id", profileHandler.AddToContacts)

	router.GET("/ws/chat/:id", chatHandler.HandleWebSocket)
}

// internal/handlers/routes.go
package handlers

import (
	"messenger/internal/repositories"
	"messenger/internal/services"
	"messenger/internal/storage"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func SetupRoutes(
	router *gin.Engine,
	db *gorm.DB,
	secret []byte,
	fileStorage storage.Storage,
	logger *zap.Logger,
	uploadsPath string,
) error {
	// Initialize repositories
	userRepo := repositories.NewUserRepo(db)
	chatRepo := repositories.NewChatRepo(db)
	messageRepo := repositories.NewMessageRepo(db)
	contactRepo := repositories.NewContactRepo(db)
	attachmentRepo := repositories.NewAttachmentRepo(db)
	unreadMessageRepo := repositories.NewUnreadMessageRepo(db)

	// Initialize services
	authService := services.NewAuthService(logger, userRepo)
	chatService := services.NewChatService(logger, chatRepo, messageRepo, unreadMessageRepo)
	contactService := services.NewContactService(logger, contactRepo)
	attachmentService := services.NewAttachmentService(logger, attachmentRepo, messageRepo, fileStorage)
	userService := services.NewUserService(logger, userRepo, fileStorage)
	presenceService := services.NewPresenceService(logger)

	// Initialize handlers
	authHandler := NewAuthHandler(authService, secret)
	chatHandler := NewChatHandler(chatService, userService)
	profileHandler := NewProfileHandler(userService, contactService, logger)
	attachmentHandler := NewAttachmentHandler(attachmentService, chatService)
	userHandler := NewUserHandler(userService)
	wsHandler := NewWebSocketHandler(
		chatService,
		presenceService,
		logger,
	)
	fileHandler := NewFileHandler(fileStorage, logger)
	healthHandler := NewHealthHandler(db)

	// Health check endpoint (before JWT middleware)
	router.GET("/health", healthHandler.GetHealth)

	// Serve static uploads (before JWT middleware for public access)
	router.Static("/uploads", uploadsPath)

	// Apply JWT middleware globally
	router.Use(JWTMiddleware(secret, logger))

	// WebSocket endpoint (global, not per-chat)
	router.GET("/ws", wsHandler.HandleWebSocket)

	// API routes
	api := router.Group("/api")
	{
		// Auth routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.RegisterAPI)
			auth.POST("/login", authHandler.LoginAPI)
			auth.POST("/logout", authHandler.LogoutAPI)
			auth.GET("/me", authHandler.GetCurrentUser)
		}

		// Chat routes
		chats := api.Group("/chats")
		{
			chats.GET("", chatHandler.ListChatsAPI)
			chats.POST("", chatHandler.CreateChatAPI)
			chats.GET("/:id", chatHandler.GetChatData)
			chats.GET("/:id/messages", chatHandler.GetChatMessagesAPI)

			// Attachment routes (nested under chats)
			chats.POST("/:chatId/messages/:messageId/attachments", attachmentHandler.UploadAttachments)
		}

		// Attachment routes
		attachments := api.Group("/attachments")
		{
			attachments.GET("/:id", attachmentHandler.DownloadAttachment)
			attachments.GET("/:id/thumbnail", attachmentHandler.GetThumbnail)
			attachments.DELETE("/:id", attachmentHandler.DeleteAttachment)
		}

		// Profile and contacts routes
		profile := api.Group("/profile")
		{
			profile.GET("", profileHandler.GetProfileAPI)
			profile.GET("/:user_id", profileHandler.GetProfileAPI)
		}

		contacts := api.Group("/contacts")
		{
			contacts.GET("", profileHandler.GetContacts)
			contacts.POST("/add/:user_id", profileHandler.AddContactAPI)
			contacts.POST("/remove/:user_id", profileHandler.RemoveContactAPI)
		}

		// User routes
		user := api.Group("/user")
		{
			user.POST("/avatar", userHandler.UploadAvatar)
			user.DELETE("/avatar", userHandler.DeleteAvatar)
		}

		// Search routes
		api.GET("/users/search", profileHandler.SearchUsers)

		// Unread messages routes
		unread := api.Group("/unread")
		{
			unread.GET("", wsHandler.GetUnreadMessagesAPI)
			unread.GET("/counts", wsHandler.GetUnreadCountsAPI)
		}

		// Presence routes
		presence := api.Group("/presence")
		{
			presence.GET("/:user_id", wsHandler.GetUserPresenceAPI)
		}

		// File serving with authorization
		api.GET("/files/:filename", fileHandler.ServeFile)
	}

	return nil
}

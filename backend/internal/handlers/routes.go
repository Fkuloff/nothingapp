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
	wsHandler := NewWebSocketHandler(chatService, presenceService, logger)
	fileHandler := NewFileHandler(fileStorage, logger)
	healthHandler := NewHealthHandler(db)

	// Public endpoints (before JWT middleware)
	router.GET("/health", healthHandler.GetHealth)
	router.Static("/uploads", uploadsPath)

	// Apply JWT middleware globally
	router.Use(JWTMiddleware(secret, logger))
	router.GET("/ws", wsHandler.HandleWebSocket)

	// API routes
	api := router.Group("/api")
	registerAuthRoutes(api, authHandler)
	registerChatRoutes(api, chatHandler, attachmentHandler)
	registerProfileRoutes(api, profileHandler)
	registerUserRoutes(api, userHandler, wsHandler, fileHandler)

	return nil
}

func registerAuthRoutes(api *gin.RouterGroup, h *AuthHandler) {
	auth := api.Group("/auth")
	auth.POST("/register", h.RegisterAPI)
	auth.POST("/login", h.LoginAPI)
	auth.POST("/logout", h.LogoutAPI)
	auth.GET("/me", h.GetCurrentUser)
}

func registerChatRoutes(api *gin.RouterGroup, chatHandler *ChatHandler, attachmentHandler *AttachmentHandler) {
	chats := api.Group("/chats")
	chats.GET("", chatHandler.ListChatsAPI)
	chats.POST("", chatHandler.CreateChatAPI)
	chats.GET("/:id", chatHandler.GetChatData)
	chats.GET("/:id/messages", chatHandler.GetChatMessagesAPI)
	chats.POST("/:chat_id/messages/:message_id/attachments", attachmentHandler.UploadAttachments)

	attachments := api.Group("/attachments")
	attachments.GET("/:id", attachmentHandler.DownloadAttachment)
	attachments.GET("/:id/thumbnail", attachmentHandler.GetThumbnail)
	attachments.DELETE("/:id", attachmentHandler.DeleteAttachment)
}

func registerProfileRoutes(api *gin.RouterGroup, h *ProfileHandler) {
	profile := api.Group("/profile")
	profile.GET("", h.GetProfileAPI)
	profile.GET("/:user_id", h.GetProfileAPI)

	contacts := api.Group("/contacts")
	contacts.GET("", h.GetContacts)
	contacts.POST("/:user_id", h.AddContactAPI)
	contacts.DELETE("/:user_id", h.RemoveContactAPI)

	api.GET("/users/search", h.SearchUsers)
}

func registerUserRoutes(api *gin.RouterGroup, userHandler *UserHandler, wsHandler *WebSocketHandler, fileHandler *FileHandler) {
	user := api.Group("/user")
	user.POST("/avatar", userHandler.UploadAvatar)
	user.DELETE("/avatar", userHandler.DeleteAvatar)

	unread := api.Group("/unread")
	unread.GET("", wsHandler.GetUnreadMessagesAPI)
	unread.GET("/counts", wsHandler.GetUnreadCountsAPI)

	presence := api.Group("/presence")
	presence.GET("/:user_id", wsHandler.GetUserPresenceAPI)

	api.GET("/files/:filename", fileHandler.ServeFile)
}

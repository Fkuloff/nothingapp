// internal/handlers/routes.go
package handlers

import (
	"messenger/internal/config"
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
	cfg *config.Config,
) error {
	// Initialize repositories
	userRepo := repositories.NewUserRepo(db)
	chatRepo := repositories.NewChatRepo(db)
	messageRepo := repositories.NewMessageRepo(db)
	contactRepo := repositories.NewContactRepo(db)
	attachmentRepo := repositories.NewAttachmentRepo(db)
	unreadMessageRepo := repositories.NewUnreadMessageRepo(db)
	pushSubRepo := repositories.NewPushSubscriptionRepo(db)
	userKeyRepo := repositories.NewUserKeyRepo(db)
	keyBackupRepo := repositories.NewKeyBackupRepo(db)

	// Initialize services
	authService := services.NewAuthService(logger, userRepo)
	chatService := services.NewChatService(db, logger, chatRepo, messageRepo, unreadMessageRepo)
	contactService := services.NewContactService(logger, contactRepo)
	attachmentService := services.NewAttachmentService(logger, attachmentRepo, messageRepo, fileStorage)
	userService := services.NewUserService(logger, userRepo, fileStorage)
	presenceService := services.NewPresenceService(logger)
	pushService := services.NewPushNotificationService(logger, pushSubRepo, cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject)
	keyService := services.NewKeyService(logger, userKeyRepo, keyBackupRepo)

	// Initialize handlers
	authHandler := NewAuthHandler(authService, userService, secret)
	chatHandler := NewChatHandler(chatService, userService)
	profileHandler := NewProfileHandler(userService, contactService, logger)
	attachmentHandler := NewAttachmentHandler(attachmentService, chatService)
	userHandler := NewUserHandler(userService)
	wsHandler := NewWebSocketHandler(chatService, presenceService, pushService, userService, logger)
	fileHandler := NewFileHandler(fileStorage, logger)
	pushHandler := NewPushHandler(pushService, logger)
	keyHandler := NewKeyHandler(keyService)
	healthHandler := NewHealthHandler(db)

	// Configure presence service to broadcast status changes via WebSocket
	presenceService.SetOnChangeCallback(wsHandler.broadcastPresenceChange)

	// Public endpoints (no JWT middleware)
	router.GET("/health", healthHandler.GetHealth)
	router.POST("/api/auth/register", authHandler.RegisterAPI)
	router.POST("/api/auth/login", authHandler.LoginAPI)

	// Public attachment endpoints (GET - no JWT, files are publicly accessible)
	router.GET("/api/attachments/:id", attachmentHandler.DownloadAttachment)

	// Public avatar endpoint
	router.GET("/api/avatars/:user_id", userHandler.GetAvatar)

	// WebSocket endpoint with JWT middleware
	router.GET("/ws", JWTMiddleware(secret, logger), wsHandler.HandleWebSocket)

	// Protected attachment endpoint (DELETE requires JWT)
	router.DELETE("/api/attachments/:id", JWTMiddleware(secret, logger), attachmentHandler.DeleteAttachment)

	// Protected API routes (JWT required)
	api := router.Group("/api")
	api.Use(JWTMiddleware(secret, logger))
	registerAuthRoutes(api, authHandler)
	registerChatRoutes(api, chatHandler, attachmentHandler)
	registerProfileRoutes(api, profileHandler)
	registerUserRoutes(api, userHandler, wsHandler, fileHandler)
	registerPushRoutes(api, pushHandler)
	registerKeyRoutes(api, keyHandler)

	return nil
}

func registerAuthRoutes(api *gin.RouterGroup, h *AuthHandler) {
	auth := api.Group("/auth")
	// register and login are public (registered before JWT middleware)
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

func registerKeyRoutes(api *gin.RouterGroup, h *KeyHandler) {
	keys := api.Group("/keys")
	keys.PUT("", h.UploadPublicKey)
	keys.GET("/:user_id", h.GetPublicKey)
	keys.PUT("/backup", h.SaveKeyBackup)
	keys.GET("/backup", h.GetKeyBackup)
	keys.DELETE("/backup", h.DeleteKeyBackup)
}

func registerPushRoutes(api *gin.RouterGroup, h *PushHandler) {
	push := api.Group("/push")
	push.GET("/vapid-key", h.GetVAPIDKey)
	push.POST("/subscribe", h.Subscribe)
	push.POST("/unsubscribe", h.Unsubscribe)
	push.GET("/status", h.GetStatus)
}

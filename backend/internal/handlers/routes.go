// internal/handlers/routes.go
package handlers

import (
	"messenger/internal/config"
	"messenger/internal/crypto"
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
	msgEncryptor *crypto.MessageEncryptor,
) error {
	// Initialize repositories
	userRepo := repositories.NewUserRepo(db)
	chatRepo := repositories.NewChatRepo(db)
	messageRepo := repositories.NewMessageRepo(db)
	contactRepo := repositories.NewContactRepo(db)
	attachmentRepo := repositories.NewAttachmentRepo(db)
	unreadMessageRepo := repositories.NewUnreadMessageRepo(db)
	pushSubRepo := repositories.NewPushSubscriptionRepo(db)
	participantRepo := repositories.NewChatParticipantRepo(db)

	// Initialize services
	authService := services.NewAuthService(logger, userRepo)
	chatService := services.NewChatService(db, logger, chatRepo, participantRepo, messageRepo, unreadMessageRepo, fileStorage, msgEncryptor)
	contactService := services.NewContactService(logger, contactRepo)
	attachmentService := services.NewAttachmentService(logger, attachmentRepo, messageRepo, fileStorage)
	userService := services.NewUserService(logger, userRepo, fileStorage)
	presenceService := services.NewPresenceService(logger)
	pushService := services.NewPushNotificationService(logger, pushSubRepo, cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject)
	groupService := services.NewGroupService(db, logger, chatRepo, participantRepo, messageRepo, unreadMessageRepo, userRepo, fileStorage)

	// Initialize handlers
	authHandler := NewAuthHandler(authService, userService, secret)
	chatHandler := NewChatHandler(chatService, userService, fileStorage)
	chatHandler.SetGroupService(groupService)
	profileHandler := NewProfileHandler(userService, contactService, logger)
	userHandler := NewUserHandler(userService)
	wsHandler := NewWebSocketHandler(chatService, presenceService, pushService, userService, logger, msgEncryptor, fileStorage)
	wsHandler.SetGroupService(groupService, participantRepo)
	attachmentHandler := NewAttachmentHandler(attachmentService, chatService, wsHandler, participantRepo, fileStorage)
	fileHandler := NewFileHandler(fileStorage, logger)
	pushHandler := NewPushHandler(pushService, logger)
	healthHandler := NewHealthHandler(db)
	groupHandler := NewGroupHandler(groupService, presenceService, userService)

	// Configure presence service to broadcast status changes via WebSocket
	presenceService.SetOnChangeCallback(wsHandler.broadcastPresenceChange)

	// Configure chat handler to broadcast chat events (clear/delete) via WebSocket
	chatHandler.SetOnChatEventCallback(wsHandler.broadcastChatEvent)

	// Configure group handler to broadcast group events via WebSocket
	groupHandler.SetOnGroupEventCallback(wsHandler.broadcastGroupEvent)

	// Public endpoints (no JWT middleware)
	router.GET("/health", healthHandler.GetHealth)
	router.POST("/api/auth/register", authHandler.RegisterAPI)
	router.POST("/api/auth/login", authHandler.LoginAPI)

	// Attachment download — JWT required (presigned URLs are the primary access method)
	router.GET("/api/attachments/:id", JWTMiddleware(secret, logger), attachmentHandler.DownloadAttachment)

	// Public avatar endpoints
	router.GET("/api/avatars/:user_id", userHandler.GetAvatar)
	router.GET("/api/group-avatars/:chat_id", groupHandler.GetGroupAvatar)

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
	registerGroupRoutes(api, groupHandler)

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
	chats.DELETE("/:id", chatHandler.DeleteChatAPI)
	chats.POST("/:id/clear", chatHandler.ClearChatAPI)
	chats.GET("/:id/messages", chatHandler.GetChatMessagesAPI)
	chats.POST("/:id/messages/:message_id/attachments", attachmentHandler.UploadAttachments)
}

func registerProfileRoutes(api *gin.RouterGroup, h *ProfileHandler) {
	profile := api.Group("/profile")
	profile.GET("", h.GetProfileAPI)
	profile.PUT("", h.UpdateProfileAPI)
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

func registerPushRoutes(api *gin.RouterGroup, h *PushHandler) {
	push := api.Group("/push")
	push.GET("/vapid-key", h.GetVAPIDKey)
	push.POST("/subscribe", h.Subscribe)
	push.POST("/unsubscribe", h.Unsubscribe)
	push.GET("/status", h.GetStatus)
}

func registerGroupRoutes(api *gin.RouterGroup, h *GroupHandler) {
	groups := api.Group("/groups")
	groups.POST("", h.CreateGroupAPI)
	groups.GET("/:id", h.GetGroupInfoAPI)
	groups.PUT("/:id", h.UpdateGroupInfoAPI)
	groups.DELETE("/:id", h.DeleteGroupAPI)
	groups.POST("/:id/avatar", h.UploadGroupAvatarAPI)
	groups.DELETE("/:id/avatar", h.DeleteGroupAvatarAPI)
	groups.POST("/:id/members", h.AddMembersAPI)
	groups.DELETE("/:id/members/:user_id", h.RemoveMemberAPI)
	groups.POST("/:id/leave", h.LeaveGroupAPI)
	groups.PUT("/:id/members/:user_id/role", h.ChangeRoleAPI)
}

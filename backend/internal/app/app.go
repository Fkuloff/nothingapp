package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"messenger/internal/config"
	"messenger/internal/crypto"
	"messenger/internal/handlers"
	"messenger/internal/logger"
	"messenger/internal/models"
	"messenger/internal/shutdown"
	"messenger/internal/storage"

	ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// Infrastructure constants.
const (
	defaultPort          = "8080"
	readHeaderTimeout    = 30 * time.Second
	shutdownTimeout      = 30 * time.Second
	dbMaxIdleConns       = 50
	dbMaxOpenConns       = 200
	dbConnMaxLifetime    = time.Hour
	dbConnMaxIdleTime    = 10 * time.Minute
	dbSlowQueryThreshold = 200 * time.Millisecond
	rateLimitPerSecond   = 30
)

// Run initializes all dependencies and starts the HTTP server with graceful shutdown.
func Run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Initialize logger
	log := logger.MustNew(os.Getenv("DEBUG") == "true")
	defer log.Sync()

	// Initialize message encryptor
	encryptor, err := crypto.NewMessageEncryptor(cfg.MessageEncryptionKey)
	if err != nil {
		return fmt.Errorf("initialize message encryptor: %w", err)
	}

	// Initialize storage
	fileStorage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("initialize storage: %w", err)
	}

	// Database connection
	db, err := initDatabase(cfg.DBURL, log)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}

	// Run migrations
	if err := runMigrations(db, log); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	// Initialize Gin router
	router := setupRouter(log)

	// Setup routes with dependencies
	if err := handlers.SetupRoutes(router, db, []byte(cfg.JWTSecret), fileStorage, log, cfg, encryptor); err != nil {
		return fmt.Errorf("setup routes: %w", err)
	}

	// HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	// Graceful shutdown
	shutdownMgr := shutdown.New(log, shutdownTimeout)
	shutdownMgr.Register(func(ctx context.Context) error {
		log.Info("stopping HTTP server...")
		return srv.Shutdown(ctx)
	})
	shutdownMgr.Register(func(ctx context.Context) error {
		log.Info("closing database...")
		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("get database instance: %w", err)
		}
		return sqlDB.Close()
	})

	// Start server in goroutine
	go func() {
		log.Info("starting server", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	shutdownMgr.Wait()
	log.Info("server stopped")
	return nil
}

func initDatabase(dbURL string, log *zap.Logger) (*gorm.DB, error) {
	gormConfig := &gorm.Config{
		Logger: gormLogger.New(
			&zapGormLogger{log: log},
			gormLogger.Config{
				SlowThreshold:             dbSlowQueryThreshold,
				LogLevel:                  gormLogger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
		PrepareStmt:                              true,
		QueryFields:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	log.Info("connecting to database...")
	db, err := gorm.Open(postgres.Open(dbURL), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(dbMaxIdleConns)
	sqlDB.SetMaxOpenConns(dbMaxOpenConns)
	sqlDB.SetConnMaxLifetime(dbConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(dbConnMaxIdleTime)

	log.Info("database connection pool configured")
	return db, nil
}

type zapGormLogger struct {
	log *zap.Logger
}

func (l *zapGormLogger) Printf(format string, args ...interface{}) {
	l.log.Sugar().Infof(format, args...)
}

func runMigrations(db *gorm.DB, log *zap.Logger) error {
	log.Info("running database migrations...")

	// Drop legacy FK constraints on chats.user1_id/user2_id BEFORE AutoMigrate.
	// Group chats use chat_participants and leave these fields as NULL, which breaks
	// FK constraints. Must drop first so AutoMigrate doesn't fail trying to re-add them.
	if err := db.Exec("ALTER TABLE chats DROP CONSTRAINT IF EXISTS fk_chats_user1").Error; err != nil {
		log.Warn("failed to drop fk_chats_user1 constraint", zap.Error(err))
	}
	if err := db.Exec("ALTER TABLE chats DROP CONSTRAINT IF EXISTS fk_chats_user2").Error; err != nil {
		log.Warn("failed to drop fk_chats_user2 constraint", zap.Error(err))
	}

	// Migrate group chats: set user1_id/user2_id from 0 to NULL.
	// With *uint fields, NULL values don't violate the idx_chat_users unique index,
	// allowing multiple group chats to coexist.
	if err := db.Exec("UPDATE chats SET user1_id = NULL, user2_id = NULL WHERE is_group = true AND (user1_id = 0 OR user2_id = 0)").Error; err != nil {
		log.Warn("failed to nullify group chat user IDs", zap.Error(err))
	}

	return db.AutoMigrate(
		&models.User{},
		&models.Chat{},
		&models.Message{},
		&models.Contact{},
		&models.Attachment{},
		&models.UnreadMessage{},
		&models.PushSubscription{},
		&models.ChatParticipant{},
		&models.PinnedMessage{},
	)
}

func setupRouter(log *zap.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Custom logger middleware using zap
	router.Use(ginZapLogger(log))
	router.Use(gin.Recovery())

	// CORS
	allowedOrigins := []string{"http://localhost:8080", "http://localhost:5173", "http://127.0.0.1:5173"}
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		allowedOrigins = strings.Split(origins, ",")
	}
	log.Info("CORS allowed origins", zap.Strings("origins", allowedOrigins))

	router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Rate limiting
	store := ratelimit.InMemoryStore(&ratelimit.InMemoryOptions{
		Rate:  time.Second,
		Limit: rateLimitPerSecond,
	})
	rateLimiterMiddleware := ratelimit.RateLimiter(store, &ratelimit.Options{
		ErrorHandler: func(c *gin.Context, info ratelimit.Info) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
		},
		KeyFunc: func(c *gin.Context) string {
			return c.ClientIP()
		},
	})

	router.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/ws") {
			c.Next()
			return
		}
		rateLimiterMiddleware(c)
	})

	return router
}

func ginZapLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)

		if len(c.Errors) > 0 {
			log.Error("request error",
				zap.String("path", path),
				zap.String("query", query),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("latency", latency),
				zap.Strings("errors", c.Errors.Errors()),
			)
		} else {
			log.Info("request",
				zap.String("path", path),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("latency", latency),
			)
		}
	}
}

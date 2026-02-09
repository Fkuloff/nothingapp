// internal/app/app.go
package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"messenger/internal/config"
	"messenger/internal/handlers"
	"messenger/internal/logger"
	"messenger/internal/models"
	"messenger/internal/shutdown"
	"messenger/internal/storage"

	ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

func Run() error {
	cfg := config.LoadConfig()

	// Initialize logger
	log := logger.MustNew(os.Getenv("DEBUG") == "true")
	defer log.Sync()

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
	if err := handlers.SetupRoutes(router, db, []byte(cfg.JWTSecret), fileStorage, log); err != nil {
		return fmt.Errorf("setup routes: %w", err)
	}

	// HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 30 * time.Second,
	}

	// Graceful shutdown
	shutdownMgr := shutdown.New(log, 30*time.Second)
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
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  gormLogger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
		PrepareStmt: true,
		QueryFields: true,
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

	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

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
	return db.AutoMigrate(
		&models.User{},
		&models.Chat{},
		&models.Message{},
		&models.Contact{},
		&models.Attachment{},
		&models.UnreadMessage{},
	)
}

func setupRouter(log *zap.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Custom logger middleware using zap
	router.Use(ginZapLogger(log))
	router.Use(gin.Recovery())

	// CORS
	allowedOrigins := []string{"http://localhost:8080"}
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		allowedOrigins = strings.Split(origins, ",")
		log.Info("CORS allowed origins", zap.Strings("origins", allowedOrigins))
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: false,
	}))

	// Rate limiting
	store := ratelimit.InMemoryStore(&ratelimit.InMemoryOptions{
		Rate:  time.Second,
		Limit: 30,
	})
	rateLimiterMiddleware := ratelimit.RateLimiter(store, &ratelimit.Options{
		ErrorHandler: func(c *gin.Context, info ratelimit.Info) {
			c.JSON(429, gin.H{"error": "too many requests"})
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

// JWTMiddleware for token validation with enhanced security checks
func JWTMiddleware(secret []byte, log *zap.Logger) gin.HandlerFunc {
	publicPaths := map[string]bool{
		"/api/auth/login":    true,
		"/api/auth/register": true,
		"/api/auth/logout":   true,
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		isAPIPath := strings.HasPrefix(path, "/api")
		isWebsocket := strings.HasPrefix(path, "/ws")

		// Allow public endpoints
		if publicPaths[path] {
			c.Next()
			return
		}

		tokenString := extractToken(c, isAPIPath || isWebsocket)
		if tokenString == "" {
			handleUnauthorized(c, isAPIPath || isWebsocket)
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			handleUnauthorized(c, isAPIPath || isWebsocket)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			handleUnauthorized(c, isAPIPath || isWebsocket)
			return
		}

		// Validate issuer and audience
		if iss, issOk := claims["iss"].(string); issOk && iss != "messenger-app" {
			handleUnauthorized(c, isAPIPath || isWebsocket)
			return
		}
		if aud, audOk := claims["aud"].(string); audOk && aud != "messenger-users" {
			handleUnauthorized(c, isAPIPath || isWebsocket)
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			handleUnauthorized(c, isAPIPath || isWebsocket)
			return
		}
		userID := uint(userIDFloat)

		c.Set("user_id", userID)
		c.Next()
	}
}

// handleUnauthorized responds with JSON
func handleUnauthorized(c *gin.Context, _respondJSON bool) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
}

// extractToken gets token from Authorization header for API/WS
func extractToken(c *gin.Context, isAPIOrWS bool) string {
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}

	// Fallback для WebSocket (query param)
	if isAPIOrWS && strings.HasPrefix(c.Request.URL.Path, "/ws") {
		if token := c.Query("token"); token != "" {
			return token
		}
	}

	return ""
}

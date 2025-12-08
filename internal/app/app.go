// internal/app/app.go
package app

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"messenger/internal/config"
	"messenger/internal/handlers"
	"messenger/internal/models"

	ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Run() {
	cfg := config.LoadConfig()

	var err error
	DB, err = gorm.Open(postgres.Open(cfg.DBURL), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	err = DB.AutoMigrate(&models.User{}, &models.Chat{}, &models.Message{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	router := gin.Default()

	router.SetFuncMap(template.FuncMap{
		"truncate": func(s string, n int) string {
			if len(s) > n {
				return s[:n] + "..."
			}
			return s
		},
		"safeJS": func(s string) template.JS {
			return template.JS(template.JSEscapeString(s))
		},
	})

	allowedOrigins := []string{"http://localhost:8080"}
	lanOrigins := os.Getenv("ALLOWED_ORIGINS")
	if lanOrigins != "" {
		allowedOrigins = append(allowedOrigins, strings.Split(lanOrigins, ",")...)
	} else {
		// Для быстрого dev: "*" (небезопасно в prod)
		allowedOrigins = []string{"*"}
		log.Println("CORS: Allowing all origins for dev (set ALLOWED_ORIGINS for specific)")
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	keyFunc := func(c *gin.Context) string {
		return c.ClientIP()
	}
	store := ratelimit.InMemoryStore(&ratelimit.InMemoryOptions{
		Rate:  time.Second,
		Limit: 5,
	})
	router.Use(ratelimit.RateLimiter(store, &ratelimit.Options{
		ErrorHandler: func(c *gin.Context, info ratelimit.Info) {
			c.String(429, "Too many requests. Try again in "+time.Until(info.ResetTime).String())
		},
		KeyFunc: keyFunc,
	}))

	router.Static("/static", "./static")

	router.LoadHTMLGlob("templates/*.html")

	router.Use(JWTMiddleware([]byte(cfg.JWTSecret)))

	handlers.SetupRoutes(router, DB, []byte(cfg.JWTSecret))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	err = router.Run(":" + port)
	if err != nil {
		log.Fatal("Failed to run server:", err)
	}
}

// JWTMiddleware for token validation
func JWTMiddleware(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/login" || c.Request.URL.Path == "/register" {
			c.Next()
			return
		}

		tokenString, err := c.Cookie("jwt_token")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		userID := uint(userIDFloat)

		c.Set("user_id", userID)
		c.Next()
	}
}

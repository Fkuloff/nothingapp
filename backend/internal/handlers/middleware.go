package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// jwtMiddleware returns a Gin middleware for JWT token validation.
func jwtMiddleware(secret []byte, log *zap.Logger) gin.HandlerFunc {
	publicPaths := map[string]bool{
		"/api/auth/login":    true,
		"/api/auth/register": true,
		"/api/auth/logout":   true,
		"/health":            true,
	}

	// Public path prefixes (attachments are public - no chat membership validation needed)
	publicPrefixes := []string{
		"/api/attachments/",
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

		// Allow public prefixes (attachments are validated internally by chat membership)
		for _, prefix := range publicPrefixes {
			if strings.HasPrefix(path, prefix) {
				c.Next()
				return
			}
		}

		tokenString := extractToken(c, isAPIPath || isWebsocket)
		if tokenString == "" {
			log.Warn("no token found",
				zap.String("path", path),
				zap.Bool("is_websocket", isWebsocket),
			)
			handleUnauthorized(c)
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			log.Warn("invalid token",
				zap.String("path", path),
				zap.Error(err),
			)
			handleUnauthorized(c)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			handleUnauthorized(c)
			return
		}

		// Validate issuer and audience
		if iss, issOk := claims["iss"].(string); issOk && iss != "messenger-app" {
			log.Warn("invalid issuer", zap.String("iss", iss))
			handleUnauthorized(c)
			return
		}
		if aud, audOk := claims["aud"].(string); audOk && aud != "messenger-users" {
			log.Warn("invalid audience", zap.String("aud", aud))
			handleUnauthorized(c)
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			handleUnauthorized(c)
			return
		}
		userID := uint(userIDFloat)

		c.Set("user_id", userID)
		c.Next()
	}
}

// handleUnauthorized responds with JSON
func handleUnauthorized(c *gin.Context) {
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

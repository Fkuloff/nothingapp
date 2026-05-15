package handlers

import (
	"net/http"
	"strings"
	"time"

	"messenger/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// refreshTokenHeader is the response header used to send a freshly-minted token to the
// client when the active token is approaching staleness. The frontend reads this header
// and replaces the stored token, sliding the session forward.
const refreshTokenHeader = "X-Refresh-Token"

// jwtMiddleware returns a Gin middleware for JWT token validation.
//
// If `tokenTTL > 0` and the request carries a valid token whose `iat` is older than
// `config.TokenRefreshThresholdSeconds`, the middleware mints a fresh token (same userID,
// new exp + new jti) and returns it via the `X-Refresh-Token` response header. This
// implements a sliding session: active users never see their session expire, while
// inactive users still get the full initial TTL window (default 10 years).
func jwtMiddleware(secret []byte, log *zap.Logger, tokenTTL time.Duration) gin.HandlerFunc {
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

		// Sliding refresh: if the token is older than the threshold, issue a fresh one.
		// Skip for WebSocket upgrades — the response is already a 101 handshake and any
		// X-Refresh-Token header would be invisible to a JS WebSocket client.
		if tokenTTL > 0 && !isWebsocket {
			if iatFloat, iatOk := claims["iat"].(float64); iatOk {
				age := time.Now().Unix() - int64(iatFloat)
				if age >= config.TokenRefreshThresholdSeconds() {
					if fresh, mintErr := issueJWT(secret, userID, tokenTTL); mintErr == nil {
						c.Header(refreshTokenHeader, fresh)
					} else {
						log.Warn("failed to mint refresh token", zap.Error(mintErr))
					}
				}
			}
		}

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

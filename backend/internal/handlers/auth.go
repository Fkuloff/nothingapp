// internal/handlers/auth.go
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"

	"messenger/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	authService *services.AuthService
	secret      []byte
}

func NewAuthHandler(authService *services.AuthService, secret []byte) *AuthHandler {
	return &AuthHandler{authService: authService, secret: secret}
}

// validatePasswordStrength checks if password meets security requirements
func validatePasswordStrength(password string) error {
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	return nil
}

// generateJTI creates a unique JWT ID for token revocation tracking
func generateJTI() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateJWT creates a JWT token with proper security claims
func (h *AuthHandler) generateJWT(userID uint) (string, error) {
	now := time.Now()
	jti, err := generateJTI()
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"iss":     "messenger-app",                // Issuer
		"aud":     "messenger-users",              // Audience
		"exp":     now.Add(time.Hour * 24).Unix(), // Expiration
		"iat":     now.Unix(),                     // Issued at
		"jti":     jti,                            // JWT ID for revocation
	})

	return token.SignedString(h.secret)
}

// RegisterAPI handles JSON registration
func (h *AuthHandler) RegisterAPI(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=20"`
		Password string `json:"password" binding:"required,min=6"`
		Name     string `json:"name" binding:"required,min=2,max=50"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Parse validation errors
		errMsg := "Проверьте введённые данные"
		errStr := err.Error()
		if strings.Contains(errStr, "Username") {
			errMsg = "Username: от 3 до 20 символов"
		} else if strings.Contains(errStr, "Password") {
			errMsg = "Пароль: минимум 6 символов"
		} else if strings.Contains(errStr, "Name") {
			errMsg = "Имя: от 2 до 50 символов"
		}
		sendBadRequest(c, errMsg)
		return
	}

	// Validate password strength
	if err := validatePasswordStrength(req.Password); err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	// Validate username format
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !usernameRegex.MatchString(req.Username) {
		sendBadRequest(c, "Username может содержать только буквы, цифры и _")
		return
	}

	req.Name = strings.TrimSpace(req.Name)

	err := h.authService.Register(c.Request.Context(), req.Username, req.Password, req.Name)
	if err != nil {
		if strings.Contains(err.Error(), "username") {
			sendBadRequest(c, "Этот username уже занят")
		} else {
			sendBadRequest(c, "Ошибка регистрации")
		}
		return
	}

	// Auto-login after registration
	user, loginErr := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if loginErr != nil {
		sendInternalError(c, "Registration successful, but login failed")
		return
	}

	tokenString, err := h.generateJWT(user.ID)
	if err != nil {
		sendInternalError(c, "Failed to generate token")
		return
	}

	sendSuccess(c, gin.H{
		"user_id":  user.ID,
		"username": user.Username,
		"name":     user.Name,
		"token":    tokenString,
	})
}

// LoginAPI handles JSON login
func (h *AuthHandler) LoginAPI(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Invalid input")
		return
	}

	user, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		sendUnauthorized(c)
		return
	}

	tokenString, err := h.generateJWT(user.ID)
	if err != nil {
		sendInternalError(c, "Failed to generate token")
		return
	}

	sendSuccess(c, gin.H{
		"user_id":  user.ID,
		"username": user.Username,
		"name":     user.Name,
		"token":    tokenString,
	})
}

// LogoutAPI handles JSON logout
func (h *AuthHandler) LogoutAPI(c *gin.Context) {
	sendSuccess(c, gin.H{
		"message": "Logged out successfully",
	})
}

// GetCurrentUser returns current user info
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		sendNotFound(c, "User not found")
		return
	}

	sendSuccess(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"name":       user.Name,
		"avatar_url": user.AvatarURL,
	})
}

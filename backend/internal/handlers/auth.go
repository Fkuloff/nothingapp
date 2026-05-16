package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"

	"messenger/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// authHandler handles HTTP requests for authentication (register, login, logout).
type authHandler struct {
	authService *services.AuthService
	userService *services.UserService
	secret      []byte
	tokenTTL    time.Duration
}

// newAuthHandler creates a new authHandler.
func newAuthHandler(authService *services.AuthService, userService *services.UserService, secret []byte, tokenTTL time.Duration) *authHandler {
	return &authHandler{
		authService: authService,
		userService: userService,
		secret:      secret,
		tokenTTL:    tokenTTL,
	}
}

// validatePasswordStrength checks if password meets security requirements
func validatePasswordStrength(password string) error {
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	if strings.TrimSpace(password) == "" {
		return errors.New("password cannot be only whitespace")
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
func (h *authHandler) generateJWT(userID uint) (string, error) {
	return issueJWT(h.secret, userID, h.tokenTTL)
}

// issueJWT mints a new JWT for the given user id. Shared between the auth handler
// (login/register) and the middleware's sliding-window refresh.
func issueJWT(secret []byte, userID uint, ttl time.Duration) (string, error) {
	now := time.Now()
	jti, err := generateJTI()
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"iss":     "messenger-app",
		"aud":     "messenger-users",
		"exp":     now.Add(ttl).Unix(),
		"iat":     now.Unix(),
		"jti":     jti,
	})

	return token.SignedString(secret)
}

// RegisterAPI handles JSON registration
func (h *authHandler) RegisterAPI(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=20"`
		Password string `json:"password" binding:"required,min=6"`
		Name     string `json:"name" binding:"required,min=2,max=50"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Parse validation errors
		errMsg := "Invalid input data"
		errStr := err.Error()
		if strings.Contains(errStr, "Username") {
			errMsg = "Username must be 3-20 characters"
		} else if strings.Contains(errStr, "Password") {
			errMsg = "Password must be at least 6 characters"
		} else if strings.Contains(errStr, "Name") {
			errMsg = "Name must be 2-50 characters"
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
		sendBadRequest(c, "Username can only contain letters, numbers and underscore")
		return
	}

	req.Name = strings.TrimSpace(req.Name)

	// Validate trimmed name length and content
	if len(req.Name) < 2 {
		sendBadRequest(c, "Name must be at least 2 characters (excluding spaces)")
		return
	}
	hasAlphanumeric := false
	for _, r := range req.Name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			hasAlphanumeric = true
			break
		}
	}
	if !hasAlphanumeric {
		sendBadRequest(c, "Name must contain at least one letter or digit")
		return
	}

	err := h.authService.Register(c.Request.Context(), req.Username, req.Password, req.Name)
	if err != nil {
		if strings.Contains(err.Error(), "username") {
			sendBadRequest(c, "Username already taken")
		} else {
			sendBadRequest(c, "Registration failed")
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

	sendCreated(c, gin.H{
		"user_id":               user.ID,
		"username":              user.Username,
		"name":                  user.Name,
		"token":                 tokenString,
		"vault_salt":            user.VaultSalt,
		"encrypted_account_key": user.EncryptedAccountKey,
		"public_key":            user.PublicKey,
	})
}

// LoginAPI handles JSON login
func (h *authHandler) LoginAPI(c *gin.Context) {
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
		"user_id":               user.ID,
		"username":              user.Username,
		"name":                  user.Name,
		"token":                 tokenString,
		"vault_salt":            user.VaultSalt,
		"encrypted_account_key": user.EncryptedAccountKey,
		"public_key":            user.PublicKey,
	})
}

// ChangePasswordAPI handles password change for authenticated users
func (h *authHandler) ChangePasswordAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Old password and new password are required")
		return
	}

	if err := validatePasswordStrength(req.NewPassword); err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	err := h.authService.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, services.ErrInvalidPassword) {
			sendBadRequest(c, "Неверный текущий пароль")
			return
		}
		sendInternalError(c, "Failed to change password")
		return
	}

	sendSuccess(c, gin.H{
		"message": "Password changed successfully",
	})
}

// LogoutAPI handles JSON logout
func (h *authHandler) LogoutAPI(c *gin.Context) {
	sendSuccess(c, gin.H{
		"message": "Logged out successfully",
	})
}

// GetCurrentUser returns current user info
func (h *authHandler) GetCurrentUser(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		sendNotFound(c, "User not found")
		return
	}

	// Refresh avatar URL for S3 presigned URLs
	h.userService.RefreshUserAvatarURL(user)

	// Vault material is included whenever it has been provisioned. Clients use it on
	// app start to derive the vault key, unwrap the account key, and re-derive the
	// X25519 keypair. Absence (all three fields null) means the user hasn't opted
	// into E2E yet — their messages will stay scheme=1.
	sendSuccess(c, gin.H{
		"id":                    user.ID,
		"username":              user.Username,
		"name":                  user.Name,
		"avatar_url":            user.AvatarURL,
		"vault_salt":            user.VaultSalt,
		"encrypted_account_key": user.EncryptedAccountKey,
		"public_key":            user.PublicKey,
	})
}

// UpdateVaultAPI accepts the user's E2E vault material:
//
//   - vault_salt: PBKDF2 salt used to derive vault_key from the password.
//   - encrypted_account_key: account_key sealed under vault_key (AES-GCM).
//   - public_key: X25519 public key derived deterministically from account_key.
//     Published openly so other users can ECDH against it.
//
// All three are stored as opaque base64 strings. The server never sees the
// vault_key, the cleartext account_key, or the X25519 private half.
//
// Pass empty strings for all three to opt out of E2E. Half-state is rejected —
// other users could otherwise ECDH against a public_key that no one can match.
func (h *authHandler) UpdateVaultAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req struct {
		VaultSalt           string `json:"vault_salt"`
		EncryptedAccountKey string `json:"encrypted_account_key"`
		PublicKey           string `json:"public_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Invalid request body")
		return
	}

	if err := h.userService.UpdateVault(c.Request.Context(), userID, req.VaultSalt, req.EncryptedAccountKey, req.PublicKey); err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	sendSuccess(c, gin.H{"success": true})
}

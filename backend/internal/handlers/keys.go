package handlers

import (
	"errors"

	"messenger/internal/services"

	"github.com/gin-gonic/gin"
)

type KeyHandler struct {
	keyService *services.KeyService
}

func NewKeyHandler(keyService *services.KeyService) *KeyHandler {
	return &KeyHandler{keyService: keyService}
}

// --- Public Key Endpoints ---

type uploadPublicKeyRequest struct {
	PublicKey string `json:"public_key" binding:"required"`
}

// UploadPublicKey stores the authenticated user's ECDH public key
func (h *KeyHandler) UploadPublicKey(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req uploadPublicKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Invalid request: public_key is required")
		return
	}

	if err := h.keyService.SavePublicKey(c.Request.Context(), userID, req.PublicKey); err != nil {
		sendInternalError(c, "Failed to save public key")
		return
	}

	sendSuccess(c, gin.H{"message": "Public key saved"})
}

// GetPublicKey returns the public key for a given user
func (h *KeyHandler) GetPublicKey(c *gin.Context) {
	_, ok := requireUserID(c)
	if !ok {
		return
	}

	targetUserID, err := parseUintParam(c, "user_id")
	if err != nil {
		sendBadRequest(c, "Invalid user ID")
		return
	}

	key, err := h.keyService.GetPublicKey(c.Request.Context(), targetUserID)
	if err != nil {
		if errors.Is(err, services.ErrPublicKeyNotFound) {
			sendNotFound(c, "Public key not found for this user")
			return
		}
		sendInternalError(c, "Failed to get public key")
		return
	}

	sendSuccess(c, gin.H{
		"user_id":    key.UserID,
		"public_key": key.PublicKey,
	})
}

// --- Key Backup Endpoints ---

type saveKeyBackupRequest struct {
	EncryptedKey string `json:"encrypted_key" binding:"required"`
	Salt         string `json:"salt" binding:"required"`
	IV           string `json:"iv" binding:"required"`
}

// SaveKeyBackup stores an encrypted private key backup for multi-device support
func (h *KeyHandler) SaveKeyBackup(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req saveKeyBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Invalid request: encrypted_key, salt, and iv are required")
		return
	}

	if err := h.keyService.SaveKeyBackup(c.Request.Context(), userID, req.EncryptedKey, req.Salt, req.IV); err != nil {
		sendInternalError(c, "Failed to save key backup")
		return
	}

	sendSuccess(c, gin.H{"message": "Key backup saved"})
}

// GetKeyBackup returns the encrypted key backup for the authenticated user
func (h *KeyHandler) GetKeyBackup(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	backup, err := h.keyService.GetKeyBackup(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, services.ErrKeyBackupNotFound) {
			sendNotFound(c, "No key backup found")
			return
		}
		sendInternalError(c, "Failed to get key backup")
		return
	}

	sendSuccess(c, gin.H{
		"encrypted_key": backup.EncryptedKey,
		"salt":          backup.Salt,
		"iv":            backup.IV,
	})
}

// DeleteKeyBackup removes the encrypted key backup for the authenticated user
func (h *KeyHandler) DeleteKeyBackup(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	if err := h.keyService.DeleteKeyBackup(c.Request.Context(), userID); err != nil {
		sendInternalError(c, "Failed to delete key backup")
		return
	}

	sendSuccess(c, gin.H{"message": "Key backup deleted"})
}

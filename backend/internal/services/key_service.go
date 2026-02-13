package services

import (
	"context"
	"errors"
	"fmt"

	"messenger/internal/models"
	"messenger/internal/repositories"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrPublicKeyNotFound = errors.New("public key not found")
	ErrKeyBackupNotFound = errors.New("key backup not found")
)

type KeyService struct {
	logger        *zap.Logger
	userKeyRepo   *repositories.UserKeyRepo
	keyBackupRepo *repositories.KeyBackupRepo
}

func NewKeyService(logger *zap.Logger, userKeyRepo *repositories.UserKeyRepo, keyBackupRepo *repositories.KeyBackupRepo) *KeyService {
	return &KeyService{
		logger:        logger,
		userKeyRepo:   userKeyRepo,
		keyBackupRepo: keyBackupRepo,
	}
}

// SavePublicKey stores or updates a user's public ECDH key
func (s *KeyService) SavePublicKey(ctx context.Context, userID uint, publicKeyJWK string) error {
	key := &models.UserKey{
		UserID:    userID,
		PublicKey: publicKeyJWK,
	}
	if err := s.userKeyRepo.Upsert(ctx, key); err != nil {
		return fmt.Errorf("save public key: %w", err)
	}
	return nil
}

// GetPublicKey returns a user's public key
func (s *KeyService) GetPublicKey(ctx context.Context, userID uint) (*models.UserKey, error) {
	key, err := s.userKeyRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPublicKeyNotFound
		}
		return nil, fmt.Errorf("get public key: %w", err)
	}
	return key, nil
}

// GetPublicKeys returns public keys for multiple users
func (s *KeyService) GetPublicKeys(ctx context.Context, userIDs []uint) ([]models.UserKey, error) {
	keys, err := s.userKeyRepo.FindByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("get public keys: %w", err)
	}
	return keys, nil
}

// SaveKeyBackup stores an encrypted private key backup
func (s *KeyService) SaveKeyBackup(ctx context.Context, userID uint, encryptedKey, salt, iv string) error {
	backup := &models.KeyBackup{
		UserID:       userID,
		EncryptedKey: encryptedKey,
		Salt:         salt,
		IV:           iv,
	}
	if err := s.keyBackupRepo.Upsert(ctx, backup); err != nil {
		return fmt.Errorf("save key backup: %w", err)
	}
	return nil
}

// GetKeyBackup returns a user's encrypted key backup
func (s *KeyService) GetKeyBackup(ctx context.Context, userID uint) (*models.KeyBackup, error) {
	backup, err := s.keyBackupRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrKeyBackupNotFound
		}
		return nil, fmt.Errorf("get key backup: %w", err)
	}
	return backup, nil
}

// DeleteKeyBackup removes a user's key backup
func (s *KeyService) DeleteKeyBackup(ctx context.Context, userID uint) error {
	if err := s.keyBackupRepo.Delete(ctx, userID); err != nil {
		return fmt.Errorf("delete key backup: %w", err)
	}
	return nil
}

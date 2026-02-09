package services

import (
	"context"
	"fmt"
	"messenger/internal/models"
	"messenger/internal/repositories"
	"messenger/internal/storage"
	"mime/multipart"
	"strings"

	"go.uber.org/zap"
)

// UserService handles business logic for users
type UserService struct {
	logger       *zap.Logger
	userRepo     *repositories.UserRepo
	storage      storage.Storage
	thumbnailGen *ThumbnailGenerator
	validator    *FileValidator
}

// NewUserService creates a new UserService instance
func NewUserService(
	logger *zap.Logger,
	userRepo *repositories.UserRepo,
	storage storage.Storage,
) *UserService {
	return &UserService{
		logger:       logger,
		userRepo:     userRepo,
		storage:      storage,
		thumbnailGen: NewThumbnailGenerator(storage),
		validator:    &FileValidator{},
	}
}

// UploadAvatar uploads a user avatar
func (s *UserService) UploadAvatar(ctx context.Context, userID uint, fileHeader *multipart.FileHeader) (string, error) {
	// Validate file
	if err := s.validator.ValidateAvatar(fileHeader); err != nil {
		return "", fmt.Errorf("invalid avatar file: %w", err)
	}

	// Open file
	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get content type
	contentType := fileHeader.Header.Get("Content-Type")

	// Delete old avatar if exists
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}

	var oldAvatarKey string
	if user.AvatarURL != nil && *user.AvatarURL != "" {
		// Extract storage key from URL if it's a local storage URL
		// For local storage, URL format is: /uploads/avatars/filename
		// We need to extract the key part
		oldAvatarKey = extractStorageKey(*user.AvatarURL)
	}

	// Save to storage with prefix for avatars
	metadata, err := s.storage.Save(
		file,
		"avatar_"+fileHeader.Filename,
		contentType,
		fileHeader.Size,
	)
	if err != nil {
		return "", fmt.Errorf("failed to save avatar: %w", err)
	}

	// Generate avatar URL
	avatarURL := metadata.URL

	// Update user avatar in database
	if err := s.userRepo.UpdateAvatar(ctx, userID, &avatarURL); err != nil {
		// Rollback: delete uploaded file
		if delErr := s.storage.Delete(metadata.Key); delErr != nil {
			s.logger.Warn("failed to rollback avatar upload", zap.Error(delErr), zap.String("key", metadata.Key))
		}
		return "", fmt.Errorf("failed to update user avatar: %w", err)
	}

	// Delete old avatar after successful update
	if oldAvatarKey != "" {
		if err := s.storage.Delete(oldAvatarKey); err != nil {
			s.logger.Warn("failed to delete old avatar", zap.Error(err), zap.String("key", oldAvatarKey))
		}
	}

	return avatarURL, nil
}

// DeleteAvatar removes user avatar
func (s *UserService) DeleteAvatar(ctx context.Context, userID uint) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if user.AvatarURL == nil || *user.AvatarURL == "" {
		return fmt.Errorf("user has no avatar")
	}

	// Extract storage key from URL
	storageKey := extractStorageKey(*user.AvatarURL)

	// Delete from storage
	if err := s.storage.Delete(storageKey); err != nil {
		// Log error but continue with database update
		s.logger.Warn("failed to delete avatar from storage", zap.Error(err), zap.String("storage_key", storageKey))
	}

	// Update database
	if err := s.userRepo.UpdateAvatar(ctx, userID, nil); err != nil {
		return fmt.Errorf("failed to update user avatar: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, userID uint) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

// FindByUsername finds a user by username
func (s *UserService) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

// SearchUsers searches for users by username or name
func (s *UserService) SearchUsers(ctx context.Context, query string) ([]*models.User, error) {
	users, err := s.userRepo.SearchByUsernameOrName(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	return users, nil
}

// extractStorageKey extracts storage key from URL
// For local storage: http://localhost:8080/uploads/files/2025/12/14/uuid.ext -> files/2025/12/14/uuid.ext
func extractStorageKey(url string) string {
	// URL format: http://localhost:8080/uploads/files/YYYY/MM/DD/uuid.ext
	// We need to extract: files/YYYY/MM/DD/uuid.ext

	// Find "/uploads/" in the URL
	uploadsPrefix := "/uploads/"
	idx := strings.Index(url, uploadsPrefix)
	if idx == -1 {
		// If no /uploads/ found, might be relative path already
		return url
	}

	// Return everything after /uploads/
	return url[idx+len(uploadsPrefix):]
}

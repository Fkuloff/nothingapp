package services

import (
	"context"
	"fmt"
	"strings"

	"messenger/internal/models"
	"messenger/internal/repositories"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles user authentication (registration, login, lookup).
type AuthService struct {
	logger   *zap.Logger
	userRepo *repositories.UserRepo
}

// NewAuthService creates a new AuthService.
func NewAuthService(logger *zap.Logger, userRepo *repositories.UserRepo) *AuthService {
	return &AuthService{
		logger:   logger,
		userRepo: userRepo,
	}
}

// Register creates a new user account with a hashed password.
func (s *AuthService) Register(ctx context.Context, username, password, name string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Normalize username to lowercase for case-insensitive uniqueness
	user := &models.User{
		Username: strings.ToLower(strings.TrimSpace(username)),
		Password: string(hashedPassword),
		Name:     strings.TrimSpace(name),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return fmt.Errorf("username already exists")
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// Login validates credentials and returns the authenticated user.
func (s *AuthService) Login(ctx context.Context, username, password string) (*models.User, error) {
	// Normalize username to lowercase for case-insensitive login
	user, err := s.userRepo.FindByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID.
func (s *AuthService) GetUserByID(ctx context.Context, userID uint) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	return user, nil
}

package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"messenger/internal/models"
	"messenger/internal/repositories"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidPassword is returned when the provided password does not match.
var ErrInvalidPassword = errors.New("invalid password")

// bcryptCost is the work factor for new password hashes. Existing hashes with
// a lower cost are transparently rehashed on successful login.
const bcryptCost = 12

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
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
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

	s.rehashIfStale(ctx, user, password)

	return user, nil
}

// rehashIfStale upgrades a user's password hash to the current bcryptCost.
// Called on successful login so users transparently move to the stronger cost.
// Best-effort: failures are logged but do not fail the login.
func (s *AuthService) rehashIfStale(ctx context.Context, user *models.User, plaintext string) {
	cost, err := bcrypt.Cost([]byte(user.Password))
	if err != nil || cost >= bcryptCost {
		return
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcryptCost)
	if err != nil {
		s.logger.Warn("bcrypt rehash failed", zap.Uint("user_id", user.ID), zap.Error(err))
		return
	}
	if err := s.userRepo.UpdatePassword(ctx, user.ID, string(newHash)); err != nil {
		s.logger.Warn("bcrypt rehash persist failed", zap.Uint("user_id", user.ID), zap.Error(err))
		return
	}
	user.Password = string(newHash)
	s.logger.Info("bcrypt rehashed", zap.Uint("user_id", user.ID), zap.Int("old_cost", cost), zap.Int("new_cost", bcryptCost))
}

// ChangePassword validates the old password and updates to a new one.
func (s *AuthService) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return ErrInvalidPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, string(hashedPassword)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by their ID.
func (s *AuthService) GetUserByID(ctx context.Context, userID uint) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	return user, nil
}

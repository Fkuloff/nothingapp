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

type AuthService struct {
	logger   *zap.Logger
	userRepo *repositories.UserRepo
}

func NewAuthService(logger *zap.Logger, userRepo *repositories.UserRepo) *AuthService {
	return &AuthService{
		logger:   logger,
		userRepo: userRepo,
	}
}

func (s *AuthService) Register(ctx context.Context, username, password, name, phone string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Username: username,
		Password: string(hashedPassword),
		Name:     strings.TrimSpace(name),
	}

	// Phone is optional
	if phone := strings.TrimSpace(phone); phone != "" {
		user.Phone = &phone
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return fmt.Errorf("username already exists")
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*models.User, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

func (s *AuthService) GetUserByID(ctx context.Context, userID uint) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	return user, nil
}

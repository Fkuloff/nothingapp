package services

import (
	"errors"
	"strings"

	"messenger/internal/models"
	"messenger/internal/repositories"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	UserRepo *repositories.UserRepo
}

func NewAuthService(userRepo *repositories.UserRepo) *AuthService {
	return &AuthService{UserRepo: userRepo}
}

func (s *AuthService) Register(username, password, name, phone string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &models.User{
		Username: username,
		Password: string(hashedPassword),
		Name:     strings.TrimSpace(name),
		Phone:    strings.TrimSpace(phone),
	}

	return s.UserRepo.Create(user)
}

func (s *AuthService) Login(username, password string) (*models.User, error) {
	user, err := s.UserRepo.FindByUsername(username)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("invalid password")
	}

	return user, nil
}

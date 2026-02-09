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

type ContactService struct {
	logger      *zap.Logger
	contactRepo *repositories.ContactRepo
}

func NewContactService(logger *zap.Logger, contactRepo *repositories.ContactRepo) *ContactService {
	return &ContactService{
		logger:      logger,
		contactRepo: contactRepo,
	}
}

func (s *ContactService) AddContact(ctx context.Context, userID, contactUserID uint) error {
	if userID == contactUserID {
		return fmt.Errorf("cannot add self to contacts")
	}

	_, err := s.contactRepo.FindByUsers(ctx, userID, contactUserID)
	if err == nil {
		return fmt.Errorf("already in contacts")
	}

	contact := &models.Contact{
		UserID:        userID,
		ContactUserID: contactUserID,
	}

	if err := s.contactRepo.Create(ctx, contact); err != nil {
		return fmt.Errorf("failed to create contact: %w", err)
	}

	return nil
}

func (s *ContactService) IsContact(ctx context.Context, userID, contactUserID uint) (bool, error) {
	_, err := s.contactRepo.FindByUsers(ctx, userID, contactUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check contact: %w", err)
	}
	return true, nil
}

// GetUserContacts returns all contacts for a user
func (s *ContactService) GetUserContacts(ctx context.Context, userID uint) ([]models.Contact, error) {
	contacts, err := s.contactRepo.GetUserContacts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user contacts: %w", err)
	}
	return contacts, nil
}

// RemoveContact removes a contact
func (s *ContactService) RemoveContact(ctx context.Context, userID, contactUserID uint) error {
	contact, err := s.contactRepo.FindByUsers(ctx, userID, contactUserID)
	if err != nil {
		return fmt.Errorf("contact not found")
	}

	if err := s.contactRepo.Delete(ctx, contact); err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	return nil
}

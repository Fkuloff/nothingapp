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
	// ErrCannotAddSelf is returned when attempting to add yourself as a contact
	ErrCannotAddSelf = errors.New("cannot add self to contacts")

	// ErrAlreadyInContacts is returned when the contact relationship already exists
	ErrAlreadyInContacts = errors.New("already in contacts")

	// ErrContactNotFound is returned when the requested contact does not exist
	ErrContactNotFound = errors.New("contact not found")
)

// ContactService handles business logic for user contacts.
type ContactService struct {
	logger      *zap.Logger
	contactRepo *repositories.ContactRepo
}

// NewContactService creates a new ContactService.
func NewContactService(logger *zap.Logger, contactRepo *repositories.ContactRepo) *ContactService {
	return &ContactService{
		logger:      logger,
		contactRepo: contactRepo,
	}
}

// AddContact adds a user to the contact list.
func (s *ContactService) AddContact(ctx context.Context, userID, contactUserID uint) error {
	if userID == contactUserID {
		return ErrCannotAddSelf
	}

	_, err := s.contactRepo.FindByUsers(ctx, userID, contactUserID)
	if err == nil {
		return ErrAlreadyInContacts
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

// IsContact checks if a user is in the contact list.
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrContactNotFound
		}
		return fmt.Errorf("failed to find contact: %w", err)
	}

	if err := s.contactRepo.Delete(ctx, contact); err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	return nil
}

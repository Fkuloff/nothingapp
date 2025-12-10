package services

import (
	"errors"

	"messenger/internal/models"
	"messenger/internal/repositories"
)

type ContactService struct {
	contactRepo *repositories.ContactRepo
}

func NewContactService(contactRepo *repositories.ContactRepo) *ContactService {
	return &ContactService{contactRepo: contactRepo}
}

func (s *ContactService) AddContact(userID, contactUserID uint) error {
	if userID == contactUserID {
		return errors.New("cannot add self to contacts")
	}

	_, err := s.contactRepo.FindByUsers(userID, contactUserID)
	if err == nil {
		return errors.New("already in contacts")
	}

	contact := &models.Contact{
		UserID:        userID,
		ContactUserID: contactUserID,
	}

	return s.contactRepo.Create(contact)
}

func (s *ContactService) IsContact(userID, contactUserID uint) (bool, error) {
	_, err := s.contactRepo.FindByUsers(userID, contactUserID)
	if err != nil {
		return false, nil
	}
	return true, nil
}

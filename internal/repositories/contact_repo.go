package repositories

import (
	"messenger/internal/models"

	"gorm.io/gorm"
)

type ContactRepo struct {
	db *gorm.DB
}

func NewContactRepo(db *gorm.DB) *ContactRepo {
	return &ContactRepo{db: db}
}

func (r *ContactRepo) Create(contact *models.Contact) error {
	return r.db.Create(contact).Error
}

func (r *ContactRepo) FindByUsers(userID, contactUserID uint) (*models.Contact, error) {
	var contact models.Contact
	err := r.db.Where("user_id = ? AND contact_user_id = ?", userID, contactUserID).First(&contact).Error
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

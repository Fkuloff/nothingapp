package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
)

// ContactRepo handles database operations for user contacts.
type ContactRepo struct {
	db *gorm.DB
}

// NewContactRepo creates a new ContactRepo instance.
func NewContactRepo(db *gorm.DB) *ContactRepo {
	return &ContactRepo{db: db}
}

// Create stores a new contact record.
func (r *ContactRepo) Create(ctx context.Context, contact *models.Contact) error {
	return r.db.WithContext(ctx).Create(contact).Error
}

// FindByUsers finds a contact record between two users.
func (r *ContactRepo) FindByUsers(ctx context.Context, userID, contactUserID uint) (*models.Contact, error) {
	var contact models.Contact
	err := r.db.WithContext(ctx).Where("user_id = ? AND contact_user_id = ?", userID, contactUserID).First(&contact).Error
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

// GetUserContacts returns all contacts for a user with user details
func (r *ContactRepo) GetUserContacts(ctx context.Context, userID uint) ([]models.Contact, error) {
	var contacts []models.Contact
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Preload("ContactUser").
		Find(&contacts).Error
	return contacts, err
}

// Delete removes a contact
func (r *ContactRepo) Delete(ctx context.Context, contact *models.Contact) error {
	return r.db.WithContext(ctx).Delete(contact).Error
}

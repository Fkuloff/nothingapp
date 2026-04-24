package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
)

// AttachmentRepo handles database operations for attachments
type AttachmentRepo struct {
	db *gorm.DB
}

// NewAttachmentRepo creates a new AttachmentRepo instance
func NewAttachmentRepo(db *gorm.DB) *AttachmentRepo {
	return &AttachmentRepo{db: db}
}

// Create creates a single attachment
func (r *AttachmentRepo) Create(ctx context.Context, attachment *models.Attachment) error {
	return r.db.WithContext(ctx).Create(attachment).Error
}

// CreateBatch creates multiple attachments
func (r *AttachmentRepo) CreateBatch(ctx context.Context, attachments []*models.Attachment) error {
	if len(attachments) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(attachments).Error
}

// FindByID finds an attachment by its ID
func (r *AttachmentRepo) FindByID(ctx context.Context, id uint) (*models.Attachment, error) {
	var attachment models.Attachment
	err := r.db.WithContext(ctx).First(&attachment, id).Error
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

// FindByIDWithMessage finds an attachment by ID and preloads its parent Message (for access checks).
func (r *AttachmentRepo) FindByIDWithMessage(ctx context.Context, id uint) (*models.Attachment, error) {
	var attachment models.Attachment
	err := r.db.WithContext(ctx).Preload("Message").First(&attachment, id).Error
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

// Delete deletes an attachment by ID
func (r *AttachmentRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Attachment{}, id).Error
}

// GetStorageKeysByMessageID returns storage keys for every attachment belonging to the given message.
// Used before deleting so the caller can remove the underlying files from object storage.
func (r *AttachmentRepo) GetStorageKeysByMessageID(ctx context.Context, messageID uint) ([]string, error) {
	var keys []string
	err := r.db.WithContext(ctx).Model(&models.Attachment{}).
		Where("message_id = ?", messageID).
		Pluck("storage_key", &keys).Error
	return keys, err
}

// DeleteByMessageID hard-deletes every attachment row for a given message.
func (r *AttachmentRepo) DeleteByMessageID(ctx context.Context, messageID uint) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("message_id = ?", messageID).
		Delete(&models.Attachment{}).Error
}

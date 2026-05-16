package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
)

// AttachmentEnvelopeRepo handles per-recipient wrapped file_keys for
// scheme=2 attachments. See models.AttachmentEnvelope for the rationale.
type AttachmentEnvelopeRepo struct {
	db *gorm.DB
}

// NewAttachmentEnvelopeRepo creates a new repo instance.
func NewAttachmentEnvelopeRepo(db *gorm.DB) *AttachmentEnvelopeRepo {
	return &AttachmentEnvelopeRepo{db: db}
}

// WithTx returns a repo bound to the given transaction.
func (r *AttachmentEnvelopeRepo) WithTx(tx *gorm.DB) *AttachmentEnvelopeRepo {
	return &AttachmentEnvelopeRepo{db: tx}
}

// CreateBatch inserts every envelope for a single attachment in one round-trip.
// Empty input is a no-op so callers don't have to guard.
func (r *AttachmentEnvelopeRepo) CreateBatch(ctx context.Context, envelopes []models.AttachmentEnvelope) error {
	if len(envelopes) == 0 {
		return nil
	}
	const batchSize = 100
	return r.db.WithContext(ctx).CreateInBatches(envelopes, batchSize).Error
}

// FindForRecipient returns the envelope addressed to the given recipient for
// each of the requested attachment IDs, keyed by attachment_id. Attachments
// without an envelope for this user (e.g. they joined the group after the file
// was uploaded) are simply absent — caller treats that as "🔒 placeholder".
// Shape mirrors MessageEnvelopeRepo.FindForRecipient.
//
//nolint:dupl // intentional, see comment above
func (r *AttachmentEnvelopeRepo) FindForRecipient(ctx context.Context, recipientID uint, attachmentIDs []uint) (map[uint]models.AttachmentEnvelope, error) {
	out := make(map[uint]models.AttachmentEnvelope, len(attachmentIDs))
	if len(attachmentIDs) == 0 {
		return out, nil
	}
	var rows []models.AttachmentEnvelope
	if err := r.db.WithContext(ctx).
		Where("recipient_id = ? AND attachment_id IN ?", recipientID, attachmentIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.AttachmentID] = row
	}
	return out, nil
}

// FindAllForAttachment returns every envelope tied to one attachment. Used by
// the WS broadcast path for attachments_added — clients filter to their own
// envelope by recipient_id.
func (r *AttachmentEnvelopeRepo) FindAllForAttachment(ctx context.Context, attachmentID uint) ([]models.AttachmentEnvelope, error) {
	var rows []models.AttachmentEnvelope
	err := r.db.WithContext(ctx).
		Where("attachment_id = ?", attachmentID).
		Find(&rows).Error
	return rows, err
}

// DeleteByAttachmentIDs is the bulk cascade variant for message deletes /
// chat clears: we already know the doomed attachment_ids, wipe their envelopes.
func (r *AttachmentEnvelopeRepo) DeleteByAttachmentIDs(ctx context.Context, attachmentIDs []uint) error {
	if len(attachmentIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Where("attachment_id IN ?", attachmentIDs).
		Delete(&models.AttachmentEnvelope{}).Error
}

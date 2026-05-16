package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
)

// MessageEnvelopeRepo handles per-recipient ciphertexts for group scheme=2
// messages. See models.MessageEnvelope for the data model rationale.
type MessageEnvelopeRepo struct {
	db *gorm.DB
}

// NewMessageEnvelopeRepo creates a new MessageEnvelopeRepo instance.
func NewMessageEnvelopeRepo(db *gorm.DB) *MessageEnvelopeRepo {
	return &MessageEnvelopeRepo{db: db}
}

// WithTx creates a new MessageEnvelopeRepo bound to the given transaction.
func (r *MessageEnvelopeRepo) WithTx(tx *gorm.DB) *MessageEnvelopeRepo {
	return &MessageEnvelopeRepo{db: tx}
}

// CreateBatch inserts all envelopes for a single message in one round-trip.
// Empty input is a no-op so callers don't have to guard against it. GORM's
// CreateInBatches keeps the statement size sane for groups with hundreds of
// members (rare today, but the batch size keeps us forward-compatible).
func (r *MessageEnvelopeRepo) CreateBatch(ctx context.Context, envelopes []models.MessageEnvelope) error {
	if len(envelopes) == 0 {
		return nil
	}
	const batchSize = 100
	return r.db.WithContext(ctx).CreateInBatches(envelopes, batchSize).Error
}

// FindForRecipient returns the envelope addressed to the given recipient for
// each of the requested message IDs, keyed by message_id. Messages without an
// envelope (because the recipient wasn't a member when the message was sent,
// or the message wasn't a group scheme=2 message) are simply absent from the
// map — callers should treat that as "no per-user ciphertext, fall back to
// Message.Text/IV". Shape mirrors AttachmentEnvelopeRepo.FindForRecipient;
// can't generic over the two envelope tables without reflection, so we
// accept the small duplication for clarity.
//
//nolint:dupl // intentional, see comment above
func (r *MessageEnvelopeRepo) FindForRecipient(ctx context.Context, recipientID uint, messageIDs []uint) (map[uint]models.MessageEnvelope, error) {
	out := make(map[uint]models.MessageEnvelope, len(messageIDs))
	if len(messageIDs) == 0 {
		return out, nil
	}
	var rows []models.MessageEnvelope
	if err := r.db.WithContext(ctx).
		Where("recipient_id = ? AND message_id IN ?", recipientID, messageIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.MessageID] = row
	}
	return out, nil
}

// FindAllForMessage returns every envelope tied to a message (all recipients).
// Used by the WebSocket broadcast path — each connected client picks the
// envelope addressed to their own user_id.
func (r *MessageEnvelopeRepo) FindAllForMessage(ctx context.Context, messageID uint) ([]models.MessageEnvelope, error) {
	var rows []models.MessageEnvelope
	err := r.db.WithContext(ctx).
		Where("message_id = ?", messageID).
		Find(&rows).Error
	return rows, err
}

// DeleteByMessageID hard-deletes every envelope for a message. Called from the
// Message delete + ClearChat / DeleteChat cascades so we don't leak per-user
// ciphertexts after the parent message is gone.
func (r *MessageEnvelopeRepo) DeleteByMessageID(ctx context.Context, messageID uint) error {
	return r.db.WithContext(ctx).
		Where("message_id = ?", messageID).
		Delete(&models.MessageEnvelope{}).Error
}

// DeleteByMessageIDs is the bulk variant used by ClearChat / DeleteChat where
// we already know the full set of doomed message IDs.
func (r *MessageEnvelopeRepo) DeleteByMessageIDs(ctx context.Context, messageIDs []uint) error {
	if len(messageIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Where("message_id IN ?", messageIDs).
		Delete(&models.MessageEnvelope{}).Error
}

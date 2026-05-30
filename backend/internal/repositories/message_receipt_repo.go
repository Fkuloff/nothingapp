package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MessageReceiptRepo handles the per-recipient delivered/read pointers.
type MessageReceiptRepo struct {
	db *gorm.DB
}

// NewMessageReceiptRepo creates a new message receipt repository.
func NewMessageReceiptRepo(db *gorm.DB) *MessageReceiptRepo {
	return &MessageReceiptRepo{db: db}
}

// UpsertDelivered bumps the recipient's delivered pointer to messageID, creating
// the row on first delivery. GREATEST guards against out-of-order acks so the
// pointer never regresses.
func (r *MessageReceiptRepo) UpsertDelivered(ctx context.Context, chatID, userID, messageID uint) error {
	receipt := models.MessageReceipt{
		ChatID:                 chatID,
		UserID:                 userID,
		LastDeliveredMessageID: messageID,
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "chat_id"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"last_delivered_message_id": gorm.Expr(
					"GREATEST(message_receipts.last_delivered_message_id, EXCLUDED.last_delivered_message_id)"),
			}),
		}).
		Create(&receipt).Error
}

// UpsertRead bumps the recipient's read pointer to messageID. Read implies
// delivered, so the delivered pointer is bumped in the same statement. Both use
// GREATEST so neither pointer ever regresses.
func (r *MessageReceiptRepo) UpsertRead(ctx context.Context, chatID, userID, messageID uint) error {
	receipt := models.MessageReceipt{
		ChatID:                 chatID,
		UserID:                 userID,
		LastDeliveredMessageID: messageID,
		LastReadMessageID:      messageID,
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "chat_id"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"last_delivered_message_id": gorm.Expr(
					"GREATEST(message_receipts.last_delivered_message_id, EXCLUDED.last_delivered_message_id)"),
				"last_read_message_id": gorm.Expr(
					"GREATEST(message_receipts.last_read_message_id, EXCLUDED.last_read_message_id)"),
			}),
		}).
		Create(&receipt).Error
}

// GetPeerReceipt returns peerUserID's receipt row in chatID. Returns
// gorm.ErrRecordNotFound when the peer has no receipt yet (nothing delivered or
// read) — callers treat that the same as "no pointers" (delivered/read = 0).
func (r *MessageReceiptRepo) GetPeerReceipt(ctx context.Context, chatID, peerUserID uint) (*models.MessageReceipt, error) {
	var receipt models.MessageReceipt
	if err := r.db.WithContext(ctx).
		Where("chat_id = ? AND user_id = ?", chatID, peerUserID).
		First(&receipt).Error; err != nil {
		return nil, err
	}
	return &receipt, nil
}

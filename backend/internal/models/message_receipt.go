package models

import "gorm.io/gorm"

// MessageReceipt tracks how far a recipient has progressed through a chat's
// messages: the highest message ID they've been delivered, and the highest
// they've read. Message IDs are monotonic, so a single "up to N" pointer
// implicitly covers every earlier message — no per-message receipt rows needed.
//
// UserID is the *recipient* whose pointers these are. In a 1-on-1 chat the
// message author reads the peer's row to decide whether each sent bubble shows
// ✓ (sent), ✓✓ grey (delivered) or ✓✓ blue (read). The unique index on
// (chat_id, user_id) keeps it to one row per participant.
//
// Scope v1: only written/read for 1-on-1 chats. The schema already generalises
// to groups (one row per member), but the emit/read paths are gated on
// !chat.IsGroup until group aggregation is designed.
type MessageReceipt struct {
	gorm.Model
	ChatID                 uint `gorm:"uniqueIndex:idx_receipt_chat_user;not null"`
	UserID                 uint `gorm:"uniqueIndex:idx_receipt_chat_user;not null"`
	LastDeliveredMessageID uint `gorm:"not null;default:0"`
	LastReadMessageID      uint `gorm:"not null;default:0"`
}

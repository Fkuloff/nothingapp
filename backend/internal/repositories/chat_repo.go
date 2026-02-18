package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
)

// ChatRepo handles database operations for chats.
type ChatRepo struct {
	db *gorm.DB
}

// NewChatRepo creates a new ChatRepo instance.
func NewChatRepo(db *gorm.DB) *ChatRepo {
	return &ChatRepo{db: db}
}

// WithTx creates a new ChatRepo with the given transaction
func (r *ChatRepo) WithTx(tx *gorm.DB) *ChatRepo {
	return &ChatRepo{db: tx}
}

// Create stores a new chat record.
func (r *ChatRepo) Create(ctx context.Context, chat *models.Chat) error {
	return r.db.WithContext(ctx).Create(chat).Error
}

// FindByUsers finds a 1-on-1 chat between two users regardless of who initiated it.
func (r *ChatRepo) FindByUsers(ctx context.Context, user1ID, user2ID uint) (*models.Chat, error) {
	var chat models.Chat
	err := r.db.WithContext(ctx).Where("(user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)", user1ID, user2ID, user2ID, user1ID).First(&chat).Error
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

// GetUserChats retrieves all chats for a user, sorted by updated_at descending (most recent first)
// preloadUsers: if true, preloads User1 and User2 (use for display); if false, only loads IDs (use for presence/routing)
func (r *ChatRepo) GetUserChats(ctx context.Context, userID uint, preloadUsers bool) ([]models.Chat, error) {
	var chats []models.Chat
	query := r.db.WithContext(ctx).
		Where("user1_id = ? OR user2_id = ?", userID, userID).
		Order("updated_at DESC")

	if preloadUsers {
		query = query.Preload("User1").Preload("User2")
	}

	err := query.Find(&chats).Error
	return chats, err
}

// FindByID finds a chat by ID with User1 and User2 preloaded.
func (r *ChatRepo) FindByID(ctx context.Context, id uint) (*models.Chat, error) {
	var chat models.Chat
	err := r.db.WithContext(ctx).Preload("User1").Preload("User2").First(&chat, id).Error
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

// FindByIDLight finds a chat by ID without preloading users (for access checks)
func (r *ChatRepo) FindByIDLight(ctx context.Context, id uint) (*models.Chat, error) {
	var chat models.Chat
	err := r.db.WithContext(ctx).First(&chat, id).Error
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

// FindByIDWithParticipants finds a chat by ID with participants and their user data preloaded
func (r *ChatRepo) FindByIDWithParticipants(ctx context.Context, id uint) (*models.Chat, error) {
	var chat models.Chat
	err := r.db.WithContext(ctx).
		Preload("Participants").
		Preload("Participants.User").
		First(&chat, id).Error
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

// GetUserChatsIncludingGroups retrieves both 1-on-1 and group chats for a user.
// groupChatIDs should come from ChatParticipantRepo.GetUserGroupChatIDs.
func (r *ChatRepo) GetUserChatsIncludingGroups(ctx context.Context, userID uint, groupChatIDs []uint, preloadUsers bool) ([]models.Chat, error) {
	var chats []models.Chat
	query := r.db.WithContext(ctx).Order("updated_at DESC")

	if len(groupChatIDs) > 0 {
		query = query.Where(
			"(is_group = false AND (user1_id = ? OR user2_id = ?)) OR (id IN (?))",
			userID, userID, groupChatIDs,
		)
	} else {
		query = query.Where(
			"is_group = false AND (user1_id = ? OR user2_id = ?)",
			userID, userID,
		)
	}

	if preloadUsers {
		query = query.Preload("User1").Preload("User2")
	}

	err := query.Find(&chats).Error
	return chats, err
}

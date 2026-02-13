package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type KeyBackupRepo struct {
	db *gorm.DB
}

func NewKeyBackupRepo(db *gorm.DB) *KeyBackupRepo {
	return &KeyBackupRepo{db: db}
}

// Upsert creates or updates a user's key backup
func (r *KeyBackupRepo) Upsert(ctx context.Context, backup *models.KeyBackup) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"encrypted_key", "salt", "iv", "updated_at"}),
		}).
		Create(backup).Error
}

// FindByUserID returns the key backup for a user
func (r *KeyBackupRepo) FindByUserID(ctx context.Context, userID uint) (*models.KeyBackup, error) {
	var backup models.KeyBackup
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&backup).Error; err != nil {
		return nil, err
	}
	return &backup, nil
}

// Delete removes a user's key backup
func (r *KeyBackupRepo) Delete(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.KeyBackup{}).Error
}

package repositories

import (
	"context"
	"errors"

	"messenger/internal/models"

	"gorm.io/gorm"
)

// AppReleaseRepo handles database operations for the app_releases table.
type AppReleaseRepo struct {
	db *gorm.DB
}

// NewAppReleaseRepo creates a new repository.
func NewAppReleaseRepo(db *gorm.DB) *AppReleaseRepo {
	return &AppReleaseRepo{db: db}
}

// GetLatest returns the highest-VersionCode release for the given platform.
// Returns (nil, nil) when no release rows exist yet — callers should treat
// this as "up to date" so a fresh deploy without any releases registered
// doesn't surface phantom update banners.
func (r *AppReleaseRepo) GetLatest(ctx context.Context, platform string) (*models.AppRelease, error) {
	var rel models.AppRelease
	err := r.db.WithContext(ctx).
		Where("platform = ?", platform).
		Order("version_code DESC").
		First(&rel).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rel, nil
}

// Create inserts a new release row. Caller is responsible for setting
// ReleasedAt (the model has a NOT NULL constraint with no DB default to
// keep release-time accuracy server-clock-bound).
func (r *AppReleaseRepo) Create(ctx context.Context, rel *models.AppRelease) error {
	return r.db.WithContext(ctx).Create(rel).Error
}

// List returns the N most recent releases for a platform, newest first.
// Used by the admin dashboard / release-history view.
func (r *AppReleaseRepo) List(ctx context.Context, platform string, limit int) ([]models.AppRelease, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rels []models.AppRelease
	err := r.db.WithContext(ctx).
		Where("platform = ?", platform).
		Order("version_code DESC").
		Limit(limit).
		Find(&rels).Error
	return rels, err
}

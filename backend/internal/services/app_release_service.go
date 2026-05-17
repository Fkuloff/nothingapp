package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"messenger/internal/models"
	"messenger/internal/repositories"

	"go.uber.org/zap"
)

// PlatformAndroid is the only platform we currently ship binaries for.
// Kept as a typed constant so adding ios later doesn't sprinkle string
// literals across handlers.
const PlatformAndroid = "android"

// Bounds on user-controllable fields so a misbehaving admin client can't
// pollute the DB with multi-MB changelogs or pathological version strings.
const (
	maxVersionNameLen = 32
	maxChangelogLen   = 8 * 1024
	maxURLLen         = 2048
)

// sha256Re matches a lowercase hex SHA-256 — 64 chars. Releases must arrive
// with the digest already computed (CI computes it from the signed APK);
// we just sanity-check the shape here so callers fail fast on typos.
var sha256Re = regexp.MustCompile(`^[a-f0-9]{64}$`)

// ErrReleaseConflict is returned when the (platform, version_code) tuple
// already exists. Composite unique index on the table catches this, but
// we want a typed error so the handler can return 409 instead of 500.
var ErrReleaseConflict = errors.New("release with this version_code already exists for platform")

// AppReleaseService is the business-logic layer between the admin handler
// and the repository. Validation lives here, not in the handler.
type AppReleaseService struct {
	log  *zap.Logger
	repo *repositories.AppReleaseRepo
}

// NewAppReleaseService constructs the service.
func NewAppReleaseService(log *zap.Logger, repo *repositories.AppReleaseRepo) *AppReleaseService {
	return &AppReleaseService{log: log, repo: repo}
}

// GetLatest returns the most recent release for the platform, or (nil, nil)
// if no releases exist. The handler renders nil → 204 No Content so the
// client can treat it as "no update available, you're on the freshest build".
func (s *AppReleaseService) GetLatest(ctx context.Context, platform string) (*models.AppRelease, error) {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		platform = PlatformAndroid
	}
	return s.repo.GetLatest(ctx, platform)
}

// CreateReleaseRequest is the wire shape for POST /api/admin/releases.
type CreateReleaseRequest struct {
	Platform                string `json:"platform"`
	VersionName             string `json:"version_name"`
	VersionCode             int    `json:"version_code"`
	MinSupportedVersionCode int    `json:"min_supported_version_code"`
	URL                     string `json:"url"`
	SHA256                  string `json:"sha256"`
	SizeBytes               int64  `json:"size_bytes"`
	Changelog               string `json:"changelog"`
}

// CreateRelease validates + inserts a new release row. Auth happens upstream
// at the handler (admin-only via X-Admin-Key); this function trusts the
// caller has already cleared that gate, but does not trust the payload.
func (s *AppReleaseService) CreateRelease(ctx context.Context, req CreateReleaseRequest) (*models.AppRelease, error) {
	platform := strings.TrimSpace(req.Platform)
	if platform == "" {
		platform = PlatformAndroid
	}
	if platform != PlatformAndroid {
		return nil, fmt.Errorf("unsupported platform %q", platform)
	}

	versionName := strings.TrimSpace(req.VersionName)
	if versionName == "" || len(versionName) > maxVersionNameLen {
		return nil, fmt.Errorf("version_name must be 1..%d chars", maxVersionNameLen)
	}

	if req.VersionCode <= 0 {
		return nil, errors.New("version_code must be positive")
	}
	if req.MinSupportedVersionCode <= 0 || req.MinSupportedVersionCode > req.VersionCode {
		return nil, errors.New("min_supported_version_code must be in (0, version_code]")
	}

	url := strings.TrimSpace(req.URL)
	if url == "" || len(url) > maxURLLen {
		return nil, fmt.Errorf("url must be 1..%d chars", maxURLLen)
	}
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return nil, errors.New("url must be http(s)://")
	}

	digest := strings.ToLower(strings.TrimSpace(req.SHA256))
	if !sha256Re.MatchString(digest) {
		return nil, errors.New("sha256 must be 64 lowercase hex chars")
	}

	if req.SizeBytes <= 0 {
		return nil, errors.New("size_bytes must be positive")
	}

	changelog := req.Changelog
	if len(changelog) > maxChangelogLen {
		return nil, fmt.Errorf("changelog must be ≤ %d bytes", maxChangelogLen)
	}

	rel := &models.AppRelease{
		Platform:                platform,
		VersionName:             versionName,
		VersionCode:             req.VersionCode,
		MinSupportedVersionCode: req.MinSupportedVersionCode,
		URL:                     url,
		SHA256:                  digest,
		SizeBytes:               req.SizeBytes,
		Changelog:               changelog,
		ReleasedAt:              time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, rel); err != nil {
		if strings.Contains(err.Error(), "idx_releases_platform_code") ||
			strings.Contains(err.Error(), "duplicate key") {
			return nil, ErrReleaseConflict
		}
		return nil, fmt.Errorf("persist release: %w", err)
	}
	s.log.Info("registered new app release",
		zap.String("platform", platform),
		zap.String("version_name", versionName),
		zap.Int("version_code", rel.VersionCode),
		zap.Int("min_supported", rel.MinSupportedVersionCode),
		zap.Int64("size_bytes", rel.SizeBytes),
	)
	return rel, nil
}

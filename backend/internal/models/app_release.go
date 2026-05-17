package models

import (
	"time"

	"gorm.io/gorm"
)

// AppRelease describes one released build of the mobile (or future iOS) app
// that the in-app self-update flow can offer to clients. One row per release.
//
//   - VersionCode is the monotonic integer Android uses to compare versions;
//     it MUST strictly increase across releases.
//   - MinSupportedVersionCode is the floor: clients with VersionCode strictly
//     below this value MUST update before they can keep using the app. Used
//     to push security fixes that can't be left running on old builds.
//   - SHA256 is the hex-encoded digest of the APK at URL; the client verifies
//     this byte-for-byte after download, before handing the file to the
//     Android PackageInstaller. Defends against MITM during download and
//     against accidental corruption.
//   - URL is the public download address (typically a presigned MinIO URL or
//     a fronted CDN path). Clients must be able to GET it without auth.
//
// Composite unique index `idx_releases_platform_code` guarantees one
// (platform, version_code) tuple — no accidental duplicate releases.
type AppRelease struct {
	gorm.Model
	Platform                string    `gorm:"type:varchar(16);not null;uniqueIndex:idx_releases_platform_code,priority:1" json:"platform"`
	VersionName             string    `gorm:"type:varchar(32);not null"                                                   json:"version_name"`
	VersionCode             int       `gorm:"not null;uniqueIndex:idx_releases_platform_code,priority:2"                  json:"version_code"`
	MinSupportedVersionCode int       `gorm:"not null"                                                                    json:"min_supported_version_code"`
	URL                     string    `gorm:"type:text;not null"                                                          json:"url"`
	SHA256                  string    `gorm:"type:char(64);not null"                                                      json:"sha256"`
	SizeBytes               int64     `gorm:"not null"                                                                    json:"size_bytes"`
	Changelog               string    `gorm:"type:text"                                                                   json:"changelog"`
	ReleasedAt              time.Time `gorm:"not null;index"                                                              json:"released_at"`
}

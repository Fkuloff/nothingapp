// internal/storage/storage.go
package storage

import (
	"io"
	"time"
)

// FileMetadata contains metadata about a stored file
type FileMetadata struct {
	UploadedAt   time.Time
	Key          string
	FileName     string
	ContentType  string
	URL          string
	ThumbnailURL string
	Size         int64
}

// Storage defines the interface for file storage operations
// This interface allows easy switching between local and cloud storage (S3, etc.)
type Storage interface {
	// Save stores a file and returns metadata
	Save(reader io.Reader, fileName, contentType string, size int64) (*FileMetadata, error)

	// Get retrieves a file by its storage key
	Get(key string) (io.ReadCloser, error)

	// Delete removes a file from storage
	Delete(key string) error

	// GetURL returns the access URL for a file
	GetURL(key string) string

	// GetThumbnailURL returns the access URL for a thumbnail
	GetThumbnailURL(key string) string
}

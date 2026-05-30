// internal/storage/storage.go
package storage

import "io"

// FileMetadata contains metadata about a stored file
type FileMetadata struct {
	Key         string
	FileName    string
	ContentType string
	URL         string
	Size        int64
}

// Storage defines the interface for file storage operations
// This interface allows easy switching between local and cloud storage (S3, etc.)
type Storage interface {
	// Save stores a file and returns metadata
	Save(reader io.Reader, fileName, contentType string, size int64) (*FileMetadata, error)

	// Get retrieves a file by its storage key
	Get(key string) (io.ReadCloser, error)

	// Copy duplicates the object at sourceKey into a freshly-generated key and
	// returns the new key. Used by message forwarding so the destination chat
	// gets an independent copy of an (encrypted) attachment body without the
	// client re-uploading it. Server-side copy — bytes never transit the app.
	Copy(sourceKey string) (string, error)

	// Delete removes a file from storage
	Delete(key string) error

	// GetURL returns the access URL for a file (presigned for private files)
	GetURL(key string) string
}

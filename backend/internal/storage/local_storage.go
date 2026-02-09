// internal/storage/local_storage.go
package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// LocalStorage implements Storage interface for local filesystem
type LocalStorage struct {
	basePath      string // Base directory for uploads (e.g., "./uploads")
	baseURL       string // Base URL for serving files (e.g., "http://localhost:8080/uploads")
	filesPath     string // Directory for files (basePath/files)
	thumbnailPath string // Directory for thumbnails (basePath/thumbnails)
}

// NewLocalStorage creates a new LocalStorage instance
func NewLocalStorage(basePath, baseURL string) (*LocalStorage, error) {
	ls := &LocalStorage{
		basePath:      basePath,
		baseURL:       baseURL,
		filesPath:     filepath.Join(basePath, "files"),
		thumbnailPath: filepath.Join(basePath, "thumbnails"),
	}

	// Create base directories
	if err := os.MkdirAll(ls.filesPath, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create files directory: %w", err)
	}
	if err := os.MkdirAll(ls.thumbnailPath, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create thumbnails directory: %w", err)
	}

	return ls, nil
}

// Save stores a file and returns metadata
func (ls *LocalStorage) Save(reader io.Reader, fileName, contentType string, size int64) (*FileMetadata, error) {
	// Generate unique filename with UUID
	ext := filepath.Ext(fileName)
	uniqueID := uuid.New().String()
	uniqueFileName := fmt.Sprintf("%s%s", uniqueID, ext)

	// Create date-based directory structure: YYYY/MM/DD
	now := time.Now()
	dateDir := filepath.Join(
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fmt.Sprintf("%02d", now.Day()),
	)

	fullDir := filepath.Join(ls.filesPath, dateDir)
	if err := os.MkdirAll(fullDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Full file path
	filePath := filepath.Join(fullDir, uniqueFileName)

	// Create and write file
	file, err := os.Create(filePath) //nolint:gosec // G304: File path is controlled by storage logic
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log error but don't obscure original error
			if err == nil {
				err = fmt.Errorf("failed to close file: %w", closeErr)
			}
		}
	}()

	written, err := io.Copy(file, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Storage key is relative path from files directory
	storageKey := filepath.Join("files", dateDir, uniqueFileName)

	metadata := &FileMetadata{
		Key:         storageKey,
		FileName:    fileName,
		ContentType: contentType,
		Size:        written,
		URL:         ls.GetURL(storageKey),
		UploadedAt:  now,
	}

	return metadata, nil
}

// Get retrieves a file by its storage key
func (ls *LocalStorage) Get(key string) (io.ReadCloser, error) {
	filePath := filepath.Join(ls.basePath, key)

	file, err := os.Open(filePath) //nolint:gosec // G304: File path is controlled by storage logic
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete removes a file from storage
func (ls *LocalStorage) Delete(key string) error {
	filePath := filepath.Join(ls.basePath, key)

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetURL returns the access URL for a file
func (ls *LocalStorage) GetURL(key string) string {
	// Convert Windows backslashes to forward slashes for URLs
	urlPath := filepath.ToSlash(key)
	return fmt.Sprintf("%s/%s", ls.baseURL, urlPath)
}

// GetThumbnailURL returns the access URL for a thumbnail
func (ls *LocalStorage) GetThumbnailURL(key string) string {
	// Convert Windows backslashes to forward slashes for URLs
	urlPath := filepath.ToSlash(key)
	return fmt.Sprintf("%s/%s", ls.baseURL, urlPath)
}

// SaveThumbnail saves a thumbnail file and returns its metadata
func (ls *LocalStorage) SaveThumbnail(reader io.Reader, originalKey string) (*FileMetadata, error) {
	// Extract date path from original key
	// originalKey format: files/YYYY/MM/DD/uuid.ext
	// Get date directory (YYYY/MM/DD) from the path structure
	var dateDir string

	// Try to extract date directory from original key path
	// Get the directory path and extract the last 3 components (YYYY/MM/DD)
	dirPath := filepath.Dir(originalKey)
	parts := filepath.SplitList(dirPath)

	// Check if path has enough components
	if len(parts) >= 3 {
		// Take last 3 parts for YYYY/MM/DD
		dateDir = filepath.Join(parts[len(parts)-3], parts[len(parts)-2], parts[len(parts)-1])
	} else {
		// Fallback to current date
		now := time.Now()
		dateDir = filepath.Join(
			fmt.Sprintf("%04d", now.Year()),
			fmt.Sprintf("%02d", now.Month()),
			fmt.Sprintf("%02d", now.Day()),
		)
	}

	// Create thumbnail directory
	fullDir := filepath.Join(ls.thumbnailPath, dateDir)
	if err := os.MkdirAll(fullDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	// Generate unique thumbnail filename
	uniqueID := uuid.New().String()
	thumbFileName := fmt.Sprintf("%s_thumb.jpg", uniqueID)
	thumbPath := filepath.Join(fullDir, thumbFileName)

	// Create and write thumbnail file
	file, err := os.Create(thumbPath) //nolint:gosec // G304: File path is controlled by storage logic
	if err != nil {
		return nil, fmt.Errorf("failed to create thumbnail: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log error but don't obscure original error
			if err == nil {
				err = fmt.Errorf("failed to close file: %w", closeErr)
			}
		}
	}()

	written, err := io.Copy(file, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to write thumbnail: %w", err)
	}

	// Storage key for thumbnail
	storageKey := filepath.Join("thumbnails", dateDir, thumbFileName)

	metadata := &FileMetadata{
		Key:         storageKey,
		FileName:    thumbFileName,
		ContentType: "image/jpeg",
		Size:        written,
		URL:         ls.GetThumbnailURL(storageKey),
		UploadedAt:  time.Now(),
	}

	return metadata, nil
}

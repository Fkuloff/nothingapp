package services

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"messenger/internal/models"
)

// File size limits
const (
	MaxImageSize    = 100 * 1024 * 1024 // 100MB
	MaxVideoSize    = 500 * 1024 * 1024 // 500MB
	MaxDocumentSize = 50 * 1024 * 1024  // 50MB
	MaxAvatarSize   = 10 * 1024 * 1024  // 10MB
)

// Allowed MIME types by category
var (
	AllowedImageTypes = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}

	AllowedVideoTypes = map[string]bool{
		"video/mp4":       true,
		"video/quicktime": true,
		"video/x-msvideo": true,
		"video/webm":      true,
	}

	AllowedDocumentTypes = map[string]bool{
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
		"text/plain": true,
	}
)

// FileValidator provides centralized file validation logic
type FileValidator struct{}

// ValidateAttachment validates a file for attachment upload
func (v *FileValidator) ValidateAttachment(fileHeader *multipart.FileHeader) error {
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		return fmt.Errorf("content type not specified")
	}

	// Check if file type is allowed
	if !v.isAllowedMimeType(contentType) {
		return fmt.Errorf("unsupported file type: %s", contentType)
	}

	// Check file size based on type
	maxSize := v.getMaxFileSize(contentType)
	if fileHeader.Size > maxSize {
		return fmt.Errorf("file too large (max %d MB)", maxSize/(1024*1024))
	}

	// Validate filename for security
	if err := v.validateFilename(fileHeader.Filename); err != nil {
		return err
	}

	return nil
}

// ValidateAvatar validates a file for avatar upload
func (v *FileValidator) ValidateAvatar(fileHeader *multipart.FileHeader) error {
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		return fmt.Errorf("content type not specified")
	}

	// Check if it's an image
	if !AllowedImageTypes[contentType] {
		return fmt.Errorf("avatar must be an image (JPEG, PNG, GIF, or WebP)")
	}

	// Check size
	if fileHeader.Size > MaxAvatarSize {
		return fmt.Errorf("avatar too large (max 10 MB)")
	}

	// Validate filename
	if err := v.validateFilename(fileHeader.Filename); err != nil {
		return err
	}

	return nil
}

// isAllowedMimeType checks if a MIME type is allowed for any attachment type
func (v *FileValidator) isAllowedMimeType(mimeType string) bool {
	return AllowedImageTypes[mimeType] || AllowedVideoTypes[mimeType] || AllowedDocumentTypes[mimeType]
}

// getMaxFileSize returns the maximum allowed size for a given MIME type
func (v *FileValidator) getMaxFileSize(mimeType string) int64 {
	if AllowedImageTypes[mimeType] {
		return MaxImageSize
	}
	if AllowedVideoTypes[mimeType] {
		return MaxVideoSize
	}
	if AllowedDocumentTypes[mimeType] {
		return MaxDocumentSize
	}
	return 0
}

// DetermineFileType returns the attachment type based on MIME type
func (v *FileValidator) DetermineFileType(mimeType string) models.AttachmentType {
	if AllowedImageTypes[mimeType] {
		return models.AttachmentTypeImage
	}
	if AllowedVideoTypes[mimeType] {
		return models.AttachmentTypeVideo
	}
	return models.AttachmentTypeDocument
}

// validateFilename checks for path traversal and other security issues
func (v *FileValidator) validateFilename(filename string) error {
	// Prevent path traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return fmt.Errorf("invalid filename")
	}

	// Ensure filename has an extension
	ext := filepath.Ext(filename)
	if ext == "" {
		return fmt.Errorf("filename must have an extension")
	}

	return nil
}

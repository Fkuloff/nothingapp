package services

import (
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"messenger/internal/models"
)

// File size limits
const (
	MaxFileSize   = 20 * 1024 * 1024 // 20MB - единый лимит для всех типов файлов
	MaxAvatarSize = 10 * 1024 * 1024 // 10MB
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

// ValidateAttachment validates a file for attachment upload.
func (v *FileValidator) ValidateAttachment(fileHeader *multipart.FileHeader) error {
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		return errors.New("content type not specified")
	}

	// Check if file type is allowed
	if !v.isAllowedMimeType(contentType) {
		return fmt.Errorf("unsupported file type: %s", contentType)
	}

	// Check file size
	if fileHeader.Size > MaxFileSize {
		return fmt.Errorf("file too large (max %d MB)", MaxFileSize/(1024*1024))
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
		return errors.New("content type not specified")
	}

	// Check if it's an image
	if !AllowedImageTypes[contentType] {
		return errors.New("avatar must be an image (JPEG, PNG, GIF, or WebP)")
	}

	// Check size
	if fileHeader.Size > MaxAvatarSize {
		return errors.New("avatar too large (max 10 MB)")
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
		return errors.New("invalid filename")
	}

	// Ensure filename has an extension
	if filepath.Ext(filename) == "" {
		return errors.New("filename must have an extension")
	}

	return nil
}

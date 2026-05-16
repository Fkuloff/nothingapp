package services

import (
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
)

// File size limits.
const (
	maxFileSize          = 20 * 1024 * 1024  // 20MB — legacy server-side validation limit (mime-checked)
	maxEncryptedFileSize = 100 * 1024 * 1024 // 100MB — E2E ciphertext, server only checks size
	maxAvatarSize        = 10 * 1024 * 1024  // 10MB
)

// Allowed MIME types by category.
var (
	allowedImageTypes = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}

	allowedVideoTypes = map[string]bool{
		"video/mp4":       true,
		"video/quicktime": true,
		"video/x-msvideo": true,
		"video/webm":      true,
	}

	allowedDocumentTypes = map[string]bool{
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
		"text/plain": true,
	}
)

// fileValidator provides centralized file validation logic
type fileValidator struct{}

// validateAttachment validates a file for attachment upload.
func (v *fileValidator) validateAttachment(fileHeader *multipart.FileHeader) error {
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		return errors.New("content type not specified")
	}

	// Check if file type is allowed
	if !v.isAllowedMimeType(contentType) {
		return fmt.Errorf("unsupported file type: %s", contentType)
	}

	// Check file size
	if fileHeader.Size > maxFileSize {
		return fmt.Errorf("file too large (max %d MB)", maxFileSize/(1024*1024))
	}

	// Validate filename for security
	if err := v.validateFilename(fileHeader.Filename); err != nil {
		return err
	}

	return nil
}

// validateAttachmentSizeOnly validates only what's still meaningful for an
// E2E (scheme=2) attachment: file size + path-traversal safety. We can't
// inspect the content type (opaque ciphertext) and we deliberately don't
// require a filename extension here — the frontend sends a fixed "blob"
// for every encrypted upload so the multipart Content-Disposition (and the
// resulting S3 storage_key) carries no extension that could leak the real
// file type. The real filename + mime live inside the encrypted_metadata
// blob the receiver decrypts client-side.
func (v *fileValidator) validateAttachmentSizeOnly(fileHeader *multipart.FileHeader) error {
	if fileHeader.Size > maxEncryptedFileSize {
		return fmt.Errorf("encrypted file too large (max %d MB)", maxEncryptedFileSize/(1024*1024))
	}
	// Path-traversal guard — same as validateFilename minus the extension
	// requirement. Defensive even though the storage layer uses UUIDs.
	name := fileHeader.Filename
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return errors.New("invalid filename")
	}
	return nil
}

// validateAvatar validates a file for avatar upload
func (v *fileValidator) validateAvatar(fileHeader *multipart.FileHeader) error {
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		return errors.New("content type not specified")
	}

	// Check if it's an image
	if !allowedImageTypes[contentType] {
		return errors.New("avatar must be an image (JPEG, PNG, GIF, or WebP)")
	}

	// Check size
	if fileHeader.Size > maxAvatarSize {
		return errors.New("avatar too large (max 10 MB)")
	}

	// Validate filename
	if err := v.validateFilename(fileHeader.Filename); err != nil {
		return err
	}

	return nil
}

// isAllowedMimeType checks if a MIME type is allowed for any attachment type
func (v *fileValidator) isAllowedMimeType(mimeType string) bool {
	return allowedImageTypes[mimeType] || allowedVideoTypes[mimeType] || allowedDocumentTypes[mimeType]
}

// validateFilename checks for path traversal and other security issues
func (v *fileValidator) validateFilename(filename string) error {
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

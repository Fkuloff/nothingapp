package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	"messenger/internal/models"
	"messenger/internal/repositories"
	"messenger/internal/storage"

	"go.uber.org/zap"
)

var (
	ErrMessageNotFound    = errors.New("message not found")
	ErrNotMessageOwner    = errors.New("unauthorized: not message owner")
	ErrAttachmentNotFound = errors.New("attachment not found")
)

// AttachmentCryptoMeta holds E2E encryption metadata for file uploads.
// All fields are JSON strings: {"original_filename": "value", ...}
type AttachmentCryptoMeta struct {
	FileIVs       string // JSON map of filename → base64 IV
	OriginalTypes string // JSON map of filename → original MIME type
	OriginalNames string // JSON map of filename → original filename
}

// AttachmentService handles business logic for attachments
type AttachmentService struct {
	logger         *zap.Logger
	attachmentRepo *repositories.AttachmentRepo
	messageRepo    *repositories.MessageRepo
	storage        storage.Storage
	validator      *FileValidator
}

// NewAttachmentService creates a new AttachmentService instance
func NewAttachmentService(
	logger *zap.Logger,
	attachmentRepo *repositories.AttachmentRepo,
	messageRepo *repositories.MessageRepo,
	storage storage.Storage,
) *AttachmentService {
	return &AttachmentService{
		logger:         logger,
		attachmentRepo: attachmentRepo,
		messageRepo:    messageRepo,
		storage:        storage,
		validator:      &FileValidator{},
	}
}

// UploadAttachments uploads multiple files and creates attachment records.
// cryptoMeta is optional — when present, files are treated as E2E encrypted.
func (s *AttachmentService) UploadAttachments(
	ctx context.Context,
	messageID uint,
	userID uint,
	files []*multipart.FileHeader,
	cryptoMeta *AttachmentCryptoMeta,
) ([]*models.Attachment, error) {
	// Verify message exists and user owns it
	message, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return nil, ErrMessageNotFound
	}
	if message.UserID != userID {
		return nil, ErrNotMessageOwner
	}

	// Parse E2E crypto metadata if provided
	ivMap, origTypeMap, origNameMap := s.parseCryptoMeta(cryptoMeta)

	var attachments []*models.Attachment
	var uploadedKeys []string // Track uploaded files for rollback

	isEncrypted := cryptoMeta != nil && len(ivMap) > 0

	for _, fileHeader := range files {
		attachment, storageKey, err := s.processFileUpload(fileHeader, messageID, isEncrypted, ivMap, origTypeMap, origNameMap)
		if err != nil {
			s.rollbackUploads(uploadedKeys)
			return nil, err
		}
		uploadedKeys = append(uploadedKeys, storageKey)
		attachments = append(attachments, attachment)
	}

	// Save all attachments in batch
	if err := s.attachmentRepo.CreateBatch(ctx, attachments); err != nil {
		// Rollback: delete all uploaded files
		s.rollbackUploads(uploadedKeys)
		return nil, fmt.Errorf("save attachments to database: %w", err)
	}

	return attachments, nil
}

// processFileUpload validates, uploads, and creates an attachment record for a single file.
func (s *AttachmentService) processFileUpload(
	fileHeader *multipart.FileHeader,
	messageID uint,
	isEncrypted bool,
	ivMap, origTypeMap, origNameMap map[string]string,
) (*models.Attachment, string, error) {
	if err := s.validator.ValidateAttachment(fileHeader, isEncrypted); err != nil {
		return nil, "", fmt.Errorf("invalid file %s: %w", fileHeader.Filename, err)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, "", fmt.Errorf("open file %s: %w", fileHeader.Filename, err)
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if origType, ok := origTypeMap[fileHeader.Filename]; ok && origType != "" {
		contentType = origType
	}
	fileType := s.validator.DetermineFileType(contentType)

	metadata, err := s.storage.Save(file, fileHeader.Filename, contentType, fileHeader.Size)
	if closeErr := file.Close(); closeErr != nil {
		s.logger.Warn("failed to close file after upload", zap.Error(closeErr), zap.String("filename", fileHeader.Filename))
	}
	if err != nil {
		return nil, "", fmt.Errorf("save file %s: %w", fileHeader.Filename, err)
	}

	attachment := &models.Attachment{
		MessageID:    messageID,
		FileType:     fileType,
		StorageKey:   metadata.Key,
		FileName:     metadata.FileName,
		FileSize:     metadata.Size,
		MimeType:     metadata.ContentType,
		IV:           ivMap[fileHeader.Filename],
		OriginalType: origTypeMap[fileHeader.Filename],
		OriginalName: origNameMap[fileHeader.Filename],
	}

	return attachment, metadata.Key, nil
}

// DeleteAttachment deletes an attachment
func (s *AttachmentService) DeleteAttachment(ctx context.Context, attachmentID, userID uint) error {
	attachment, err := s.attachmentRepo.FindByID(ctx, attachmentID)
	if err != nil {
		return ErrAttachmentNotFound
	}

	// Verify ownership through message
	message, err := s.messageRepo.FindByID(ctx, attachment.MessageID)
	if err != nil || message.UserID != userID {
		return ErrNotMessageOwner
	}

	// Delete from storage
	if err := s.storage.Delete(attachment.StorageKey); err != nil {
		// Log error but continue with database deletion
		s.logger.Warn("failed to delete file from storage", zap.Error(err), zap.String("storage_key", attachment.StorageKey))
	}

	// Delete from database
	if err := s.attachmentRepo.Delete(ctx, attachmentID); err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}

	return nil
}

// GetAttachment retrieves an attachment and its file reader
func (s *AttachmentService) GetAttachment(ctx context.Context, attachmentID uint) (*models.Attachment, io.ReadCloser, error) {
	attachment, err := s.attachmentRepo.FindByID(ctx, attachmentID)
	if err != nil {
		return nil, nil, ErrAttachmentNotFound
	}

	reader, err := s.storage.Get(attachment.StorageKey)
	if err != nil {
		return nil, nil, fmt.Errorf("retrieve file: %w", err)
	}

	return attachment, reader, nil
}

// parseCryptoMeta parses E2E encryption metadata JSON strings into maps.
// Returns empty maps if metadata is nil or fields are empty.
func (s *AttachmentService) parseCryptoMeta(meta *AttachmentCryptoMeta) (map[string]string, map[string]string, map[string]string) {
	ivs := make(map[string]string)
	origTypes := make(map[string]string)
	origNames := make(map[string]string)

	if meta == nil {
		return ivs, origTypes, origNames
	}

	if meta.FileIVs != "" {
		if err := json.Unmarshal([]byte(meta.FileIVs), &ivs); err != nil {
			s.logger.Warn("parse file_ivs metadata", zap.Error(err))
		}
	}
	if meta.OriginalTypes != "" {
		if err := json.Unmarshal([]byte(meta.OriginalTypes), &origTypes); err != nil {
			s.logger.Warn("parse original_types metadata", zap.Error(err))
		}
	}
	if meta.OriginalNames != "" {
		if err := json.Unmarshal([]byte(meta.OriginalNames), &origNames); err != nil {
			s.logger.Warn("parse original_names metadata", zap.Error(err))
		}
	}

	return ivs, origTypes, origNames
}

// rollbackUploads deletes uploaded files in case of error
func (s *AttachmentService) rollbackUploads(keys []string) {
	for _, key := range keys {
		if err := s.storage.Delete(key); err != nil {
			s.logger.Warn("failed to delete file during rollback", zap.Error(err), zap.String("storage_key", key))
		}
	}
}

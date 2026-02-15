package services

import (
	"context"
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
func (s *AttachmentService) UploadAttachments(
	ctx context.Context,
	messageID uint,
	userID uint,
	files []*multipart.FileHeader,
) ([]*models.Attachment, error) {
	// Verify message exists and user owns it
	message, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return nil, ErrMessageNotFound
	}
	if message.UserID != userID {
		return nil, ErrNotMessageOwner
	}

	var attachments []*models.Attachment
	var uploadedKeys []string // Track uploaded files for rollback

	for _, fileHeader := range files {
		attachment, storageKey, err := s.processFileUpload(fileHeader, messageID)
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
) (*models.Attachment, string, error) {
	if err := s.validator.ValidateAttachment(fileHeader); err != nil {
		return nil, "", fmt.Errorf("invalid file %s: %w", fileHeader.Filename, err)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, "", fmt.Errorf("open file %s: %w", fileHeader.Filename, err)
	}

	contentType := fileHeader.Header.Get("Content-Type")
	fileType := s.validator.DetermineFileType(contentType)

	metadata, err := s.storage.Save(file, fileHeader.Filename, contentType, fileHeader.Size)
	if closeErr := file.Close(); closeErr != nil {
		s.logger.Warn("failed to close file after upload", zap.Error(closeErr), zap.String("filename", fileHeader.Filename))
	}
	if err != nil {
		return nil, "", fmt.Errorf("save file %s: %w", fileHeader.Filename, err)
	}

	attachment := &models.Attachment{
		MessageID:  messageID,
		FileType:   fileType,
		StorageKey: metadata.Key,
		FileName:   metadata.FileName,
		FileSize:   metadata.Size,
		MimeType:   metadata.ContentType,
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

// rollbackUploads deletes uploaded files in case of error
func (s *AttachmentService) rollbackUploads(keys []string) {
	for _, key := range keys {
		if err := s.storage.Delete(key); err != nil {
			s.logger.Warn("failed to delete file during rollback", zap.Error(err), zap.String("storage_key", key))
		}
	}
}

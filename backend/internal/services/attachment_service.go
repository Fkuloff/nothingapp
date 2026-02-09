package services

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"messenger/internal/models"
	"messenger/internal/repositories"
	"messenger/internal/storage"

	"go.uber.org/zap"
)

// AttachmentService handles business logic for attachments
type AttachmentService struct {
	logger         *zap.Logger
	attachmentRepo *repositories.AttachmentRepo
	messageRepo    *repositories.MessageRepo
	storage        storage.Storage
	thumbnailGen   *ThumbnailGenerator
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
		thumbnailGen:   NewThumbnailGenerator(storage),
		validator:      &FileValidator{},
	}
}

// UploadAttachments uploads multiple files and creates attachment records
func (s *AttachmentService) UploadAttachments(
	ctx context.Context,
	messageID uint,
	userID uint,
	files []*multipart.FileHeader,
) ([]*models.Attachment, error) {
	// Verify message exists and user owns it
	message, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("message not found")
	}
	if message.UserID != userID {
		return nil, fmt.Errorf("unauthorized: not message owner")
	}

	var attachments []*models.Attachment
	var uploadedKeys []string // Track uploaded files for rollback

	for _, fileHeader := range files {
		// Validate file
		if err := s.validator.ValidateAttachment(fileHeader); err != nil {
			// Rollback: delete already uploaded files
			s.rollbackUploads(uploadedKeys)
			return nil, fmt.Errorf("invalid file %s: %w", fileHeader.Filename, err)
		}

		// Open file
		file, err := fileHeader.Open()
		if err != nil {
			s.rollbackUploads(uploadedKeys)
			return nil, fmt.Errorf("failed to open file %s: %w", fileHeader.Filename, err)
		}

		// Determine file type
		contentType := fileHeader.Header.Get("Content-Type")
		fileType := s.validator.DetermineFileType(contentType)

		// Save to storage
		metadata, err := s.storage.Save(
			file,
			fileHeader.Filename,
			contentType,
			fileHeader.Size,
		)
		if closeErr := file.Close(); closeErr != nil {
			s.logger.Warn("failed to close file after upload", zap.Error(closeErr), zap.String("filename", fileHeader.Filename))
		}

		if err != nil {
			s.rollbackUploads(uploadedKeys)
			return nil, fmt.Errorf("failed to save file %s: %w", fileHeader.Filename, err)
		}

		uploadedKeys = append(uploadedKeys, metadata.Key)

		// Create attachment record
		attachment := &models.Attachment{
			MessageID:  messageID,
			FileType:   fileType,
			StorageKey: metadata.Key,
			FileName:   metadata.FileName,
			FileSize:   metadata.Size,
			MimeType:   metadata.ContentType,
		}

		// Generate thumbnail for images/videos
		if attachment.RequiresThumbnail() {
			thumbMeta, err := s.thumbnailGen.Generate(metadata.Key, 300, 300)
			if err == nil && thumbMeta != nil {
				attachment.ThumbnailKey = &thumbMeta.Key
				uploadedKeys = append(uploadedKeys, thumbMeta.Key)
			}
			// Don't fail if thumbnail generation fails
		}

		attachments = append(attachments, attachment)
	}

	// Save all attachments in batch
	if err := s.attachmentRepo.CreateBatch(ctx, attachments); err != nil {
		// Rollback: delete all uploaded files
		s.rollbackUploads(uploadedKeys)
		return nil, fmt.Errorf("failed to save attachments to database: %w", err)
	}

	return attachments, nil
}

// DeleteAttachment deletes an attachment
func (s *AttachmentService) DeleteAttachment(ctx context.Context, attachmentID, userID uint) error {
	attachment, err := s.attachmentRepo.FindByID(ctx, attachmentID)
	if err != nil {
		return fmt.Errorf("attachment not found")
	}

	// Verify ownership through message
	message, err := s.messageRepo.FindByID(ctx, attachment.MessageID)
	if err != nil || message.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Delete from storage
	if err := s.storage.Delete(attachment.StorageKey); err != nil {
		// Log error but continue with database deletion
		s.logger.Warn("failed to delete file from storage", zap.Error(err), zap.String("storage_key", attachment.StorageKey))
	}

	// Delete thumbnail if exists
	if attachment.ThumbnailKey != nil {
		if err := s.storage.Delete(*attachment.ThumbnailKey); err != nil {
			s.logger.Warn("failed to delete thumbnail from storage", zap.Error(err), zap.String("thumbnail_key", *attachment.ThumbnailKey))
		}
	}

	// Delete from database
	if err := s.attachmentRepo.Delete(ctx, attachmentID); err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	return nil
}

// GetAttachment retrieves an attachment and its file reader
func (s *AttachmentService) GetAttachment(ctx context.Context, attachmentID uint) (*models.Attachment, io.ReadCloser, error) {
	attachment, err := s.attachmentRepo.FindByID(ctx, attachmentID)
	if err != nil {
		return nil, nil, fmt.Errorf("attachment not found")
	}

	reader, err := s.storage.Get(attachment.StorageKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve file: %w", err)
	}

	return attachment, reader, nil
}

// GetThumbnail retrieves a thumbnail
func (s *AttachmentService) GetThumbnail(ctx context.Context, attachmentID uint) (*models.Attachment, io.ReadCloser, error) {
	attachment, err := s.attachmentRepo.FindByID(ctx, attachmentID)
	if err != nil {
		return nil, nil, fmt.Errorf("attachment not found")
	}

	if attachment.ThumbnailKey == nil {
		return nil, nil, fmt.Errorf("thumbnail not available")
	}

	reader, err := s.storage.Get(*attachment.ThumbnailKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve thumbnail: %w", err)
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

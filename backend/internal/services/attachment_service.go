package services

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"

	"messenger/internal/models"
	"messenger/internal/repositories"
	"messenger/internal/storage"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrMessageNotFound      = errors.New("message not found")
	ErrNotMessageOwner      = errors.New("unauthorized: not message owner")
	ErrAttachmentNotFound   = errors.New("attachment not found")
	ErrAttachmentMetaShape  = errors.New("attachment metadata count must equal file count")
	ErrAttachmentMetaFields = errors.New("attachment metadata: file_iv and envelopes are required")
)

// AttachmentEnvelopeInput is one per-recipient wrapped file_key, mirrored from
// the wire-level multipart metadata into the service-layer shape.
type AttachmentEnvelopeInput struct {
	RecipientID      uint
	EncryptedFileKey string
	IV               string
}

// AttachmentMetaInput is the per-file metadata accompanying a multipart upload:
// the file body's own AES-GCM IV (so recipients can decrypt the blob once they
// have file_key) plus the per-recipient wrapped file_key envelopes.
//
// MimeType is what the client claims the original file is. The server records
// it but cannot verify (the body is ciphertext). FileName + FileSize come
// from the multipart part itself.
type AttachmentMetaInput struct {
	FileIV    string
	MimeType  string
	Envelopes []AttachmentEnvelopeInput
}

// AttachmentService handles business logic for attachments
type AttachmentService struct {
	db              *gorm.DB
	logger          *zap.Logger
	attachmentRepo  *repositories.AttachmentRepo
	envelopeRepo    *repositories.AttachmentEnvelopeRepo
	messageRepo     *repositories.MessageRepo
	chatRepo        *repositories.ChatRepo
	participantRepo *repositories.ChatParticipantRepo
	storage         storage.Storage
	validator       *fileValidator
}

// NewAttachmentService creates a new AttachmentService instance
func NewAttachmentService(
	db *gorm.DB,
	logger *zap.Logger,
	attachmentRepo *repositories.AttachmentRepo,
	envelopeRepo *repositories.AttachmentEnvelopeRepo,
	messageRepo *repositories.MessageRepo,
	chatRepo *repositories.ChatRepo,
	participantRepo *repositories.ChatParticipantRepo,
	storage storage.Storage,
) *AttachmentService {
	return &AttachmentService{
		db:              db,
		logger:          logger,
		attachmentRepo:  attachmentRepo,
		envelopeRepo:    envelopeRepo,
		messageRepo:     messageRepo,
		chatRepo:        chatRepo,
		participantRepo: participantRepo,
		storage:         storage,
		validator:       &fileValidator{},
	}
}

// UploadAttachments uploads multiple files, encrypted client-side, plus their
// per-recipient envelopes. The flow is:
//
//  1. Validate ownership of the parent message + envelope shape (every chat
//     participant addressed exactly once, sender included).
//  2. Persist each ciphertext blob to S3.
//  3. In a single transaction, insert the Attachment rows and their envelope
//     rows. On tx failure, rollback the storage uploads.
//
// The server stores opaque blobs (the encrypted files) and opaque envelopes
// (wrapped file_keys); it can read neither without a participant's
// account_key. The body's own AES-GCM IV (FileIV) is fine to store in the
// clear — IVs don't need to be secret, just unique per (key, message).
func (s *AttachmentService) UploadAttachments(
	ctx context.Context,
	chatID, messageID, userID uint,
	files []*multipart.FileHeader,
	metas []AttachmentMetaInput,
) ([]*models.Attachment, error) {
	if len(metas) != len(files) {
		return nil, ErrAttachmentMetaShape
	}

	message, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return nil, ErrMessageNotFound
	}
	if message.UserID != userID {
		return nil, ErrNotMessageOwner
	}
	if message.ChatID != chatID {
		return nil, ErrNotMessageOwner
	}

	if err := s.validateAllMetas(ctx, chatID, userID, metas); err != nil {
		return nil, err
	}

	attachments, uploadedKeys, err := s.uploadCipherBlobs(files, messageID, metas)
	if err != nil {
		return nil, err
	}

	// Persist attachments + envelopes atomically so a partial commit can't
	// leave orphan rows on either side.
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// gorm Create on []*Attachment writes back the IDs; we need them to
		// link to envelopes below.
		if createErr := tx.Create(&attachments).Error; createErr != nil {
			return fmt.Errorf("save attachments: %w", createErr)
		}
		var envelopes []models.AttachmentEnvelope
		for i, att := range attachments {
			for _, e := range metas[i].Envelopes {
				envelopes = append(envelopes, models.AttachmentEnvelope{
					AttachmentID:     att.ID,
					RecipientID:      e.RecipientID,
					EncryptedFileKey: e.EncryptedFileKey,
					IV:               e.IV,
				})
			}
		}
		if len(envelopes) == 0 {
			return nil
		}
		return s.envelopeRepo.WithTx(tx).CreateBatch(ctx, envelopes)
	}); err != nil {
		s.rollbackUploads(uploadedKeys)
		return nil, err
	}

	return attachments, nil
}

// validateAllMetas pre-validates every per-file envelope set against the
// chat's expected recipient set. Extracted from UploadAttachments to keep
// its cognitive-complexity score in check.
func (s *AttachmentService) validateAllMetas(ctx context.Context, chatID, userID uint, metas []AttachmentMetaInput) error {
	expected, err := s.chatRecipientSet(ctx, chatID, userID)
	if err != nil {
		return err
	}
	for i := range metas {
		if vErr := validateAttachmentMeta(metas[i], expected); vErr != nil {
			return fmt.Errorf("file %d: %w", i, vErr)
		}
	}
	return nil
}

// uploadCipherBlobs writes every encrypted file to object storage and returns
// the matching Attachment rows (not yet persisted) plus the list of storage
// keys so a caller can rollback on subsequent DB failure. Extracted to keep
// UploadAttachments under the linter's cognitive-complexity threshold.
func (s *AttachmentService) uploadCipherBlobs(files []*multipart.FileHeader, messageID uint, metas []AttachmentMetaInput) ([]*models.Attachment, []string, error) {
	attachments := make([]*models.Attachment, 0, len(files))
	uploadedKeys := make([]string, 0, len(files))
	for i, fileHeader := range files {
		attachment, storageKey, procErr := s.processEncryptedFileUpload(fileHeader, messageID, metas[i])
		if procErr != nil {
			s.rollbackUploads(uploadedKeys)
			return nil, nil, procErr
		}
		uploadedKeys = append(uploadedKeys, storageKey)
		attachments = append(attachments, attachment)
	}
	return attachments, uploadedKeys, nil
}

// chatRecipientSet returns the set of user_ids that must appear in any
// per-recipient envelope for this chat: both members of a 1-on-1, or every
// participant of a group. Sender membership is also asserted here.
func (s *AttachmentService) chatRecipientSet(ctx context.Context, chatID, senderID uint) (map[uint]struct{}, error) {
	chat, err := s.chatRepo.FindByIDLight(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("find chat: %w", err)
	}
	expected := make(map[uint]struct{})
	if chat.IsGroup {
		ids, repoErr := s.participantRepo.GetParticipantUserIDs(ctx, chatID)
		if repoErr != nil {
			return nil, fmt.Errorf("fetch participants: %w", repoErr)
		}
		for _, id := range ids {
			expected[id] = struct{}{}
		}
	} else {
		expected[chat.GetUser1ID()] = struct{}{}
		expected[chat.GetUser2ID()] = struct{}{}
	}
	if _, ok := expected[senderID]; !ok {
		return nil, errors.New("sender is not a participant in this chat")
	}
	return expected, nil
}

// validateAttachmentMeta checks the envelope set covers every chat recipient
// exactly once and the body's own IV is populated. Same strict policy as
// message envelopes: no duplicates, no missing, no extras.
func validateAttachmentMeta(meta AttachmentMetaInput, expected map[uint]struct{}) error {
	if meta.FileIV == "" || len(meta.Envelopes) == 0 {
		return ErrAttachmentMetaFields
	}
	seen := make(map[uint]struct{}, len(meta.Envelopes))
	for _, e := range meta.Envelopes {
		if e.EncryptedFileKey == "" || e.IV == "" {
			return errors.New("envelope encrypted_file_key and iv must be non-empty")
		}
		if _, dup := seen[e.RecipientID]; dup {
			return fmt.Errorf("duplicate envelope for recipient %d", e.RecipientID)
		}
		if _, ok := expected[e.RecipientID]; !ok {
			return fmt.Errorf("envelope for non-participant %d", e.RecipientID)
		}
		seen[e.RecipientID] = struct{}{}
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("envelopes cover %d of %d recipients", len(seen), len(expected))
	}
	return nil
}

// processEncryptedFileUpload writes one encrypted blob to S3 and returns the
// Attachment row (not yet persisted) plus the storage_key for potential
// rollback. The file's actual mime is what the client claims — we record it
// verbatim because the body is opaque ciphertext. File type bucket (image /
// video / document) is inferred from the claimed mime so the UI can pick
// the right renderer; this is a hint, not a security boundary.
func (s *AttachmentService) processEncryptedFileUpload(
	fileHeader *multipart.FileHeader,
	messageID uint,
	meta AttachmentMetaInput,
) (*models.Attachment, string, error) {
	if err := s.validator.validateAttachmentSizeOnly(fileHeader); err != nil {
		return nil, "", fmt.Errorf("invalid file %s: %w", fileHeader.Filename, err)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, "", fmt.Errorf("open file %s: %w", fileHeader.Filename, err)
	}

	// The body is ciphertext, so we tell storage it's application/octet-stream.
	// The client-supplied original mime lives on the Attachment row only.
	const cipherContentType = "application/octet-stream"
	fileType := s.validator.determineFileType(meta.MimeType)
	metadata, err := s.storage.Save(file, fileHeader.Filename, cipherContentType, fileHeader.Size)
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
		MimeType:   meta.MimeType,
		// The body's AES-GCM nonce — same for every recipient, different per
		// attachment. Without persisting this the read path can't decrypt:
		// the client needs (encrypted_file_key, envelope_iv, file_iv) all
		// three, and an empty file_iv silently falls back to the
		// "non-E2E" branch in the UI which then tries to render ciphertext
		// bytes as a real image / video / pdf.
		FileIV: meta.FileIV,
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

	// Delete envelope rows + attachment row in one tx so we don't orphan
	// either side.
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if delErr := s.envelopeRepo.WithTx(tx).DeleteByAttachmentIDs(ctx, []uint{attachmentID}); delErr != nil {
			return fmt.Errorf("delete attachment envelopes: %w", delErr)
		}
		if delErr := tx.Delete(&models.Attachment{}, attachmentID).Error; delErr != nil {
			return fmt.Errorf("delete attachment: %w", delErr)
		}
		return nil
	})
}

// FindAttachmentWithMessage retrieves attachment with its parent message (for access checks)
func (s *AttachmentService) FindAttachmentWithMessage(ctx context.Context, attachmentID uint) (*models.Attachment, error) {
	attachment, err := s.attachmentRepo.FindByIDWithMessage(ctx, attachmentID)
	if err != nil {
		return nil, ErrAttachmentNotFound
	}
	return attachment, nil
}

// ResolveAttachmentEnvelopes is the read-path helper used when building API
// responses: for each attachment ID in the list, look up the envelope
// addressed to recipientID. Missing entries mean the recipient wasn't a
// participant at the time the file was uploaded (e.g. joined the group
// later) — UI renders "🔒 placeholder" for those.
func (s *AttachmentService) ResolveAttachmentEnvelopes(ctx context.Context, recipientID uint, attachmentIDs []uint) (map[uint]models.AttachmentEnvelope, error) {
	return s.envelopeRepo.FindForRecipient(ctx, recipientID, attachmentIDs)
}

// AllEnvelopesForAttachment returns every envelope for one attachment; used
// by the WS broadcast for attachments_added so each client picks their own.
func (s *AttachmentService) AllEnvelopesForAttachment(ctx context.Context, attachmentID uint) ([]models.AttachmentEnvelope, error) {
	return s.envelopeRepo.FindAllForAttachment(ctx, attachmentID)
}

// rollbackUploads deletes uploaded files in case of error
func (s *AttachmentService) rollbackUploads(keys []string) {
	for _, key := range keys {
		if err := s.storage.Delete(key); err != nil {
			s.logger.Warn("failed to delete file during rollback", zap.Error(err), zap.String("storage_key", key))
		}
	}
}

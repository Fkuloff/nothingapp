package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"messenger/internal/models"
	"messenger/internal/repositories"
	"messenger/internal/services"
	"messenger/internal/storage"

	"github.com/gin-gonic/gin"
)

// attachmentHandler handles HTTP requests for attachments
type attachmentHandler struct {
	attachmentService *services.AttachmentService
	chatService       *services.ChatService
	wsHandler         *webSocketHandler
	participantRepo   *repositories.ChatParticipantRepo
	storage           storage.Storage
}

// newAttachmentHandler creates a new attachmentHandler instance
func newAttachmentHandler(
	attachmentService *services.AttachmentService,
	chatService *services.ChatService,
	wsHandler *webSocketHandler,
	participantRepo *repositories.ChatParticipantRepo,
	fileStorage storage.Storage,
) *attachmentHandler {
	return &attachmentHandler{
		attachmentService: attachmentService,
		chatService:       chatService,
		wsHandler:         wsHandler,
		participantRepo:   participantRepo,
		storage:           fileStorage,
	}
}

// wireAttachmentEnvelope is one envelope as it arrives over the multipart wire.
type wireAttachmentEnvelope struct {
	RecipientID      uint   `json:"recipient_id"`
	EncryptedFileKey string `json:"encrypted_file_key"`
	IV               string `json:"iv"`
}

// wireAttachmentMeta is the per-file metadata block in the `metadata` form
// field. The N-th entry corresponds to the N-th file in `attachments`.
type wireAttachmentMeta struct {
	FileIV    string                   `json:"file_iv"`
	MimeType  string                   `json:"mime_type"`
	Envelopes []wireAttachmentEnvelope `json:"envelopes"`
}

// UploadAttachments handles multipart file uploads for scheme=2 (E2E)
// attachments. The client encrypts each file with a random file_key before
// upload and ships:
//
//   - `attachments`: one multipart file part per encrypted blob
//   - `metadata`:    a single JSON form field whose value is an array of
//     {file_iv, mime_type, envelopes:[{recipient_id, encrypted_file_key, iv}]},
//     one entry per file, in the same order
//
// The server never sees the file_key — it only stores the ciphertext and the
// per-recipient wrapped keys.
func (h *attachmentHandler) UploadAttachments(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid chat ID")
		return
	}

	messageID, err := parseUintParam(c, "message_id")
	if err != nil {
		sendBadRequest(c, "Invalid message ID")
		return
	}

	// Verify user has access to chat
	if err := h.verifyChatMembership(c, chatID, userID); err != nil {
		return
	}

	// Parse multipart form
	if parseErr := c.Request.ParseMultipartForm(multipartFormSizeAttachment); parseErr != nil {
		sendBadRequest(c, "Failed to parse form")
		return
	}

	form := c.Request.MultipartForm
	files := form.File["attachments"]

	if len(files) == 0 {
		sendBadRequest(c, "No files uploaded")
		return
	}

	if len(files) > maxAttachmentsPerMessage {
		sendBadRequest(c, "Too many files per message")
		return
	}

	metaRaw := c.PostForm("metadata")
	if metaRaw == "" {
		sendBadRequest(c, "metadata is required for E2E attachments")
		return
	}
	var wireMetas []wireAttachmentMeta
	if jsonErr := json.Unmarshal([]byte(metaRaw), &wireMetas); jsonErr != nil {
		sendBadRequest(c, "invalid metadata JSON")
		return
	}
	if len(wireMetas) != len(files) {
		sendBadRequest(c, "metadata count must match attachments count")
		return
	}
	metas := make([]services.AttachmentMetaInput, len(wireMetas))
	for i, m := range wireMetas {
		envs := make([]services.AttachmentEnvelopeInput, len(m.Envelopes))
		for j, e := range m.Envelopes {
			envs[j] = services.AttachmentEnvelopeInput{
				RecipientID:      e.RecipientID,
				EncryptedFileKey: e.EncryptedFileKey,
				IV:               e.IV,
			}
		}
		metas[i] = services.AttachmentMetaInput{
			FileIV:    m.FileIV,
			MimeType:  m.MimeType,
			Envelopes: envs,
		}
	}

	// Upload attachments
	attachments, err := h.attachmentService.UploadAttachments(c.Request.Context(), chatID, messageID, userID, files, metas)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMessageNotFound):
			sendNotFound(c, "Message not found")
		case errors.Is(err, services.ErrNotMessageOwner):
			sendForbidden(c, "Access denied")
		case errors.Is(err, services.ErrAttachmentMetaShape),
			errors.Is(err, services.ErrAttachmentMetaFields):
			sendBadRequest(c, err.Error())
		default:
			sendBadRequest(c, err.Error())
		}
		return
	}

	// Broadcast attachments_added event to all chat participants via WebSocket
	serialized := serializeAttachmentsForBroadcast(c, h.attachmentService, attachments, metas, h.storage)
	if h.wsHandler != nil {
		h.wsHandler.BroadcastAttachmentsAdded(chatID, messageID, serialized)
	}

	sendSuccess(c, gin.H{
		"success":     true,
		"attachments": serialized,
	})
}

// DownloadAttachment redirects to a presigned S3 URL for direct download (JWT required)
func (h *attachmentHandler) DownloadAttachment(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	attachmentID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid attachment ID")
		return
	}

	attachment, err := h.attachmentService.FindAttachmentWithMessage(c.Request.Context(), attachmentID)
	if err != nil {
		sendNotFound(c, "Attachment not found")
		return
	}

	// Verify user has access to the chat containing this attachment
	if attachment.Message == nil {
		sendInternalError(c, "Attachment has no associated message")
		return
	}

	if err := h.verifyChatMembership(c, attachment.Message.ChatID, userID); err != nil {
		return
	}

	presignedURL := h.storage.GetURL(attachment.StorageKey)
	c.Redirect(http.StatusFound, presignedURL)
}

// DeleteAttachment removes an attachment
func (h *attachmentHandler) DeleteAttachment(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	attachmentID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid attachment ID")
		return
	}

	if err := h.attachmentService.DeleteAttachment(c.Request.Context(), attachmentID, userID); err != nil {
		switch {
		case errors.Is(err, services.ErrAttachmentNotFound):
			sendNotFound(c, "Attachment not found")
		case errors.Is(err, services.ErrNotMessageOwner):
			sendForbidden(c, "Access denied")
		default:
			sendInternalError(c, "Failed to delete attachment")
		}
		return
	}

	sendSuccess(c, gin.H{"success": true})
}

// verifyChatMembership checks that the user is a participant of the given chat.
// Responds with an HTTP error and returns a non-nil error on failure.
func (h *attachmentHandler) verifyChatMembership(c *gin.Context, chatID, userID uint) error {
	chat, err := h.chatService.FindChatByIDLight(c.Request.Context(), chatID)
	if err != nil {
		sendForbidden(c, "Access denied")
		return err
	}

	if chat.IsGroup {
		if h.participantRepo == nil {
			sendForbidden(c, "Access denied")
			return errAccessDenied
		}
		if _, pErr := h.participantRepo.FindByUserAndChat(c.Request.Context(), chatID, userID); pErr != nil {
			sendForbidden(c, "Access denied")
			return pErr
		}
	} else if !chat.HasUser(userID) {
		sendForbidden(c, "Access denied")
		return errAccessDenied
	}

	return nil
}

// serializeAttachmentSlice converts a value-type attachment slice to maps with presigned URLs.
func serializeAttachmentSlice(attachments []models.Attachment, s storage.Storage) []map[string]any {
	result := make([]map[string]any, 0, len(attachments))
	for i := range attachments {
		result = append(result, serializeAttachment(&attachments[i], s))
	}
	return result
}

// serializeAttachment converts a single attachment model to a map with a presigned URL.
// Includes file_iv (the body's AES-GCM nonce; identical across recipients, so
// part of the attachment row itself, not the per-recipient envelope). Does not
// include per-recipient envelope material — callers that need it should use
// serializeAttachmentSliceForUser (read path: pre-resolved per user) or
// serializeAttachmentsForBroadcast (write path: full envelope set, each client
// filters to its own).
func serializeAttachment(att *models.Attachment, s storage.Storage) map[string]any {
	return map[string]any{
		"id":        att.ID,
		"file_type": att.FileType,
		"file_name": att.FileName,
		"file_size": att.FileSize,
		"mime_type": att.MimeType,
		"url":       s.GetURL(att.StorageKey),
		"file_iv":   att.FileIV,
	}
}

// serializeAttachmentsForBroadcast serializes attachments right after the
// upload, packing every envelope in the response. Receiving clients filter to
// their own envelope by recipient_id. Used in the attachments_added WS event
// and in the upload-call's HTTP response (sender sees their own envelope plus
// the full set is just there too).
func serializeAttachmentsForBroadcast(
	c *gin.Context,
	svc *services.AttachmentService,
	attachments []*models.Attachment,
	metas []services.AttachmentMetaInput,
	s storage.Storage,
) []map[string]any {
	result := make([]map[string]any, 0, len(attachments))
	for i, att := range attachments {
		out := serializeAttachment(att, s)
		out["file_iv"] = ""
		if i < len(metas) {
			out["file_iv"] = metas[i].FileIV
			// We already have the envelopes in memory — no need to re-fetch
			// them from the repo.
			envs := make([]map[string]any, 0, len(metas[i].Envelopes))
			for _, e := range metas[i].Envelopes {
				envs = append(envs, map[string]any{
					"recipient_id":       e.RecipientID,
					"encrypted_file_key": e.EncryptedFileKey,
					"iv":                 e.IV,
				})
			}
			out["envelopes"] = envs
		} else if svc != nil {
			// Defensive fallback: hit the repo if metas wasn't provided.
			ctx := c.Request.Context()
			if envs, err := svc.AllEnvelopesForAttachment(ctx, att.ID); err == nil {
				envOut := make([]map[string]any, 0, len(envs))
				for _, e := range envs {
					envOut = append(envOut, map[string]any{
						"recipient_id":       e.RecipientID,
						"encrypted_file_key": e.EncryptedFileKey,
						"iv":                 e.IV,
					})
				}
				out["envelopes"] = envOut
			}
		}
		result = append(result, out)
	}
	return result
}

// serializeAttachmentSliceForUser pre-resolves the caller's envelope so the
// response includes only the wrapped file_key + iv addressed to them, not the
// whole set. Used by the read path (chat-data, message-list) where each
// request belongs to exactly one user.
func serializeAttachmentSliceForUser(
	ctx context.Context,
	svc *services.AttachmentService,
	attachments []models.Attachment,
	userID uint,
	s storage.Storage,
) []map[string]any {
	result := make([]map[string]any, 0, len(attachments))
	if len(attachments) == 0 {
		return result
	}
	ids := make([]uint, 0, len(attachments))
	for i := range attachments {
		ids = append(ids, attachments[i].ID)
	}
	envs, envErr := svc.ResolveAttachmentEnvelopes(ctx, userID, ids)
	if envErr != nil {
		// Non-fatal — render attachments without envelope data; the client
		// will show "🔒 placeholder" for them.
		envs = nil
	}
	for i := range attachments {
		att := &attachments[i]
		out := serializeAttachment(att, s)
		if env, ok := envs[att.ID]; ok {
			out["encrypted_file_key"] = env.EncryptedFileKey
			out["envelope_iv"] = env.IV
		}
		result = append(result, out)
	}
	return result
}

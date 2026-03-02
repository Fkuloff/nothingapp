package handlers

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"
)

// handleCallSignaling relays WebRTC signaling messages between two users in a 1-on-1 chat.
// The server is stateless — it only validates access and forwards the payload.
func (h *webSocketHandler) handleCallSignaling(ctx context.Context, userID uint, rawMsg []byte) error {
	var ca callAction
	if err := json.Unmarshal(rawMsg, &ca); err != nil {
		return &wsError{message: "Invalid call signaling format"}
	}

	if ca.ChatID == 0 {
		return &wsError{message: "chat_id is required for call signaling"}
	}
	if ca.CallID == "" {
		return &wsError{message: "call_id is required for call signaling"}
	}

	chat, err := h.chatService.FindChatByIDLight(ctx, ca.ChatID)
	if err != nil {
		return &wsError{message: "Chat not found"}
	}
	if chat.IsGroup {
		return &wsError{message: "Calls are only supported in 1-on-1 chats"}
	}
	if !chat.HasUser(userID) {
		return &wsError{message: "Access denied to this chat"}
	}

	otherUserID := chat.GetUser1ID()
	if otherUserID == userID {
		otherUserID = chat.GetUser2ID()
	}

	if ca.Action == "call_offer" && !h.presenceService.IsUserOnline(otherUserID) {
		return &wsError{message: "User is offline"}
	}

	// Track active calls for disconnect notification
	switch ca.Action {
	case "call_offer":
		h.registerCall(userID, otherUserID, ca.CallID, ca.ChatID)
	case "call_hangup", "call_reject":
		h.unregisterCall(userID)
	}

	relayData := map[string]any{
		"action":  ca.Action,
		"chat_id": ca.ChatID,
		"call_id": ca.CallID,
		"user_id": userID,
	}
	if ca.SDP != "" {
		relayData["sdp"] = ca.SDP
		relayData["sdp_type"] = ca.SDPType
	}
	if ca.Candidate != "" {
		relayData["candidate"] = ca.Candidate
	}

	msgJSON, err := json.Marshal(relayData)
	if err != nil {
		return &wsError{message: "Failed to marshal call signal"}
	}

	h.broadcastToUser(otherUserID, msgJSON)

	h.logger.Debug("relayed call signal",
		zap.String("action", ca.Action),
		zap.Uint("from", userID),
		zap.Uint("to", otherUserID),
		zap.String("call_id", ca.CallID),
	)

	return nil
}

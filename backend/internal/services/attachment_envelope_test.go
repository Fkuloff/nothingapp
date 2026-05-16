package services

import (
	"errors"
	"strings"
	"testing"
)

// The envelope contract is the most security-critical part of the attachment
// upload path: the server has no way to verify the file body (it's
// ciphertext) and has to trust the client about mime-type and so on. The one
// thing the server CAN and MUST enforce is the envelope shape — every chat
// recipient addressed exactly once, sender included, no extras. If this
// validation is wrong we either leak (envelope addressed to a non-participant)
// or break delivery (missing envelope).

// helper: build a participant set the way validateAttachmentMeta expects.
func recipientSet(ids ...uint) map[uint]struct{} {
	m := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		m[id] = struct{}{}
	}
	return m
}

func TestValidateAttachmentMeta_Happy(t *testing.T) {
	// 1-on-1 chat between users 1 and 2. Both envelopes present, no extras.
	meta := AttachmentMetaInput{
		FileIV: "base64-file-iv",
		Envelopes: []AttachmentEnvelopeInput{
			{RecipientID: 1, EncryptedFileKey: "ct1", IV: "iv1"},
			{RecipientID: 2, EncryptedFileKey: "ct2", IV: "iv2"},
		},
	}
	if err := validateAttachmentMeta(meta, recipientSet(1, 2)); err != nil {
		t.Fatalf("happy path rejected: %v", err)
	}
}

func TestValidateAttachmentMeta_MissingFileIV(t *testing.T) {
	// AES-GCM body without the body's own IV is undecryptable. Reject up front.
	meta := AttachmentMetaInput{
		FileIV: "",
		Envelopes: []AttachmentEnvelopeInput{
			{RecipientID: 1, EncryptedFileKey: "ct1", IV: "iv1"},
		},
	}
	err := validateAttachmentMeta(meta, recipientSet(1))
	if !errors.Is(err, ErrAttachmentMetaFields) {
		t.Fatalf("expected ErrAttachmentMetaFields, got %v", err)
	}
}

func TestValidateAttachmentMeta_NoEnvelopes(t *testing.T) {
	// Envelope set must be non-empty — otherwise the file_key is unrecoverable
	// for everyone and the attachment is dead on arrival.
	meta := AttachmentMetaInput{FileIV: "iv", Envelopes: nil}
	err := validateAttachmentMeta(meta, recipientSet(1))
	if !errors.Is(err, ErrAttachmentMetaFields) {
		t.Fatalf("expected ErrAttachmentMetaFields, got %v", err)
	}
}

func TestValidateAttachmentMeta_EmptyEnvelopeFields(t *testing.T) {
	// Each envelope must have ciphertext + iv. Empty either side means the
	// recipient can't unwrap their file_key — fail fast rather than persist
	// a poison row.
	meta := AttachmentMetaInput{
		FileIV: "iv",
		Envelopes: []AttachmentEnvelopeInput{
			{RecipientID: 1, EncryptedFileKey: "", IV: "iv1"},
		},
	}
	err := validateAttachmentMeta(meta, recipientSet(1))
	if err == nil || !strings.Contains(err.Error(), "must be non-empty") {
		t.Fatalf("expected non-empty error, got %v", err)
	}
}

func TestValidateAttachmentMeta_DuplicateRecipient(t *testing.T) {
	// Two envelopes for the same recipient is ambiguous (which one do we
	// serve them?). Reject so the client doesn't accidentally write garbage.
	meta := AttachmentMetaInput{
		FileIV: "iv",
		Envelopes: []AttachmentEnvelopeInput{
			{RecipientID: 1, EncryptedFileKey: "ct1", IV: "iv1"},
			{RecipientID: 1, EncryptedFileKey: "ct1b", IV: "iv1b"},
			{RecipientID: 2, EncryptedFileKey: "ct2", IV: "iv2"},
		},
	}
	err := validateAttachmentMeta(meta, recipientSet(1, 2))
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestValidateAttachmentMeta_StrangerRecipient(t *testing.T) {
	// Envelope for someone who isn't in the chat. Server enforces that
	// envelopes only address current participants — otherwise a sender could
	// leak file_key to arbitrary user_ids by stuffing extras.
	meta := AttachmentMetaInput{
		FileIV: "iv",
		Envelopes: []AttachmentEnvelopeInput{
			{RecipientID: 1, EncryptedFileKey: "ct1", IV: "iv1"},
			{RecipientID: 2, EncryptedFileKey: "ct2", IV: "iv2"},
			{RecipientID: 999, EncryptedFileKey: "ctx", IV: "ivx"},
		},
	}
	err := validateAttachmentMeta(meta, recipientSet(1, 2))
	if err == nil || !strings.Contains(err.Error(), "non-participant") {
		t.Fatalf("expected non-participant error, got %v", err)
	}
}

func TestValidateAttachmentMeta_MissingParticipant(t *testing.T) {
	// Envelope set covers only a subset of participants. Under the strict
	// policy (which the lazy-vault modal makes safe in practice) the client
	// must address every participant; partial drops break delivery for them
	// in a way that's not obvious to the sender. Reject.
	meta := AttachmentMetaInput{
		FileIV: "iv",
		Envelopes: []AttachmentEnvelopeInput{
			{RecipientID: 1, EncryptedFileKey: "ct1", IV: "iv1"},
		},
	}
	err := validateAttachmentMeta(meta, recipientSet(1, 2, 3))
	if err == nil || !strings.Contains(err.Error(), "cover") {
		t.Fatalf("expected coverage error, got %v", err)
	}
}

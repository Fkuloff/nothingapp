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

// validMeta is the canonical valid payload for the happy path. All other
// tests start from this and mutate one field — keeps the negative cases
// focused on the specific invariant they're checking.
func validMeta() AttachmentMetaInput {
	return AttachmentMetaInput{
		FileIV:            "base64-file-iv",
		EncryptedMetadata: "encrypted-name-and-mime",
		MetadataIV:        "metadata-iv",
		Envelopes: []AttachmentEnvelopeInput{
			{RecipientID: 1, EncryptedFileKey: "ct1", IV: "iv1"},
			{RecipientID: 2, EncryptedFileKey: "ct2", IV: "iv2"},
		},
	}
}

func TestValidateAttachmentMeta_Happy(t *testing.T) {
	if err := validateAttachmentMeta(validMeta(), recipientSet(1, 2)); err != nil {
		t.Fatalf("happy path rejected: %v", err)
	}
}

func TestValidateAttachmentMeta_MissingFileIV(t *testing.T) {
	// AES-GCM body without the body's own IV is undecryptable. Reject up front.
	meta := validMeta()
	meta.FileIV = ""
	err := validateAttachmentMeta(meta, recipientSet(1, 2))
	if !errors.Is(err, ErrAttachmentMetaFields) {
		t.Fatalf("expected ErrAttachmentMetaFields, got %v", err)
	}
}

func TestValidateAttachmentMeta_NoEnvelopes(t *testing.T) {
	// Envelope set must be non-empty — otherwise the file_key is unrecoverable
	// for everyone and the attachment is dead on arrival.
	meta := validMeta()
	meta.Envelopes = nil
	err := validateAttachmentMeta(meta, recipientSet(1))
	if !errors.Is(err, ErrAttachmentMetaFields) {
		t.Fatalf("expected ErrAttachmentMetaFields, got %v", err)
	}
}

func TestValidateAttachmentMeta_EmptyEnvelopeFields(t *testing.T) {
	// Each envelope must have ciphertext + iv. Empty either side means the
	// recipient can't unwrap their file_key — fail fast rather than persist
	// a poison row.
	meta := validMeta()
	meta.Envelopes = []AttachmentEnvelopeInput{
		{RecipientID: 1, EncryptedFileKey: "", IV: "iv1"},
		{RecipientID: 2, EncryptedFileKey: "ct2", IV: "iv2"},
	}
	err := validateAttachmentMeta(meta, recipientSet(1, 2))
	if err == nil || !strings.Contains(err.Error(), "must be non-empty") {
		t.Fatalf("expected non-empty error, got %v", err)
	}
}

func TestValidateAttachmentMeta_DuplicateRecipient(t *testing.T) {
	// Two envelopes for the same recipient is ambiguous (which one do we
	// serve them?). Reject so the client doesn't accidentally write garbage.
	meta := validMeta()
	meta.Envelopes = []AttachmentEnvelopeInput{
		{RecipientID: 1, EncryptedFileKey: "ct1", IV: "iv1"},
		{RecipientID: 1, EncryptedFileKey: "ct1b", IV: "iv1b"},
		{RecipientID: 2, EncryptedFileKey: "ct2", IV: "iv2"},
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
	meta := validMeta()
	meta.Envelopes = []AttachmentEnvelopeInput{
		{RecipientID: 1, EncryptedFileKey: "ct1", IV: "iv1"},
		{RecipientID: 2, EncryptedFileKey: "ct2", IV: "iv2"},
		{RecipientID: 999, EncryptedFileKey: "ctx", IV: "ivx"},
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
	meta := validMeta()
	meta.Envelopes = []AttachmentEnvelopeInput{
		{RecipientID: 1, EncryptedFileKey: "ct1", IV: "iv1"},
	}
	err := validateAttachmentMeta(meta, recipientSet(1, 2, 3))
	if err == nil || !strings.Contains(err.Error(), "cover") {
		t.Fatalf("expected coverage error, got %v", err)
	}
}

func TestValidateAttachmentMeta_RequiresEncryptedMetadata(t *testing.T) {
	// After the legacy plaintext-filename removal, every new upload MUST
	// ship encrypted_metadata + metadata_iv. Without them the row is
	// unrenderable on the receiver (no way to recover filename / mime)
	// and the server can't synthesize fallback values either, since the
	// columns are gone. Reject at upload time, not at render time.
	for _, tc := range []struct {
		name    string
		mutator func(*AttachmentMetaInput)
	}{
		{"missing both", func(m *AttachmentMetaInput) { m.EncryptedMetadata = ""; m.MetadataIV = "" }},
		{"missing ciphertext", func(m *AttachmentMetaInput) { m.EncryptedMetadata = "" }},
		{"missing iv", func(m *AttachmentMetaInput) { m.MetadataIV = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			meta := validMeta()
			tc.mutator(&meta)
			err := validateAttachmentMeta(meta, recipientSet(1, 2))
			if err == nil || !strings.Contains(err.Error(), "encrypted_metadata") {
				t.Fatalf("expected encrypted_metadata error, got %v", err)
			}
		})
	}
}

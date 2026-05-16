package services

import (
	"errors"
	"testing"

	"messenger/internal/models"
)

// After the scheme=1 removal, prepareMessageBody only accepts two valid shapes:
//   - 1-on-1 scheme=2: Text + IV both non-empty.
//   - Group scheme=2 pairwise: Envelopes non-empty (Text/IV ignored on the row).
// Everything else is rejected up front.

func TestPrepareMessageBody_ClientSide_PassThrough(t *testing.T) {
	svc := &ChatService{}

	clientCT := "opaque-client-ciphertext"
	clientIV := "client-iv-base64"

	ct, iv, scheme, err := svc.prepareMessageBody(SendMessageInput{
		Text:   clientCT,
		IV:     clientIV,
		Scheme: models.SchemeClientSide,
	})
	if err != nil {
		t.Fatalf("prepareMessageBody: %v", err)
	}
	if scheme != models.SchemeClientSide {
		t.Fatalf("scheme = %d, want %d", scheme, models.SchemeClientSide)
	}
	if ct != clientCT || iv != clientIV {
		t.Fatalf("ciphertext/iv mutated: got (%q, %q)", ct, iv)
	}
}

// scheme=2 + no Envelopes + empty IV is impossible to decrypt on the receiver
// side, so the service rejects it instead of writing garbage to the DB.
func TestPrepareMessageBody_ClientSide_RequiresIV(t *testing.T) {
	svc := &ChatService{}

	_, _, _, err := svc.prepareMessageBody(SendMessageInput{
		Text:   "ciphertext",
		IV:     "",
		Scheme: models.SchemeClientSide,
	})
	if !errors.Is(err, errMissingIVForClientScheme) {
		t.Fatalf("expected errMissingIVForClientScheme, got %v", err)
	}
}

// Group pairwise: Envelopes set → row Text/IV intentionally empty (the real
// ciphertexts live in message_envelopes). prepareMessageBody returns empty
// strings + scheme=2.
func TestPrepareMessageBody_GroupEnvelopes(t *testing.T) {
	svc := &ChatService{}

	ct, iv, scheme, err := svc.prepareMessageBody(SendMessageInput{
		Scheme: models.SchemeClientSide,
		Envelopes: []MessageEnvelopeInput{
			{RecipientID: 1, Ciphertext: "ct1", IV: "iv1"},
			{RecipientID: 2, Ciphertext: "ct2", IV: "iv2"},
		},
	})
	if err != nil {
		t.Fatalf("prepareMessageBody (group envelopes): %v", err)
	}
	if scheme != models.SchemeClientSide {
		t.Fatalf("scheme = %d, want %d", scheme, models.SchemeClientSide)
	}
	if ct != "" || iv != "" {
		t.Fatalf("group envelope path must leave row text/iv empty, got (%q, %q)", ct, iv)
	}
}

// Any send that doesn't declare scheme=2 is rejected — the server no longer
// encrypts on behalf of clients. Covers scheme=1 (legacy explicit) and scheme=0
// (default / pre-migration) equally.
func TestPrepareMessageBody_RejectsServerSide(t *testing.T) {
	svc := &ChatService{}

	for _, s := range []uint8{0, models.SchemeServerSide} {
		_, _, _, err := svc.prepareMessageBody(SendMessageInput{
			Text:   "plaintext",
			Scheme: s,
		})
		if !errors.Is(err, errServerSchemeRemoved) {
			t.Fatalf("scheme=%d: expected errServerSchemeRemoved, got %v", s, err)
		}
	}
}

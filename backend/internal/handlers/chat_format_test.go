package handlers

import (
	"errors"
	"strings"
	"testing"

	"messenger/internal/models"
)

// formatLastMessage is what the chat-list endpoint shows under each row's
// "last message" preview. Three behaviors that matter to the UI:
//
//   1. scheme=2 messages MUST NOT leak ciphertext into the preview — the
//      server can't decrypt them, and the client doing it on the list endpoint
//      would mean a round-trip per chat for every chat-list refresh. We
//      substitute a lock-emoji placeholder.
//   2. scheme=1 (legacy) messages render their plaintext, truncated to
//      maxChatListPreview runes.
//   3. Errors (no message yet, deleted-only chat) render an empty string.

func TestFormatLastMessage_SchemeClientSideRendersPlaceholder(t *testing.T) {
	msg := &models.Message{
		Text:   "AAAAAAciphertext-this-should-never-be-shown",
		Scheme: models.SchemeClientSide,
	}

	got := formatLastMessage(msg, nil)
	if !strings.Contains(got, "Зашифрованное сообщение") {
		t.Fatalf("scheme=2 preview should be the placeholder, got %q", got)
	}
	if strings.Contains(got, "AAAAAA") {
		t.Fatalf("preview leaked ciphertext: %q", got)
	}
}

func TestFormatLastMessage_SchemeServerSideShowsText(t *testing.T) {
	msg := &models.Message{
		Text:   "Привет, это обычное сообщение",
		Scheme: models.SchemeServerSide,
	}

	got := formatLastMessage(msg, nil)
	if got != "Привет, это обычное сообщение" {
		t.Fatalf("expected verbatim text, got %q", got)
	}
}

// Rows that pre-date the scheme column read back as scheme=0 in Go (the DB
// default kicks in only on INSERT, not on SELECT of a NULL column). They were
// all server-side encrypted, so the preview must treat them as scheme=1.
func TestFormatLastMessage_PreSchemeColumnRows(t *testing.T) {
	msg := &models.Message{
		Text:   "legacy plaintext",
		Scheme: 0,
	}
	got := formatLastMessage(msg, nil)
	if got != "legacy plaintext" {
		t.Fatalf("scheme=0 rows should render text unchanged, got %q", got)
	}
}

func TestFormatLastMessage_NilMessageReturnsEmpty(t *testing.T) {
	if got := formatLastMessage(nil, nil); got != "" {
		t.Fatalf("nil message should be empty preview, got %q", got)
	}
}

func TestFormatLastMessage_ErrorReturnsEmpty(t *testing.T) {
	if got := formatLastMessage(nil, errors.New("not found")); got != "" {
		t.Fatalf("err path should be empty preview, got %q", got)
	}
}

// Long scheme=1 plaintexts are truncated at maxChatListPreview runes (not
// bytes — multibyte chars count once). The placeholder path doesn't go through
// the truncator because the placeholder is fixed-length.
func TestFormatLastMessage_TruncatesLongServerSide(t *testing.T) {
	long := strings.Repeat("ы", maxChatListPreview+10) // Cyrillic, multibyte
	msg := &models.Message{Text: long, Scheme: models.SchemeServerSide}

	got := formatLastMessage(msg, nil)
	if runes := []rune(got); len(runes) != maxChatListPreview {
		t.Fatalf("expected truncation to %d runes, got %d", maxChatListPreview, len(runes))
	}
}

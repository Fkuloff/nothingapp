package services

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// UpdateVault runs all of its validation (length caps + all-or-nothing) before
// touching the repository. To exercise the rejection paths in isolation we build
// a UserService with a nil repo — the validation errors short-circuit the method
// before any repo call could nil-deref. The "happy path" can't be tested here
// without a real DB, but it's covered transitively by the auth handler tests
// when the full app is wired up.

func newValidationOnlyUserService() *UserService {
	return &UserService{}
}

func TestUpdateVault_RejectsVaultSaltTooLong(t *testing.T) {
	svc := newValidationOnlyUserService()
	longSalt := strings.Repeat("A", 65)

	err := svc.UpdateVault(context.Background(), 1, longSalt, "valid-blob", "valid-key")
	if err == nil || !strings.Contains(err.Error(), "vault_salt too long") {
		t.Fatalf("expected vault_salt-too-long error, got %v", err)
	}
}

func TestUpdateVault_RejectsEncryptedAccountKeyTooLong(t *testing.T) {
	svc := newValidationOnlyUserService()
	longBlob := strings.Repeat("A", 4097)

	err := svc.UpdateVault(context.Background(), 1, "salt", longBlob, "key")
	if err == nil || !strings.Contains(err.Error(), "encrypted_account_key too long") {
		t.Fatalf("expected encrypted_account_key-too-long error, got %v", err)
	}
}

func TestUpdateVault_RejectsPublicKeyTooLong(t *testing.T) {
	svc := newValidationOnlyUserService()
	longPub := strings.Repeat("A", 65)

	err := svc.UpdateVault(context.Background(), 1, "salt", "blob", longPub)
	if err == nil || !strings.Contains(err.Error(), "public_key too long") {
		t.Fatalf("expected public_key-too-long error, got %v", err)
	}
}

// The "all three must move together" rule — exercise every partial combination.
// Half-state would let other users ECDH against a public_key whose private half
// nobody can match, so it must be rejected at the service layer.
func TestUpdateVault_RejectsPartialState(t *testing.T) {
	cases := []struct {
		name                      string
		vaultSalt, encKey, pubKey string
	}{
		{"only vault_salt", "salt", "", ""},
		{"only encrypted_account_key", "", "blob", ""},
		{"only public_key", "", "", "key"},
		{"salt + encrypted, no public", "salt", "blob", ""},
		{"salt + public, no encrypted", "salt", "", "key"},
		{"encrypted + public, no salt", "", "blob", "key"},
	}

	wantSubstr := "must all be set or all cleared"
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := newValidationOnlyUserService()
			err := svc.UpdateVault(context.Background(), 1, c.vaultSalt, c.encKey, c.pubKey)
			if err == nil {
				t.Fatalf("expected error for partial state, got nil")
			}
			if !strings.Contains(err.Error(), wantSubstr) {
				t.Errorf("error %q does not mention %q", err, wantSubstr)
			}
		})
	}
}

// Clearing all three (opt-out path) passes validation. We can't assert the repo
// call here without a fake repo, so verify only that we get past validation —
// the nil repo would NPE on a UpdateVault call, so a "nil pointer dereference"
// panic confirms the call happened. To avoid the panic in CI, recover and assert
// no validation error reached us.
func TestUpdateVault_AllowsAllCleared(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when nil repo is called — got nothing, which means validation rejected the all-cleared case")
		}
	}()

	svc := newValidationOnlyUserService()
	// All three empty → validation passes → repo call → nil dereference panic.
	// We catch the panic via defer/recover above; the assertion is "panic happened",
	// which proves validation didn't bounce us early.
	_ = svc.UpdateVault(context.Background(), 1, "", "", "")
}

// Same for the all-three-set happy path.
func TestUpdateVault_AllowsAllSet(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when nil repo is called for the all-set case")
		}
	}()

	svc := newValidationOnlyUserService()
	_ = svc.UpdateVault(context.Background(), 1, "salt", "encryptedBlob", "publicKey")
}

// Sanity: ensure the "too long" errors aren't masking the partial-state error
// when both conditions could apply. Length checks come first, so a too-long
// vault_salt with empty other fields should yield "too long", not "partial".
func TestUpdateVault_LengthChecksFirst(t *testing.T) {
	svc := newValidationOnlyUserService()
	longSalt := strings.Repeat("A", 65)

	err := svc.UpdateVault(context.Background(), 1, longSalt, "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "vault_salt too long") {
		t.Errorf("length check should win over partial-state check, got: %v", err)
	}
	// Just to make sure we didn't accidentally pull in errors.New somewhere with
	// surprising wrapping that we couldn't detect via Contains:
	if errors.Unwrap(err) != nil {
		t.Logf("note: error has wrapped chain — that's fine, just unexpected")
	}
}

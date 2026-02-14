package config

import (
	"errors"
	"testing"
)

// validEncKey is a base64-encoded 32-byte key for testing (openssl rand -base64 32).
const validEncKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestLoadConfig_Valid(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("JWT_SECRET", "this-is-a-very-secure-secret-key-32+")
	t.Setenv("MESSAGE_ENCRYPTION_KEY", validEncKey)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.DBURL != "postgres://user:pass@localhost:5432/testdb" {
		t.Errorf("DBURL = %q, want postgres://...", cfg.DBURL)
	}
	if cfg.JWTSecret != "this-is-a-very-secure-secret-key-32+" {
		t.Errorf("JWTSecret mismatch")
	}
	if cfg.MessageEncryptionKey != validEncKey {
		t.Errorf("MessageEncryptionKey mismatch")
	}
}

func TestLoadConfig_MissingDBURL(t *testing.T) {
	t.Setenv("DB_URL", "")
	t.Setenv("JWT_SECRET", "this-is-a-very-secure-secret-key-32+")
	t.Setenv("MESSAGE_ENCRYPTION_KEY", validEncKey)

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing DB_URL")
	}
	if !errors.Is(err, ErrDBURLNotSet) {
		t.Errorf("expected ErrDBURLNotSet, got: %v", err)
	}
}

func TestLoadConfig_MissingJWTSecret(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("MESSAGE_ENCRYPTION_KEY", validEncKey)

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing JWT_SECRET")
	}
	if !errors.Is(err, ErrJWTSecretNotSet) {
		t.Errorf("expected ErrJWTSecretNotSet, got: %v", err)
	}
}

func TestLoadConfig_ShortJWTSecret(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("JWT_SECRET", "too-short")
	t.Setenv("MESSAGE_ENCRYPTION_KEY", validEncKey)

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for short JWT_SECRET")
	}
	if !errors.Is(err, ErrJWTSecretTooShort) {
		t.Errorf("expected ErrJWTSecretTooShort, got: %v", err)
	}
}

func TestLoadConfig_VAPIDKeysOptional(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("JWT_SECRET", "this-is-a-very-secure-secret-key-32+")
	t.Setenv("MESSAGE_ENCRYPTION_KEY", validEncKey)
	// Explicitly unset VAPID keys
	t.Setenv("VAPID_PUBLIC_KEY", "")
	t.Setenv("VAPID_PRIVATE_KEY", "")
	t.Setenv("VAPID_SUBJECT", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() should not error when VAPID keys absent: %v", err)
	}
	if cfg.VAPIDPublicKey != "" {
		t.Error("VAPIDPublicKey should be empty")
	}
}

func TestLoadConfig_JWTSecretExactly32Chars(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("JWT_SECRET", "abcdefghijklmnopqrstuvwxyz123456") // exactly 32
	t.Setenv("MESSAGE_ENCRYPTION_KEY", validEncKey)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() should accept 32-char secret: %v", err)
	}
	if cfg.JWTSecret != "abcdefghijklmnopqrstuvwxyz123456" {
		t.Error("JWTSecret mismatch")
	}
}

func TestLoadConfig_MissingEncryptionKey(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("JWT_SECRET", "this-is-a-very-secure-secret-key-32+")
	t.Setenv("MESSAGE_ENCRYPTION_KEY", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing MESSAGE_ENCRYPTION_KEY")
	}
	if !errors.Is(err, ErrMsgEncKeyNotSet) {
		t.Errorf("expected ErrMsgEncKeyNotSet, got: %v", err)
	}
}

func TestLoadConfig_InvalidEncryptionKey(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("JWT_SECRET", "this-is-a-very-secure-secret-key-32+")
	t.Setenv("MESSAGE_ENCRYPTION_KEY", "not-valid-base64!!!")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for invalid MESSAGE_ENCRYPTION_KEY")
	}
	if !errors.Is(err, ErrMsgEncKeyInvalid) {
		t.Errorf("expected ErrMsgEncKeyInvalid, got: %v", err)
	}
}

func TestLoadConfig_ShortEncryptionKey(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("JWT_SECRET", "this-is-a-very-secure-secret-key-32+")
	t.Setenv("MESSAGE_ENCRYPTION_KEY", "dG9vLXNob3J0") // "too-short" base64, only 9 bytes

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for short MESSAGE_ENCRYPTION_KEY")
	}
	if !errors.Is(err, ErrMsgEncKeyInvalid) {
		t.Errorf("expected ErrMsgEncKeyInvalid, got: %v", err)
	}
}

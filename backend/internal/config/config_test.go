package config

import (
	"errors"
	"testing"
)

func TestLoadConfig_Valid(t *testing.T) {
	t.Setenv("DB_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("JWT_SECRET", "this-is-a-very-secure-secret-key-32+")

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
}

func TestLoadConfig_MissingDBURL(t *testing.T) {
	t.Setenv("DB_URL", "")
	t.Setenv("JWT_SECRET", "this-is-a-very-secure-secret-key-32+")

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

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() should accept 32-char secret: %v", err)
	}
	if cfg.JWTSecret != "abcdefghijklmnopqrstuvwxyz123456" {
		t.Error("JWTSecret mismatch")
	}
}

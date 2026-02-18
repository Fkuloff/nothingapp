package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"messenger/internal/storage"

	"github.com/joho/godotenv"
)

// Config holds application configuration.
type Config struct {
	Storage              *storage.StorageConfig
	DBURL                string
	JWTSecret            string
	MessageEncryptionKey string
	VAPIDPublicKey       string
	VAPIDPrivateKey      string
	VAPIDSubject         string
}

// Sentinel errors for configuration validation.
var (
	errDBURLNotSet       = errors.New("DB_URL is not set")
	errJWTSecretNotSet   = errors.New("JWT_SECRET is not set")
	errJWTSecretTooShort = errors.New("JWT_SECRET must be at least 32 characters long")
	errMsgEncKeyNotSet   = errors.New("MESSAGE_ENCRYPTION_KEY is not set")
	errMsgEncKeyInvalid  = errors.New("MESSAGE_ENCRYPTION_KEY must be valid base64 encoding exactly 32 bytes")
)

// LoadConfig loads configuration from environment variables.
func LoadConfig() (*Config, error) {
	// Load .env file if present (ignore error if not found)
	_ = godotenv.Load() //nolint:errcheck // intentionally ignoring - .env is optional

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, errDBURLNotSet
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, errJWTSecretNotSet
	}

	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("%w: got %d characters", errJWTSecretTooShort, len(jwtSecret))
	}

	msgEncKey := os.Getenv("MESSAGE_ENCRYPTION_KEY")
	if msgEncKey == "" {
		return nil, errMsgEncKeyNotSet
	}
	keyBytes, err := base64.StdEncoding.DecodeString(msgEncKey)
	if err != nil || len(keyBytes) != 32 {
		return nil, errMsgEncKeyInvalid
	}

	return &Config{
		DBURL:                dbURL,
		JWTSecret:            jwtSecret,
		MessageEncryptionKey: msgEncKey,
		Storage:              storage.LoadStorageConfig(),
		VAPIDPublicKey:       os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey:      os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDSubject:         os.Getenv("VAPID_SUBJECT"),
	}, nil
}

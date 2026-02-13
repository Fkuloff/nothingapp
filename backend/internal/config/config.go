// internal/config/config.go
package config

import (
	"errors"
	"fmt"
	"os"

	"messenger/internal/storage"

	"github.com/joho/godotenv"
)

// Config holds application configuration
type Config struct {
	Storage         *storage.StorageConfig
	DBURL           string
	JWTSecret       string
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string
}

// Sentinel errors for configuration validation
var (
	ErrDBURLNotSet       = errors.New("DB_URL is not set")
	ErrJWTSecretNotSet   = errors.New("JWT_SECRET is not set")
	ErrJWTSecretTooShort = errors.New("JWT_SECRET must be at least 32 characters long")
)

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if present (ignore error if not found)
	_ = godotenv.Load() //nolint:errcheck // intentionally ignoring - .env is optional

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, ErrDBURLNotSet
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, ErrJWTSecretNotSet
	}

	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("%w: got %d characters", ErrJWTSecretTooShort, len(jwtSecret))
	}

	return &Config{
		DBURL:           dbURL,
		JWTSecret:       jwtSecret,
		Storage:         storage.LoadStorageConfig(),
		VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDSubject:    os.Getenv("VAPID_SUBJECT"),
	}, nil
}

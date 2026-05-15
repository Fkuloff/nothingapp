package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"

	"messenger/internal/storage"

	"github.com/joho/godotenv"
)

// JWT lifetime constants.
//
// We chose long-lived tokens (default 10 years) because the product is a personal
// messenger and the UX requirement is "sessions never expire, even if the user doesn't
// open the app for half a year". For active users, the auth middleware silently issues
// a refreshed token via the X-Refresh-Token response header every 24 hours, sliding the
// expiration forward. For inactive users, the long initial TTL covers the gap.
//
// The 10-year horizon is the worst-case validity of a stolen device-stored token.
// Acceptable for this product; users can rotate by changing their password and
// triggering JWT_SECRET rotation if needed.
const (
	defaultJWTExpiryDays  = 3650  // ~10 years
	tokenRefreshThreshold = 86400 // seconds; reissue when the current token is older than a day
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
	FCMCredentialsPath   string
	JWTExpiryDays        int
}

// TokenRefreshThresholdSeconds returns the age (in seconds) at which the auth middleware
// rotates an active token. Exposed as a function so callers don't import the unexported
// constant directly.
func TokenRefreshThresholdSeconds() int64 {
	return tokenRefreshThreshold
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

	jwtExpiryDays := defaultJWTExpiryDays
	if raw := os.Getenv("JWT_EXPIRY_DAYS"); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 {
			jwtExpiryDays = parsed
		}
	}

	return &Config{
		DBURL:                dbURL,
		JWTSecret:            jwtSecret,
		MessageEncryptionKey: msgEncKey,
		Storage:              storage.LoadStorageConfig(),
		VAPIDPublicKey:       os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey:      os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDSubject:         os.Getenv("VAPID_SUBJECT"),
		FCMCredentialsPath:   os.Getenv("FCM_CREDENTIALS_PATH"),
		JWTExpiryDays:        jwtExpiryDays,
	}, nil
}

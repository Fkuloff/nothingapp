package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"messenger/internal/secret"
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
	Storage            *storage.StorageConfig
	DBURL              string
	JWTSecret          string
	VAPIDPublicKey     string
	VAPIDPrivateKey    string
	VAPIDSubject       string
	FCMCredentialsPath string
	// AdminAPIKey gates the /api/admin/* endpoints (currently just the release
	// registration POST). Compared via constant-time match against the
	// X-Admin-Key header. Empty value disables every admin endpoint (returns
	// 503) so accidental empty-env deploys don't expose an open-write API.
	AdminAPIKey   string
	JWTExpiryDays int
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
)

// LoadConfig loads configuration from environment variables.
//
// Sensitive values (DB_URL, JWT_SECRET, VAPID_PRIVATE_KEY) also support the
// `*_FILE` convention — set e.g. JWT_SECRET_FILE=/run/secrets/jwt_secret and
// the value will be read from the file. Used to plug into Docker Compose secrets
// without leaking the cleartext via `cat .env` or `docker inspect`.
//
// Note: MESSAGE_ENCRYPTION_KEY used to be required for legacy server-side
// (scheme=1) message encryption. It was removed after the full migration to
// client-side E2E (scheme=2). Old scheme=1 rows in the DB are no longer
// readable by the server — clients render them as a "🔒 encrypted message"
// placeholder. New deploys: don't set this variable.
func LoadConfig() (*Config, error) {
	// Load .env file if present (ignore error if not found)
	_ = godotenv.Load() //nolint:errcheck // intentionally ignoring - .env is optional

	dbURL, err := secret.ReadEnvOrFile("DB_URL")
	if err != nil {
		return nil, err
	}
	if dbURL == "" {
		return nil, errDBURLNotSet
	}

	jwtSecret, err := secret.ReadEnvOrFile("JWT_SECRET")
	if err != nil {
		return nil, err
	}
	if jwtSecret == "" {
		return nil, errJWTSecretNotSet
	}
	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("%w: got %d characters", errJWTSecretTooShort, len(jwtSecret))
	}

	vapidPrivateKey, err := secret.ReadEnvOrFile("VAPID_PRIVATE_KEY")
	if err != nil {
		return nil, err
	}

	// Optional. Empty value disables admin endpoints rather than crashing
	// on boot, so production deploys can run before a release-pipeline
	// secret is provisioned.
	adminAPIKey, err := secret.ReadEnvOrFile("ADMIN_API_KEY")
	if err != nil {
		return nil, err
	}

	jwtExpiryDays := defaultJWTExpiryDays
	if raw := os.Getenv("JWT_EXPIRY_DAYS"); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 {
			jwtExpiryDays = parsed
		}
	}

	return &Config{
		DBURL:              dbURL,
		JWTSecret:          jwtSecret,
		Storage:            storage.LoadStorageConfig(),
		VAPIDPublicKey:     os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey:    vapidPrivateKey,
		VAPIDSubject:       os.Getenv("VAPID_SUBJECT"),
		FCMCredentialsPath: os.Getenv("FCM_CREDENTIALS_PATH"),
		AdminAPIKey:        adminAPIKey,
		JWTExpiryDays:      jwtExpiryDays,
	}, nil
}

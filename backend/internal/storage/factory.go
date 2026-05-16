// internal/storage/factory.go
package storage

import (
	"fmt"
	"os"
	"strconv"

	"messenger/internal/secret"
)

// StorageConfig holds configuration for S3-compatible storage backend
type StorageConfig struct {
	S3Endpoint        string // Internal endpoint for backend (e.g., http://minio:9000)
	S3PublicEndpoint  string // Public endpoint for presigned URLs (e.g., http://localhost:9000)
	S3Bucket          string
	S3Region          string
	S3AccessKey       string
	S3SecretKey       string
	S3PresignedExpiry int // Presigned URL expiry in seconds
}

// NewStorage creates a new Storage instance (S3-compatible only)
func NewStorage(config *StorageConfig) (Storage, error) {
	// Validate required S3 config
	if config.S3Bucket == "" {
		return nil, fmt.Errorf("STORAGE_S3_BUCKET is required")
	}
	if config.S3AccessKey == "" {
		return nil, fmt.Errorf("STORAGE_S3_ACCESS_KEY is required")
	}
	if config.S3SecretKey == "" {
		return nil, fmt.Errorf("STORAGE_S3_SECRET_KEY is required")
	}

	return NewS3Storage(config)
}

// LoadStorageConfig loads S3 storage configuration from environment variables.
//
// `STORAGE_S3_ACCESS_KEY` and `STORAGE_S3_SECRET_KEY` additionally support the
// `*_FILE` Docker secrets convention — set e.g. `STORAGE_S3_SECRET_KEY_FILE=
// /run/secrets/minio_root_password` and the value is read from the file. Same
// pattern as `JWT_SECRET_FILE` etc. in the config package.
func LoadStorageConfig() *StorageConfig {
	// Parse presigned URL expiry (default: 1 hour)
	presignedExpiry := 3600
	if expiryStr := os.Getenv("STORAGE_S3_PRESIGNED_EXPIRY"); expiryStr != "" {
		if parsed, err := strconv.Atoi(expiryStr); err == nil && parsed > 0 {
			presignedExpiry = parsed
		}
	}

	// Default region if not specified
	region := os.Getenv("STORAGE_S3_REGION")
	if region == "" {
		region = "us-east-1"
	}

	endpoint := os.Getenv("STORAGE_S3_ENDPOINT")
	publicEndpoint := os.Getenv("STORAGE_S3_PUBLIC_ENDPOINT")
	// If public endpoint not set, use the same as internal endpoint
	if publicEndpoint == "" {
		publicEndpoint = endpoint
	}

	// Errors from ReadEnvOrFile are intentionally swallowed here (treated as empty value),
	// matching the prior behavior where missing env vars were tolerated and the failure
	// only surfaced in NewStorage's required-field validation. That keeps the contract
	// identical for callers — if the secret file is missing/unreadable, you'll see
	// "STORAGE_S3_SECRET_KEY is required" from NewStorage, same as before.
	accessKey, _ := secret.ReadEnvOrFile("STORAGE_S3_ACCESS_KEY") //nolint:errcheck // see comment above
	secretKey, _ := secret.ReadEnvOrFile("STORAGE_S3_SECRET_KEY") //nolint:errcheck // see comment above

	return &StorageConfig{
		S3Endpoint:        endpoint,
		S3PublicEndpoint:  publicEndpoint,
		S3Bucket:          os.Getenv("STORAGE_S3_BUCKET"),
		S3Region:          region,
		S3AccessKey:       accessKey,
		S3SecretKey:       secretKey,
		S3PresignedExpiry: presignedExpiry,
	}
}

// internal/storage/factory.go
package storage

import (
	"fmt"
	"os"
	"strconv"
)

// StorageConfig holds configuration for S3-compatible storage backend
type StorageConfig struct {
	S3Endpoint        string // Internal endpoint for backend (e.g., http://minio:9000)
	S3PublicEndpoint  string // Public endpoint for presigned URLs (e.g., http://localhost:9000)
	S3Bucket          string
	S3Region          string
	S3AccessKey       string
	S3SecretKey       string
	S3UseSSL          bool
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

// LoadStorageConfig loads S3 storage configuration from environment variables
func LoadStorageConfig() *StorageConfig {
	// Parse presigned URL expiry (default: 1 hour)
	presignedExpiry := 3600
	if expiryStr := os.Getenv("STORAGE_S3_PRESIGNED_EXPIRY"); expiryStr != "" {
		if parsed, err := strconv.Atoi(expiryStr); err == nil && parsed > 0 {
			presignedExpiry = parsed
		}
	}

	// Parse SSL flag (accepts: 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False)
	useSSL := false
	if sslStr := os.Getenv("STORAGE_S3_USE_SSL"); sslStr != "" {
		if parsed, err := strconv.ParseBool(sslStr); err == nil {
			useSSL = parsed
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

	return &StorageConfig{
		S3Endpoint:        endpoint,
		S3PublicEndpoint:  publicEndpoint,
		S3Bucket:          os.Getenv("STORAGE_S3_BUCKET"),
		S3Region:          region,
		S3AccessKey:       os.Getenv("STORAGE_S3_ACCESS_KEY"),
		S3SecretKey:       os.Getenv("STORAGE_S3_SECRET_KEY"),
		S3UseSSL:          useSSL,
		S3PresignedExpiry: presignedExpiry,
	}
}

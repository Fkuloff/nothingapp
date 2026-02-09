// internal/storage/factory.go
package storage

import (
	"fmt"
	"os"
)

// StorageType defines the type of storage backend
type StorageType string

const (
	StorageTypeLocal StorageType = "local"
	StorageTypeS3    StorageType = "s3"
)

// StorageConfig holds configuration for storage backend
type StorageConfig struct {
	Type StorageType

	// Local storage config
	LocalBasePath string
	LocalBaseURL  string

	// S3 storage config (for future use)
	S3Bucket     string
	S3Region     string
	S3CloudFront string
}

// NewStorage creates a new Storage instance based on configuration
func NewStorage(config *StorageConfig) (Storage, error) {
	switch config.Type {
	case StorageTypeLocal:
		return NewLocalStorage(config.LocalBasePath, config.LocalBaseURL)
	case StorageTypeS3:
		// TODO: Implement S3Storage when needed
		return nil, fmt.Errorf("S3 storage not yet implemented")
	default:
		return nil, fmt.Errorf("unknown storage type: %s", config.Type)
	}
}

// LoadStorageConfig loads storage configuration from environment variables
func LoadStorageConfig() *StorageConfig {
	storageType := os.Getenv("STORAGE_TYPE")
	if storageType == "" {
		storageType = "local" // Default to local
	}

	return &StorageConfig{
		Type:          StorageType(storageType),
		LocalBasePath: getEnv("STORAGE_LOCAL_PATH", "./uploads"),
		LocalBaseURL:  getEnv("STORAGE_LOCAL_URL", "http://localhost:8080/uploads"),
		S3Bucket:      os.Getenv("STORAGE_S3_BUCKET"),
		S3Region:      os.Getenv("STORAGE_S3_REGION"),
		S3CloudFront:  os.Getenv("STORAGE_S3_CLOUDFRONT"),
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

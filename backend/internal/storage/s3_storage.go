package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// S3Storage implements Storage interface for S3-compatible storage (MinIO, AWS S3)
type S3Storage struct {
	client          *s3.Client
	bucket          string
	region          string
	presignedExpiry time.Duration
	presignClient   *s3.PresignClient
	publicEndpoint  string // Public endpoint for presigned URLs
}

// Verify interface compliance at compile time
var _ Storage = (*S3Storage)(nil)

// NewS3Storage creates a new S3Storage instance
func NewS3Storage(config *StorageConfig) (*S3Storage, error) {
	// Create AWS config
	awsConfig := aws.Config{
		Region: config.S3Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			config.S3AccessKey,
			config.S3SecretKey,
			"",
		),
	}

	// Create S3 client with internal endpoint for backend operations
	clientOptions := func(o *s3.Options) {
		o.UsePathStyle = true // Critical for MinIO
		if config.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(config.S3Endpoint)
		}
	}

	client := s3.NewFromConfig(awsConfig, clientOptions)

	// Create separate client for presigned URLs with public endpoint
	publicClientOptions := func(o *s3.Options) {
		o.UsePathStyle = true
		if config.S3PublicEndpoint != "" {
			o.BaseEndpoint = aws.String(config.S3PublicEndpoint)
		}
	}

	publicClient := s3.NewFromConfig(awsConfig, publicClientOptions)
	presignClient := s3.NewPresignClient(publicClient)

	return &S3Storage{
		client:          client,
		bucket:          config.S3Bucket,
		region:          config.S3Region,
		presignedExpiry: time.Duration(config.S3PresignedExpiry) * time.Second,
		presignClient:   presignClient,
		publicEndpoint:  config.S3PublicEndpoint,
	}, nil
}

// Save stores a file in S3 and returns metadata
func (s *S3Storage) Save(reader io.Reader, fileName, contentType string, size int64) (*FileMetadata, error) {
	ctx := context.Background()

	// Generate unique filename
	ext := filepath.Ext(fileName)
	uniqueID := uuid.New().String()
	uniqueFileName := fmt.Sprintf("%s%s", uniqueID, ext)

	// Create date-based directory structure
	now := time.Now()
	storageKey := filepath.ToSlash(filepath.Join(
		"files",
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fmt.Sprintf("%02d", now.Day()),
		uniqueFileName,
	))

	// Read data into buffer
	var buf bytes.Buffer
	written, err := io.Copy(&buf, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	// Upload to S3
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(storageKey),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	return &FileMetadata{
		Key:         storageKey,
		FileName:    fileName,
		ContentType: contentType,
		Size:        written,
		URL:         s.GetURL(storageKey),
		UploadedAt:  now,
	}, nil
}

// Get retrieves a file from S3
func (s *S3Storage) Get(key string) (io.ReadCloser, error) {
	ctx := context.Background()

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}

	return result.Body, nil
}

// Delete removes a file from S3
func (s *S3Storage) Delete(key string) error {
	ctx := context.Background()

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from S3: %w", err)
	}

	return nil
}

// GetURL returns a presigned URL for file access
func (s *S3Storage) GetURL(key string) string {
	ctx := context.Background()

	presignedReq, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = s.presignedExpiry
	})

	if err != nil {
		return fmt.Sprintf("s3://%s/%s", s.bucket, key)
	}

	return presignedReq.URL
}

// GetThumbnailURL returns a presigned URL for thumbnail access
func (s *S3Storage) GetThumbnailURL(key string) string {
	return s.GetURL(key)
}

// SaveThumbnail saves a thumbnail file to S3
func (s *S3Storage) SaveThumbnail(reader io.Reader, originalKey string) (*FileMetadata, error) {
	ctx := context.Background()

	// Extract date path from original key
	// Format: files/YYYY/MM/DD/uuid.ext → thumbnails/YYYY/MM/DD/uuid_thumb.jpg
	var dateDir string
	parts := strings.Split(filepath.ToSlash(originalKey), "/")
	if len(parts) >= 4 && parts[0] == "files" {
		dateDir = filepath.Join(parts[1], parts[2], parts[3])
	} else {
		now := time.Now()
		dateDir = filepath.Join(
			fmt.Sprintf("%04d", now.Year()),
			fmt.Sprintf("%02d", now.Month()),
			fmt.Sprintf("%02d", now.Day()),
		)
	}

	// Generate thumbnail filename
	uniqueID := uuid.New().String()
	thumbFileName := fmt.Sprintf("%s_thumb.jpg", uniqueID)
	storageKey := filepath.ToSlash(filepath.Join("thumbnails", dateDir, thumbFileName))

	// Read thumbnail data
	var buf bytes.Buffer
	written, err := io.Copy(&buf, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read thumbnail data: %w", err)
	}

	// Upload to S3
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(storageKey),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("image/jpeg"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload thumbnail to S3: %w", err)
	}

	return &FileMetadata{
		Key:         storageKey,
		FileName:    thumbFileName,
		ContentType: "image/jpeg",
		Size:        written,
		URL:         s.GetThumbnailURL(storageKey),
		UploadedAt:  time.Now(),
	}, nil
}

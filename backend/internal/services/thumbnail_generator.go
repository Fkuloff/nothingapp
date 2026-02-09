package services

import (
	"bytes"
	"image"
	"image/jpeg"

	"messenger/internal/storage"

	"github.com/disintegration/imaging"
)

// ThumbnailGenerator generates thumbnails for images and videos
type ThumbnailGenerator struct {
	storage storage.Storage
}

// NewThumbnailGenerator creates a new ThumbnailGenerator instance
func NewThumbnailGenerator(storage storage.Storage) *ThumbnailGenerator {
	return &ThumbnailGenerator{storage: storage}
}

// Generate creates a thumbnail from an image file
func (tg *ThumbnailGenerator) Generate(storageKey string, width, height int) (*storage.FileMetadata, error) {
	// Get original file
	reader, err := tg.storage.Get(storageKey)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// Decode image
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	// Resize maintaining aspect ratio
	thumb := imaging.Fit(img, width, height, imaging.Lanczos)

	// Encode to JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, thumb, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}

	// Save thumbnail using LocalStorage's SaveThumbnail method
	localStorage, ok := tg.storage.(*storage.LocalStorage)
	if !ok {
		// For non-local storage, save as regular file
		metadata, saveErr := tg.storage.Save(
			bytes.NewReader(buf.Bytes()),
			"thumbnail.jpg",
			"image/jpeg",
			int64(buf.Len()),
		)
		return metadata, saveErr
	}

	// Use LocalStorage's specialized method
	metadata, saveErr := localStorage.SaveThumbnail(bytes.NewReader(buf.Bytes()), storageKey)
	return metadata, saveErr
}

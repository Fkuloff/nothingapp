package services

import (
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"

	"messenger/internal/models"
)

// newFileHeader creates a multipart.FileHeader for testing.
func newFileHeader(filename, contentType string, size int64) *multipart.FileHeader {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", contentType)
	return &multipart.FileHeader{
		Filename: filename,
		Header:   header,
		Size:     size,
	}
}

func TestFileValidator_ValidateAttachment(t *testing.T) {
	v := &FileValidator{}

	tests := []struct {
		name      string
		fh        *multipart.FileHeader
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "valid JPEG image",
			fh:      newFileHeader("photo.jpg", "image/jpeg", 1024),
			wantErr: false,
		},
		{
			name:    "valid PDF document",
			fh:      newFileHeader("doc.pdf", "application/pdf", 5*1024*1024),
			wantErr: false,
		},
		{
			name:    "valid MP4 video",
			fh:      newFileHeader("clip.mp4", "video/mp4", 10*1024*1024),
			wantErr: false,
		},
		{
			name:      "unsupported MIME type",
			fh:        newFileHeader("script.exe", "application/x-executable", 1024),
			wantErr:   true,
			errSubstr: "unsupported file type",
		},
		{
			name:      "empty content type",
			fh:        newFileHeader("file.txt", "", 1024),
			wantErr:   true,
			errSubstr: "content type not specified",
		},
		{
			name:      "oversized file",
			fh:        newFileHeader("big.jpg", "image/jpeg", MaxFileSize+1),
			wantErr:   true,
			errSubstr: "file too large",
		},
		{
			name:      "path traversal with dots",
			fh:        newFileHeader("../../../etc/passwd", "image/jpeg", 1024),
			wantErr:   true,
			errSubstr: "invalid filename",
		},
		{
			name:      "path traversal with forward slash",
			fh:        newFileHeader("dir/file.jpg", "image/jpeg", 1024),
			wantErr:   true,
			errSubstr: "invalid filename",
		},
		{
			name:      "path traversal with backslash",
			fh:        newFileHeader("dir\\file.jpg", "image/jpeg", 1024),
			wantErr:   true,
			errSubstr: "invalid filename",
		},
		{
			name:      "filename without extension",
			fh:        newFileHeader("noext", "image/jpeg", 1024),
			wantErr:   true,
			errSubstr: "filename must have an extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateAttachment(tt.fh)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateAttachment() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errSubstr != "" {
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
			}
		})
	}
}

func TestFileValidator_ValidateAvatar(t *testing.T) {
	v := &FileValidator{}

	tests := []struct {
		name      string
		fh        *multipart.FileHeader
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "valid JPEG avatar",
			fh:      newFileHeader("avatar.jpg", "image/jpeg", 1024),
			wantErr: false,
		},
		{
			name:    "valid PNG avatar",
			fh:      newFileHeader("avatar.png", "image/png", 2*1024*1024),
			wantErr: false,
		},
		{
			name:      "non-image type rejected",
			fh:        newFileHeader("doc.pdf", "application/pdf", 1024),
			wantErr:   true,
			errSubstr: "avatar must be an image",
		},
		{
			name:      "oversized avatar",
			fh:        newFileHeader("big.jpg", "image/jpeg", MaxAvatarSize+1),
			wantErr:   true,
			errSubstr: "avatar too large",
		},
		{
			name:      "empty content type",
			fh:        newFileHeader("avatar.jpg", "", 1024),
			wantErr:   true,
			errSubstr: "content type not specified",
		},
		{
			name:      "video MIME rejected",
			fh:        newFileHeader("vid.mp4", "video/mp4", 1024),
			wantErr:   true,
			errSubstr: "avatar must be an image",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateAvatar(tt.fh)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateAvatar() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errSubstr != "" {
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
			}
		})
	}
}

func TestFileValidator_DetermineFileType(t *testing.T) {
	v := &FileValidator{}

	tests := []struct {
		name     string
		mimeType string
		want     models.AttachmentType
	}{
		{"image/jpeg → Image", "image/jpeg", models.AttachmentTypeImage},
		{"image/png → Image", "image/png", models.AttachmentTypeImage},
		{"image/gif → Image", "image/gif", models.AttachmentTypeImage},
		{"image/webp → Image", "image/webp", models.AttachmentTypeImage},
		{"video/mp4 → Video", "video/mp4", models.AttachmentTypeVideo},
		{"video/webm → Video", "video/webm", models.AttachmentTypeVideo},
		{"application/pdf → Document", "application/pdf", models.AttachmentTypeDocument},
		{"text/plain → Document", "text/plain", models.AttachmentTypeDocument},
		{"unknown type → Document", "application/x-unknown", models.AttachmentTypeDocument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.DetermineFileType(tt.mimeType); got != tt.want {
				t.Errorf("DetermineFileType(%q) = %q, want %q", tt.mimeType, got, tt.want)
			}
		})
	}
}

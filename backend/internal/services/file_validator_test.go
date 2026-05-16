package services

import (
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"
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

func TestFileValidator_validateAttachment(t *testing.T) {
	v := &fileValidator{}

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
			fh:        newFileHeader("big.jpg", "image/jpeg", maxFileSize+1),
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
		{
			name:    "file exactly at max size",
			fh:      newFileHeader("exact.jpg", "image/jpeg", maxFileSize),
			wantErr: false,
		},
		{
			name:    "zero size file",
			fh:      newFileHeader("empty.jpg", "image/jpeg", 0),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.validateAttachment(tt.fh)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateAttachment() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errSubstr != "" {
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
			}
		})
	}
}

func TestFileValidator_validateAvatar(t *testing.T) {
	v := &fileValidator{}

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
			fh:        newFileHeader("big.jpg", "image/jpeg", maxAvatarSize+1),
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
		{
			name:    "avatar exactly at max size",
			fh:      newFileHeader("exact.png", "image/png", maxAvatarSize),
			wantErr: false,
		},
		{
			name:    "valid GIF avatar",
			fh:      newFileHeader("anim.gif", "image/gif", 512),
			wantErr: false,
		},
		{
			name:    "valid WebP avatar",
			fh:      newFileHeader("photo.webp", "image/webp", 512),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.validateAvatar(tt.fh)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateAvatar() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errSubstr != "" {
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
			}
		})
	}
}

// determineFileType + AttachmentType were removed when attachment metadata
// became client-side encrypted. The render bucket (image / video / document)
// is now derived on the frontend from the decrypted mime — no equivalent test
// to keep here. See e2e.test.ts:bucketFromMime.

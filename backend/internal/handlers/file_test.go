package handlers

import "testing"

func TestValidateFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{"simple filename", "photo.jpg", false},
		{"filename with dash", "my-file.pdf", false},
		{"filename with underscore", "my_file.png", false},
		{"path traversal with dots", "../etc/passwd", true},
		{"path traversal nested", "../../secret.txt", true},
		{"backslash path traversal", "dir\\file.jpg", true},
		{"absolute path unix", "/etc/passwd", true},
		{"dot dot in middle", "foo..bar.txt", true},
		{"path with extra slash", "subdir/file.jpg", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilename(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFilename(%q) error = %v, wantErr %v", tt.filename, err, tt.wantErr)
			}
		})
	}
}

func TestGetContentTypeFromExtension(t *testing.T) {
	tests := []struct {
		name string
		ext  string
		want string
	}{
		{"jpeg", ".jpg", "image/jpeg"},
		{"jpeg long", ".jpeg", "image/jpeg"},
		{"png", ".png", "image/png"},
		{"gif", ".gif", "image/gif"},
		{"webp", ".webp", "image/webp"},
		{"svg", ".svg", "image/svg+xml"},
		{"pdf", ".pdf", "application/pdf"},
		{"mp4", ".mp4", "video/mp4"},
		{"txt", ".txt", "text/plain"},
		{"zip", ".zip", "application/zip"},
		{"unknown extension", ".xyz", "application/octet-stream"},
		{"empty extension", "", "application/octet-stream"},
		{"no dot", "jpg", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getContentTypeFromExtension(tt.ext); got != tt.want {
				t.Errorf("getContentTypeFromExtension(%q) = %q, want %q", tt.ext, got, tt.want)
			}
		})
	}
}

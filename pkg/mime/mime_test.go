package mime

import (
	"testing"
)

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		path    string
		isImage bool
	}{
		{"photo.jpg", true},
		{"photo.png", true},
		{"photo.gif", true},
		{"photo.webp", true},
		{"doc.txt", false},
		{"code.go", false},
		{"unknown.xyz", false},
	}
	for _, tt := range tests {
		if got := IsImageFile(tt.path); got != tt.isImage {
			t.Errorf("IsImageFile(%q) = %v, want %v", tt.path, got, tt.isImage)
		}
	}
}

func TestIsTextFile(t *testing.T) {
	tests := []struct {
		path   string
		isText bool
	}{
		{"readme.md", true},
		{"config.json", true},
		{"main.go", true},
		{"photo.jpg", false},
		{"binary.dat", false},
	}
	for _, tt := range tests {
		if got := IsTextFile(tt.path); got != tt.isText {
			t.Errorf("IsTextFile(%q) = %v, want %v", tt.path, got, tt.isText)
		}
	}
}

func TestGetMimeType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"photo.jpg", "image/jpeg"},
		{"photo.png", "image/png"},
		{"readme.md", "text/markdown"},
		{"config.json", "application/json"},
		{"main.go", "text/x-go"},
		{"unknown.xyz", "application/octet-stream"},
	}
	for _, tt := range tests {
		if got := GetMimeType(tt.path); got != tt.expected {
			t.Errorf("GetMimeType(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}

func TestIsBinaryFile(t *testing.T) {
	if IsBinaryFile("main.go") {
		t.Error("Go files should not be binary")
	}
	if !IsBinaryFile("unknown.dat") {
		t.Error("Unknown files should be binary")
	}
}

func TestDetectSupportedImageMimeTypeFromFile(t *testing.T) {
	if m := DetectSupportedImageMimeTypeFromFile("test.PNG"); m != "image/png" {
		t.Errorf("Expected case-insensitive match for PNG, got %q", m)
	}
	if m := DetectSupportedImageMimeTypeFromFile("test.txt"); m != "" {
		t.Errorf("Expected empty string for non-image, got %q", m)
	}
}

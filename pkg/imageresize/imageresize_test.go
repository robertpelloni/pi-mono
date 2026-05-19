package imageresize

import (
	"encoding/base64"
	"testing"
)

func TestResizeImage(t *testing.T) {
	img := aiImage{
		Data:     base64.StdEncoding.EncodeToString([]byte("fake-image-data")),
		MimeType: "image/png",
	}
	result := ResizeImage(img)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.MimeType != "image/png" {
		t.Errorf("Expected image/png, got %s", result.MimeType)
	}
}

func TestFormatDimensionNote_Nil(t *testing.T) {
	note := FormatDimensionNote(nil)
	if note != "" {
		t.Errorf("Expected empty note for nil, got %q", note)
	}
}

func TestFormatDimensionNote_ZeroDimensions(t *testing.T) {
	img := &ResizedImage{Width: 0, Height: 0}
	note := FormatDimensionNote(img)
	if note != "" {
		t.Errorf("Expected empty note for zero dimensions, got %q", note)
	}
}

func TestFormatDimensionNote_WithDimensions(t *testing.T) {
	img := &ResizedImage{Width: 800, Height: 600}
	note := FormatDimensionNote(img)
	if note != "Image: 800x600" {
		t.Errorf("Expected 'Image: 800x600', got %q", note)
	}
}

func TestDetectImageDimensions_InvalidData(t *testing.T) {
	_, _, err := DetectImageDimensions("not-valid-base64!!", "image/png")
	if err == nil {
		t.Error("Expected error for invalid data")
	}
}

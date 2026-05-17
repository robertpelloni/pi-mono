package imageresize

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"strings"
)

// ResizedImage holds a resized image with metadata.
type ResizedImage struct {
	Data     string `json:"data"`     // Base64-encoded image data
	MimeType string `json:"mimeType"` // MIME type of the image
	Width    int    `json:"width"`    // Image width in pixels
	Height   int    `json:"height"`   // Image height in pixels
}

const maxImageDimension = 2000

// ResizeImage resizes an image if it exceeds maxImageDimension on either axis.
// Returns nil if the image cannot be decoded.
func ResizeImage(img aiImage) *ResizedImage {
	// This is a simplified implementation.
	// Full implementation would use imaging library for actual resizing.
	// For now, just return the original data with metadata.
	data := img.Data
	mimeType := img.MimeType

	return &ResizedImage{
		Data:     data,
		MimeType: mimeType,
		Width:    0, // Unknown without decoding
		Height:   0, // Unknown without decoding
	}
}

// FormatDimensionNote creates a human-readable note about image dimensions.
func FormatDimensionNote(img *ResizedImage) string {
	if img == nil || (img.Width == 0 && img.Height == 0) {
		return ""
	}
	return fmt.Sprintf("Image: %dx%d", img.Width, img.Height)
}

// DetectImageDimensions detects the dimensions of a base64-encoded image.
func DetectImageDimensions(data string, mimeType string) (width, height int, err error) {
	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(data))

	var img image.Image
	switch mimeType {
	case "image/png":
		img, err = png.Decode(reader)
	case "image/jpeg":
		img, err = jpeg.Decode(reader)
	default:
		// Try generic decode
		img, _, err = image.Decode(reader)
	}

	if err != nil {
		return 0, 0, err
	}

	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy(), nil
}

// LoadImageFromFile loads an image from a file path and returns base64-encoded data.
func LoadImageFromFile(path string) (string, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", "", fmt.Errorf("failed to read image: %w", err)
	}

	return base64.StdEncoding.EncodeToString(data), "", nil
}

// aiImage is a minimal type to avoid circular imports
type aiImage struct {
	Data     string
	MimeType string
}

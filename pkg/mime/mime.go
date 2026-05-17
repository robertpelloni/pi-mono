package mime

import (
	"path/filepath"
	"strings"
)

// SupportedImageMIMETypes maps file extensions to their MIME types.
var SupportedImageMIMETypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
	".bmp":  "image/bmp",
	".ico":  "image/x-icon",
	".tiff": "image/tiff",
	".tif":  "image/tiff",
}

// TextMIMETypes maps file extensions to text-based MIME types.
var TextMIMETypes = map[string]string{
	".txt":  "text/plain",
	".md":   "text/markdown",
	".json": "application/json",
	".yaml": "text/yaml",
	".yml":  "text/yaml",
	".xml":  "text/xml",
	".html": "text/html",
	".css":  "text/css",
	".js":   "text/javascript",
	".ts":   "text/typescript",
	".go":   "text/x-go",
	".py":   "text/x-python",
	".rs":   "text/x-rust",
	".java": "text/x-java",
	".c":    "text/x-c",
	".cpp":  "text/x-c++",
	".h":    "text/x-c",
	".sh":   "text/x-shellscript",
	".bash": "text/x-shellscript",
	".zsh":  "text/x-shellscript",
	".toml": "text/x-toml",
	".ini":  "text/x-ini",
	".csv":  "text/csv",
}

// DetectSupportedImageMimeTypeFromFile detects if a file is a supported image type.
// Returns the MIME type or empty string if not an image.
func DetectSupportedImageMimeTypeFromFile(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if mimeType, ok := SupportedImageMIMETypes[ext]; ok {
		return mimeType
	}
	return ""
}

// IsImageFile checks if a file is a supported image type.
func IsImageFile(path string) bool {
	return DetectSupportedImageMimeTypeFromFile(path) != ""
}

// IsTextFile checks if a file is likely a text file based on extension.
func IsTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := TextMIMETypes[ext]; ok {
		return true
	}
	return false
}

// GetMimeType returns the MIME type for a file path.
func GetMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	// Check image types first
	if mimeType, ok := SupportedImageMIMETypes[ext]; ok {
		return mimeType
	}

	// Check text types
	if mimeType, ok := TextMIMETypes[ext]; ok {
		return mimeType
	}

	// Default
	return "application/octet-stream"
}

// IsBinaryFile checks if a file is likely binary (not text).
func IsBinaryFile(path string) bool {
	return !IsTextFile(path) && !IsImageFile(path)
}

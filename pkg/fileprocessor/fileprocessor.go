package fileprocessor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/badlogic/pi-mono/pkg/mime"
)

// FileContent holds the processed content of a file argument.
type FileContent struct {
	Text    string
	Images  []ImageContent
	Path    string
}

// ImageContent holds image data for sending to the AI.
type ImageContent struct {
	Data     string // Base64-encoded
	MimeType string
	Path     string
}

// ProcessFileArgs processes @file arguments from the CLI.
// Each @file argument can be:
// - A text file (read as string)
// - An image file (read and base64-encoded)
// Returns combined text and any images.
func ProcessFileArgs(cwd string, fileArgs []string) (*FileContent, error) {
	result := &FileContent{}
	var textParts []string

	for _, fileArg := range fileArgs {
		path := fileArg
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, path)
		}

		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("file not found: %s", fileArg)
		}

		if info.IsDir() {
			// For directories, list the files
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read directory %s: %w", fileArg, err)
			}
			var names []string
			for _, e := range entries {
				names = append(names, e.Name())
			}
			textParts = append(textParts, fmt.Sprintf("Contents of %s/:\n%s", fileArg, strings.Join(names, "\n")))
			continue
		}

		// Check if it's an image
		if mimeType := mime.DetectSupportedImageMimeTypeFromFile(path); mimeType != "" {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read image %s: %w", fileArg, err)
			}

			encoded := encodeBase64(data)
			result.Images = append(result.Images, ImageContent{
				Data:     encoded,
				MimeType: mimeType,
				Path:     fileArg,
			})
			continue
		}

		// Read as text
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", fileArg, err)
		}

		textParts = append(textParts, string(content))
	}

	result.Text = strings.Join(textParts, "\n")
	return result, nil
}

// BuildInitialMessage combines stdin content, @file text, and CLI messages.
func BuildInitialMessage(stdinContent string, fileText string, messages []string) string {
	var parts []string

	if stdinContent != "" {
		parts = append(parts, stdinContent)
	}
	if fileText != "" {
		parts = append(parts, fileText)
	}
	if len(messages) > 0 {
		parts = append(parts, messages[0])
	}

	return strings.Join(parts, "")
}

// encodeBase64 encodes bytes to base64 string.
func encodeBase64(data []byte) string {
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	result := make([]byte, 0, (len(data)+2)/3*4)

	for i := 0; i < len(data); i += 3 {
		n := len(data) - i
		if n > 3 {
			n = 3
		}

		var b0, b1, b2 byte
		b0 = data[i]
		if n > 1 {
			b1 = data[i+1]
		}
		if n > 2 {
			b2 = data[i+2]
		}

		result = append(result, base64Chars[b0>>2])
		result = append(result, base64Chars[(b0&0x03)<<4|b1>>4])

		if n > 1 {
			result = append(result, base64Chars[(b1&0x0f)<<2|b2>>6])
		} else {
			result = append(result, '=')
		}

		if n > 2 {
			result = append(result, base64Chars[b2&0x3f])
		} else {
			result = append(result, '=')
		}
	}

	return string(result)
}

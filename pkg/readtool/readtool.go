package readtool

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/badlogic/pi-mono/pkg/truncate"
)

// ReadOperations allows plugging in custom file reading operations.
type ReadOperations interface {
	Access(absolutePath string) error
	ReadFile(absolutePath string) ([]byte, error)
}

type defaultReadOps struct{}

func (d *defaultReadOps) Access(absolutePath string) error {
	_, err := os.Stat(absolutePath)
	if err != nil {
		return fmt.Errorf("file not found: %s", absolutePath)
	}
	return nil
}

func (d *defaultReadOps) ReadFile(absolutePath string) ([]byte, error) {
	return os.ReadFile(absolutePath)
}

// ReadToolInput represents the input parameters for the read tool.
type ReadToolInput struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// ReadToolResult represents the result of the read tool.
type ReadToolResult struct {
	Content    string                    `json:"content"`
	Truncation *truncate.TruncationResult `json:"truncation,omitempty"`
	IsImage    bool                      `json:"isImage,omitempty"`
	MimeType   string                    `json:"mimeType,omitempty"`
	ImageData  string                    `json:"imageData,omitempty"`
}

// Supported image MIME types
var supportedImageTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
}

// Execute runs the read tool.
func Execute(ctx context.Context, input ReadToolInput, cwd string, ops ReadOperations) (*ReadToolResult, error) {
	absolutePath := input.Path
	if !filepath.IsAbs(absolutePath) {
		absolutePath = filepath.Join(cwd, absolutePath)
	}

	// Expand ~ to home directory
	if strings.HasPrefix(absolutePath, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			absolutePath = filepath.Join(home, absolutePath[1:])
		}
	}

	if ops == nil {
		ops = &defaultReadOps{}
	}

	// Check if file exists and is readable
	if err := ops.Access(absolutePath); err != nil {
		return nil, err
	}

	// Check if it's an image
	ext := strings.ToLower(filepath.Ext(absolutePath))
	if mimeType, ok := supportedImageTypes[ext]; ok {
		buffer, err := ops.ReadFile(absolutePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read image: %w", err)
		}
		return &ReadToolResult{
			Content:   fmt.Sprintf("Read image file [%s]", mimeType),
			IsImage:   true,
			MimeType:  mimeType,
			ImageData: base64.StdEncoding.EncodeToString(buffer),
		}, nil
	}

	// Read text content
	buffer, err := ops.ReadFile(absolutePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	textContent := string(buffer)
	allLines := strings.Split(strings.ReplaceAll(textContent, "\r\n", "\n"), "\n")
	totalFileLines := len(allLines)

	// Apply offset (1-indexed to 0-indexed)
	startLine := 0
	if input.Offset > 0 {
		startLine = input.Offset - 1
	}
	startLineDisplay := startLine + 1

	// Check if offset is out of bounds
	if startLine >= totalFileLines {
		return nil, fmt.Errorf("offset %d is beyond end of file (%d lines total)", input.Offset, totalFileLines)
	}

	var selectedContent string
	userLimitedLines := 0

	if input.Limit > 0 {
		endLine := startLine + input.Limit
		if endLine > totalFileLines {
			endLine = totalFileLines
		}
		selectedContent = strings.Join(allLines[startLine:endLine], "\n")
		userLimitedLines = endLine - startLine
	} else {
		selectedContent = strings.Join(allLines[startLine:], "\n")
	}

	// Apply truncation
	truncation := truncate.TruncateHead(selectedContent, truncate.TruncationOptions{})
	var outputText string

	if truncation.FirstLineExceeds {
		firstLineSize := len(allLines[startLine])
		outputText = fmt.Sprintf("[Line %d is %s, exceeds %s limit. Use bash: sed -n '%dp' %s | head -c %d]",
			startLineDisplay, truncate.FormatSize(firstLineSize), truncate.FormatSize(truncation.MaxBytes),
			startLineDisplay, input.Path, truncate.DefaultMaxBytes)
	} else if truncation.Truncated {
		endLineDisplay := startLineDisplay + truncation.OutputLines - 1
		nextOffset := endLineDisplay + 1
		outputText = truncation.Content

		if truncation.TruncatedBy == "lines" {
			outputText += fmt.Sprintf("\n\n[Showing lines %d-%d of %d. Use offset=%d to continue.]",
				startLineDisplay, endLineDisplay, totalFileLines, nextOffset)
		} else {
			outputText += fmt.Sprintf("\n\n[Showing lines %d-%d of %d (%s limit). Use offset=%d to continue.]",
				startLineDisplay, endLineDisplay, totalFileLines, truncate.FormatSize(truncation.MaxBytes), nextOffset)
		}
	} else if userLimitedLines > 0 && startLine+userLimitedLines < totalFileLines {
		remaining := totalFileLines - (startLine + userLimitedLines)
		nextOffset := startLine + userLimitedLines + 1
		outputText = fmt.Sprintf("%s\n\n[%d more lines in file. Use offset=%d to continue.]",
			truncation.Content, remaining, nextOffset)
	} else {
		outputText = truncation.Content
	}

	result := &ReadToolResult{
		Content: outputText,
	}
	if truncation.Truncated || truncation.FirstLineExceeds {
		result.Truncation = &truncation
	}

	return result, nil
}

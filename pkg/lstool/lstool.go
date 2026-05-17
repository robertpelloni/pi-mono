package lstool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/badlogic/pi-mono/pkg/truncate"
)

// LsOperations allows plugging in custom directory listing operations.
type LsOperations interface {
	Exists(absolutePath string) bool
	IsDirectory(absolutePath string) (bool, error)
	ReadDir(absolutePath string) ([]string, error)
	Stat(absolutePath string) (os.FileInfo, error)
}

type defaultLsOps struct{}

func (d *defaultLsOps) Exists(absolutePath string) bool {
	_, err := os.Stat(absolutePath)
	return err == nil
}

func (d *defaultLsOps) IsDirectory(absolutePath string) (bool, error) {
	info, err := os.Stat(absolutePath)
	if err != nil {
		return false, fmt.Errorf("path not found: %s", absolutePath)
	}
	return info.IsDir(), nil
}

func (d *defaultLsOps) ReadDir(absolutePath string) ([]string, error) {
	entries, err := os.ReadDir(absolutePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory: %w", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

func (d *defaultLsOps) Stat(absolutePath string) (os.FileInfo, error) {
	return os.Stat(absolutePath)
}

// LsToolInput represents the input parameters for the ls tool.
type LsToolInput struct {
	Path  string `json:"path,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

// LsToolResult represents the result of the ls tool.
type LsToolResult struct {
	Content           string                    `json:"content"`
	Truncation        *truncate.TruncationResult `json:"truncation,omitempty"`
	EntryLimitReached int                       `json:"entryLimitReached,omitempty"`
}

const defaultLsLimit = 500

// Execute runs the ls tool.
func Execute(ctx context.Context, input LsToolInput, cwd string, ops LsOperations) (*LsToolResult, error) {
	dirPath := input.Path
	if dirPath == "" {
		dirPath = "."
	}
	if !filepath.IsAbs(dirPath) {
		dirPath = filepath.Join(cwd, dirPath)
	}

	effectiveLimit := input.Limit
	if effectiveLimit <= 0 {
		effectiveLimit = defaultLsLimit
	}

	if ops == nil {
		ops = &defaultLsOps{}
	}

	// Check if path exists
	if !ops.Exists(dirPath) {
		return nil, fmt.Errorf("path not found: %s", dirPath)
	}

	// Check if it's a directory
	isDir, err := ops.IsDirectory(dirPath)
	if err != nil {
		return nil, err
	}
	if !isDir {
		return nil, fmt.Errorf("not a directory: %s", dirPath)
	}

	// Read directory entries
	entries, err := ops.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	// Sort alphabetically, case-insensitive
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i]) < strings.ToLower(entries[j])
	})

	// Format entries with directory indicator
	var results []string
	entryLimitReached := false

	for _, entry := range entries {
		if len(results) >= effectiveLimit {
			entryLimitReached = true
			break
		}

		fullPath := filepath.Join(dirPath, entry)
		info, err := ops.Stat(fullPath)
		if err != nil {
			continue
		}
		if info.IsDir() {
			results = append(results, entry+"/")
		} else {
			results = append(results, entry)
		}
	}

	if len(results) == 0 {
		return &LsToolResult{Content: "(empty directory)"}, nil
	}

	rawOutput := strings.Join(results, "\n")

	// Apply byte truncation
	truncation := truncate.TruncateHead(rawOutput, truncate.TruncationOptions{})
	output := truncation.Content

	var notices []string
	if entryLimitReached {
		notices = append(notices, fmt.Sprintf("%d entries limit reached. Use limit=%d for more", effectiveLimit, effectiveLimit*2))
	}
	if truncation.Truncated {
		notices = append(notices, fmt.Sprintf("%s limit reached", truncate.FormatSize(truncation.MaxBytes)))
	}
	if len(notices) > 0 {
		output += "\n\n[" + strings.Join(notices, ". ") + "]"
	}

	result := &LsToolResult{
		Content: output,
	}
	if truncation.Truncated {
		result.Truncation = &truncation
	}
	if entryLimitReached {
		result.EntryLimitReached = effectiveLimit
	}

	return result, nil
}

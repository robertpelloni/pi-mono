package findtool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/badlogic/pi-mono/pkg/truncate"
)

// FindOperations allows plugging in custom file search operations.
type FindOperations interface {
	// Exists checks if the path exists.
	Exists(absolutePath string) bool
	// Glob finds files matching a glob pattern. Returns relative or absolute paths.
	Glob(pattern string, cwd string, ignore []string, limit int) ([]string, error)
}

// defaultFindOps implements local filesystem find operations.
type defaultFindOps struct{}

func (d *defaultFindOps) Exists(absolutePath string) bool {
	_, err := os.Stat(absolutePath)
	return err == nil
}

func (d *defaultFindOps) Glob(pattern string, cwd string, ignore []string, limit int) ([]string, error) {
	var results []string
	// Use filepath.Glob with double-star support
	err := filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip ignored directories
		rel, _ := filepath.Rel(cwd, path)
		for _, ign := range ignore {
			if strings.Contains(rel, ign) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Try matching the pattern
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched && !info.IsDir() {
			results = append(results, path)
			if len(results) >= limit {
				return fmt.Errorf("limit reached")
			}
		}

		// Also try matching with ** patterns
		if strings.Contains(pattern, "**") {
			// Simple **/ glob matching
			globMatch := matchGlob(pattern, rel)
			if globMatch && !info.IsDir() {
				// Check if already added
				for _, r := range results {
					if r == path {
						return nil
					}
				}
				results = append(results, path)
				if len(results) >= limit {
					return fmt.Errorf("limit reached")
				}
			}
		}

		return nil
	})
	if err != nil && err.Error() != "limit reached" {
		return nil, err
	}
	return results, nil
}

// matchGlob performs simple glob matching with ** support.
func matchGlob(pattern, name string) bool {
	pattern = filepath.ToSlash(pattern)
	name = filepath.ToSlash(name)

	// Split pattern into parts
	parts := strings.Split(pattern, "/")
	nameParts := strings.Split(name, "/")

	return matchGlobParts(parts, nameParts)
}

func matchGlobParts(pattern, name []string) bool {
	for len(pattern) > 0 {
		part := pattern[0]
		switch {
		case part == "**":
			// ** matches zero or more path segments
			pattern = pattern[1:]
			if len(pattern) == 0 {
				return true
			}
			for i := 0; i <= len(name); i++ {
				if matchGlobParts(pattern, name[i:]) {
					return true
				}
			}
			return false
		case len(name) == 0:
			return false
		default:
			matched, _ := filepath.Match(part, name[0])
			if !matched {
				return false
			}
			pattern = pattern[1:]
			name = name[1:]
		}
	}
	return len(name) == 0
}

// FindToolInput represents the input parameters for the find tool.
type FindToolInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// FindToolResult represents the result of the find tool.
type FindToolResult struct {
	Content           string                  `json:"content"`
	Truncation        *truncate.TruncationResult `json:"truncation,omitempty"`
	ResultLimitReached int                    `json:"resultLimitReached,omitempty"`
}

const defaultFindLimit = 1000

// Execute runs the find tool.
func Execute(ctx context.Context, input FindToolInput, cwd string, ops FindOperations) (*FindToolResult, error) {
	searchDir := input.Path
	if searchDir == "" {
		searchDir = "."
	}
	if !filepath.IsAbs(searchDir) {
		searchDir = filepath.Join(cwd, searchDir)
	}

	effectiveLimit := input.Limit
	if effectiveLimit <= 0 {
		effectiveLimit = defaultFindLimit
	}

	if ops == nil {
		ops = &defaultFindOps{}
	}

	// Check path exists
	if !ops.Exists(searchDir) {
		return nil, fmt.Errorf("path not found: %s", searchDir)
	}

	// Execute glob search
	results, err := ops.Glob(input.Pattern, searchDir, []string{"node_modules", ".git"}, effectiveLimit)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &FindToolResult{Content: "No files found matching pattern"}, nil
	}

	// Relativize paths
	var relativized []string
	for _, p := range results {
		rel, err := filepath.Rel(searchDir, p)
		if err != nil {
			rel = p
		}
		rel = filepath.ToSlash(rel)
		relativized = append(relativized, rel)
	}
	sort.Strings(relativized)

	resultLimitReached := len(relativized) >= effectiveLimit
	rawOutput := strings.Join(relativized, "\n")

	// Apply byte truncation
	truncation := truncate.TruncateHead(rawOutput, truncate.TruncationOptions{})
	output := truncation.Content

	var notices []string
	if resultLimitReached {
		notices = append(notices, fmt.Sprintf("%d results limit reached. Use limit=%d for more, or refine pattern", effectiveLimit, effectiveLimit*2))
	}
	if truncation.Truncated {
		notices = append(notices, fmt.Sprintf("%s limit reached", truncate.FormatSize(truncate.DefaultMaxBytes)))
	}

	if len(notices) > 0 {
		output += "\n\n[" + strings.Join(notices, ". ") + "]"
	}

	result := &FindToolResult{
		Content:    output,
		Truncation: nil,
	}
	if truncation.Truncated {
		result.Truncation = &truncation
	}
	if resultLimitReached {
		result.ResultLimitReached = effectiveLimit
	}

	return result, nil
}

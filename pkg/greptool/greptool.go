package greptool

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/badlogic/pi-mono/pkg/truncate"
)

// GrepOperations allows plugging in custom grep operations.
type GrepOperations interface {
	IsDirectory(absolutePath string) (bool, error)
	ReadFile(absolutePath string) (string, error)
}

type defaultGrepOps struct{}

func (d *defaultGrepOps) IsDirectory(absolutePath string) (bool, error) {
	info, err := os.Stat(absolutePath)
	if err != nil {
		return false, fmt.Errorf("path not found: %s", absolutePath)
	}
	return info.IsDir(), nil
}

func (d *defaultGrepOps) ReadFile(absolutePath string) (string, error) {
	data, err := os.ReadFile(absolutePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GrepToolInput represents the input parameters for the grep tool.
type GrepToolInput struct {
	Pattern    string `json:"pattern"`
	Path       string `json:"path,omitempty"`
	Glob       string `json:"glob,omitempty"`
	IgnoreCase bool   `json:"ignoreCase,omitempty"`
	Literal    bool   `json:"literal,omitempty"`
	Context    int    `json:"context,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// GrepToolResult represents the result of the grep tool.
type GrepToolResult struct {
	Content           string                    `json:"content"`
	Truncation        *truncate.TruncationResult `json:"truncation,omitempty"`
	MatchLimitReached int                       `json:"matchLimitReached,omitempty"`
	LinesTruncated    bool                      `json:"linesTruncated,omitempty"`
}

const defaultGrepLimit = 100

// grepMatch represents a single grep match.
type grepMatch struct {
	filePath   string
	lineNumber int
}

// Execute runs the grep tool.
func Execute(ctx context.Context, input GrepToolInput, cwd string, ops GrepOperations) (*GrepToolResult, error) {
	searchDir := input.Path
	if searchDir == "" {
		searchDir = "."
	}
	if !filepath.IsAbs(searchDir) {
		searchDir = filepath.Join(cwd, searchDir)
	}

	effectiveLimit := input.Limit
	if effectiveLimit <= 0 {
		effectiveLimit = defaultGrepLimit
	}

	contextValue := input.Context
	if contextValue < 0 {
		contextValue = 0
	}

	if ops == nil {
		ops = &defaultGrepOps{}
	}

	// Check if search path exists and is a directory
	isDir, err := ops.IsDirectory(searchDir)
	if err != nil {
		return nil, err
	}

	// Try to use ripgrep if available
	if rgPath, _ := exec.LookPath("rg"); rgPath != "" {
		return executeWithRg(ctx, rgPath, input, searchDir, isDir, effectiveLimit, contextValue, ops)
	}

	// Fallback to Go-native implementation
	return executeNative(ctx, input, searchDir, isDir, effectiveLimit, contextValue, ops)
}

func executeWithRg(ctx context.Context, rgPath string, input GrepToolInput, searchDir string, isDir bool, effectiveLimit int, contextValue int, ops GrepOperations) (*GrepToolResult, error) {
	args := []string{"--json", "--line-number", "--color=never", "--hidden"}
	if input.IgnoreCase {
		args = append(args, "--ignore-case")
	}
	if input.Literal {
		args = append(args, "--fixed-strings")
	}
	if input.Glob != "" {
		args = append(args, "--glob", input.Glob)
	}
	args = append(args, input.Pattern, searchDir)

	cmd := exec.CommandContext(ctx, rgPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to run ripgrep: %w", err)
	}

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ripgrep: %w", err)
	}

	var matches []grepMatch
	matchCount := 0
	matchLimitReached := false

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		if eventType, ok := event["type"].(string); ok && eventType == "match" {
			if matchCount >= effectiveLimit {
				matchLimitReached = true
				break
			}
			data, _ := event["data"].(map[string]interface{})
			if data != nil {
				pathText, _ := data["path"].(map[string]interface{})
				if pathText != nil {
					if filePath, ok := pathText["text"].(string); ok {
						if lineNumber, ok := data["line_number"].(float64); ok {
							matches = append(matches, grepMatch{filePath: filePath, lineNumber: int(lineNumber)})
							matchCount++
							if matchCount >= effectiveLimit {
								matchLimitReached = true
							}
						}
					}
				}
			}
		}
	}

	cmd.Process.Kill()
	cmd.Wait()

	if matchCount == 0 {
		return &GrepToolResult{Content: "No matches found"}, nil
	}

	return formatGrepOutput(matches, searchDir, isDir, contextValue, effectiveLimit, matchLimitReached, ops)
}

// executeNative is a pure Go fallback when rg is not available.
func executeNative(ctx context.Context, input GrepToolInput, searchDir string, isDir bool, effectiveLimit int, contextValue int, ops GrepOperations) (*GrepToolResult, error) {
	var pattern *regexp.Regexp
	var err error

	if input.Literal {
		pattern = regexp.MustCompile(regexp.QuoteMeta(input.Pattern))
	} else {
		pattern, err = regexp.Compile(input.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	if input.IgnoreCase {
		insensitivePattern := "(?i)" + pattern.String()
		pattern, err = regexp.Compile(insensitivePattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	var matches []grepMatch
	matchLimitReached := false

	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Skip .git and node_modules
		rel, _ := filepath.Rel(searchDir, path)
		relSlash := filepath.ToSlash(rel)
		if strings.Contains(relSlash, "node_modules") || strings.Contains(relSlash, ".git") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// Apply glob filter
		if input.Glob != "" {
			matched, _ := filepath.Match(input.Glob, filepath.Base(path))
			if !matched {
				return nil
			}
		}

		// Search file contents
		content, err := ops.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
		for i, line := range lines {
			if pattern.MatchString(line) {
				matches = append(matches, grepMatch{filePath: path, lineNumber: i + 1})
				if len(matches) >= effectiveLimit {
					matchLimitReached = true
					return fmt.Errorf("limit reached")
				}
			}
		}

		return nil
	}

	if isDir {
		filepath.WalkDir(searchDir, walkFunc)
	} else {
		// Search single file
		content, err := ops.ReadFile(searchDir)
		if err != nil {
			return nil, fmt.Errorf("cannot read file: %w", err)
		}
		lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
		for i, line := range lines {
			if pattern.MatchString(line) {
				matches = append(matches, grepMatch{filePath: searchDir, lineNumber: i + 1})
				if len(matches) >= effectiveLimit {
					matchLimitReached = true
				}
			}
		}
	}

	if len(matches) == 0 {
		return &GrepToolResult{Content: "No matches found"}, nil
	}

	return formatGrepOutput(matches, searchDir, isDir, contextValue, effectiveLimit, matchLimitReached, ops)
}

func formatGrepOutput(matches []grepMatch, searchDir string, isDir bool, contextValue int, effectiveLimit int, matchLimitReached bool, ops GrepOperations) (*GrepToolResult, error) {
	formatPath := func(filePath string) string {
		if isDir {
			rel, err := filepath.Rel(searchDir, filePath)
			if err == nil && !strings.HasPrefix(rel, "..") {
				return filepath.ToSlash(rel)
			}
		}
		return filepath.Base(filePath)
	}

	fileCache := make(map[string][]string)
	getFileLines := func(filePath string) []string {
		if lines, ok := fileCache[filePath]; ok {
			return lines
		}
		content, err := ops.ReadFile(filePath)
		if err != nil {
			return nil
		}
		lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
		fileCache[filePath] = lines
		return lines
	}

	var outputLines []string
	linesTruncated := false

	for _, m := range matches {
		relativePath := formatPath(m.filePath)
		lines := getFileLines(m.filePath)
		if lines == nil {
			outputLines = append(outputLines, fmt.Sprintf("%s:%d: (unable to read file)", relativePath, m.lineNumber))
			continue
		}

		start := m.lineNumber - contextValue
		if start < 1 {
			start = 1
		}
		end := m.lineNumber + contextValue
		if end > len(lines) {
			end = len(lines)
		}

		for current := start; current <= end; current++ {
			lineText := strings.ReplaceAll(lines[current-1], "\r", "")
			isMatchLine := current == m.lineNumber

			truncatedText, wasTruncated := truncate.TruncateLine(lineText)
			if wasTruncated {
				linesTruncated = true
			}

			if isMatchLine {
				outputLines = append(outputLines, fmt.Sprintf("%s:%d: %s", relativePath, current, truncatedText))
			} else {
				outputLines = append(outputLines, fmt.Sprintf("%s-%d- %s", relativePath, current, truncatedText))
			}
		}
	}

	rawOutput := strings.Join(outputLines, "\n")
	truncation := truncate.TruncateHead(rawOutput, truncate.TruncationOptions{})
	output := truncation.Content

	var notices []string
	if matchLimitReached {
		notices = append(notices, fmt.Sprintf("%d matches limit reached. Use limit=%d for more, or refine pattern", effectiveLimit, effectiveLimit*2))
	}
	if truncation.Truncated {
		notices = append(notices, fmt.Sprintf("%s limit reached", truncate.FormatSize(truncation.MaxBytes)))
	}
	if linesTruncated {
		notices = append(notices, fmt.Sprintf("Some lines truncated to %d chars. Use read tool to see full lines", truncate.GrepMaxLineLength))
	}
	if len(notices) > 0 {
		output += "\n\n[" + strings.Join(notices, ". ") + "]"
	}

	result := &GrepToolResult{
		Content:        output,
		LinesTruncated: linesTruncated,
	}
	if truncation.Truncated {
		result.Truncation = &truncation
	}
	if matchLimitReached {
		result.MatchLimitReached = effectiveLimit
	}

	return result, nil
}

package nativetools

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// NativeReadTool reads files using Go's os package instead of shelling out.
// This matches the TypeScript read tool behavior more closely.
func NativeReadTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "read",
		Label:       "read",
		Description: "Read the contents of a file. Supports text files and images (jpg, png, gif, webp). Images are sent as attachments. For text files, output is truncated to 2000 lines or 50KB (whichever is hit first). Use offset/limit for large files.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to read (relative or absolute)",
				},
				"offset": map[string]interface{}{
					"type":        "number",
					"description": "Line number to start reading from (1-indexed)",
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of lines to read",
				},
			},
			"required": []string{"path"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			path, _ := params["path"].(string)
			if path == "" {
				return agent.AgentToolResult{}, fmt.Errorf("missing path parameter")
			}

			absPath := resolvePath(cwd, path)

			info, err := os.Stat(absPath)
			if err != nil {
				return agent.AgentToolResult{}, fmt.Errorf("cannot access %q: %w", path, err)
			}

			if info.IsDir() {
				return readDirectory(absPath)
			}

			// Check if it's an image
			ext := strings.ToLower(filepath.Ext(absPath))
			if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" {
				return readImageFile(absPath, ext)
			}

			// Read as text file
			return readTextFile(absPath, params)
		},
	}
}

func resolvePath(cwd, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cwd, path)
}

func readDirectory(path string) (agent.AgentToolResult, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return agent.AgentToolResult{}, err
	}

	var lines []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		info, err := entry.Info()
		if err == nil {
			lines = append(lines, fmt.Sprintf("%s\t%d", name, info.Size()))
		} else {
			lines = append(lines, name)
		}
	}

	return agent.AgentToolResult{
		Content: []ai.Content{ai.TextContent{Text: strings.Join(lines, "\n")}},
	}, nil
}

func readTextFile(path string, params map[string]any) (agent.AgentToolResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return agent.AgentToolResult{}, err
	}

	const maxSize = 50 * 1024 // 50KB
	const maxLines = 2000

	content := string(data)
	lines := strings.Split(content, "\n")

	// Apply offset/limit
	offset := 1
	if o, ok := params["offset"].(float64); ok && int(o) > 0 {
		offset = int(o)
	}
	limit := maxLines
	if l, ok := params["limit"].(float64); ok && int(l) > 0 {
		limit = int(l)
	}

	// Adjust for 1-indexed offset
	startIdx := offset - 1
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx >= len(lines) {
		return agent.AgentToolResult{
			Content: []ai.Content{ai.TextContent{Text: fmt.Sprintf("Offset %d exceeds file length (%d lines)", offset, len(lines))}},
		}, nil
	}

	endIdx := startIdx + limit
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	selectedLines := lines[startIdx:endIdx]
	result := strings.Join(selectedLines, "\n")

	// Truncate if too large
	if len(result) > maxSize {
		result = result[:maxSize] + fmt.Sprintf("\n\n... [truncated at %d bytes, file is %d bytes total]", maxSize, len(content))
	}

	// Add line numbers
	numberedLines := make([]string, len(selectedLines))
	for i, line := range selectedLines {
		numberedLines[i] = fmt.Sprintf("%6d\t%s", startIdx+i+1, line)
	}
	result = strings.Join(numberedLines, "\n")

	// Add file info header
	header := fmt.Sprintf("File: %s (%d lines, %d bytes)\n", path, len(lines), len(content))
	if startIdx > 0 || endIdx < len(lines) {
		header += fmt.Sprintf("Showing lines %d-%d of %d\n", startIdx+1, endIdx, len(lines))
	}

	return agent.AgentToolResult{
		Content: []ai.Content{ai.TextContent{Text: header + result}},
		Details: map[string]any{
			"path":  path,
			"lines": len(lines),
			"size":  len(content),
		},
	}, nil
}

func readImageFile(path, ext string) (agent.AgentToolResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return agent.AgentToolResult{}, err
	}

	mimeType := "image/jpeg"
	switch ext {
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	}

	// Return as image content
	return agent.AgentToolResult{
		Content: []ai.Content{
			ai.ImageContent{
				Data:     fmt.Sprintf("data:%s;base64,", mimeType) + encodeBase64(data),
				MimeType: mimeType,
			},
		},
		Details: map[string]any{
			"path": path,
			"size": len(data),
			"type": "image",
		},
	}, nil
}

func encodeBase64(data []byte) string {
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var result bytes.Buffer

	for i := 0; i < len(data); i += 3 {
		remaining := len(data) - i
		b0 := data[i]
		b1 := byte(0)
		b2 := byte(0)
		if remaining > 1 {
			b1 = data[i+1]
		}
		if remaining > 2 {
			b2 = data[i+2]
		}

		result.WriteByte(base64Chars[b0>>2])
		result.WriteByte(base64Chars[(b0&0x03)<<4|b1>>4])
		if remaining > 1 {
			result.WriteByte(base64Chars[(b1&0x0f)<<2|b2>>6])
		} else {
			result.WriteByte('=')
		}
		if remaining > 2 {
			result.WriteByte(base64Chars[b2&0x3f])
		} else {
			result.WriteByte('=')
		}
	}

	return result.String()
}

// NativeGlobTool finds files by glob pattern using Go's filepath.Glob.
func NativeGlobTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "glob",
		Label:       "glob",
		Description: "Search for files by glob pattern. Returns matching file paths relative to the search directory. Respects .gitignore.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Glob pattern to match files, e.g. '*.ts' or '**/*.spec.ts'",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory to search in (default: current directory)",
				},
			},
			"required": []string{"pattern"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			pattern, _ := params["pattern"].(string)
			if pattern == "" {
				return agent.AgentToolResult{}, fmt.Errorf("missing pattern parameter")
			}

			searchDir := cwd
			if p, ok := params["path"].(string); ok && p != "" {
				searchDir = resolvePath(cwd, p)
			}

			// Use filepath.Glob for simple patterns
			matches, err := filepath.Glob(filepath.Join(searchDir, pattern))
			if err != nil {
				return agent.AgentToolResult{}, fmt.Errorf("invalid glob pattern: %w", err)
			}

			// For ** patterns, walk the directory tree
			if strings.Contains(pattern, "**") {
				matches, err = globWalk(searchDir, pattern)
				if err != nil {
					return agent.AgentToolResult{}, err
				}
			}

			// Make paths relative
			var relativeMatches []string
			for _, m := range matches {
				rel, err := filepath.Rel(cwd, m)
				if err == nil {
					relativeMatches = append(relativeMatches, rel)
				} else {
					relativeMatches = append(relativeMatches, m)
				}
			}

			sort.Strings(relativeMatches)

			result := fmt.Sprintf("Found %d files:\n%s", len(relativeMatches), strings.Join(relativeMatches, "\n"))
			if len(relativeMatches) == 0 {
				result = "No files found matching pattern."
			}

			return agent.AgentToolResult{
				Content: []ai.Content{ai.TextContent{Text: result}},
			}, nil
		},
	}
}

// globWalk implements recursive glob matching using filepath.WalkDir.
func globWalk(root, pattern string) ([]string, error) {
	var matches []string

	// Convert glob pattern to regex for matching
	regexPattern := globToRegex(pattern)
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		rel = filepath.ToSlash(rel)
		if re.MatchString(rel) {
			matches = append(matches, path)
		}

		return nil
	})

	return matches, err
}

// globToRegex converts a simple glob pattern to a regex.
func globToRegex(pattern string) string {
	pattern = filepath.ToSlash(pattern)
	regex := "^"
	for _, ch := range pattern {
		switch ch {
		case '*':
			regex += "[^/]*"
		case '?':
			regex += "[^/]"
		case '.':
			regex += "\\."
		case '+':
			regex += "\\+"
		case '(':
			regex += "\\("
		case ')':
			regex += "\\)"
		default:
			regex += string(ch)
		}
	}
	regex += "$"

	// Replace ** pattern (match any path segments)
	regex = strings.ReplaceAll(regex, "[^/]*[^/]*", ".*")

	return regex
}

// NativeGrepTool searches file contents using Go instead of shelling out.
func NativeGrepTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "grep",
		Label:       "grep",
		Description: "Search file contents for a pattern. Returns matching lines with file paths and line numbers. Respects .gitignore.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Search pattern (regex or literal string)",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory or file to search in (default: current directory)",
				},
				"ignoreCase": map[string]interface{}{
					"type":        "boolean",
					"description": "Case-insensitive search (default: false)",
				},
				"literal": map[string]interface{}{
					"type":        "boolean",
					"description": "Treat pattern as literal string instead of regex (default: false)",
				},
				"context": map[string]interface{}{
					"type":        "number",
					"description": "Number of lines to show before and after each match (default: 0)",
				},
			},
			"required": []string{"pattern"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			pattern, _ := params["pattern"].(string)
			if pattern == "" {
				return agent.AgentToolResult{}, fmt.Errorf("missing pattern parameter")
			}

			searchPath := cwd
			if p, ok := params["path"].(string); ok && p != "" {
				searchPath = resolvePath(cwd, p)
			}

			ignoreCase := false
			if ic, ok := params["ignoreCase"].(bool); ok {
				ignoreCase = ic
			}

			literal := false
			if l, ok := params["literal"].(bool); ok {
				literal = l
			}

			contextLines := 0
			if c, ok := params["context"].(float64); ok {
				contextLines = int(c)
			}

			// Build regex
			regexPattern := pattern
			if literal {
				regexPattern = regexp.QuoteMeta(pattern)
			}
			if ignoreCase {
				regexPattern = "(?i)" + regexPattern
			}

			re, err := regexp.Compile(regexPattern)
			if err != nil {
				return agent.AgentToolResult{}, fmt.Errorf("invalid pattern: %w", err)
			}

			// Search
			var results []string
			matchCount := 0
			const maxMatches = 100

			err = filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					// Skip common non-code directories
					if d != nil && d.IsDir() {
						name := d.Name()
						if name == ".git" || name == "node_modules" || name == "vendor" ||
							name == "__pycache__" || name == ".venv" || name == "dist" || name == "build" {
							return filepath.SkipDir
						}
					}
					return nil
				}

				if matchCount >= maxMatches {
					return filepath.SkipDir
				}

				// Skip binary files
				if isBinaryFile(path) {
					return nil
				}

				data, err := os.ReadFile(path)
				if err != nil {
					return nil
				}

				lines := strings.Split(string(data), "\n")
				relPath, _ := filepath.Rel(cwd, path)

				for i, line := range lines {
					if re.MatchString(line) {
						if contextLines > 0 {
							start := i - contextLines
							if start < 0 {
								start = 0
							}
							end := i + contextLines + 1
							if end > len(lines) {
								end = len(lines)
							}
							for j := start; j < end; j++ {
								prefix := "  "
								if j == i {
									prefix = "> "
								}
								results = append(results, fmt.Sprintf("%s:%d:%s%s", relPath, j+1, prefix, lines[j]))
							}
							results = append(results, "--")
						} else {
							// Truncate long lines
							displayLine := line
							if len(displayLine) > 500 {
								displayLine = displayLine[:500] + "..."
							}
							results = append(results, fmt.Sprintf("%s:%d:%s", relPath, i+1, displayLine))
						}
						matchCount++
						if matchCount >= maxMatches {
							break
						}
					}
				}

				return nil
			})

			if err != nil {
				return agent.AgentToolResult{}, err
			}

			var output string
			if len(results) == 0 {
				output = "No matches found."
			} else {
				output = strings.Join(results, "\n")
				if matchCount >= maxMatches {
					output += fmt.Sprintf("\n\n... [truncated at %d matches]", maxMatches)
				}
			}

			return agent.AgentToolResult{
				Content: []ai.Content{ai.TextContent{Text: output}},
			}, nil
		},
	}
}

// NativeBashTool executes commands with a timeout.
func NativeBashTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "bash",
		Label:       "bash",
		Description: "Execute a bash command in the current working directory. Returns stdout and stderr. Supports timeout.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The bash command to run",
				},
				"timeout": map[string]interface{}{
					"type":        "number",
					"description": "Optional timeout in seconds",
				},
			},
			"required": []string{"command"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			command, _ := params["command"].(string)
			if command == "" {
				return agent.AgentToolResult{}, fmt.Errorf("missing command parameter")
			}

			// Block destructive commands
			blockedPatterns := []string{"rm -rf /", "rm -rf ~", "mkfs.", "dd if=", "format "}
			lowerCmd := strings.ToLower(command)
			for _, pattern := range blockedPatterns {
				if strings.Contains(lowerCmd, pattern) {
					return agent.AgentToolResult{}, fmt.Errorf("blocked: potentially destructive command")
				}
			}

			// Set timeout
			timeoutSec := 120 // Default 2 minutes
			if t, ok := params["timeout"].(float64); ok && int(t) > 0 {
				timeoutSec = int(t)
			}

			timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
			defer cancel()

			cmd := exec.CommandContext(timeoutCtx, "bash", "-c", command)
			cmd.Dir = cwd

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			output := stdout.String()
			errOutput := stderr.String()

			isError := err != nil
			if isError && output == "" && errOutput == "" {
				errOutput = err.Error()
			}

			result := output
			if errOutput != "" {
				if result != "" {
					result += "\n"
				}
				result += "[stderr]\n" + errOutput
			}

			exitCode := 0
			if cmd.ProcessState != nil {
				exitCode = cmd.ProcessState.ExitCode()
			}

			return agent.AgentToolResult{
				Content: []ai.Content{ai.TextContent{Text: result}},
				Details: map[string]any{
					"command":  command,
					"exitCode": exitCode,
					"isError":  isError,
				},
			}, nil
		},
	}
}

// NativeLsTool lists directory contents natively.
func NativeLsTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "ls",
		Label:       "ls",
		Description: "List directory contents. Returns entries sorted alphabetically, with '/' suffix for directories. Includes dotfiles.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory to list (default: current directory)",
				},
			},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			path := cwd
			if p, ok := params["path"].(string); ok && p != "" {
				path = resolvePath(cwd, p)
			}

			entries, err := os.ReadDir(path)
			if err != nil {
				return agent.AgentToolResult{}, err
			}

			var lines []string
			for _, entry := range entries {
				name := entry.Name()
				if entry.IsDir() {
					name += "/"
				}
				info, err := entry.Info()
				if err == nil {
					lines = append(lines, fmt.Sprintf("%-50s %10d  %s", name, info.Size(), info.ModTime().Format("2006-01-02 15:04")))
				} else {
					lines = append(lines, name)
				}
			}

			sort.Strings(lines)

			return agent.AgentToolResult{
				Content: []ai.Content{ai.TextContent{Text: strings.Join(lines, "\n")}},
			}, nil
		},
	}
}

// isBinaryFile checks if a file appears to be binary.
func isBinaryFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".webp": true, ".ico": true, ".bmp": true, ".tiff": true,
		".zip": true, ".tar": true, ".gz": true, ".bz2": true,
		".xz": true, ".7z": true, ".rar": true, ".pdf": true,
		".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
		".ppt": true, ".pptx": true, ".woff": true, ".woff2": true,
		".ttf": true, ".eot": true, ".otf": true, ".mp3": true,
		".mp4": true, ".avi": true, ".mov": true, ".wav": true,
		".sqlite": true, ".db": true, ".pyc": true, ".o": true,
		".a": true, ".class": true, ".jar": true, ".wasm": true,
	}
	return binaryExts[ext]
}



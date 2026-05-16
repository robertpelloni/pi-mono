package truncate

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Default truncation limits matching the TypeScript version.
const (
	DefaultMaxLines       = 2000
	DefaultMaxBytes       = 50 * 1024 // 50KB
	GrepMaxLineLength     = 500
)

// TruncationResult describes the outcome of a truncation operation.
type TruncationResult struct {
	Content            string `json:"content"`
	Truncated          bool   `json:"truncated"`
	TruncatedBy        string `json:"truncatedBy"` // "lines", "bytes", or ""
	TotalLines         int    `json:"totalLines"`
	TotalBytes         int    `json:"totalBytes"`
	OutputLines        int    `json:"outputLines"`
	OutputBytes        int    `json:"outputBytes"`
	LastLinePartial    bool   `json:"lastLinePartial"`
	FirstLineExceeds   bool   `json:"firstLineExceedsLimit"`
	MaxLines           int    `json:"maxLines"`
	MaxBytes           int    `json:"maxBytes"`
}

// TruncationOptions configures truncation behavior.
type TruncationOptions struct {
	MaxLines int `json:"maxLines"`
	MaxBytes int `json:"maxBytes"`
}

// FormatSize formats a byte count as a human-readable string.
func FormatSize(b int) string {
	if b < 1024 {
		return fmt.Sprintf("%dB", b)
	} else if b < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(b)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
}

// getOpts returns options with defaults applied.
func getOpts(opts TruncationOptions) (int, int) {
	maxLines := opts.MaxLines
	if maxLines == 0 {
		maxLines = DefaultMaxLines
	}
	maxBytes := opts.MaxBytes
	if maxBytes == 0 {
		maxBytes = DefaultMaxBytes
	}
	return maxLines, maxBytes
}

// TruncateHead truncates content from the head (keeps first N lines/bytes).
// Suitable for file reads where you want to see the beginning.
// Never returns partial lines.
func TruncateHead(content string, opts ...TruncationOptions) TruncationResult {
	maxLines, maxBytes := DefaultMaxLines, DefaultMaxBytes
	if len(opts) > 0 {
		maxLines, maxBytes = getOpts(opts[0])
	}

	totalBytes := len([]byte(content))
	lines := splitLines(content)
	totalLines := len(lines)

	if totalLines <= maxLines && totalBytes <= maxBytes {
		return TruncationResult{
			Content:     content,
			Truncated:   false,
			TruncatedBy: "",
			TotalLines:  totalLines,
			TotalBytes:  totalBytes,
			OutputLines: totalLines,
			OutputBytes: totalBytes,
			MaxLines:    maxLines,
			MaxBytes:    maxBytes,
		}
	}

	// Check if first line alone exceeds byte limit
	if len([]byte(lines[0])) > maxBytes {
		return TruncationResult{
			Content:          "",
			Truncated:        true,
			TruncatedBy:      "bytes",
			TotalLines:       totalLines,
			TotalBytes:       totalBytes,
			OutputLines:      0,
			OutputBytes:      0,
			FirstLineExceeds: true,
			MaxLines:         maxLines,
			MaxBytes:         maxBytes,
		}
	}

	var outputLines []string
	outputBytesCount := 0
	truncatedBy := "lines"

	for i := 0; i < len(lines) && i < maxLines; i++ {
		line := lines[i]
		lineBytes := len([]byte(line))
		if i > 0 {
			lineBytes++ // newline
		}
		if outputBytesCount+lineBytes > maxBytes {
			truncatedBy = "bytes"
			break
		}
		outputLines = append(outputLines, line)
		outputBytesCount += lineBytes
	}

	if len(outputLines) >= maxLines && outputBytesCount <= maxBytes {
		truncatedBy = "lines"
	}

	output := joinLines(outputLines)
	return TruncationResult{
		Content:     output,
		Truncated:   true,
		TruncatedBy: truncatedBy,
		TotalLines:  totalLines,
		TotalBytes:  totalBytes,
		OutputLines: len(outputLines),
		OutputBytes: len([]byte(output)),
		MaxLines:    maxLines,
		MaxBytes:    maxBytes,
	}
}

// TruncateTail truncates content from the tail (keeps last N lines/bytes).
// Suitable for bash output where you want to see the end.
// May return partial first line if the last line exceeds the byte limit.
func TruncateTail(content string, opts ...TruncationOptions) TruncationResult {
	maxLines, maxBytes := DefaultMaxLines, DefaultMaxBytes
	if len(opts) > 0 {
		maxLines, maxBytes = getOpts(opts[0])
	}

	totalBytes := len([]byte(content))
	lines := splitLines(content)
	totalLines := len(lines)

	if totalLines <= maxLines && totalBytes <= maxBytes {
		return TruncationResult{
			Content:     content,
			Truncated:   false,
			TruncatedBy: "",
			TotalLines:  totalLines,
			TotalBytes:  totalBytes,
			OutputLines: totalLines,
			OutputBytes: totalBytes,
			MaxLines:    maxLines,
			MaxBytes:    maxBytes,
		}
	}

	var outputLines []string
	outputBytesCount := 0
	truncatedBy := "lines"
	lastLinePartial := false

	for i := len(lines) - 1; i >= 0 && len(outputLines) < maxLines; i-- {
		line := lines[i]
		lineBytes := len([]byte(line))
		if len(outputLines) > 0 {
			lineBytes++ // newline
		}
		if outputBytesCount+lineBytes > maxBytes {
			truncatedBy = "bytes"
			// Edge case: if no lines yet and this line exceeds maxBytes,
			// take the end of the line (partial)
			if len(outputLines) == 0 {
				truncated := truncateStringToBytesFromEnd(line, maxBytes)
				outputLines = append([]string{truncated}, outputLines...)
				outputBytesCount = len([]byte(truncated))
				lastLinePartial = true
			}
			break
		}
		outputLines = append([]string{line}, outputLines...)
		outputBytesCount += lineBytes
	}

	if len(outputLines) >= maxLines && outputBytesCount <= maxBytes {
		truncatedBy = "lines"
	}

	output := joinLines(outputLines)
	return TruncationResult{
		Content:         output,
		Truncated:       true,
		TruncatedBy:     truncatedBy,
		TotalLines:      totalLines,
		TotalBytes:      totalBytes,
		OutputLines:     len(outputLines),
		OutputBytes:     len([]byte(output)),
		LastLinePartial: lastLinePartial,
		MaxLines:        maxLines,
		MaxBytes:        maxBytes,
	}
}

// TruncateLine truncates a single line to maxChars, adding "..." suffix.
func TruncateLine(line string, maxChars ...int) (text string, wasTruncated bool) {
	limit := GrepMaxLineLength
	if len(maxChars) > 0 {
		limit = maxChars[0]
	}
	if utf8.RuneCountInString(line) <= limit {
		return line, false
	}
	runes := []rune(line)
	return string(runes[:limit]) + "... [truncated]", true
}

// truncateStringToBytesFromEnd truncates a string to fit within a byte limit,
// taking from the end and respecting UTF-8 boundaries.
func truncateStringToBytesFromEnd(s string, maxBytes int) string {
	b := []byte(s)
	if len(b) <= maxBytes {
		return s
	}

	start := len(b) - maxBytes
	// Find valid UTF-8 boundary
	for start < len(b) && (b[start]&0xC0) == 0x80 {
		start++
	}
	return string(b[start:])
}

// splitLines splits content into lines, preserving the behavior of
// JavaScript's string.split("\n").
func splitLines(content string) []string {
	if content == "" {
		return []string{""}
	}
	return strings.Split(content, "\n")
}

// joinLines joins lines with newlines.
func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}



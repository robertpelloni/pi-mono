package editdiff

import (
	"fmt"
	"strings"
	"unicode"
)

// Edit represents a single oldText -> newText replacement.
type Edit struct {
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
}

// AppliedEditsResult contains the content before and after edits.
type AppliedEditsResult struct {
	BaseContent string `json:"baseContent"`
	NewContent  string `json:"newContent"`
}

// DiffResult contains a unified diff string and the first changed line number.
type DiffResult struct {
	Diff             string `json:"diff"`
	FirstChangedLine int    `json:"firstChangedLine"` // 0 if none
}

// DetectLineEnding returns the dominant line ending in content.
func DetectLineEnding(content string) string {
	crlfIdx := strings.Index(content, "\r\n")
	lfIdx := strings.Index(content, "\n")
	if lfIdx == -1 || crlfIdx == -1 || crlfIdx > lfIdx {
		return "\n"
	}
	return "\r\n"
}

// NormalizeToLF converts all line endings to LF.
func NormalizeToLF(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

// RestoreLineEndings converts LF back to the original line ending.
func RestoreLineEndings(text, ending string) string {
	if ending == "\r\n" {
		return strings.ReplaceAll(text, "\n", "\r\n")
	}
	return text
}

// NormalizeForFuzzyMatch applies progressive transformations for fuzzy matching:
// - Strip trailing whitespace from each line
// - Normalize smart quotes to ASCII
// - Normalize Unicode dashes to ASCII hyphen
// - Normalize special spaces to regular space
func NormalizeForFuzzyMatch(text string) string {
	// NFKC normalization would go here in full impl; we handle common cases
	// Strip trailing whitespace per line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	result := strings.Join(lines, "\n")

	// Smart single quotes → '
	result = strings.Map(func(r rune) rune {
		switch r {
		case '\u2018', '\u2019', '\u201A', '\u201B':
			return '\''
		case '\u201C', '\u201D', '\u201E', '\u201F':
			return '"'
		case '\u2010', '\u2011', '\u2012', '\u2013', '\u2014', '\u2015', '\u2212':
			return '-'
		case '\u00A0', '\u2002', '\u2003', '\u2004', '\u2005', '\u2006',
			'\u2007', '\u2008', '\u2009', '\u200A', '\u202F', '\u205F', '\u3000':
			return ' '
		default:
			return r
		}
	}, result)

	return result
}

// FuzzyMatchResult describes the outcome of a fuzzy text search.
type FuzzyMatchResult struct {
	Found               bool   `json:"found"`
	Index               int    `json:"index"`
	MatchLength         int    `json:"matchLength"`
	UsedFuzzyMatch      bool   `json:"usedFuzzyMatch"`
	ContentForReplacement string `json:"contentForReplacement"`
}

// FuzzyFindText finds oldText in content, trying exact match first, then fuzzy.
func FuzzyFindText(content, oldText string) FuzzyMatchResult {
	// Try exact match first
	idx := strings.Index(content, oldText)
	if idx != -1 {
		return FuzzyMatchResult{
			Found:               true,
			Index:               idx,
			MatchLength:         len(oldText),
			UsedFuzzyMatch:      false,
			ContentForReplacement: content,
		}
	}

	// Try fuzzy match
	fuzzyContent := NormalizeForFuzzyMatch(content)
	fuzzyOldText := NormalizeForFuzzyMatch(oldText)
	fuzzyIdx := strings.Index(fuzzyContent, fuzzyOldText)
	if fuzzyIdx == -1 {
		return FuzzyMatchResult{
			Found: false,
		}
	}

	return FuzzyMatchResult{
		Found:               true,
		Index:               fuzzyIdx,
		MatchLength:         len(fuzzyOldText),
		UsedFuzzyMatch:      true,
		ContentForReplacement: fuzzyContent,
	}
}

// StripBom removes a UTF-8 BOM if present.
// The UTF-8 BOM is the byte sequence 0xEF,0xBB,0xBF.
func StripBom(content string) (bom, text string) {
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		return content[:3], content[3:]
	}
	return "", content
}

// CountOccurrences counts how many times oldText appears in content.
func CountOccurrences(content, oldText string) int {
	fuzzyContent := NormalizeForFuzzyMatch(content)
	fuzzyOldText := NormalizeForFuzzyMatch(oldText)
	return strings.Count(fuzzyContent, fuzzyOldText)
}

// ApplyEditsToNormalizedContent applies exact-text replacements to LF-normalized content.
// All edits are matched against the same original content.
// Replacements are applied in reverse order so offsets remain stable.
func ApplyEditsToNormalizedContent(normalizedContent string, edits []Edit, path string) (*AppliedEditsResult, error) {
	normalizedEdits := make([]Edit, len(edits))
	for i, edit := range edits {
		normalizedEdits[i] = Edit{
			OldText: NormalizeToLF(edit.OldText),
			NewText: NormalizeToLF(edit.NewText),
		}
	}

	// Validate: no empty oldText
	for i, edit := range normalizedEdits {
		if len(edit.OldText) == 0 {
			if len(normalizedEdits) == 1 {
				return nil, fmt.Errorf("oldText must not be empty in %s", path)
			}
			return nil, fmt.Errorf("edits[%d].oldText must not be empty in %s", i, path)
		}
	}

	// Check initial matches to determine if we need fuzzy space
	initialMatches := make([]FuzzyMatchResult, len(normalizedEdits))
	needsFuzzy := false
	for i, edit := range normalizedEdits {
		initialMatches[i] = FuzzyFindText(normalizedContent, edit.OldText)
		if initialMatches[i].UsedFuzzyMatch {
			needsFuzzy = true
		}
	}

	baseContent := normalizedContent
	if needsFuzzy {
		baseContent = NormalizeForFuzzyMatch(normalizedContent)
	}

	type matchedEdit struct {
		editIndex   int
		matchIndex  int
		matchLength int
		newText     string
	}

	var matchedEdits []matchedEdit
	for i, edit := range normalizedEdits {
		matchResult := FuzzyFindText(baseContent, edit.OldText)
		if !matchResult.Found {
			if len(normalizedEdits) == 1 {
				return nil, fmt.Errorf("could not find the exact text in %s. The old text must match exactly including all whitespace and newlines", path)
			}
			return nil, fmt.Errorf("could not find edits[%d] in %s. The oldText must match exactly including all whitespace and newlines", i, path)
		}

		occurrences := CountOccurrences(baseContent, edit.OldText)
		if occurrences > 1 {
			if len(normalizedEdits) == 1 {
				return nil, fmt.Errorf("found %d occurrences of the text in %s. The text must be unique. Please provide more context to make it unique", occurrences, path)
			}
			return nil, fmt.Errorf("found %d occurrences of edits[%d] in %s. Each oldText must be unique. Please provide more context to make it unique", occurrences, i, path)
		}

		matchedEdits = append(matchedEdits, matchedEdit{
			editIndex:   i,
			matchIndex:  matchResult.Index,
			matchLength: matchResult.MatchLength,
			newText:     edit.NewText,
		})
	}

	// Sort by match index
	sortSlice(matchedEdits, func(a, b matchedEdit) bool {
		return a.matchIndex < b.matchIndex
	})

	// Check for overlaps
	for i := 1; i < len(matchedEdits); i++ {
		prev := matchedEdits[i-1]
		curr := matchedEdits[i]
		if prev.matchIndex+prev.matchLength > curr.matchIndex {
			return nil, fmt.Errorf("edits[%d] and edits[%d] overlap in %s. Merge them into one edit or target disjoint regions", prev.editIndex, curr.editIndex, path)
		}
	}

	// Apply in reverse order
	newContent := baseContent
	for i := len(matchedEdits) - 1; i >= 0; i-- {
		edit := matchedEdits[i]
		newContent = newContent[:edit.matchIndex] + edit.newText + newContent[edit.matchIndex+edit.matchLength:]
	}

	if baseContent == newContent {
		if len(normalizedEdits) == 1 {
			return nil, fmt.Errorf("no changes made to %s. The replacement produced identical content. This might indicate an issue with special characters or the text not existing as expected", path)
		}
		return nil, fmt.Errorf("no changes made to %s. The replacements produced identical content", path)
	}

	return &AppliedEditsResult{
		BaseContent: baseContent,
		NewContent:  newContent,
	}, nil
}

// GenerateDiffString creates a unified diff with line numbers and context.
func GenerateDiffString(oldContent, newContent string, contextLines ...int) DiffResult {
	numContext := 4
	if len(contextLines) > 0 {
		numContext = contextLines[0]
	}

	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	maxLineNum := len(oldLines)
	if len(newLines) > maxLineNum {
		maxLineNum = len(newLines)
	}
	lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))

	parts := diffLines(oldLines, newLines)

	var output []string
	oldLineNum := 1
	newLineNum := 1
	lastWasChange := false
	firstChangedLine := 0

	for i := 0; i < len(parts); i++ {
		part := parts[i]

		if part.changeType != 0 { // added or removed
			if firstChangedLine == 0 {
				firstChangedLine = newLineNum
			}

			for _, line := range part.lines {
				if part.changeType > 0 { // added
					output = append(output, fmt.Sprintf("+%s %s", padNum(newLineNum, lineNumWidth), line))
					newLineNum++
				} else { // removed
					output = append(output, fmt.Sprintf("-%s %s", padNum(oldLineNum, lineNumWidth), line))
					oldLineNum++
				}
			}
			lastWasChange = true
		} else { // context
			nextIsChange := i < len(parts)-1 && parts[i+1].changeType != 0
			hasLeading := lastWasChange
			hasTrailing := nextIsChange

			raw := part.lines

			if hasLeading && hasTrailing {
				if len(raw) <= numContext*2 {
					for _, line := range raw {
						output = append(output, fmt.Sprintf(" %s %s", padNum(oldLineNum, lineNumWidth), line))
						oldLineNum++
						newLineNum++
					}
				} else {
					leading := raw[:numContext]
					trailing := raw[len(raw)-numContext:]
					skipped := len(raw) - len(leading) - len(trailing)
					for _, line := range leading {
						output = append(output, fmt.Sprintf(" %s %s", padNum(oldLineNum, lineNumWidth), line))
						oldLineNum++
						newLineNum++
					}
					output = append(output, fmt.Sprintf(" %s ...", padNum(0, lineNumWidth)))
					oldLineNum += skipped
					newLineNum += skipped
					for _, line := range trailing {
						output = append(output, fmt.Sprintf(" %s %s", padNum(oldLineNum, lineNumWidth), line))
						oldLineNum++
						newLineNum++
					}
				}
			} else if hasLeading {
				shown := raw
				if len(shown) > numContext {
					shown = shown[:numContext]
				}
				skipped := len(raw) - len(shown)
				for _, line := range shown {
					output = append(output, fmt.Sprintf(" %s %s", padNum(oldLineNum, lineNumWidth), line))
					oldLineNum++
					newLineNum++
				}
				if skipped > 0 {
					output = append(output, fmt.Sprintf(" %s ...", padNum(0, lineNumWidth)))
					oldLineNum += skipped
					newLineNum += skipped
				}
			} else if hasTrailing {
				skipped := len(raw) - numContext
				if skipped < 0 {
					skipped = 0
				}
				if skipped > 0 {
					output = append(output, fmt.Sprintf(" %s ...", padNum(0, lineNumWidth)))
					oldLineNum += skipped
					newLineNum += skipped
				}
				for _, line := range raw[skipped:] {
					output = append(output, fmt.Sprintf(" %s %s", padNum(oldLineNum, lineNumWidth), line))
					oldLineNum++
					newLineNum++
				}
			} else {
				oldLineNum += len(raw)
				newLineNum += len(raw)
			}
			lastWasChange = false
		}
	}

	return DiffResult{
		Diff:             strings.Join(output, "\n"),
		FirstChangedLine: firstChangedLine,
	}
}

// --- Internal diff algorithm (Myers-like) ---

type diffPart struct {
	changeType int    // 0=equal, -1=removed, 1=added
	lines      []string
}

// diffLines computes a line-level diff between old and new content.
func diffLines(oldLines, newLines []string) []diffPart {
	// Simple LCS-based diff
	n := len(oldLines)
	m := len(newLines)

	// For very large files, fall back to simple comparison
	if n > 5000 || m > 5000 {
		return simpleDiff(oldLines, newLines)
	}

	// Build LCS table
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack to find the diff
	var parts []diffPart
	i, j := n, m
	var reversed []diffPart

	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			reversed = append(reversed, diffPart{0, []string{oldLines[i-1]}})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			reversed = append(reversed, diffPart{1, []string{newLines[j-1]}})
			j--
		} else {
			reversed = append(reversed, diffPart{-1, []string{oldLines[i-1]}})
			i--
		}
	}

	// Reverse
	for i := len(reversed) - 1; i >= 0; i-- {
		parts = append(parts, reversed[i])
	}

	// Merge consecutive parts of same type
	return mergeParts(parts)
}

func simpleDiff(oldLines, newLines []string) []diffPart {
	var parts []diffPart
	minLen := len(oldLines)
	if len(newLines) < minLen {
		minLen = len(newLines)
	}

	for i := 0; i < minLen; i++ {
		if oldLines[i] == newLines[i] {
			parts = append(parts, diffPart{0, []string{oldLines[i]}})
		} else {
			parts = append(parts, diffPart{-1, []string{oldLines[i]}})
			parts = append(parts, diffPart{1, []string{newLines[i]}})
		}
	}
	if len(oldLines) > minLen {
		parts = append(parts, diffPart{-1, oldLines[minLen:]})
	}
	if len(newLines) > minLen {
		parts = append(parts, diffPart{1, newLines[minLen:]})
	}
	return mergeParts(parts)
}

func mergeParts(parts []diffPart) []diffPart {
	if len(parts) == 0 {
		return parts
	}
	merged := []diffPart{parts[0]}
	for i := 1; i < len(parts); i++ {
		if merged[len(merged)-1].changeType == parts[i].changeType {
			merged[len(merged)-1].lines = append(merged[len(merged)-1].lines, parts[i].lines...)
		} else {
			merged = append(merged, parts[i])
		}
	}
	return merged
}

func padNum(n, width int) string {
	return fmt.Sprintf("%*d", width, n)
}

// Ensure unicode import is used
var _ = unicode.IsSpace

// sortSlice sorts a slice of matchedEdit by matchIndex.
func sortSlice[S any](slice []S, less func(a, b S) bool) {
	// Simple insertion sort for small slices
	for i := 1; i < len(slice); i++ {
		j := i
		for j > 0 && less(slice[j], slice[j-1]) {
			slice[j], slice[j-1] = slice[j-1], slice[j]
			j--
		}
	}
}

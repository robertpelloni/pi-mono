package summarizer

import (
	"strings"
	"unicode"

	"github.com/badlogic/pi-mono/pkg/memory"
)

// SummarizeContext concatenates knowledge entries into a context string limited by maxTokens.
// It preserves whole entries and stops when the token budget is exceeded.
func SummarizeContext(entries []*memory.KnowledgeEntry, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}

	var builder strings.Builder
	used := 0

	for _, e := range entries {
		text := e.Title + "\n" + e.Content + "\n"
		tokens := countTokens(text)
		if used+tokens > maxTokens {
			break
		}
		builder.WriteString(text)
		used += tokens
	}

	return strings.TrimSpace(builder.String())
}

// countTokens provides a naive token count by counting whitespace-separated words.
func countTokens(s string) int {
	if s == "" {
		return 0
	}
	count := 0
	inWord := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if inWord {
				count++
				inWord = false
			}
		} else {
			inWord = true
		}
	}
	if inWord {
		count++
	}
	return count
}

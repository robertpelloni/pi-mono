package frontmatter

import (
	"regexp"
	"strings"
)

// FrontMatter represents parsed YAML frontmatter from a markdown file.
type FrontMatter struct {
	Content string                 `json:"content"` // Content after frontmatter
	Fields  map[string]interface{} `json:"fields"`  // Parsed frontmatter fields
}

var frontmatterRegex = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n(.*)$`)

// ParseFrontMatter extracts YAML frontmatter from a markdown string.
// Returns the content after the frontmatter and any parsed fields.
func ParseFrontMatter(content string) *FrontMatter {
	matches := frontmatterRegex.FindStringSubmatch(content)
	if matches == nil {
		return &FrontMatter{
			Content: content,
			Fields:  make(map[string]interface{}),
		}
	}

	fm := &FrontMatter{
		Content: matches[2],
		Fields:  make(map[string]interface{}),
	}

	// Simple YAML key: value parsing
	lines := strings.Split(matches[1], "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			fm.Fields[key] = value
		}
	}

	return fm
}

// HasFrontMatter checks if content starts with YAML frontmatter.
func HasFrontMatter(content string) bool {
	return strings.HasPrefix(strings.TrimSpace(content), "---")
}

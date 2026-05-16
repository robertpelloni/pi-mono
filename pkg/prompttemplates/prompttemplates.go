package prompttemplates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/badlogic/pi-mono/pkg/ai"
)

// PromptTemplate is a reusable prompt definition loaded from a markdown file.
type PromptTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	FilePath    string `json:"filePath"`
	BaseDir     string `json:"baseDir"`
	Content     string `json:"-"`
}

// PromptTemplateLoader discovers and loads prompt templates from the filesystem.
type PromptTemplateLoader struct {
	mu    sync.RWMutex
	cache map[string]*LoadResult
}

// LoadResult contains loaded templates and diagnostics.
type LoadResult struct {
	Templates   []PromptTemplate      `json:"templates"`
	Diagnostics []TemplateDiagnostic  `json:"diagnostics"`
}

// TemplateDiagnostic is a non-fatal issue loading a template.
type TemplateDiagnostic struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// NewPromptTemplateLoader creates a new loader.
func NewPromptTemplateLoader() *PromptTemplateLoader {
	return &PromptTemplateLoader{
		cache: make(map[string]*LoadResult),
	}
}

// Load discovers prompt templates from the agent and project directories.
// Templates are .md files in:
//   - ~/.pi/prompts/ (global)
//   - .pi/prompts/   (project)
func (l *PromptTemplateLoader) Load(agentDir, projectDir string) *LoadResult {
	key := agentDir + ":" + projectDir
	l.mu.RLock()
	if cached, ok := l.cache[key]; ok {
		l.mu.RUnlock()
		return cached
	}
	l.mu.RUnlock()

	result := &LoadResult{}

	// Load global prompts
	l.loadFromDir(filepath.Join(agentDir, "prompts"), "global", result)

	// Load project prompts
	l.loadFromDir(filepath.Join(projectDir, ".pi", "prompts"), "project", result)

	l.mu.Lock()
	l.cache[key] = result
	l.mu.Unlock()

	return result
}

// ExpandPromptTemplate resolves a prompt template reference and returns the expanded content.
// Template references look like: /my-template or template:my-template
// If the input doesn't match a template, it's returned as-is.
func (l *PromptTemplateLoader) ExpandPromptTemplate(input string, templates []PromptTemplate) string {
	name := extractTemplateName(input)
	if name == "" {
		return input
	}

	for _, tmpl := range templates {
		if tmpl.Name == name {
			return tmpl.Content
		}
	}

	return input
}

// ClearCache clears the loader cache.
func (l *PromptTemplateLoader) ClearCache() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache = make(map[string]*LoadResult)
}

func (l *PromptTemplateLoader) loadFromDir(dir, scope string, result *LoadResult) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, TemplateDiagnostic{
				Path:    filePath,
				Message: err.Error(),
			})
			continue
		}

		content := string(data)
		fm, body := parsePromptFrontmatter(content)

		name := strings.TrimSuffix(entry.Name(), ".md")
		if n, ok := fm["name"].(string); ok && n != "" {
			name = n
		}

		description := ""
		if d, ok := fm["description"].(string); ok {
			description = d
		}

		result.Templates = append(result.Templates, PromptTemplate{
			Name:        name,
			Description: description,
			FilePath:    filePath,
			BaseDir:     dir,
			Content:     strings.TrimSpace(body),
		})
	}
}

func parsePromptFrontmatter(content string) (map[string]any, string) {
	fm := make(map[string]any)

	if !strings.HasPrefix(content, "---") {
		return fm, content
	}

	lines := strings.SplitN(content, "\n", -1)
	closingIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closingIdx = i
			break
		}
	}

	if closingIdx == -1 {
		return fm, content
	}

	for i := 1; i < closingIdx; i++ {
		line := strings.TrimSpace(lines[i])
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])
		fm[key] = value
	}

	body := strings.Join(lines[closingIdx+1:], "\n")
	return fm, body
}

func extractTemplateName(input string) string {
	input = strings.TrimSpace(input)

	// /template-name
	if strings.HasPrefix(input, "/") && !strings.Contains(input, " ") {
		return input[1:]
	}

	// template:template-name
	if strings.HasPrefix(input, "template:") {
		name := strings.TrimPrefix(input, "template:")
		if name != "" && !strings.Contains(name, " ") {
			return name
		}
	}

	return ""
}

// FormatTemplatesForPrompt generates a help string listing available templates.
func FormatTemplatesForPrompt(templates []PromptTemplate) string {
	if len(templates) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n# Prompt Templates\n\n")
	sb.WriteString("You can reference these prompt templates by name:\n\n")

	for _, tmpl := range templates {
		desc := tmpl.Description
		if desc == "" {
			desc = "(no description)"
		}
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", tmpl.Name, desc))
	}

	return sb.String()
}

// InjectTemplatesIntoContext creates a UserMessage from a template expansion
// and appends any images.
func InjectTemplatesIntoContext(text string, templates []PromptTemplate, images []ai.ImageContent) ai.UserMessage {
	expanded := NewPromptTemplateLoader().ExpandPromptTemplate(text, templates)

	content := []ai.Content{ai.TextContent{Text: expanded}}
	for _, img := range images {
		content = append(content, img)
	}

	return ai.UserMessage{
		Content:   content,
		Timestamp: 0, // Will be set by caller
	}
}


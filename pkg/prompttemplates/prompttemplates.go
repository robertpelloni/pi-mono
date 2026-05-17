package prompttemplates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/badlogic/pi-mono/pkg/frontmatter"
	"github.com/badlogic/pi-mono/pkg/sourceinfo"
)

// PromptTemplate represents a prompt template loaded from a markdown file.
type PromptTemplate struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Content     string                `json:"content"`
	SourceInfo  sourceinfo.SourceInfo `json:"sourceInfo"`
	FilePath    string                `json:"filePath"`
}

// LoadPromptTemplatesOptions configures template loading.
type LoadPromptTemplatesOptions struct {
	CWD           string
	AgentDir      string
	PromptPaths   []string
	IncludeDefaults bool
}

// LoadPromptTemplates loads all prompt templates from global, project, and explicit paths.
func LoadPromptTemplates(options LoadPromptTemplatesOptions) []PromptTemplate {
	if options.CWD == "" {
		options.CWD, _ = os.Getwd()
	}
	if options.AgentDir == "" {
		home, _ := os.UserHomeDir()
		options.AgentDir = filepath.Join(home, ".pi")
	}
	if !options.IncludeDefaults {
		options.IncludeDefaults = true
	}

	var templates []PromptTemplate

	if options.IncludeDefaults {
		globalPromptsDir := filepath.Join(options.AgentDir, "prompts")
		projectPromptsDir := filepath.Join(options.CWD, ".pi", "prompts")

		templates = append(templates, loadTemplatesFromDir(globalPromptsDir, "user", globalPromptsDir)...)
		templates = append(templates, loadTemplatesFromDir(projectPromptsDir, "project", projectPromptsDir)...)
	}

	// Load explicit prompt paths
	for _, rawPath := range options.PromptPaths {
		resolvedPath := resolvePromptPath(rawPath, options.CWD)
		if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
			continue
		}

		info, err := os.Stat(resolvedPath)
		if err != nil {
			continue
		}

		if info.IsDir() {
			templates = append(templates, loadTemplatesFromDir(resolvedPath, "explicit", resolvedPath)...)
		} else if strings.HasSuffix(resolvedPath, ".md") {
			tmpl := loadTemplateFromFile(resolvedPath, sourceinfo.SourceInfo{
				Source: "local",
				Scope:  "explicit",
				BaseDir: filepath.Dir(resolvedPath),
			})
			if tmpl != nil {
				templates = append(templates, *tmpl)
			}
		}
	}

	return templates
}

// loadTemplatesFromDir loads .md files from a directory as prompt templates.
func loadTemplatesFromDir(dir, scope, baseDir string) []PromptTemplate {
	var templates []PromptTemplate

	entries, err := os.ReadDir(dir)
	if err != nil {
		return templates
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			fullPath := filepath.Join(dir, entry.Name())
			tmpl := loadTemplateFromFile(fullPath, sourceinfo.SourceInfo{
				Source:  "local",
				Scope:   scope,
				BaseDir: baseDir,
			})
			if tmpl != nil {
				templates = append(templates, *tmpl)
			}
		}
	}

	return templates
}

// loadTemplateFromFile loads a prompt template from a markdown file.
func loadTemplateFromFile(filePath string, si sourceinfo.SourceInfo) *PromptTemplate {
	rawContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	fmResult := frontmatter.ParseFrontMatter(string(rawContent))
	name := strings.TrimSuffix(filepath.Base(filePath), ".md")

	// Get description from frontmatter or first non-empty line
	description := ""
	if fmResult.Fields != nil {
		if desc, ok := fmResult.Fields["description"].(string); ok {
			description = desc
		}
	}
	if description == "" {
		lines := strings.Split(fmResult.Content, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				description = trimmed
				if len(description) > 60 {
					description = description[:60] + "..."
				}
				break
			}
		}
	}

	return &PromptTemplate{
		Name:        name,
		Description: description,
		Content:     fmResult.Content,
		SourceInfo:  si,
		FilePath:    filePath,
	}
}

// resolvePromptPath resolves a prompt path relative to cwd.
func resolvePromptPath(p, cwd string) string {
	trimmed := strings.TrimSpace(p)
	if trimmed == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(trimmed, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, trimmed[2:])
	}
	if filepath.IsAbs(trimmed) {
		return trimmed
	}
	return filepath.Join(cwd, trimmed)
}

// ParseCommandArgs parses command arguments respecting quoted strings.
func ParseCommandArgs(argsString string) []string {
	var args []string
	var current strings.Builder
	inQuote := byte(0)

	for i := 0; i < len(argsString); i++ {
		char := argsString[i]
		if inQuote != 0 {
			if char == inQuote {
				inQuote = 0
			} else {
				current.WriteByte(char)
			}
		} else if char == '"' || char == '\'' {
			inQuote = char
		} else if char == ' ' || char == '\t' {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(char)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// SubstituteArgs replaces argument placeholders in template content.
// Supports: $1, $2, ... for positional args, $@ and $ARGUMENTS for all args,
// ${@:N} for args from Nth onwards, ${@:N:L} for L args starting from Nth.
func SubstituteArgs(content string, args []string) string {
	result := content

	// Replace $1, $2, etc. with positional args FIRST
	positionalRe := regexp.MustCompile(`\$(\d+)`)
	result = positionalRe.ReplaceAllStringFunc(result, func(match string) string {
		sub := positionalRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		index := 0
		fmt.Sscanf(sub[1], "%d", &index)
		index-- // Convert to 0-indexed
		if index >= 0 && index < len(args) {
			return args[index]
		}
		return ""
	})

	// Replace ${@:start} or ${@:start:length} with sliced args
	sliceRe := regexp.MustCompile(`\$\{@:(\d+)(?::(\d+))?\}`)
	result = sliceRe.ReplaceAllStringFunc(result, func(match string) string {
		sub := sliceRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		start := 0
		fmt.Sscanf(sub[1], "%d", &start)
		start-- // Convert to 0-indexed
		if start < 0 {
			start = 0
		}

		if len(sub) >= 3 && sub[2] != "" {
			length := 0
			fmt.Sscanf(sub[2], "%d", &length)
			if start+length <= len(args) {
				return strings.Join(args[start:start+length], " ")
			}
			return strings.Join(args[start:], " ")
		}
		return strings.Join(args[start:], " ")
	})

	// Replace $ARGUMENTS and $@ with all args joined
	allArgs := strings.Join(args, " ")
	result = strings.ReplaceAll(result, "$ARGUMENTS", allArgs)
	result = strings.ReplaceAll(result, "$@", allArgs)

	return result
}

// ExpandPromptTemplate expands a prompt template if it matches a template name.
// Returns the expanded content or the original text if not a template.
func ExpandPromptTemplate(text string, templates []PromptTemplate) string {
	if !strings.HasPrefix(text, "/") {
		return text
	}

	spaceIndex := strings.Index(text, " ")
	templateName := ""
	argsString := ""

	if spaceIndex == -1 {
		templateName = text[1:]
	} else {
		templateName = text[1:spaceIndex]
		argsString = text[spaceIndex+1:]
	}

	for _, tmpl := range templates {
		if tmpl.Name == templateName {
			args := ParseCommandArgs(argsString)
			return SubstituteArgs(tmpl.Content, args)
		}
	}

	return text
}

// Ensure fmt is used
var _ = fmt.Sprintf

// PromptTemplateDiagnostic represents a non-fatal issue loading a template.
type PromptTemplateDiagnostic struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// PromptTemplateLoadResult holds the result of loading prompt templates.
type PromptTemplateLoadResult struct {
	Templates   []PromptTemplate          `json:"templates"`
	Diagnostics []PromptTemplateDiagnostic `json:"diagnostics"`
}

// PromptTemplateLoader loads prompt templates from disk.
type PromptTemplateLoader struct{}

// NewPromptTemplateLoader creates a new prompt template loader.
func NewPromptTemplateLoader() *PromptTemplateLoader {
	return &PromptTemplateLoader{}
}

// Load loads prompt templates from agent and project directories.
func (l *PromptTemplateLoader) Load(agentDir, cwd string) PromptTemplateLoadResult {
	templates := LoadPromptTemplates(LoadPromptTemplatesOptions{
		CWD:           cwd,
		AgentDir:      agentDir,
		IncludeDefaults: true,
	})

	if templates == nil {
		templates = []PromptTemplate{}
	}

	return PromptTemplateLoadResult{
		Templates:   templates,
		Diagnostics: []PromptTemplateDiagnostic{},
	}
}

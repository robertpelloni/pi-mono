package systemprompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/badlogic/pi-mono/pkg/agent"
)

// BuildSystemPromptOptions configures how the system prompt is constructed.
type BuildSystemPromptOptions struct {
	// CustomPrompt replaces the default prompt entirely.
	CustomPrompt string
	// SelectedTools lists which tools to include. Default: [read, bash, edit, write]
	SelectedTools []string
	// ToolSnippets maps tool name → one-line description shown in the prompt.
	ToolSnippets map[string]string
	// PromptGuidelines are additional guideline bullets appended to defaults.
	PromptGuidelines []string
	// AppendSystemPrompt is text appended to the system prompt.
	AppendSystemPrompt string
	// CWD is the working directory.
	CWD string
	// ContextFiles are pre-loaded project context files.
	ContextFiles []ContextFile
	// Skills are loaded skill definitions to include in the prompt.
	Skills []SkillRef
}

// ContextFile is a project-level instruction file.
type ContextFile struct {
	Path    string
	Content string
}

// SkillRef is a lightweight reference to a skill for inclusion in the prompt.
type SkillRef struct {
	Name        string
	Description string
	FilePath    string
	Content     string
}

// BuildSystemPrompt constructs the full system prompt.
func BuildSystemPrompt(options BuildSystemPromptOptions) string {
	if options.CWD == "" {
		options.CWD, _ = os.Getwd()
	}
	promptCWD := filepath.ToSlash(options.CWD)
	date := time.Now().Format("2006-01-02")

	appendSection := ""
	if options.AppendSystemPrompt != "" {
		appendSection = "\n\n" + options.AppendSystemPrompt
	}

	// If custom prompt, use it as base
	if options.CustomPrompt != "" {
		prompt := options.CustomPrompt
		prompt += appendSection

		// Append project context files
		if len(options.ContextFiles) > 0 {
			prompt += "\n\n# Project Context\n\n"
			prompt += "Project-specific instructions and guidelines:\n\n"
			for _, cf := range options.ContextFiles {
				prompt += fmt.Sprintf("## %s\n\n%s\n\n", cf.Path, cf.Content)
			}
		}

		// Append skills
		if len(options.Skills) > 0 {
			prompt += formatSkillsForPrompt(options.Skills)
		}

		prompt += fmt.Sprintf("\nCurrent date: %s", date)
		prompt += fmt.Sprintf("\nCurrent working directory: %s", promptCWD)
		return prompt
	}

	// Default tools
	tools := options.SelectedTools
	if len(tools) == 0 {
		tools = []string{"read", "bash", "edit", "write", "ls", "grep", "find"}
	}

	// Build tools list with snippets
	visibleTools := filterVisibleTools(tools, options.ToolSnippets)
	toolsList := "(none)"
	if len(visibleTools) > 0 {
		lines := make([]string, len(visibleTools))
		for i, name := range visibleTools {
			snippet := options.ToolSnippets[name]
			lines[i] = fmt.Sprintf("- %s: %s", name, snippet)
		}
		toolsList = strings.Join(lines, "\n")
	}

	// Build guidelines
	guidelines := buildGuidelines(tools, options.PromptGuidelines)

	// Build the full prompt
	prompt := fmt.Sprintf(`You are an autonomous coding agent running in a terminal. You have access to the following tools:

%s

# Guidelines

%s`, toolsList, strings.Join(guidelines, "\n"))

	// Add context files
	if len(options.ContextFiles) > 0 {
		prompt += "\n\n# Project Context\n\n"
		prompt += "Project-specific instructions and guidelines:\n\n"
		for _, cf := range options.ContextFiles {
			prompt += fmt.Sprintf("## %s\n\n%s\n\n", cf.Path, cf.Content)
		}
	}

	// Add skills
	if len(options.Skills) > 0 {
		prompt += formatSkillsForPrompt(options.Skills)
	}

	prompt += appendSection
	prompt += fmt.Sprintf("\n\nCurrent date: %s", date)
	prompt += fmt.Sprintf("\nCurrent working directory: %s", promptCWD)

	return prompt
}

// DefaultToolSnippets returns the standard one-line descriptions for built-in tools.
func DefaultToolSnippets() map[string]string {
	return map[string]string{
		"read":  "Read the contents of a file (supports text and images, use offset/limit for large files)",
		"bash":  "Execute a bash command in the current working directory (with timeout)",
		"write": "Write content to a file (creates parent directories automatically)",
		"edit":  "Edit a single file using exact text replacement (non-overlapping targeted replacements)",
		"ls":    "List directory contents (sorted alphabetically, includes dotfiles)",
		"grep":  "Search file contents for a pattern (regex or literal, respects .gitignore)",
		"find":  "Search for files by glob pattern (respects .gitignore)",
	}
}

// BuildToolSnippetsFromAgentTools generates snippets from an agent's tool list.
func BuildToolSnippetsFromAgentTools(tools []agent.AgentTool) map[string]string {
	snippets := make(map[string]string, len(tools))
	for _, t := range tools {
		desc := t.Description
		if len(desc) > 120 {
			desc = desc[:117] + "..."
		}
		snippets[t.Name] = desc
	}
	return snippets
}

// LoadProjectContextFiles reads context files from the project directory.
// Looks for: .pi/instructions.md, CLAUDE.md, .cursorrules, .github/copilot-instructions.md
func LoadProjectContextFiles(cwd string) []ContextFile {
	var files []ContextFile

	candidates := []string{
		".pi/instructions.md",
		"CLAUDE.md",
		".cursorrules",
		".github/copilot-instructions.md",
	}

	for _, candidate := range candidates {
		path := filepath.Join(cwd, candidate)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}
		files = append(files, ContextFile{
			Path:    candidate,
			Content: content,
		})
	}

	return files
}

// --- Internal helpers ---

func filterVisibleTools(tools []string, snippets map[string]string) []string {
	if len(snippets) == 0 {
		return tools
	}
	var visible []string
	for _, name := range tools {
		if _, ok := snippets[name]; ok {
			visible = append(visible, name)
		}
	}
	return visible
}

func buildGuidelines(tools []string, extra []string) []string {
	toolSet := make(map[string]bool, len(tools))
	for _, t := range tools {
		toolSet[t] = true
	}

	guidelines := []string{}
	seen := make(map[string]bool)
	add := func(g string) {
		if !seen[g] {
			seen[g] = true
			guidelines = append(guidelines, g)
		}
	}

	// Core guidelines
	add("- Think carefully before acting. Plan your approach, then execute step by step.")
	add("- Always verify your changes by reading the file after editing or writing.")
	add("- Use tools whenever possible rather than guessing or making assumptions.")
	add("- When reading files, prefer the read tool. For large files, use offset/limit to read in chunks.")
	add("- When running commands, use the bash tool. Prefer simple, targeted commands.")

	if toolSet["bash"] {
		add("- Never execute commands that could destroy data (rm -rf /, etc.) without explicit confirmation.")
		add("- Check command output for errors before proceeding.")
	}
	if toolSet["edit"] {
		add("- When editing files, use the edit tool with exact text matching. Include enough context to make the match unique.")
		add("- Do not include large unchanged regions in your edit — only the specific lines that need to change.")
	}
	if toolSet["write"] {
		add("- When creating new files, use the write tool. It automatically creates parent directories.")
	}
	if toolSet["read"] {
		add("- Before editing a file, read it first to understand its current content and structure.")
	}
	if toolSet["grep"] {
		add("- When searching for patterns across files, use grep. It respects .gitignore rules.")
	}
	if toolSet["find"] {
		add("- When searching for files by name or pattern, use the find tool.")
	}

	add("- When a task is complete, provide a concise summary of what was done.")
	add("- If you encounter an error, read the error message carefully and try to fix it before giving up.")

	// Add extra guidelines
	for _, g := range extra {
		add(g)
	}

	return guidelines
}

func formatSkillsForPrompt(skills []SkillRef) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n# Skills\n\n")
	sb.WriteString("The following skills are available. Skills are markdown files that define reusable workflows:\n\n")

	for _, skill := range skills {
		sb.WriteString(fmt.Sprintf("## %s\n\n", skill.Name))
		if skill.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", skill.Description))
		}
		if skill.Content != "" {
			// Truncate very long skill content
			content := skill.Content
			if len(content) > 2000 {
				content = content[:1997] + "..."
			}
			sb.WriteString(content)
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}

package systemprompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSystemPromptDefault(t *testing.T) {
	prompt := BuildSystemPrompt(BuildSystemPromptOptions{
		CWD: "/home/user/project",
	})

	if !strings.Contains(prompt, "autonomous coding agent") {
		t.Error("default prompt should contain 'autonomous coding agent'")
	}
	if !strings.Contains(prompt, "/home/user/project") {
		t.Error("prompt should contain the working directory")
	}
	if !strings.Contains(prompt, "Current date:") {
		t.Error("prompt should contain current date")
	}
}

func TestBuildSystemPromptCustom(t *testing.T) {
	prompt := BuildSystemPrompt(BuildSystemPromptOptions{
		CustomPrompt: "You are a helpful assistant.",
		CWD:         "/test",
	})

	if !strings.Contains(prompt, "You are a helpful assistant.") {
		t.Error("custom prompt should be used as base")
	}
	if !strings.Contains(prompt, "/test") {
		t.Error("custom prompt should have CWD appended")
	}
}

func TestBuildSystemPromptWithTools(t *testing.T) {
	prompt := BuildSystemPrompt(BuildSystemPromptOptions{
		SelectedTools: []string{"read", "bash", "write"},
		ToolSnippets: map[string]string{
			"read":  "Read file contents",
			"bash":  "Execute commands",
			"write": "Write to files",
		},
		CWD: "/test",
	})

	if !strings.Contains(prompt, "read: Read file contents") {
		t.Error("prompt should list tool snippets")
	}
	if !strings.Contains(prompt, "bash: Execute commands") {
		t.Error("prompt should list bash snippet")
	}
}

func TestBuildSystemPromptWithContextFiles(t *testing.T) {
	prompt := BuildSystemPrompt(BuildSystemPromptOptions{
		ContextFiles: []ContextFile{
			{Path: "CLAUDE.md", Content: "Always use TypeScript strict mode."},
		},
		CWD: "/test",
	})

	if !strings.Contains(prompt, "Project Context") {
		t.Error("prompt should have project context section")
	}
	if !strings.Contains(prompt, "Always use TypeScript strict mode.") {
		t.Error("prompt should include context file content")
	}
}

func TestBuildSystemPromptWithSkills(t *testing.T) {
	prompt := BuildSystemPrompt(BuildSystemPromptOptions{
		Skills: []SkillRef{
			{Name: "debug", Description: "Debug the codebase", Content: "Steps to debug..."},
		},
		CWD: "/test",
	})

	if !strings.Contains(prompt, "# Skills") {
		t.Error("prompt should have skills section")
	}
	if !strings.Contains(prompt, "debug") {
		t.Error("prompt should include skill name")
	}
}

func TestBuildSystemPromptAppend(t *testing.T) {
	prompt := BuildSystemPrompt(BuildSystemPromptOptions{
		AppendSystemPrompt: "Extra instructions here.",
		CWD:                "/test",
	})

	if !strings.Contains(prompt, "Extra instructions here.") {
		t.Error("prompt should include appended text")
	}
}

func TestBuildSystemPromptGuidelines(t *testing.T) {
	prompt := BuildSystemPrompt(BuildSystemPromptOptions{
		SelectedTools: []string{"read", "bash", "edit"},
		CWD:          "/test",
	})

	if !strings.Contains(prompt, "Guidelines") {
		t.Error("prompt should have guidelines section")
	}
	if !strings.Contains(prompt, "Think carefully") {
		t.Error("prompt should have core guidelines")
	}
}

func TestLoadProjectContextFiles(t *testing.T) {
	// Create temp directory with context files
	tmpDir := t.TempDir()

	// Create .pi/instructions.md
	os.MkdirAll(filepath.Join(tmpDir, ".pi"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".pi", "instructions.md"), []byte("Use Go best practices."), 0644)

	// Create CLAUDE.md
	os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte("Always test your code."), 0644)

	files := LoadProjectContextFiles(tmpDir)

	if len(files) < 1 {
		t.Fatalf("expected at least 1 context file, got %d", len(files))
	}

	found := false
	for _, f := range files {
		if f.Path == "CLAUDE.md" && strings.Contains(f.Content, "Always test your code.") {
			found = true
		}
	}
	if !found {
		t.Error("CLAUDE.md should be loaded")
	}
}

func TestDefaultToolSnippets(t *testing.T) {
	snippets := DefaultToolSnippets()

	requiredTools := []string{"read", "bash", "write", "edit", "ls", "grep", "find"}
	for _, tool := range requiredTools {
		if _, ok := snippets[tool]; !ok {
			t.Errorf("missing snippet for tool: %s", tool)
		}
	}
}

func TestBuildToolSnippetsFromAgentTools(t *testing.T) {
	snippets := BuildToolSnippetsFromAgentTools(nil)
	if len(snippets) != 0 {
		t.Error("empty tools should produce empty snippets")
	}
}

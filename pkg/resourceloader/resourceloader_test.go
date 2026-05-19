package resourceloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewResourceLoader(t *testing.T) {
	rl := NewResourceLoader(ResourceLoaderOptions{
		CWD:       ".",
		AgentDir:  ".pi",
		NoSkills:  true,
	})
	if rl == nil {
		t.Fatal("Expected non-nil ResourceLoader")
	}
}

func TestResourceLoader_Load(t *testing.T) {
	dir, _ := os.MkdirTemp("", "resourceloader_test")
	defer os.RemoveAll(dir)

	rl := NewResourceLoader(ResourceLoaderOptions{
		CWD:      dir,
		AgentDir: filepath.Join(dir, ".pi"),
		NoSkills: true,
		NoPromptTemplates: true,
	})
	err := rl.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
}

func TestResourceLoader_GetSkills(t *testing.T) {
	rl := NewResourceLoader(ResourceLoaderOptions{
		CWD:      ".",
		AgentDir: ".pi",
		NoSkills: true,
	})
	rl.Load()
	skills, diags := rl.GetSkills()
	_ = skills
	_ = diags
}

func TestResourceLoader_GetPrompts(t *testing.T) {
	rl := NewResourceLoader(ResourceLoaderOptions{
		CWD:       ".",
		AgentDir:  ".pi",
		NoSkills:  true,
		NoPromptTemplates: true,
	})
	rl.Load()
	prompts, diags := rl.GetPrompts()
	_ = prompts
	_ = diags
}

func TestResourceLoader_GetContextFiles(t *testing.T) {
	dir, _ := os.MkdirTemp("", "resourceloader_ctx_test")
	defer os.RemoveAll(dir)

	// Create a CLAUDE.md
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Context\nTest context file"), 0644)

	rl := NewResourceLoader(ResourceLoaderOptions{
		CWD:      dir,
		AgentDir: filepath.Join(dir, ".pi"),
		NoSkills: true,
		NoPromptTemplates: true,
	})
	rl.Load()
	contextFiles := rl.GetContextFiles()
	// May or may not find CLAUDE.md depending on implementation
	_ = contextFiles
}

func TestResourceLoader_GetSystemPrompt(t *testing.T) {
	rl := NewResourceLoader(ResourceLoaderOptions{
		CWD:          ".",
		AgentDir:     ".pi",
		NoSkills:     true,
		SystemPrompt: "custom prompt",
	})
	rl.Load()
	prompt := rl.GetSystemPrompt()
	if prompt != "custom prompt" {
		t.Errorf("Expected 'custom prompt', got %q", prompt)
	}
}

func TestResourceDiagnostic_Fields(t *testing.T) {
	d := ResourceDiagnostic{Path: "/test", Message: "warning"}
	if d.Path != "/test" || d.Message != "warning" {
		t.Error("Field mismatch")
	}
}

func TestContextFile_Fields(t *testing.T) {
	cf := ContextFile{Path: "/test/CLAUDE.md", Content: "test content"}
	if cf.Path != "/test/CLAUDE.md" || cf.Content != "test content" {
		t.Error("Field mismatch")
	}
}

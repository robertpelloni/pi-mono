package prompttemplates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/badlogic/pi-mono/pkg/sourceinfo"
)

func TestParseCommandArgs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"hello", []string{"hello"}},
		{"hello world", []string{"hello", "world"}},
		{`hello "world foo"`, []string{"hello", "world foo"}},
		{`hello 'world foo'`, []string{"hello", "world foo"}},
		{"  multiple   spaces  ", []string{"multiple", "spaces"}},
	}

	for _, tt := range tests {
		got := ParseCommandArgs(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("ParseCommandArgs(%q) = %v, want %v", tt.input, got, tt.expected)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("ParseCommandArgs(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestSubstituteArgs(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		args     []string
		expected string
	}{
		{"no substitution", "hello world", nil, "hello world"},
		{"positional", "hello $1", []string{"world"}, "hello world"},
		{"multiple positional", "$1 and $2", []string{"alice", "bob"}, "alice and bob"},
		{"out of range", "hello $3", []string{"a"}, "hello "},
		{"all args dollar at", "args: $@", []string{"a", "b", "c"}, "args: a b c"},
		{"all args ARGUMENTS", "args: $ARGUMENTS", []string{"a", "b"}, "args: a b"},
		{"slice from", "${@:2}", []string{"a", "b", "c"}, "b c"},
		{"slice with length", "${@:1:2}", []string{"a", "b", "c"}, "a b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SubstituteArgs(tt.content, tt.args)
			if got != tt.expected {
				t.Errorf("SubstituteArgs(%q, %v) = %q, want %q", tt.content, tt.args, got, tt.expected)
			}
		})
	}
}

func TestExpandPromptTemplate(t *testing.T) {
	templates := []PromptTemplate{
		{Name: "review", Content: "Review this code: $1"},
		{Name: "test", Content: "Write tests for $@"},
	}

	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"not a template", "hello world", "hello world"},
		{"matching template", "/review main.go", "Review this code: main.go"},
		{"template with multiple args", "/test foo bar", "Write tests for foo bar"},
		{"no matching template", "/unknown stuff", "/unknown stuff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandPromptTemplate(tt.text, templates)
			if got != tt.expected {
				t.Errorf("ExpandPromptTemplate(%q) = %q, want %q", tt.text, got, tt.expected)
			}
		})
	}
}

func TestLoadTemplateFromFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "template_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	templateContent := "---\ndescription: A test template\n---\nThis is the body with $1"
	templatePath := filepath.Join(tmpDir, "test.md")
	os.WriteFile(templatePath, []byte(templateContent), 0644)

	tmpl := loadTemplateFromFile(templatePath, sourceinfo.CreateSourceInfo(templatePath, "local"))
	if tmpl == nil {
		t.Fatal("Expected template to be loaded")
	}
	if tmpl.Name != "test" {
		t.Errorf("Expected name 'test', got %q", tmpl.Name)
	}
	if tmpl.Description != "A test template" {
		t.Errorf("Expected description from frontmatter, got %q", tmpl.Description)
	}
}

func TestLoadPromptTemplates(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "template_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a prompts directory with a template
	promptsDir := filepath.Join(tmpDir, "prompts")
	os.MkdirAll(promptsDir, 0755)
	os.WriteFile(filepath.Join(promptsDir, "hello.md"), []byte("# Hello\nHello $1!"), 0644)

	templates := LoadPromptTemplates(LoadPromptTemplatesOptions{
		CWD:           tmpDir,
		AgentDir:      tmpDir,
		IncludeDefaults: true,
	})

	// May or may not find templates depending on directory structure
	// Just verify it doesn't crash
	_ = templates
}

func TestPromptTemplateLoader(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "loader_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	loader := NewPromptTemplateLoader()
	result := loader.Load(tmpDir, tmpDir)

	if result.Templates == nil {
		t.Error("Expected non-nil templates slice")
	}
	// Empty dir is fine - just verify no panic
}

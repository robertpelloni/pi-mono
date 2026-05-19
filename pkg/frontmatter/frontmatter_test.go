package frontmatter

import (
	"testing"
)

func TestParseFrontMatter_WithFrontMatter(t *testing.T) {
	content := `---
name: test
version: 1.0
---
# Hello World`

	fm := ParseFrontMatter(content)
	if fm == nil {
		t.Fatal("Expected non-nil FrontMatter")
	}
	if fm.Content != "# Hello World" {
		t.Errorf("Expected '# Hello World', got %q", fm.Content)
	}
	if fm.Fields["name"] != "test" {
		t.Errorf("Expected name='test', got %v", fm.Fields["name"])
	}
	if fm.Fields["version"] != "1.0" {
		t.Errorf("Expected version='1.0', got %v", fm.Fields["version"])
	}
}

func TestParseFrontMatter_NoFrontMatter(t *testing.T) {
	content := "# Hello World"
	fm := ParseFrontMatter(content)
	if fm.Content != "# Hello World" {
		t.Errorf("Expected full content, got %q", fm.Content)
	}
	if len(fm.Fields) != 0 {
		t.Errorf("Expected no fields, got %v", fm.Fields)
	}
}

func TestParseFrontMatter_Comments(t *testing.T) {
	content := `---
# This is a comment
name: test
---
Body`

	fm := ParseFrontMatter(content)
	if fm.Fields["name"] != "test" {
		t.Errorf("Expected name='test', got %v", fm.Fields["name"])
	}
	if _, ok := fm.Fields["#"]; ok {
		t.Error("Comments should not be parsed as fields")
	}
}

func TestHasFrontMatter(t *testing.T) {
	if !HasFrontMatter("---\nname: test\n---\nBody") {
		t.Error("Expected HasFrontMatter=true")
	}
	if HasFrontMatter("# No frontmatter") {
		t.Error("Expected HasFrontMatter=false")
	}
	if !HasFrontMatter("  ---\nname: test\n---\nBody") {
		t.Error("Expected HasFrontMatter=true with leading whitespace")
	}
}

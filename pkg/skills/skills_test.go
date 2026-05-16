package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSkillsEmpty(t *testing.T) {
	loader := NewSkillLoader()
	result := loader.LoadSkills(t.TempDir(), t.TempDir())

	if len(result.Skills) != 0 {
		t.Errorf("expected 0 skills from empty dirs, got %d", len(result.Skills))
	}
}

func TestLoadSkillsFromDir(t *testing.T) {
	// Create a skill directory
	skillDir := t.TempDir()
	globalDir := filepath.Join(skillDir, "skills")
	os.MkdirAll(globalDir, 0755)

	// Create a skill: my-skill/my-skill.md
	skillPath := filepath.Join(globalDir, "my-skill")
	os.MkdirAll(skillPath, 0755)
	os.WriteFile(filepath.Join(skillPath, "my-skill.md"), []byte(`---
name: my-skill
description: A test skill
disable-model-invocation: false
---
# My Skill

This is the skill content.
`), 0644)

	loader := NewSkillLoader()
	result := loader.LoadSkills(skillDir, t.TempDir())

	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result.Skills))
	}

	skill := result.Skills[0]
	if skill.Name != "my-skill" {
		t.Errorf("expected name 'my-skill', got '%s'", skill.Name)
	}
	if skill.Description != "A test skill" {
		t.Errorf("expected description 'A test skill', got '%s'", skill.Description)
	}
	if !strings.Contains(skill.Content, "This is the skill content") {
		t.Errorf("expected content to contain skill body, got '%s'", skill.Content)
	}
	if skill.DisableModelInvocation {
		t.Error("disable-model-invocation should be false")
	}
}

func TestLoadSkillsProjectDir(t *testing.T) {
	// Create project skill directory
	projectDir := t.TempDir()
	skillDir := filepath.Join(projectDir, ".pi", "skills")
	os.MkdirAll(skillDir, 0755)

	skillPath := filepath.Join(skillDir, "project-skill")
	os.MkdirAll(skillPath, 0755)
	os.WriteFile(filepath.Join(skillPath, "project-skill.md"), []byte(`---
name: project-skill
description: A project-specific skill
---
Project skill content.
`), 0644)

	loader := NewSkillLoader()
	result := loader.LoadSkills(t.TempDir(), projectDir)

	if len(result.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(result.Skills))
	}
	if result.Skills[0].Name != "project-skill" {
		t.Errorf("expected name 'project-skill', got '%s'", result.Skills[0].Name)
	}
}

func TestLoadSkillsNoMarkdownFile(t *testing.T) {
	skillDir := t.TempDir()
	globalDir := filepath.Join(skillDir, "skills")
	os.MkdirAll(globalDir, 0755)

	// Create a skill dir without a markdown file
	skillPath := filepath.Join(globalDir, "broken-skill")
	os.MkdirAll(skillPath, 0755)

	loader := NewSkillLoader()
	result := loader.LoadSkills(skillDir, t.TempDir())

	if len(result.Diagnostics) == 0 {
		t.Error("expected a diagnostic for missing markdown file")
	}
}

func TestToSkillRefs(t *testing.T) {
	skills := []Skill{
		{Name: "test", Description: "Test skill", Content: "Content"},
	}
	refs := ToSkillRefs(skills)

	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].Name != "test" {
		t.Errorf("expected name 'test', got '%s'", refs[0].Name)
	}
}

func TestParseFrontmatter(t *testing.T) {
	content := `---
name: my-skill
description: A test
disable-model-invocation: true
---
Body content here.`

	fm, body := parseFrontmatter(content)

	if fm["name"] != "my-skill" {
		t.Errorf("expected name 'my-skill', got '%v'", fm["name"])
	}
	if fm["description"] != "A test" {
		t.Errorf("expected description 'A test', got '%v'", fm["description"])
	}
	if fm["disable-model-invocation"] != true {
		t.Errorf("expected disable-model-invocation true, got '%v'", fm["disable-model-invocation"])
	}
	if !strings.Contains(body, "Body content here.") {
		t.Errorf("expected body to contain 'Body content here.', got '%s'", body)
	}
}

func TestParseFrontmatterNoFrontmatter(t *testing.T) {
	content := "Just some content without frontmatter."
	fm, body := parseFrontmatter(content)

	if len(fm) != 0 {
		t.Errorf("expected empty frontmatter, got %v", fm)
	}
	if body != content {
		t.Errorf("expected body to equal content")
	}
}

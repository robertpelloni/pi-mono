package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/badlogic/pi-mono/pkg/systemprompt"
)

const (
	maxNameLength        = 64
	maxDescriptionLength = 1024
	configDirName        = ".pi"
)

// Skill represents a loaded skill definition.
type Skill struct {
	Name                string `json:"name"`
	Description         string `json:"description"`
	FilePath            string `json:"filePath"`
	BaseDir             string `json:"baseDir"`
	DisableModelInvocation bool `json:"disableModelInvocation"`
	Content             string `json:"-"`
}

// LoadSkillsResult contains the loaded skills and any diagnostics.
type LoadSkillsResult struct {
	Skills      []Skill            `json:"skills"`
	Diagnostics []SkillDiagnostic  `json:"diagnostics"`
}

// SkillDiagnostic represents a non-fatal issue loading a skill.
type SkillDiagnostic struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// SkillLoader discovers and loads skill definitions from the filesystem.
type SkillLoader struct {
	mu    sync.RWMutex
	cache map[string]*LoadSkillsResult
}

// NewSkillLoader creates a new skill loader.
func NewSkillLoader() *SkillLoader {
	return &SkillLoader{
		cache: make(map[string]*LoadSkillsResult),
	}
}

// LoadSkills discovers skills from the agent directory and project directory.
// Searches in:
//   - ~/.pi/skills/  (global skills)
//   - .pi/skills/    (project skills)
//
// Each skill is a directory containing a .md file with frontmatter:
//
//	---
//	name: my-skill
//	description: Does something useful
//	disable-model-invocation: false
//	---
func (sl *SkillLoader) LoadSkills(agentDir, projectDir string) *LoadSkillsResult {
	key := agentDir + ":" + projectDir
	sl.mu.RLock()
	if cached, ok := sl.cache[key]; ok {
		sl.mu.RUnlock()
		return cached
	}
	sl.mu.RUnlock()

	result := &LoadSkillsResult{}

	// Load global skills
	sl.loadFromDir(filepath.Join(agentDir, "skills"), "global", result)

	// Load project skills
	sl.loadFromDir(filepath.Join(projectDir, configDirName, "skills"), "project", result)

	sl.mu.Lock()
	sl.cache[key] = result
	sl.mu.Unlock()

	return result
}

// ToSkillRefs converts loaded skills to SkillRef for the system prompt builder.
func ToSkillRefs(skills []Skill) []systemprompt.SkillRef {
	refs := make([]systemprompt.SkillRef, len(skills))
	for i, s := range skills {
		refs[i] = systemprompt.SkillRef{
			Name:        s.Name,
			Description: s.Description,
			FilePath:    s.FilePath,
			Content:     s.Content,
		}
	}
	return refs
}

// loadFromDir scans a directory for skill definitions.
func (sl *SkillLoader) loadFromDir(dir, scope string, result *LoadSkillsResult) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return // Directory doesn't exist, that's fine
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillDir := filepath.Join(dir, entry.Name())
		sl.loadSingleSkill(skillDir, entry.Name(), scope, result)
	}
}

// loadSingleSkill loads a single skill from a directory.
// The skill directory should contain a .md file with the same name as the directory,
// or any .md file if only one exists.
func (sl *SkillLoader) loadSingleSkill(dir, dirName, scope string, result *LoadSkillsResult) {
	// Look for markdown file
	mdFile := ""
	candidates := []string{
		filepath.Join(dir, dirName+".md"),
	}

	// Also check for any .md file
	entries, err := os.ReadDir(dir)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, SkillDiagnostic{
			Path:    dir,
			Message: fmt.Sprintf("cannot read skill directory: %v", err),
		})
		return
	}

	mdCount := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			mdCount++
			candidates = append(candidates, filepath.Join(dir, e.Name()))
		}
	}

	// Try candidates
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		mdFile = candidate
		content := string(data)

		// Parse frontmatter
		frontmatter, body := parseFrontmatter(content)

		name := dirName
		if n, ok := frontmatter["name"].(string); ok && n != "" {
			name = n
		}

		description := ""
		if d, ok := frontmatter["description"].(string); ok {
			description = d
		}

		disableModelInvocation := false
		if d, ok := frontmatter["disable-model-invocation"].(bool); ok {
			disableModelInvocation = d
		}

		// Validate
		if len(name) > maxNameLength {
			result.Diagnostics = append(result.Diagnostics, SkillDiagnostic{
				Path:    mdFile,
				Message: fmt.Sprintf("name exceeds %d characters (%d)", maxNameLength, len(name)),
			})
			name = name[:maxNameLength]
		}
		if len(description) > maxDescriptionLength {
			result.Diagnostics = append(result.Diagnostics, SkillDiagnostic{
				Path:    mdFile,
				Message: fmt.Sprintf("description exceeds %d characters (%d)", maxDescriptionLength, len(description)),
			})
			description = description[:maxDescriptionLength]
		}

		skill := Skill{
			Name:                name,
			Description:         description,
			FilePath:            mdFile,
			BaseDir:             dir,
			DisableModelInvocation: disableModelInvocation,
			Content:             strings.TrimSpace(body),
		}

		result.Skills = append(result.Skills, skill)
		return
	}

	if mdCount == 0 {
		result.Diagnostics = append(result.Diagnostics, SkillDiagnostic{
			Path:    dir,
			Message: "no markdown file found in skill directory",
		})
	}
}

// parseFrontmatter extracts YAML-like frontmatter from a markdown string.
// Returns a map of key-value pairs and the body after the frontmatter.
//
// Format:
//
//	---
//	key: value
//	another-key: true
//	---
//	Body content here
func parseFrontmatter(content string) (map[string]any, string) {
	frontmatter := make(map[string]any)

	if !strings.HasPrefix(content, "---") {
		return frontmatter, content
	}

	// Find closing ---
	lines := strings.SplitN(content, "\n", -1)
	closingIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closingIdx = i
			break
		}
	}

	if closingIdx == -1 {
		return frontmatter, content
	}

	// Parse key: value pairs
	for i := 1; i < closingIdx; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])

		// Type coercion
		switch strings.ToLower(value) {
		case "true":
			frontmatter[key] = true
		case "false":
			frontmatter[key] = false
		default:
			frontmatter[key] = value
		}
	}

	body := strings.Join(lines[closingIdx+1:], "\n")
	return frontmatter, body
}

// ClearCache clears the skill loader's cache.
func (sl *SkillLoader) ClearCache() {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.cache = make(map[string]*LoadSkillsResult)
}

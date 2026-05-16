package resourceloader

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/badlogic/pi-mono/pkg/prompttemplates"
	"github.com/badlogic/pi-mono/pkg/skills"
)

// ResourceDiagnostic represents a non-fatal issue loading a resource.
type ResourceDiagnostic struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// ContextFile is a project context file (e.g., CLAUDE.md, AGENTS.md).
type ContextFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// ResourceLoader discovers and loads all agent resources:
// - Skills from ~/.pi/skills/ and .pi/skills/
// - Prompt templates from ~/.pi/prompts/ and .pi/prompts/
// - Context files (CLAUDE.md, AGENTS.md) from project directories
// - System prompt overrides
// - Append system prompt entries
type ResourceLoader struct {
	mu sync.RWMutex

	cwd       string
	agentDir  string

	// Configuration
	noSkills         bool
	noPromptTemplates bool
	systemPromptSrc  string
	appendSystemPromptSrc string

	// Cached resources
	skills          []skills.Skill
	skillDiags      []ResourceDiagnostic
	prompts         []prompttemplates.PromptTemplate
	promptDiags     []ResourceDiagnostic
	contextFiles    []ContextFile
	agentsFiles     []ContextFile
	systemPrompt    string
	appendSystemPrompt []string
}

// ResourceLoaderOptions configures the resource loader.
type ResourceLoaderOptions struct {
	CWD                string
	AgentDir           string
	NoSkills           bool
	NoPromptTemplates  bool
	SystemPrompt       string
	AppendSystemPrompt string
}

// NewResourceLoader creates a new resource loader.
func NewResourceLoader(opts ResourceLoaderOptions) *ResourceLoader {
	agentDir := opts.AgentDir
	if agentDir == "" {
		home, _ := os.UserHomeDir()
		agentDir = filepath.Join(home, ".pi")
	}
	cwd := opts.CWD
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	return &ResourceLoader{
		cwd:                 cwd,
		agentDir:            agentDir,
		noSkills:            opts.NoSkills,
		noPromptTemplates:   opts.NoPromptTemplates,
		systemPromptSrc:     opts.SystemPrompt,
		appendSystemPromptSrc: opts.AppendSystemPrompt,
	}
}

// Load discovers and caches all resources.
func (r *ResourceLoader) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load skills
	if !r.noSkills {
		loader := skills.NewSkillLoader()
		result := loader.LoadSkills(r.agentDir, r.cwd)
		r.skills = result.Skills
		r.skillDiags = make([]ResourceDiagnostic, len(result.Diagnostics))
		for i, d := range result.Diagnostics {
			r.skillDiags[i] = ResourceDiagnostic{Path: d.Path, Message: d.Message}
		}
	}

	// Load prompt templates
	if !r.noPromptTemplates {
		loader := prompttemplates.NewPromptTemplateLoader()
		result := loader.Load(r.agentDir, r.cwd)
		r.prompts = result.Templates
		r.promptDiags = make([]ResourceDiagnostic, len(result.Diagnostics))
		for i, d := range result.Diagnostics {
			r.promptDiags[i] = ResourceDiagnostic{Path: d.Path, Message: d.Message}
		}
	}

	// Load context files
	r.contextFiles = loadProjectContextFiles(r.cwd, r.agentDir)

	// Load agents files (.pi/agents/*.md)
	r.agentsFiles = loadAgentsFiles(r.cwd, r.agentDir)

	// Resolve system prompt
	r.systemPrompt = resolvePromptInput(r.systemPromptSrc, "system prompt")

	// Resolve append system prompt
	r.appendSystemPrompt = []string{}
	if r.appendSystemPromptSrc != "" {
		resolved := resolvePromptInput(r.appendSystemPromptSrc, "append system prompt")
		if resolved != "" {
			r.appendSystemPrompt = []string{resolved}
		}
	}

	return nil
}

// GetSkills returns the loaded skills.
func (r *ResourceLoader) GetSkills() ([]skills.Skill, []ResourceDiagnostic) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sk := make([]skills.Skill, len(r.skills))
	copy(sk, r.skills)
	return sk, r.skillDiags
}

// GetPrompts returns the loaded prompt templates.
func (r *ResourceLoader) GetPrompts() ([]prompttemplates.PromptTemplate, []ResourceDiagnostic) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p := make([]prompttemplates.PromptTemplate, len(r.prompts))
	copy(p, r.prompts)
	return p, r.promptDiags
}

// GetContextFiles returns the loaded context files.
func (r *ResourceLoader) GetContextFiles() []ContextFile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cf := make([]ContextFile, len(r.contextFiles))
	copy(cf, r.contextFiles)
	return cf
}

// GetAgentsFiles returns the loaded agents files.
func (r *ResourceLoader) GetAgentsFiles() []ContextFile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	af := make([]ContextFile, len(r.agentsFiles))
	copy(af, r.agentsFiles)
	return af
}

// GetSystemPrompt returns the system prompt override.
func (r *ResourceLoader) GetSystemPrompt() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.systemPrompt
}

// GetAppendSystemPrompt returns append system prompt entries.
func (r *ResourceLoader) GetAppendSystemPrompt() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.appendSystemPrompt
}

// --- Internal helpers ---

// loadContextFileFromDir looks for CLAUDE.md or AGENTS.md in a directory.
func loadContextFileFromDir(dir string) *ContextFile {
	candidates := []string{"AGENTS.md", "CLAUDE.md"}
	for _, filename := range candidates {
		filePath := filepath.Join(dir, filename)
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		return &ContextFile{
			Path:    filePath,
			Content: string(data),
		}
	}
	return nil
}

// loadProjectContextFiles discovers context files from the project tree.
func loadProjectContextFiles(cwd, agentDir string) []ContextFile {
	var contextFiles []ContextFile
	seenPaths := make(map[string]bool)

	// Global context from agent directory
	globalContext := loadContextFileFromDir(agentDir)
	if globalContext != nil {
		contextFiles = append(contextFiles, *globalContext)
		seenPaths[globalContext.Path] = true
	}

	// Walk from root to cwd, collecting context files
	var ancestorFiles []ContextFile
	currentDir := cwd
	for {
		ctxFile := loadContextFileFromDir(currentDir)
		if ctxFile != nil && !seenPaths[ctxFile.Path] {
			ancestorFiles = append([]ContextFile{*ctxFile}, ancestorFiles...)
			seenPaths[ctxFile.Path] = true
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}
	contextFiles = append(contextFiles, ancestorFiles...)

	return contextFiles
}

// loadAgentsFiles discovers .pi/agents/*.md files.
func loadAgentsFiles(cwd, agentDir string) []ContextFile {
	var files []ContextFile

	// Global agents files
	globalDir := filepath.Join(agentDir, "agents")
	loadAgentsFromDir(globalDir, &files)

	// Project agents files
	projectDir := filepath.Join(cwd, ".pi", "agents")
	loadAgentsFromDir(projectDir, &files)

	return files
}

func loadAgentsFromDir(dir string, files *[]ContextFile) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		*files = append(*files, ContextFile{
			Path:    filePath,
			Content: string(data),
		})
	}
}

// resolvePromptInput resolves a prompt input that may be a file path or literal text.
func resolvePromptInput(input, description string) string {
	if input == "" {
		return ""
	}
	// Try reading as file
	data, err := os.ReadFile(input)
	if err == nil {
		return string(data)
	}
	// Return as literal text
	return input
}

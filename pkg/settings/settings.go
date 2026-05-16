package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Settings holds all user-configurable preferences.
type Settings struct {
	// Model
	DefaultProvider string `json:"defaultProvider,omitempty"`
	DefaultModel    string `json:"defaultModel,omitempty"`
	EnabledModels   []string `json:"enabledModels,omitempty"`

	// UI
	Theme         string `json:"theme,omitempty"`
	ShowCursor    *bool  `json:"showHardwareCursor,omitempty"`
	ClearOnShrink *bool  `json:"clearOnShrink,omitempty"`
	QuietStartup  *bool  `json:"quietStartup,omitempty"`

	// Session
	SessionDir string `json:"sessionDir,omitempty"`

	// Tools
	ImageAutoResize *bool `json:"imageAutoResize,omitempty"`

	// Extensions
	NoExtensions    *bool    `json:"noExtensions,omitempty"`
	NoSkills        *bool    `json:"noSkills,omitempty"`
	NoPromptTemplates *bool  `json:"noPromptTemplates,omitempty"`
	NoThemes        *bool    `json:"noThemes,omitempty"`

	// Custom
	Custom map[string]any `json:"custom,omitempty"`
}

// SettingsManager loads and persists settings from multiple scopes:
// 1. Global: ~/.pi/settings.json
// 2. Project: .pi/settings.json (in the project root)
// Merged with project overriding global.
type SettingsManager struct {
	mu       sync.RWMutex
	cwd      string
	agentDir string
	global   Settings
	project  Settings
	merged   Settings
	errors   []SettingsError
}

// SettingsError represents a non-fatal issue loading settings.
type SettingsError struct {
	Scope string
	Error error
}

// Create initializes a SettingsManager by loading global and project settings.
func Create(cwd, agentDir string) *SettingsManager {
	sm := &SettingsManager{
		cwd:      cwd,
		agentDir: agentDir,
		errors:   []SettingsError{},
	}

	// Load global settings
	globalPath := filepath.Join(agentDir, "settings.json")
	if data, err := os.ReadFile(globalPath); err == nil {
		if err := json.Unmarshal(data, &sm.global); err != nil {
			sm.errors = append(sm.errors, SettingsError{Scope: "global", Error: err})
		}
	}

	// Load project settings
	projectPath := filepath.Join(cwd, ".pi", "settings.json")
	if data, err := os.ReadFile(projectPath); err == nil {
		if err := json.Unmarshal(data, &sm.project); err != nil {
			sm.errors = append(sm.errors, SettingsError{Scope: "project", Error: err})
		}
	}

	sm.rebuild()
	return sm
}

// DrainErrors returns all accumulated settings errors and clears the list.
func (sm *SettingsManager) DrainErrors() []SettingsError {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	errs := sm.errors
	sm.errors = nil
	return errs
}

// --- Getters ---

func (sm *SettingsManager) GetDefaultProvider() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.merged.DefaultProvider
}

func (sm *SettingsManager) GetDefaultModel() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.merged.DefaultModel
}

func (sm *SettingsManager) GetEnabledModels() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if len(sm.merged.EnabledModels) == 0 {
		return nil
	}
	result := make([]string, len(sm.merged.EnabledModels))
	copy(result, sm.merged.EnabledModels)
	return result
}

func (sm *SettingsManager) GetTheme() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.merged.Theme == "" {
		return "default"
	}
	return sm.merged.Theme
}

func (sm *SettingsManager) GetShowHardwareCursor() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.merged.ShowCursor != nil {
		return *sm.merged.ShowCursor
	}
	return false
}

func (sm *SettingsManager) GetClearOnShrink() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.merged.ClearOnShrink != nil {
		return *sm.merged.ClearOnShrink
	}
	return true
}

func (sm *SettingsManager) GetQuietStartup() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.merged.QuietStartup != nil {
		return *sm.merged.QuietStartup
	}
	return false
}

func (sm *SettingsManager) GetSessionDir() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.merged.SessionDir != "" {
		return sm.merged.SessionDir
	}
	return ""
}

func (sm *SettingsManager) GetImageAutoResize() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.merged.ImageAutoResize != nil {
		return *sm.merged.ImageAutoResize
	}
	return true
}

// --- Setters (persist to the appropriate scope) ---

func (sm *SettingsManager) SetDefaultProvider(provider string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.global.DefaultProvider = provider
	sm.rebuild()
	sm.saveGlobal()
}

func (sm *SettingsManager) SetDefaultModel(model string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.global.DefaultModel = model
	sm.rebuild()
	sm.saveGlobal()
}

func (sm *SettingsManager) SetEnabledModels(models []string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.global.EnabledModels = models
	sm.rebuild()
	sm.saveGlobal()
}

func (sm *SettingsManager) SetTheme(theme string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.global.Theme = theme
	sm.rebuild()
	sm.saveGlobal()
}

func (sm *SettingsManager) SetCustom(key string, value any) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.global.Custom == nil {
		sm.global.Custom = make(map[string]any)
	}
	sm.global.Custom[key] = value
	sm.rebuild()
	sm.saveGlobal()
}

func (sm *SettingsManager) GetCustom(key string) (any, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.merged.Custom == nil {
		return nil, false
	}
	v, ok := sm.merged.Custom[key]
	return v, ok
}

// --- Private ---

func (sm *SettingsManager) rebuild() {
	// Start with global, overlay project
	sm.merged = sm.global

	// Simple merge: project overrides global for non-zero values
	if sm.project.DefaultProvider != "" {
		sm.merged.DefaultProvider = sm.project.DefaultProvider
	}
	if sm.project.DefaultModel != "" {
		sm.merged.DefaultModel = sm.project.DefaultModel
	}
	if len(sm.project.EnabledModels) > 0 {
		sm.merged.EnabledModels = sm.project.EnabledModels
	}
	if sm.project.Theme != "" {
		sm.merged.Theme = sm.project.Theme
	}
	if sm.project.ShowCursor != nil {
		sm.merged.ShowCursor = sm.project.ShowCursor
	}
	if sm.project.ClearOnShrink != nil {
		sm.merged.ClearOnShrink = sm.project.ClearOnShrink
	}
	if sm.project.QuietStartup != nil {
		sm.merged.QuietStartup = sm.project.QuietStartup
	}
	if sm.project.SessionDir != "" {
		sm.merged.SessionDir = sm.project.SessionDir
	}
	if sm.project.ImageAutoResize != nil {
		sm.merged.ImageAutoResize = sm.project.ImageAutoResize
	}
	if sm.project.NoExtensions != nil {
		sm.merged.NoExtensions = sm.project.NoExtensions
	}
	if sm.project.NoSkills != nil {
		sm.merged.NoSkills = sm.project.NoSkills
	}
	if sm.project.NoPromptTemplates != nil {
		sm.merged.NoPromptTemplates = sm.project.NoPromptTemplates
	}
	if sm.project.NoThemes != nil {
		sm.merged.NoThemes = sm.project.NoThemes
	}
	if sm.project.Custom != nil {
		if sm.merged.Custom == nil {
			sm.merged.Custom = make(map[string]any)
		}
		for k, v := range sm.project.Custom {
			sm.merged.Custom[k] = v
		}
	}
}

func (sm *SettingsManager) saveGlobal() {
	globalPath := filepath.Join(sm.agentDir, "settings.json")
	os.MkdirAll(filepath.Dir(globalPath), 0755)
	data, err := json.MarshalIndent(sm.global, "", "  ")
	if err != nil {
		sm.errors = append(sm.errors, SettingsError{Scope: "global", Error: err})
		return
	}
	if err := os.WriteFile(globalPath, data, 0644); err != nil {
		sm.errors = append(sm.errors, SettingsError{Scope: "global", Error: err})
	}
}

// AgentDir returns the path to the agent configuration directory.
func AgentDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".pi")
}

// InitAgentDir creates the agent configuration directory if it doesn't exist.
func InitAgentDir() (string, error) {
	dir := AgentDir()
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return "", fmt.Errorf("create agent dir: %w", err)
	}
	// Create subdirectories
	for _, sub := range []string{"sessions", "settings", "extensions"} {
		os.MkdirAll(filepath.Join(dir, sub), 0755)
	}
	return dir, nil
}

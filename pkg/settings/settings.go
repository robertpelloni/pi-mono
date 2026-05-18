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
	DefaultProvider         string   `json:"defaultProvider,omitempty"`
	DefaultModel            string   `json:"defaultModel,omitempty"`
	DefaultThinkingLevel    string   `json:"defaultThinkingLevel,omitempty"`
	EnabledModels           []string `json:"enabledModels,omitempty"`

	// UI
	Theme        string `json:"theme,omitempty"`
	ShowCursor   *bool  `json:"showHardwareCursor,omitempty"`
	ClearOnShrink *bool `json:"clearOnShrink,omitempty"`
	QuietStartup *bool  `json:"quietStartup,omitempty"`

	// Session
	SessionDir string `json:"sessionDir,omitempty"`

	// Tools
	ImageAutoResize    *bool  `json:"imageAutoResize,omitempty"`
	ShellCommandPrefix string `json:"shellCommandPrefix,omitempty"`

	// Steering & Follow-up
	SteeringMode string `json:"steeringMode,omitempty"`    // "all" or "one-at-a-time"
	FollowUpMode string `json:"followUpMode,omitempty"`    // "all" or "one-at-a-time"

	// Compaction
	CompactionEnabled   *bool    `json:"compactionEnabled,omitempty"`
	CompactionThreshold float64  `json:"compactionThreshold,omitempty"`

	// Auto-Retry
	RetryEnabled     *bool `json:"retryEnabled,omitempty"`
	RetryMaxRetries  int   `json:"retryMaxRetries,omitempty"`
	RetryBaseDelayMs int   `json:"retryBaseDelayMs,omitempty"`

	// Branch Summary
	BranchSummaryReserveTokens int `json:"branchSummaryReserveTokens,omitempty"`

	// Extensions
	NoExtensions      *bool `json:"noExtensions,omitempty"`
	NoSkills          *bool `json:"noSkills,omitempty"`
	NoPromptTemplates *bool `json:"noPromptTemplates,omitempty"`
	NoThemes          *bool `json:"noThemes,omitempty"`

	// Custom
	Custom map[string]any `json:"custom,omitempty"`
}

// RetrySettings holds auto-retry configuration.
type RetrySettings struct {
	Enabled      bool
	MaxRetries   int
	BaseDelayMs  int
}

// CompactionSettings holds compaction configuration.
type CompactionSettings struct {
	Enabled   bool
	Threshold float64
}

// BranchSummarySettings holds branch summarization configuration.
type BranchSummarySettings struct {
	ReserveTokens int
}

// SettingsManager loads and persists settings from multiple scopes:
// 1. Global: ~/.pi/settings.json
// 2. Project: .pi/settings.json (in the project root)
// Merged with project overriding global.
type SettingsManager struct {
	mu      sync.RWMutex
	cwd     string
	agentDir string
	global  Settings
	project Settings
	merged  Settings
	errors  []SettingsError
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

// Reload re-reads global and project settings from disk.
func (sm *SettingsManager) Reload() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	globalPath := filepath.Join(sm.agentDir, "settings.json")
	if data, err := os.ReadFile(globalPath); err == nil {
		if err := json.Unmarshal(data, &sm.global); err != nil {
			sm.errors = append(sm.errors, SettingsError{Scope: "global", Error: err})
		}
	}
	projectPath := filepath.Join(sm.cwd, ".pi", "settings.json")
	if data, err := os.ReadFile(projectPath); err == nil {
		if err := json.Unmarshal(data, &sm.project); err != nil {
			sm.errors = append(sm.errors, SettingsError{Scope: "project", Error: err})
		}
	}
	sm.rebuild()
	return nil
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

func (sm *SettingsManager) GetDefaultThinkingLevel() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.merged.DefaultThinkingLevel
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
	return sm.merged.SessionDir
}

func (sm *SettingsManager) GetImageAutoResize() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.merged.ImageAutoResize != nil {
		return *sm.merged.ImageAutoResize
	}
	return true
}

func (sm *SettingsManager) GetShellCommandPrefix() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.merged.ShellCommandPrefix
}

func (sm *SettingsManager) GetSteeringMode() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.merged.SteeringMode == "" {
		return "all"
	}
	return sm.merged.SteeringMode
}

func (sm *SettingsManager) GetFollowUpMode() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.merged.FollowUpMode == "" {
		return "all"
	}
	return sm.merged.FollowUpMode
}

// GetCompactionSettings returns compaction configuration.
func (sm *SettingsManager) GetCompactionSettings() CompactionSettings {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	enabled := true
	if sm.merged.CompactionEnabled != nil {
		enabled = *sm.merged.CompactionEnabled
	}
	threshold := 0.8
	if sm.merged.CompactionThreshold > 0 {
		threshold = sm.merged.CompactionThreshold
	}
	return CompactionSettings{Enabled: enabled, Threshold: threshold}
}

// GetCompactionEnabled returns whether auto-compaction is enabled.
func (sm *SettingsManager) GetCompactionEnabled() bool {
	return sm.GetCompactionSettings().Enabled
}

// GetRetrySettings returns auto-retry configuration.
func (sm *SettingsManager) GetRetrySettings() RetrySettings {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	enabled := true
	if sm.merged.RetryEnabled != nil {
		enabled = *sm.merged.RetryEnabled
	}
	maxRetries := 3
	if sm.merged.RetryMaxRetries > 0 {
		maxRetries = sm.merged.RetryMaxRetries
	}
	baseDelay := 1000
	if sm.merged.RetryBaseDelayMs > 0 {
		baseDelay = sm.merged.RetryBaseDelayMs
	}
	return RetrySettings{Enabled: enabled, MaxRetries: maxRetries, BaseDelayMs: baseDelay}
}

// GetRetryEnabled returns whether auto-retry is enabled.
func (sm *SettingsManager) GetRetryEnabled() bool {
	return sm.GetRetrySettings().Enabled
}

// GetBranchSummarySettings returns branch summarization configuration.
func (sm *SettingsManager) GetBranchSummarySettings() BranchSummarySettings {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	reserve := 4000
	if sm.merged.BranchSummaryReserveTokens > 0 {
		reserve = sm.merged.BranchSummaryReserveTokens
	}
	return BranchSummarySettings{ReserveTokens: reserve}
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

// SetDefaultModelAndProvider sets both the default model and provider.
func (sm *SettingsManager) SetDefaultModelAndProvider(provider, model string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.global.DefaultProvider = provider
	sm.global.DefaultModel = model
	sm.rebuild()
	sm.saveGlobal()
}

// SetDefaultThinkingLevel sets the default thinking level.
func (sm *SettingsManager) SetDefaultThinkingLevel(level string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.global.DefaultThinkingLevel = level
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

// SetSteeringMode sets the steering message mode.
func (sm *SettingsManager) SetSteeringMode(mode string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.global.SteeringMode = mode
	sm.rebuild()
	sm.saveGlobal()
}

// SetFollowUpMode sets the follow-up message mode.
func (sm *SettingsManager) SetFollowUpMode(mode string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.global.FollowUpMode = mode
	sm.rebuild()
	sm.saveGlobal()
}

// SetCompactionEnabled toggles auto-compaction.
func (sm *SettingsManager) SetCompactionEnabled(enabled bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.global.CompactionEnabled = &enabled
	sm.rebuild()
	sm.saveGlobal()
}

// SetRetryEnabled toggles auto-retry.
func (sm *SettingsManager) SetRetryEnabled(enabled bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.global.RetryEnabled = &enabled
	sm.rebuild()
	sm.saveGlobal()
}

// SetShellCommandPrefix sets the shell command prefix.
func (sm *SettingsManager) SetShellCommandPrefix(prefix string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.global.ShellCommandPrefix = prefix
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
	sm.merged = sm.global
	if sm.project.DefaultProvider != "" {
		sm.merged.DefaultProvider = sm.project.DefaultProvider
	}
	if sm.project.DefaultModel != "" {
		sm.merged.DefaultModel = sm.project.DefaultModel
	}
	if sm.project.DefaultThinkingLevel != "" {
		sm.merged.DefaultThinkingLevel = sm.project.DefaultThinkingLevel
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
	if sm.project.ShellCommandPrefix != "" {
		sm.merged.ShellCommandPrefix = sm.project.ShellCommandPrefix
	}
	if sm.project.SteeringMode != "" {
		sm.merged.SteeringMode = sm.project.SteeringMode
	}
	if sm.project.FollowUpMode != "" {
		sm.merged.FollowUpMode = sm.project.FollowUpMode
	}
	if sm.project.CompactionEnabled != nil {
		sm.merged.CompactionEnabled = sm.project.CompactionEnabled
	}
	if sm.project.CompactionThreshold > 0 {
		sm.merged.CompactionThreshold = sm.project.CompactionThreshold
	}
	if sm.project.RetryEnabled != nil {
		sm.merged.RetryEnabled = sm.project.RetryEnabled
	}
	if sm.project.RetryMaxRetries > 0 {
		sm.merged.RetryMaxRetries = sm.project.RetryMaxRetries
	}
	if sm.project.RetryBaseDelayMs > 0 {
		sm.merged.RetryBaseDelayMs = sm.project.RetryBaseDelayMs
	}
	if sm.project.BranchSummaryReserveTokens > 0 {
		sm.merged.BranchSummaryReserveTokens = sm.project.BranchSummaryReserveTokens
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
	for _, sub := range []string{"sessions", "settings", "extensions"} {
		os.MkdirAll(filepath.Join(dir, sub), 0755)
	}
	return dir, nil
}

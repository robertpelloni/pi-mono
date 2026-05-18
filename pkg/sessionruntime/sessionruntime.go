package sessionruntime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/agentsession"
	"github.com/badlogic/pi-mono/pkg/compaction"
	"github.com/badlogic/pi-mono/pkg/modelresolver"
	"github.com/badlogic/pi-mono/pkg/session"
	"github.com/badlogic/pi-mono/pkg/settings"
	"github.com/badlogic/pi-mono/pkg/skills"
	"github.com/badlogic/pi-mono/pkg/slashcommands"
	"github.com/badlogic/pi-mono/pkg/systemprompt"
)

// DiagnosticType classifies non-fatal issues collected during runtime creation.
type DiagnosticType string

const (
	DiagInfo    DiagnosticType = "info"
	DiagWarning DiagnosticType = "warning"
	DiagError   DiagnosticType = "error"
)

// AgentSessionRuntimeDiagnostic represents a non-fatal issue.
type AgentSessionRuntimeDiagnostic struct {
	Type    DiagnosticType `json:"type"`
	Message string         `json:"message"`
}

// CreateAgentSessionRuntimeResult holds the result of creating a runtime.
type CreateAgentSessionRuntimeResult struct {
	AgentSession       *agentsession.AgentSession
	Services           AgentSessionServices
	Diagnostics        []AgentSessionRuntimeDiagnostic
	ModelFallbackMessage *string
}

// AgentSessionServices holds cwd-bound runtime services.
type AgentSessionServices struct {
	CWD             string                    `json:"cwd"`
	AgentDir        string                    `json:"agentDir"`
	SettingsManager *settings.SettingsManager
	ModelRegistry   *modelresolver.ModelRegistry
	SkillLoader     *skills.SkillLoader
	SlashRegistry   *slashcommands.Registry
	Diagnostics     []AgentSessionRuntimeDiagnostic `json:"diagnostics"`
}

// CreateAgentSessionRuntimeFactory creates a full runtime for a target cwd.
type CreateAgentSessionRuntimeFactory func(options CreateAgentSessionRuntimeOptions) (*CreateAgentSessionRuntimeResult, error)

// CreateAgentSessionRuntimeOptions configures runtime creation.
type CreateAgentSessionRuntimeOptions struct {
	CWD             string
	AgentDir        string
	SessionManager  *session.SessionManager
	SessionStartEvent *SessionStartEvent
	// Override agent/stream function for custom runtimes
	StreamFn ai.StreamFunction
	// Override tools for custom runtimes
	Tools []agent.AgentTool
	// Initial model override
	Model *ai.ModelInfo
	// Pre-built services (optional - created if nil)
	SettingsManager *settings.SettingsManager
	ModelRegistry   *modelresolver.ModelRegistry
	SkillLoader     *skills.SkillLoader
	SlashRegistry   *slashcommands.Registry
}

// SessionStartEvent is fired when a session starts.
type SessionStartEvent struct {
	Type                SessionStartReason `json:"type"`
	PreviousSessionFile *string            `json:"previousSessionFile,omitempty"`
}

// SessionStartReason indicates why a session started.
type SessionStartReason string

const (
	ReasonStartup SessionStartReason = "startup"
	ReasonReload  SessionStartReason = "reload"
	ReasonNew     SessionStartReason = "new"
	ReasonResume  SessionStartReason = "resume"
	ReasonFork    SessionStartReason = "fork"
)

// AgentSessionRuntime owns the current AgentSession plus its cwd-bound services.
type AgentSessionRuntime struct {
	mu                chan struct{} // mutex via channel
	agentSession      *agentsession.AgentSession
	services          AgentSessionServices
	createRuntime     CreateAgentSessionRuntimeFactory
	diagnostics       []AgentSessionRuntimeDiagnostic
	modelFallbackMessage *string
}

// NewAgentSessionRuntime creates a new runtime wrapper.
func NewAgentSessionRuntime(
	sess *agentsession.AgentSession,
	services AgentSessionServices,
	createRuntime CreateAgentSessionRuntimeFactory,
	diagnostics []AgentSessionRuntimeDiagnostic,
	modelFallbackMessage *string,
) *AgentSessionRuntime {
	return &AgentSessionRuntime{
		agentSession:      sess,
		services:          services,
		createRuntime:     createRuntime,
		diagnostics:       diagnostics,
		modelFallbackMessage: modelFallbackMessage,
	}
}

// Services returns the cwd-bound services.
func (r *AgentSessionRuntime) Services() *AgentSessionServices {
	return &r.services
}

// AgentSession returns the current agent session.
func (r *AgentSessionRuntime) AgentSession() *agentsession.AgentSession {
	return r.agentSession
}

// CWD returns the current working directory.
func (r *AgentSessionRuntime) CWD() string {
	return r.services.CWD
}

// Diagnostics returns collected diagnostics.
func (r *AgentSessionRuntime) Diagnostics() []AgentSessionRuntimeDiagnostic {
	return r.diagnostics
}

// ModelFallbackMessage returns the model fallback message if any.
func (r *AgentSessionRuntime) ModelFallbackMessage() *string {
	return r.modelFallbackMessage
}

// SwitchSession switches to a different session file.
func (r *AgentSessionRuntime) SwitchSession(sessionPath string, cwdOverride *string) error {
	sm := session.OpenSession(sessionPath, r.services.AgentDir, cwdOverride)
	result, err := r.createRuntime(CreateAgentSessionRuntimeOptions{
		CWD:            sm.GetCWD(),
		AgentDir:       r.services.AgentDir,
		SessionManager: sm,
		SessionStartEvent: &SessionStartEvent{
			Type: ReasonResume,
		},
	})
	if err != nil {
		return err
	}
	r.apply(result)
	return nil
}

// NewSession creates a new session.
func (r *AgentSessionRuntime) NewSession(parentSession *string) error {
	sessionDir := ""
	if r.services.SettingsManager != nil {
		sessionDir = r.services.SettingsManager.GetSessionDir()
	}
	sm := session.CreateSession(r.services.CWD, sessionDir)
	if parentSession != nil {
		sm.NewSession(&session.NewSessionOptions{ParentSession: parentSession})
	}
	result, err := r.createRuntime(CreateAgentSessionRuntimeOptions{
		CWD:            r.services.CWD,
		AgentDir:       r.services.AgentDir,
		SessionManager: sm,
		SessionStartEvent: &SessionStartEvent{
			Type: ReasonNew,
		},
	})
	if err != nil {
		return err
	}
	r.apply(result)
	return nil
}

// ImportFromJsonl imports a session from a JSONL file.
func (r *AgentSessionRuntime) ImportFromJsonl(inputPath string, cwdOverride *string) error {
	resolvedPath, _ := filepath.Abs(inputPath)
	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", resolvedPath)
	}
	sessionDir := ""
	if r.services.SettingsManager != nil {
		sessionDir = r.services.SettingsManager.GetSessionDir()
	}
	sm := session.OpenSession(resolvedPath, sessionDir, cwdOverride)
	result, err := r.createRuntime(CreateAgentSessionRuntimeOptions{
		CWD:            sm.GetCWD(),
		AgentDir:       r.services.AgentDir,
		SessionManager: sm,
		SessionStartEvent: &SessionStartEvent{
			Type: ReasonResume,
		},
	})
	if err != nil {
		return err
	}
	r.apply(result)
	return nil
}

// Dispose shuts down the runtime and releases resources.
func (r *AgentSessionRuntime) Dispose() {
	if r.agentSession != nil {
		r.agentSession.Dispose()
	}
}

// Prompt sends a prompt to the agent session.
func (r *AgentSessionRuntime) Prompt(ctx context.Context, text string) error {
	if r.agentSession == nil {
		return fmt.Errorf("no active agent session")
	}
	return r.agentSession.Prompt(ctx, text)
}

// Reload reloads the runtime (settings, skills, system prompt).
func (r *AgentSessionRuntime) Reload() error {
	if r.agentSession == nil {
		return fmt.Errorf("no active agent session")
	}
	return r.agentSession.Reload()
}

func (r *AgentSessionRuntime) apply(result *CreateAgentSessionRuntimeResult) {
	if cwd := result.Services.CWD; cwd != "" {
		if wd, _ := os.Getwd(); wd != cwd {
			os.Chdir(cwd)
		}
	}
	r.agentSession = result.AgentSession
	r.services = result.Services
	r.diagnostics = result.Diagnostics
	r.modelFallbackMessage = result.ModelFallbackMessage
}

// CreateAgentSessionRuntime creates the initial runtime.
func CreateAgentSessionRuntime(
	createRuntime CreateAgentSessionRuntimeFactory,
	options CreateAgentSessionRuntimeOptions,
) (*AgentSessionRuntime, error) {
	result, err := createRuntime(options)
	if err != nil {
		return nil, err
	}
	if cwd := result.Services.CWD; cwd != "" {
		if wd, _ := os.Getwd(); wd != cwd {
			os.Chdir(cwd)
		}
	}
	return NewAgentSessionRuntime(
		result.AgentSession,
		result.Services,
		createRuntime,
		result.Diagnostics,
		result.ModelFallbackMessage,
	), nil
}

// DefaultCreateRuntime is the default factory that creates a runtime with
// standard tool definitions, system prompt, and session services.
func DefaultCreateRuntime(options CreateAgentSessionRuntimeOptions) (*CreateAgentSessionRuntimeResult, error) {
	var diagnostics []AgentSessionRuntimeDiagnostic

	// Initialize settings if not provided
	settingsMgr := options.SettingsManager
	if settingsMgr == nil {
		settingsMgr = settings.Create(options.CWD, options.AgentDir)
	}

	// Initialize model registry
	modelRegistry := options.ModelRegistry
	if modelRegistry == nil {
		modelRegistry = modelresolver.NewModelRegistry()
	}

	// Initialize skill loader
	skillLoader := options.SkillLoader
	if skillLoader == nil {
		skillLoader = skills.NewSkillLoader()
	}

	// Initialize slash command registry
	slashReg := options.SlashRegistry
	if slashReg == nil {
		slashReg = slashcommands.NewRegistry()
	}

	// Initialize compactor
	compactor := compaction.NewCompactor(compaction.DefaultCompactionConfig())

	// Resolve the model
	var activeModel ai.ModelInfo
	if options.Model != nil {
		activeModel = *options.Model
	} else {
		// Try to get default model from settings
		provider := settingsMgr.GetDefaultProvider()
		modelID := settingsMgr.GetDefaultModel()
		if provider != "" && modelID != "" {
			resolved, err := modelresolver.ResolveWithProvider(provider, modelID, modelRegistry)
			if err == nil {
				activeModel = resolved.Model
			}
		}
	}

	// Build tools
	tools := options.Tools
	if tools == nil {
		tools = agent.DefaultTools()
	}

	// Build stream function
	streamFn := options.StreamFn
	if streamFn == nil {
		streamFn = ai.DefaultStreamFunction
	}

	// Create agent
	ag := agent.NewAgent(activeModel, tools, streamFn, agent.DefaultLoopConfig())

	// Build system prompt
	loadedSkills := skillLoader.LoadSkills(options.AgentDir, options.CWD)
	contextFiles := systemprompt.LoadProjectContextFiles(options.CWD)
	var toolNames []string
	for _, t := range tools {
		toolNames = append(toolNames, t.Name)
	}
	toolSnippets := systemprompt.DefaultToolSnippets()
	prompt := systemprompt.BuildSystemPrompt(systemprompt.BuildSystemPromptOptions{
		SelectedTools: toolNames,
		ToolSnippets:  toolSnippets,
		ContextFiles:  contextFiles,
		Skills:        skills.ToSkillRefs(loadedSkills.Skills),
		CWD:           options.CWD,
	})
	ag.SetSystemPrompt(prompt)

	// Create session manager if not provided
	sessionMgr := options.SessionManager
	if sessionMgr == nil {
		sessionDir := settingsMgr.GetSessionDir()
		sessionMgr = session.CreateSession(options.CWD, sessionDir)
	}

	// Create agent session
	as := agentsession.NewAgentSession(agentsession.AgentSessionConfig{
		Agent:          ag,
		SessionManager: sessionMgr,
		Settings:       settingsMgr,
		ModelRegistry:  modelRegistry,
		SkillLoader:    skillLoader,
		Compactor:      compactor,
		SlashCommands:  slashReg,
		CWD:            options.CWD,
		AgentDir:       options.AgentDir,
	})

	return &CreateAgentSessionRuntimeResult{
		AgentSession:       as,
		Services: AgentSessionServices{
			CWD:             options.CWD,
			AgentDir:        options.AgentDir,
			SettingsManager: settingsMgr,
			ModelRegistry:   modelRegistry,
			SkillLoader:     skillLoader,
			SlashRegistry:   slashReg,
			Diagnostics:     diagnostics,
		},
		Diagnostics:          diagnostics,
		ModelFallbackMessage: nil,
	}, nil
}

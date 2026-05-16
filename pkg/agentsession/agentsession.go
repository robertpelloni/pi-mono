package agentsession

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/compaction"
	"github.com/badlogic/pi-mono/pkg/modelresolver"
	"github.com/badlogic/pi-mono/pkg/session"
	"github.com/badlogic/pi-mono/pkg/settings"
	"github.com/badlogic/pi-mono/pkg/skills"
	"github.com/badlogic/pi-mono/pkg/slashcommands"
	"github.com/badlogic/pi-mono/pkg/systemprompt"
)

// AgentSessionConfig holds the configuration for creating an AgentSession.
type AgentSessionConfig struct {
	// Agent is the core agent loop.
	Agent *agent.Agent
	// Session is the persistent conversation history.
	Session *session.Session
	// Settings is the settings manager.
	Settings *settings.SettingsManager
	// ModelRegistry is the model discovery service.
	ModelRegistry *modelresolver.ModelRegistry
	// SkillLoader discovers and loads skills.
	SkillLoader *skills.SkillLoader
	// Compactor handles context window compaction.
	Compactor *compaction.Compactor
	// SlashCommands manages slash command registration.
	SlashCommands *slashcommands.Registry
	// CWD is the working directory.
	CWD string
	// AgentDir is the configuration directory (~/.pi).
	AgentDir string
}

// AgentSession is the core runtime that manages the agent lifecycle,
// session persistence, model switching, compaction, and event routing.
// It's the Go equivalent of the TypeScript AgentSession class.
type AgentSession struct {
	mu            sync.RWMutex
	config        AgentSessionConfig
	activeModel   ai.ModelInfo
	thinkingLevel ai.ThinkingLevel
	listeners     []AgentSessionEventListener
	stats         SessionStats
}

// AgentSessionEvent represents events emitted by the AgentSession.
type AgentSessionEvent struct {
	Type    string         `json:"type"`
	Data    any            `json:"data,omitempty"`
	Error   error          `json:"error,omitempty"`
}

// AgentSessionEventListener is a callback for session events.
type AgentSessionEventListener func(event AgentSessionEvent)

// SessionStats tracks session statistics.
type SessionStats struct {
	UserMessages      int `json:"userMessages"`
	AssistantMessages int `json:"assistantMessages"`
	ToolCalls         int `json:"toolCalls"`
	ToolResults       int `json:"toolResults"`
	Compactions       int `json:"compactions"`
	TotalTokensIn     int `json:"totalTokensIn"`
	TotalTokensOut    int `json:"totalTokensOut"`
	TotalCost         float64 `json:"totalCost"`
}

// NewAgentSession creates a new AgentSession from the given config.
func NewAgentSession(config AgentSessionConfig) *AgentSession {
	as := &AgentSession{
		config:      config,
		activeModel: config.Agent.Model(),
	}

	// Subscribe to agent events for statistics tracking
	config.Agent.Subscribe(func(event agent.AgentEvent) {
		as.onAgentEvent(event)
	})

	return as
}

// Prompt sends a user message through the agent loop.
// It handles session persistence, compaction checks, and event routing.
func (as *AgentSession) Prompt(ctx context.Context, text string) error {
	// Check if compaction is needed before sending
	if as.config.Compactor != nil {
		messages := as.config.Agent.Messages()
		if as.config.Compactor.ShouldCompact(messages) {
			as.emit(AgentSessionEvent{Type: "compaction_start", Data: "threshold"})
			compacted, err := as.config.Compactor.Compact(ctx, messages)
			if err == nil {
				as.config.Agent.SetMessages(compacted)
				as.mu.Lock()
				as.stats.Compactions++
				as.mu.Unlock()
			}
			as.emit(AgentSessionEvent{Type: "compaction_end", Data: "threshold"})
		}
	}

	// Build user message
	userMsg := ai.UserMessage{
		Content:   []ai.Content{ai.TextContent{Text: text}},
		Timestamp: time.Now().UnixMilli(),
	}

	// Persist to session
	if as.config.Session != nil {
		as.config.Session.AppendMessage(userMsg)
	}

	// Update stats
	as.mu.Lock()
	as.stats.UserMessages++
	as.mu.Unlock()

	// Run the agent loop
	err := as.config.Agent.Prompt(ctx, userMsg)
	if err != nil {
		as.emit(AgentSessionEvent{Type: "prompt_error", Error: err})
	}
	return err
}

// SwitchModel changes the active model.
func (as *AgentSession) SwitchModel(modelID string) error {
	if as.config.ModelRegistry == nil {
		return fmt.Errorf("model registry not available")
	}

	provider := string(as.activeModel.Provider)
	resolved, err := modelresolver.ResolveWithProvider(provider, modelID, as.config.ModelRegistry)
	if err != nil {
		return fmt.Errorf("failed to resolve model %q: %w", modelID, err)
	}

	as.mu.Lock()
	as.activeModel = resolved.Model
	as.thinkingLevel = resolved.ThinkingLevel
	as.mu.Unlock()

	as.config.Agent.SetModel(resolved.Model)
	if resolved.ThinkingLevel != "" {
		as.config.Agent.SetThinkingLevel(resolved.ThinkingLevel)
	}

	// Update stream function based on provider
	switch resolved.Model.Provider {
	case ai.ProviderAnthropic:
		// Stream function would be set by the caller
	case ai.ProviderGoogle:
		// Same
	default:
		// OpenAI
	}

	as.emit(AgentSessionEvent{Type: "model_switch", Data: modelID})
	return nil
}

// SwitchProvider changes the active provider and optionally the model.
func (as *AgentSession) SwitchProvider(providerName string) error {
	as.mu.Lock()
	as.activeModel.Provider = ai.Provider(providerName)
	as.activeModel.API = providerToAPI(ai.Provider(providerName))
	as.mu.Unlock()

	as.config.Agent.SetModel(as.activeModel)
	as.emit(AgentSessionEvent{Type: "provider_switch", Data: providerName})
	return nil
}

// Compact manually triggers context compaction.
func (as *AgentSession) Compact(ctx context.Context) error {
	if as.config.Compactor == nil {
		return fmt.Errorf("compactor not available")
	}

	as.emit(AgentSessionEvent{Type: "compaction_start", Data: "manual"})
	messages := as.config.Agent.Messages()
	compacted, err := as.config.Compactor.Compact(ctx, messages)
	if err != nil {
		as.emit(AgentSessionEvent{Type: "compaction_end", Data: "manual", Error: err})
		return err
	}

	as.config.Agent.SetMessages(compacted)
	as.mu.Lock()
	as.stats.Compactions++
	as.mu.Unlock()

	as.emit(AgentSessionEvent{Type: "compaction_end", Data: "manual"})
	return nil
}

// NewSession creates a fresh session, resetting conversation history.
func (as *AgentSession) NewSession() {
	as.config.Agent.SetMessages(nil)
	if as.config.Session != nil {
		as.config.Session = session.NewSession(as.config.CWD, as.config.Settings.GetSessionDir())
	}
	as.mu.Lock()
	as.stats = SessionStats{}
	as.mu.Unlock()
	as.emit(AgentSessionEvent{Type: "new_session"})
}

// Reload refreshes skills, settings, and extensions.
func (as *AgentSession) Reload() error {
	// Reload skills
	if as.config.SkillLoader != nil {
		as.config.SkillLoader.ClearCache()
	}

	// Rebuild system prompt
	loadedSkills := as.config.SkillLoader.LoadSkills(as.config.AgentDir, as.config.CWD)
	contextFiles := systemprompt.LoadProjectContextFiles(as.config.CWD)

	var toolNames []string
	for _, t := range as.config.Agent.Tools() {
		toolNames = append(toolNames, t.Name)
	}

	toolSnippets := systemprompt.DefaultToolSnippets()
	prompt := systemprompt.BuildSystemPrompt(systemprompt.BuildSystemPromptOptions{
		SelectedTools: toolNames,
		ToolSnippets:  toolSnippets,
		ContextFiles:  contextFiles,
		Skills:        skills.ToSkillRefs(loadedSkills.Skills),
		CWD:           as.config.CWD,
	})

	as.config.Agent.SetSystemPrompt(prompt)

	as.emit(AgentSessionEvent{Type: "reload"})
	return nil
}

// Stats returns the current session statistics.
func (as *AgentSession) Stats() SessionStats {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.stats
}

// Model returns the current active model.
func (as *AgentSession) Model() ai.ModelInfo {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.activeModel
}

// ThinkingLevel returns the current thinking level.
func (as *AgentSession) ThinkingLevel() ai.ThinkingLevel {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.thinkingLevel
}

// SetThinkingLevel changes the thinking level.
func (as *AgentSession) SetThinkingLevel(level ai.ThinkingLevel) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.thinkingLevel = level
	as.config.Agent.SetThinkingLevel(level)
}

// Subscribe adds an event listener.
func (as *AgentSession) Subscribe(listener AgentSessionEventListener) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.listeners = append(as.listeners, listener)
}

// Session returns the session object.
func (as *AgentSession) Session() *session.Session {
	return as.config.Session
}

// Agent returns the underlying agent.
func (as *AgentSession) Agent() *agent.Agent {
	return as.config.Agent
}

// onAgentEvent processes agent events and updates stats/persistence.
func (as *AgentSession) onAgentEvent(event agent.AgentEvent) {
	as.mu.Lock()
	defer as.mu.Unlock()

	switch event.Type {
	case agent.EventMessageEnd:
		if event.Message != nil {
			switch event.Message.GetRole() {
			case ai.RoleAssistant:
				as.stats.AssistantMessages++
				// Track usage if available
				if am, ok := event.Message.(ai.AssistantMessage); ok {
					as.stats.TotalTokensIn += am.Usage.Input
					as.stats.TotalTokensOut += am.Usage.Output
					as.stats.TotalCost += am.Usage.Cost.Total
				}
			case ai.RoleTool:
				as.stats.ToolResults++
			}
		}
	case agent.EventToolExecutionEnd:
		as.stats.ToolCalls++
	}

	// Forward to session event listeners
	for _, l := range as.listeners {
		l(AgentSessionEvent{
			Type: string(event.Type),
			Data: event,
		})
	}
}

func (as *AgentSession) emit(event AgentSessionEvent) {
	as.mu.RLock()
	listeners := as.listeners
	as.mu.RUnlock()

	for _, l := range listeners {
		l(event)
	}
}

// providerToAPI returns the default API type for a provider.
func providerToAPI(provider ai.Provider) ai.Api {
	switch provider {
	case ai.ProviderAnthropic:
		return ai.ApiAnthropicMessages
	case ai.ProviderGoogle:
		return ai.ApiGoogleGenerativeAI
	case ai.ProviderOpenAI:
		return ai.ApiOpenAIResponses
	default:
		return ai.ApiOpenAICompletions
	}
}

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/agentsession"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/compaction"
	"github.com/badlogic/pi-mono/pkg/modelresolver"
	"github.com/badlogic/pi-mono/pkg/session"
	"github.com/badlogic/pi-mono/pkg/settings"
	"github.com/badlogic/pi-mono/pkg/skills"
	"github.com/badlogic/pi-mono/pkg/slashcommands"
	"github.com/badlogic/pi-mono/pkg/tools"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ────────────────────────────────────────────────────────────────
// App - the Wails-bound desktop application
// ────────────────────────────────────────────────────────────────

type App struct {
	ctx           context.Context
	agentSession  *agentsession.AgentSession
	agentLoop     *agent.Agent
	mu            sync.Mutex
	isGenerating  bool
	version       string
	cwd           string
	agentDir      string
}

func NewApp() *App {
	cwd, _ := os.Getwd()
	return &App{
		version: "0.97.0",
		cwd:     cwd,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ────────────────────────────────────────────────────────────────
// Initialization
// ────────────────────────────────────────────────────────────────

// InitAgent initializes the full agent stack with the given provider and model.
func (a *App) InitAgent(provider, modelID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.agentSession != nil {
		return nil
	}

	cwd := a.cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	// ── Agent Dir ──
	agentDir, err := settings.InitAgentDir()
	if err != nil {
		agentDir = settings.AgentDir()
	}
	a.agentDir = agentDir

	// ── Settings ──
	settingsManager := settings.Create(cwd, agentDir)

	// ── Auth ──
	authPath := filepath.Join(agentDir, "auth.json")
	_ = authPath // auth handled via env vars in desktop mode

	// ── Model Registry ──
	modelRegistry := modelresolver.NewModelRegistryWithDefaults()

	// Load models.json
	modelsJSONPath := modelresolver.GetModelsJSONPath(agentDir)
	_ = modelRegistry.LoadFromModelsJSON(modelsJSONPath, nil)

	// Load provider cache
	providerCachePath := filepath.Join(agentDir, "provider-cache.json")
	if _, err := os.Stat(providerCachePath); err == nil {
		if data, err := os.ReadFile(providerCachePath); err == nil {
			_ = data // provider cache models loaded if possible
		}
	}

	// ── Resolve Provider / Model ──
	if provider == "" && modelID != "" {
		provider = modelresolver.DetectProvider(modelID)
	}
	if provider == "" {
		provider = settingsManager.GetDefaultProvider()
	}
	if provider == "" {
		provider = "openai"
	}
	if modelID == "" {
		modelID = settingsManager.GetDefaultModel()
	}
	if modelID == "" {
		switch provider {
		case "anthropic":
			modelID = "claude-sonnet-4-20250514"
		case "google", "gemini":
			modelID = "gemini-2.5-pro"
		default:
			modelID = "gpt-4o"
		}
	}

	// ── Resolve ModelInfo ──
	var modelInfo ai.ModelInfo
	if resolved := modelRegistry.Find(provider, modelID); resolved != nil {
		modelInfo = *resolved
	} else {
		modelInfo = ai.ModelInfo{
			ID:       modelID,
			Provider: ai.Provider(provider),
			API:      providerToAPI(ai.Provider(provider)),
		}
	}

	// ── Stream Function ──
	var streamFunc ai.StreamFunction
	switch provider {
	case "anthropic":
		streamFunc = ai.StreamAnthropic
	case "google", "gemini":
		streamFunc = ai.StreamGoogle
	default:
		streamFunc = ai.StreamOpenAIResponses
	}

	// ── Thinking Level ──
	thinkingLevel := ai.ThinkingLevel("medium")
	if !modelInfo.Reasoning {
		thinkingLevel = ai.ThinkingLevel("off")
	}

	// ── Tools ──
	toolList := tools.CreateAllTools(cwd)

	// ── Agent Loop ──
	agentConfig := agent.AgentLoopConfig{
		ToolExecution: agent.ToolExecutionParallel,
	}

	// Compaction
	compactor := compaction.NewCompactor(compaction.CompactionConfig{
		MaxTokens: 80000,
		Strategy:  compaction.StrategySummarize,
		KeepLastN: 6,
		SummarizeFn: func(ctx context.Context, messages []ai.Message) (string, error) {
			result := compaction.GenerateDefaultSummary(messages, nil, nil, compaction.NewFileOps())
			return result.Summary, nil
		},
	})
	agentConfig.TransformContext = func(ctx context.Context, messages []ai.Message) ([]ai.Message, error) {
		if compactor.ShouldCompact(messages) {
			return compactor.Compact(ctx, messages)
		}
		return messages, nil
	}

	a.agentLoop = agent.NewAgent(modelInfo, toolList, streamFunc, agentConfig)
	a.agentLoop.SetThinkingLevel(thinkingLevel)

	// ── Session ──
	sessionDir := settingsManager.GetSessionDir()
	sess := session.CreateSession(cwd, sessionDir)

	// Replay existing messages
	if ctx := sess.BuildSessionContext(); len(ctx.Messages) > 0 {
		a.agentLoop.SetMessages(ctx.Messages)
	}

	// Persist messages
	a.agentLoop.Subscribe(func(event agent.AgentEvent) {
		if event.Type == agent.EventMessageEnd && event.Message != nil {
			sess.AppendMessage(event.Message)
		}
	})

	// ── Slash Commands ──
	slashRegistry := slashcommands.NewRegistry()
	slashRegistry.Register(slashcommands.SlashCommandInfo{
		Name:        "model",
		Description: "Switch the active model",
		Source:      slashcommands.SourceBuiltin,
	}, func(args string) (slashcommands.SlashCommandResult, error) {
		if args == "" {
			return slashcommands.SlashCommandResult{Info: "Usage: /model <model-id>"}, nil
		}
		return slashcommands.SlashCommandResult{SwitchModel: args}, nil
	})

	// ── Skills ──
	skillLoader := skills.NewSkillLoader()
	_ = skillLoader.LoadSkills(agentDir, cwd)

	// ── AgentSession ──
	a.agentSession = agentsession.NewAgentSession(agentsession.AgentSessionConfig{
		Agent:         a.agentLoop,
		SessionManager: sess,
		Settings:      settingsManager,
		ModelRegistry: modelRegistry,
		SkillLoader:   skillLoader,
		Compactor:     compactor,
		SlashCommands: slashRegistry,
		CWD:           cwd,
		AgentDir:      agentDir,
	})

	// ── Forward events to frontend ──
	a.agentLoop.Subscribe(func(event agent.AgentEvent) {
		payload := a.agentEventToMap(event)
		runtime.EventsEmit(a.ctx, "agent-event", payload)
	})

	a.agentSession.Subscribe(func(event agentsession.AgentSessionEvent) {
		payload := a.sessionEventToMap(event)
		runtime.EventsEmit(a.ctx, "session-event", payload)
	})

	// Notify frontend
	runtime.EventsEmit(a.ctx, "agent-ready", map[string]string{
		"provider": string(modelInfo.Provider),
		"model":    modelInfo.ID,
		"session":  sess.GetSessionID(),
	})

	return nil
}

// ────────────────────────────────────────────────────────────────
// Chat
// ────────────────────────────────────────────────────────────────

// SendMessage sends a user message and returns immediately.
func (a *App) SendMessage(text string) error {
	if a.agentSession == nil {
		return fmt.Errorf("agent not initialized - call InitAgent first")
	}

	a.mu.Lock()
	if a.isGenerating {
		a.mu.Unlock()
		return fmt.Errorf("agent is already generating")
	}
	a.isGenerating = true
	a.mu.Unlock()

	go func() {
		defer func() {
			a.mu.Lock()
			a.isGenerating = false
			a.mu.Unlock()
			runtime.EventsEmit(a.ctx, "generation-done", nil)
		}()

		err := a.agentSession.Prompt(a.ctx, text)
		if err != nil {
			runtime.EventsEmit(a.ctx, "agent-error", err.Error())
		}
	}()

	return nil
}

// AbortGeneration aborts the current generation.
func (a *App) AbortGeneration() {
	if a.agentSession != nil {
		a.agentSession.Abort()
	}
}

// ────────────────────────────────────────────────────────────────
// Model / Provider
// ────────────────────────────────────────────────────────────────

// GetModelInfo returns the current model info.
func (a *App) GetModelInfo() map[string]string {
	if a.agentSession == nil {
		return map[string]string{"provider": "", "model": ""}
	}
	model := a.agentSession.Model()
	return map[string]string{
		"provider": string(model.Provider),
		"model":    model.ID,
	}
}

// CycleModel cycles to the next model.
func (a *App) CycleModel() string {
	if a.agentSession == nil {
		return ""
	}
	res := a.agentSession.CycleModel("forward")
	if res != nil {
		return fmt.Sprintf("%s/%s", res.Model.Provider, res.Model.ID)
	}
	return ""
}

// SetThinkingLevel sets the thinking level.
func (a *App) SetThinkingLevel(level string) {
	if a.agentSession != nil {
		a.agentSession.SetThinkingLevel(level)
	}
}

// ────────────────────────────────────────────────────────────────
// Session
// ────────────────────────────────────────────────────────────────

// GetSessionID returns the current session ID.
func (a *App) GetSessionID() string {
	if a.agentSession == nil {
		return ""
	}
	return a.agentSession.SessionID()
}

// NewSession starts a new session.
func (a *App) NewSession() {
	if a.agentSession != nil {
		a.agentSession.NewSession()
	}
}

// ExportSession exports the session to HTML.
func (a *App) ExportSession(path string) string {
	if a.agentSession == nil {
		return "no active session"
	}
	result, err := a.agentSession.ExportToHTML(strings.TrimSpace(path))
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return result
}

// ────────────────────────────────────────────────────────────────
// Stats
// ────────────────────────────────────────────────────────────────

// GetStats returns current session stats.
func (a *App) GetStats() map[string]any {
	if a.agentSession == nil {
		return nil
	}
	stats := a.agentSession.GetSessionStats()
	ctxUsage := a.agentSession.GetContextUsage()
	return map[string]any{
		"tokensIn":       stats.TokensInput,
		"tokensOut":      stats.TokensOutput,
		"cost":           stats.Cost,
		"toolCalls":      stats.ToolCalls,
		"contextPercent": ctxUsage.Percent,
		"contextWindow":  ctxUsage.ContextWindow,
	}
}

// GetVersion returns the app version.
func (a *App) GetVersion() string {
	return a.version
}

// IsGenerating returns whether the agent is currently generating.
func (a *App) IsGenerating() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.isGenerating
}

// ────────────────────────────────────────────────────────────────
// Event serialization
// ────────────────────────────────────────────────────────────────

func (a *App) agentEventToMap(event agent.AgentEvent) map[string]any {
	m := map[string]any{
		"type": string(event.Type),
	}

	switch event.Type {
	case agent.EventMessageUpdate:
		if event.AssistantMessageEvent != nil {
			switch event.AssistantMessageEvent.Type {
			case ai.EventTextDelta:
				if event.AssistantMessageEvent.Delta != nil {
					m["text"] = *event.AssistantMessageEvent.Delta
				}
				m["eventType"] = "text"
			case ai.EventThinkingStart:
				m["eventType"] = "thinking-start"
			case ai.EventThinkingDelta:
				if event.AssistantMessageEvent.Delta != nil {
					m["text"] = *event.AssistantMessageEvent.Delta
				}
				m["eventType"] = "thinking"
			case ai.EventThinkingEnd:
				m["eventType"] = "thinking-end"
			case ai.EventToolCallStart:
				m["eventType"] = "tool-call-start"
			}
		}
	case agent.EventToolExecutionStart:
		m["tool"] = event.ToolName
		m["args"] = event.Args
	case agent.EventToolExecutionEnd:
		m["tool"] = event.ToolName
		m["isError"] = event.IsError
		content := extractResultText(event.Result)
		if content != "" {
			m["result"] = content
		}
	}

	return m
}

func (a *App) sessionEventToMap(event agentsession.AgentSessionEvent) map[string]any {
	return map[string]any{
		"type": event.Type,
		"data": event.Data,
	}
}

func extractResultText(result agent.AgentToolResult) string {
	var sb strings.Builder
	for _, c := range result.Content {
		if txt, ok := c.(ai.TextContent); ok {
			sb.WriteString(txt.Text)
		}
	}
	return sb.String()
}

// ────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────

// providerToAPI maps a provider name to its API type.
func providerToAPI(provider ai.Provider) ai.Api {
	switch provider {
	case ai.ProviderAnthropic:
		return ai.ApiAnthropicMessages
	case ai.ProviderGoogle:
		return ai.ApiGoogleGenerativeAI
	case ai.ProviderOllama:
		return ai.ApiOpenAICompletions
	default:
		return ai.ApiOpenAIResponses
	}
}

// GetCurrentTime returns the current time as a formatted string.
func (a *App) GetCurrentTime() string {
	return time.Now().Format("15:04:05")
}

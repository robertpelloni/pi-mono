package agentsession

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/compaction"
	"github.com/badlogic/pi-mono/pkg/export"
	"github.com/badlogic/pi-mono/pkg/modelresolver"
	"github.com/badlogic/pi-mono/pkg/session"
	"github.com/badlogic/pi-mono/pkg/settings"
	"github.com/badlogic/pi-mono/pkg/skills"
	"github.com/badlogic/pi-mono/pkg/slashcommands"
	"github.com/badlogic/pi-mono/pkg/systemprompt"
)

// ============================================================================
// Types
// ============================================================================

// ThinkingLevel constants matching the TypeScript THINKING_LEVELS array.
const (
	ThinkingOff     = "off"
	ThinkingMinimal = "minimal"
	ThinkingLow     = "low"
	ThinkingMedium  = "medium"
	ThinkingHigh    = "high"
	ThinkingXHigh   = "xhigh"
)

// Standard thinking levels
var ThinkingLevels = []string{ThinkingOff, ThinkingMinimal, ThinkingLow, ThinkingMedium, ThinkingHigh}

// Thinking levels including xhigh (for supported models)
var ThinkingLevelsWithXHigh = []string{ThinkingOff, ThinkingMinimal, ThinkingLow, ThinkingMedium, ThinkingHigh, ThinkingXHigh}

// CompactionReason indicates why compaction was triggered.
type CompactionReason string

const (
	CompactionManual   CompactionReason = "manual"
	CompactionThreshold CompactionReason = "threshold"
	CompactionOverflow  CompactionReason = "overflow"
)

// ModelCycleResult is returned from CycleModel operations.
type ModelCycleResult struct {
	Model         ai.ModelInfo
	ThinkingLevel string
	IsScoped      bool
}

// ParsedSkillBlock represents a parsed skill block from a user message.
type ParsedSkillBlock struct {
	Name        string
	Location    string
	Content     string
	UserMessage string
}

// ContextUsage provides information about the current context window utilization.
type ContextUsage struct {
	Tokens        *int     `json:"tokens"`
	ContextWindow int      `json:"contextWindow"`
	Percent       *float64 `json:"percent"`
}

// SessionStats tracks session statistics.
type SessionStats struct {
	SessionFile      string  `json:"sessionFile"`
	SessionID        string  `json:"sessionId"`
	UserMessages     int     `json:"userMessages"`
	AssistantMessages int    `json:"assistantMessages"`
	ToolCalls        int     `json:"toolCalls"`
	ToolResults      int     `json:"toolResults"`
	TotalMessages    int     `json:"totalMessages"`
	TokensInput      int     `json:"tokensInput"`
	TokensOutput     int     `json:"tokensOutput"`
	TokensCacheRead  int     `json:"tokensCacheRead"`
	TokensCacheWrite int     `json:"tokensCacheWrite"`
	TokensTotal      int     `json:"tokensTotal"`
	Cost             float64 `json:"cost"`
	ContextUsage     *ContextUsage `json:"contextUsage,omitempty"`
}

// AgentSessionConfig holds the configuration for creating an AgentSession.
type AgentSessionConfig struct {
	Agent          *agent.Agent
	SessionManager *session.SessionManager
	Settings       *settings.SettingsManager
	ModelRegistry  *modelresolver.ModelRegistry
	SkillLoader    *skills.SkillLoader
	Compactor      *compaction.Compactor
	SlashCommands  *slashcommands.Registry
	CWD            string
	AgentDir       string
	ScopedModels   []ai.ModelInfo
}

// AgentSessionEvent represents events emitted by the AgentSession.
type AgentSessionEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
	Error error       `json:"error,omitempty"`
}

// AgentSessionEventListener is a callback for session events.
type AgentSessionEventListener func(event AgentSessionEvent)

// AgentSession is the core runtime that manages the agent lifecycle,
// session persistence, model switching, compaction, auto-retry,
// bash execution, and event routing.
// It's the Go equivalent of the TypeScript AgentSession class.
type AgentSession struct {
	mu             sync.RWMutex
	config         AgentSessionConfig
	activeModel    ai.ModelInfo
	thinkingLevel  string
	listeners      []AgentSessionEventListener
	stats          SessionStats
	steeringMessages  []string
	followUpMessages  []string
	pendingNextTurnMessages []ai.Message

	// Compaction state
	compactionAbortController    context.CancelFunc
	autoCompactionAbortController context.CancelFunc
	overflowRecoveryAttempted    bool

	// Branch summary state
	branchSummaryAbortController context.CancelFunc

	// Retry state
	retryAbortController context.CancelFunc
	retryAttempt         int
	retryDone            chan struct{}
	retryMu              sync.Mutex

	// Bash execution state
	bashAbortController  context.CancelFunc
	pendingBashMessages  []ai.Message

	// Turn tracking
	turnIndex int

	// Last assistant message for auto-compaction check
	lastAssistantMessage *ai.AssistantMessage

	// Base system prompt (without extension appends)
	baseSystemPrompt string

	// Agent event unsubscribe
	unsubscribeAgent func()
}

// NewAgentSession creates a new AgentSession from the given config.
func NewAgentSession(config AgentSessionConfig) *AgentSession {
	as := &AgentSession{
		config:       config,
		activeModel:  config.Agent.Model(),
		thinkingLevel: string(config.Agent.ThinkingLevel()),
	}

	// Subscribe to agent events for internal handling
	as.unsubscribeAgent = config.Agent.Subscribe(func(event agent.AgentEvent) {
		as.handleAgentEvent(event)
	})

	return as
}

// ============================================================================
// Event Subscription
// ============================================================================

// Subscribe adds an event listener. Returns an unsubscribe function.
func (as *AgentSession) Subscribe(listener AgentSessionEventListener) func() {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.listeners = append(as.listeners, listener)
	return func() {
		as.mu.Lock()
		defer as.mu.Unlock()
		for i, l := range as.listeners {
			if &l == &listener {
				as.listeners = append(as.listeners[:i], as.listeners[i+1:]...)
				break
			}
		}
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

func (as *AgentSession) emitQueueUpdate() {
	as.mu.RLock()
	steering := make([]string, len(as.steeringMessages))
	copy(steering, as.steeringMessages)
	followUp := make([]string, len(as.followUpMessages))
	copy(followUp, as.followUpMessages)
	as.mu.RUnlock()

	as.emit(AgentSessionEvent{
		Type: "queue_update",
		Data: map[string]interface{}{
			"steering": steering,
			"followUp": followUp,
		},
	})
}

// ============================================================================
// Agent Event Handling
// ============================================================================

// handleAgentEvent processes agent events for session persistence,
// auto-compaction, auto-retry, and event forwarding.
func (as *AgentSession) handleAgentEvent(event agent.AgentEvent) {
	// Create retry promise synchronously for agent_end before async processing
	if event.Type == agent.EventAgentEnd {
		as.createRetryPromiseForAgentEnd(event)
	}

	// Process the event (session persistence, compaction checks, etc.)
	as.processAgentEvent(event)

	// Forward to session event listeners
	as.emit(AgentSessionEvent{
		Type: string(event.Type),
		Data: event,
	})
}

func (as *AgentSession) createRetryPromiseForAgentEnd(event agent.AgentEvent) {
	if event.Type != agent.EventAgentEnd {
		return
	}
	as.retryMu.Lock()
	defer as.retryMu.Unlock()
	if as.retryDone != nil {
		return // Already have a retry promise
	}
	retrySettings := as.config.Settings.GetRetrySettings()
	if !retrySettings.Enabled {
		return
	}
	lastAssistant := as.findLastAssistantInMessages(event.Messages)
	if lastAssistant == nil || !as.isRetryableError(*lastAssistant) {
		return
	}
	as.retryDone = make(chan struct{})
}

func (as *AgentSession) processAgentEvent(event agent.AgentEvent) {
	switch event.Type {
	case agent.EventMessageStart:
		if msg, ok := event.Message.(ai.UserMessage); ok {
			as.mu.Lock()
			as.overflowRecoveryAttempted = false
			as.mu.Unlock()
			// Check if this message came from steering or follow-up queue
			text := as.getUserMessageText(msg)
			if text != "" {
				as.mu.Lock()
				// Check steering queue first
				idx := -1
				for i, s := range as.steeringMessages {
					if s == text {
						idx = i
						break
					}
				}
				if idx != -1 {
					as.steeringMessages = append(as.steeringMessages[:idx], as.steeringMessages[idx+1:]...)
					as.emitQueueUpdate()
				} else {
					// Check follow-up queue
					idx = -1
					for i, f := range as.followUpMessages {
						if f == text {
							idx = i
							break
						}
					}
					if idx != -1 {
						as.followUpMessages = append(as.followUpMessages[:idx], as.followUpMessages[idx+1:]...)
						as.emitQueueUpdate()
					}
				}
				as.mu.Unlock()
			}
		}

	case agent.EventMessageEnd:
		if event.Message != nil {
			// Persist to session
			if as.config.SessionManager != nil {
				switch event.Message.GetRole() {
				case ai.RoleUser, ai.RoleAssistant, ai.RoleTool:
					as.config.SessionManager.AppendMessage(event.Message)
				}
			}

			// Track assistant message for auto-compaction
			if am, ok := event.Message.(ai.AssistantMessage); ok {
				as.mu.Lock()
				as.lastAssistantMessage = &am
				as.mu.Unlock()

				// Track stats
				as.mu.Lock()
				switch am.GetRole() {
				case ai.RoleAssistant:
					as.stats.AssistantMessages++
					as.stats.TokensInput += am.Usage.Input
					as.stats.TokensOutput += am.Usage.Output
					as.stats.TokensCacheRead += am.Usage.CacheRead
					as.stats.TokensCacheWrite += am.Usage.CacheWrite
					as.stats.Cost += am.Usage.Cost.Total
				case ai.RoleTool:
					as.stats.ToolResults++
				}
				as.mu.Unlock()

				// Reset retry counter on successful response
				if am.StopReason != ai.StopReasonError && as.retryAttempt > 0 {
					as.emit(AgentSessionEvent{
						Type: "auto_retry_end",
						Data: map[string]interface{}{
							"success": true,
							"attempt": as.retryAttempt,
						},
					})
					as.retryAttempt = 0
				}
			}

			// Track user message stats
			if event.Message.GetRole() == ai.RoleUser {
				as.mu.Lock()
				as.stats.UserMessages++
				as.mu.Unlock()
			}
		}

	case agent.EventToolExecutionEnd:
		as.mu.Lock()
		as.stats.ToolCalls++
		as.mu.Unlock()

	case agent.EventAgentEnd:
		// Check auto-retry and auto-compaction
		as.mu.Lock()
		msg := as.lastAssistantMessage
		as.lastAssistantMessage = nil
		as.mu.Unlock()

		if msg != nil {
			// Check retryable errors first
			if as.isRetryableError(*msg) {
				didRetry := as.handleRetryableError(*msg)
				if didRetry {
					return // Retry initiated, don't proceed to compaction
				}
			}
			// Check compaction
			as.checkCompaction(*msg)
		}
		as.resolveRetry()
	}
}

// ============================================================================
// Prompting
// ============================================================================

// Prompt sends a user message through the agent loop.
// It handles session persistence, compaction checks, retry logic, and event routing.
func (as *AgentSession) Prompt(ctx context.Context, text string) error {
	// Check if compaction is needed before sending
	lastAssistant := as.findLastAssistantMessage()
	if lastAssistant != nil {
		as.checkCompaction(*lastAssistant)
	}

	// Flush any pending bash messages before the new prompt
	as.flushPendingBashMessages()

	// Build user message
	userContent := []ai.Content{ai.TextContent{Text: text}}
	userMsg := ai.UserMessage{
		Content:   userContent,
		Timestamp: time.Now().UnixMilli(),
	}

	// Persist to session
	if as.config.SessionManager != nil {
		as.config.SessionManager.AppendMessage(userMsg)
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

// ============================================================================
// Steering & Follow-up
// ============================================================================

// Steer adds a steering message that interrupts the current turn.
func (as *AgentSession) Steer(text string) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.steeringMessages = append(as.steeringMessages, text)
	as.emitQueueUpdate()
	// Forward to agent
	as.config.Agent.Steer(ai.UserMessage{
		Content:   []ai.Content{ai.TextContent{Text: text}},
		Timestamp: time.Now().UnixMilli(),
	})
}

// FollowUp adds a follow-up message that waits for the current turn to complete.
func (as *AgentSession) FollowUp(text string) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.followUpMessages = append(as.followUpMessages, text)
	as.emitQueueUpdate()
	// Forward to agent
	as.config.Agent.FollowUp(ai.UserMessage{
		Content:   []ai.Content{ai.TextContent{Text: text}},
		Timestamp: time.Now().UnixMilli(),
	})
}

// GetSteeringMessages returns and clears pending steering messages.
func (as *AgentSession) GetSteeringMessages() []string {
	as.mu.Lock()
	defer as.mu.Unlock()
	msgs := as.steeringMessages
	as.steeringMessages = nil
	return msgs
}

// GetFollowUpMessages returns and clears pending follow-up messages.
func (as *AgentSession) GetFollowUpMessages() []string {
	as.mu.Lock()
	defer as.mu.Unlock()
	msgs := as.followUpMessages
	as.followUpMessages = nil
	return msgs
}

// ClearQueue clears all queued messages and returns them.
func (as *AgentSession) ClearQueue() (steering []string, followUp []string) {
	as.mu.Lock()
	defer as.mu.Unlock()
	steering = make([]string, len(as.steeringMessages))
	copy(steering, as.steeringMessages)
	followUp = make([]string, len(as.followUpMessages))
	copy(followUp, as.followUpMessages)
	as.steeringMessages = nil
	as.followUpMessages = nil
	as.emitQueueUpdate()
	return
}

// PendingMessageCount returns the number of pending messages.
func (as *AgentSession) PendingMessageCount() int {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return len(as.steeringMessages) + len(as.followUpMessages)
}

// ============================================================================
// Model Management
// ============================================================================

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
	previousModel := as.activeModel
	as.mu.Lock()
	as.activeModel = resolved.Model
	as.thinkingLevel = string(resolved.ThinkingLevel)
	as.mu.Unlock()

	as.config.Agent.SetModel(resolved.Model)
	if resolved.ThinkingLevel != "" {
		as.config.Agent.SetThinkingLevel(ai.ThinkingLevel(resolved.ThinkingLevel))
	}

	// Persist model change to session
	if as.config.SessionManager != nil {
		as.config.SessionManager.AppendModelChange(string(resolved.Model.Provider), resolved.Model.ID)
	}
	if as.config.Settings != nil {
		as.config.Settings.SetDefaultModelAndProvider(string(resolved.Model.Provider), resolved.Model.ID)
	}

	as.emit(AgentSessionEvent{Type: "model_select", Data: map[string]interface{}{
		"model":     resolved.Model,
		"previous":  previousModel,
		"source":    "set",
	}})
	return nil
}

// SetModel sets the active model directly.
func (as *AgentSession) SetModel(model ai.ModelInfo) error {
	previousModel := as.activeModel
	thinkingLevel := as.getThinkingLevelForModelSwitch(nil)

	as.mu.Lock()
	as.activeModel = model
	as.thinkingLevel = thinkingLevel
	as.mu.Unlock()

	as.config.Agent.SetModel(model)

	// Persist
	if as.config.SessionManager != nil {
		as.config.SessionManager.AppendModelChange(string(model.Provider), model.ID)
	}
	if as.config.Settings != nil {
		as.config.Settings.SetDefaultModelAndProvider(string(model.Provider), model.ID)
	}

	as.SetThinkingLevel(thinkingLevel)

	as.emit(AgentSessionEvent{Type: "model_select", Data: map[string]interface{}{
		"model":    model,
		"previous": previousModel,
		"source":   "set",
	}})
	return nil
}

// CycleModel cycles through available models in the registry.
func (as *AgentSession) CycleModel(direction string) *ModelCycleResult {
	// Use scoped models if available
	if len(as.config.ScopedModels) > 0 {
		return as.cycleScopedModel(direction)
	}
	return as.cycleAvailableModel(direction)
}

func (as *AgentSession) cycleScopedModel(direction string) *ModelCycleResult {
	models := as.config.ScopedModels
	if len(models) <= 1 {
		return nil
	}

	currentIdx := -1
	for i, m := range models {
		if ai.ModelsAreEqual(&m, &as.activeModel) {
			currentIdx = i
			break
		}
	}
	if currentIdx == -1 {
		currentIdx = 0
	}

	var nextIdx int
	if direction == "backward" {
		nextIdx = (currentIdx - 1 + len(models)) % len(models)
	} else {
		nextIdx = (currentIdx + 1) % len(models)
	}

	nextModel := models[nextIdx]
	previousModel := as.activeModel

	as.mu.Lock()
	as.activeModel = nextModel
	as.mu.Unlock()
	as.config.Agent.SetModel(nextModel)

	if as.config.SessionManager != nil {
		as.config.SessionManager.AppendModelChange(string(nextModel.Provider), nextModel.ID)
	}
	if as.config.Settings != nil {
		as.config.Settings.SetDefaultModelAndProvider(string(nextModel.Provider), nextModel.ID)
	}

	as.emit(AgentSessionEvent{Type: "model_select", Data: map[string]interface{}{
		"model":    nextModel,
		"previous": previousModel,
		"source":   "cycle",
	}})

	return &ModelCycleResult{
		Model:         nextModel,
		ThinkingLevel: as.thinkingLevel,
		IsScoped:      true,
	}
}

func (as *AgentSession) cycleAvailableModel(direction string) *ModelCycleResult {
	if as.config.ModelRegistry == nil {
		return nil
	}
	models := as.config.ModelRegistry.AllModels()
	if len(models) <= 1 {
		return nil
	}

	currentIdx := -1
	for i, m := range models {
		if m.ID == as.activeModel.ID {
			currentIdx = i
			break
		}
	}
	if currentIdx == -1 {
		currentIdx = 0
	}

	var nextIdx int
	if direction == "backward" {
		nextIdx = (currentIdx - 1 + len(models)) % len(models)
	} else {
		nextIdx = (currentIdx + 1) % len(models)
	}

	nextModel := models[nextIdx]
	previousModel := as.activeModel

	as.mu.Lock()
	as.activeModel = nextModel
	as.mu.Unlock()
	as.config.Agent.SetModel(nextModel)

	if as.config.SessionManager != nil {
		as.config.SessionManager.AppendModelChange(string(nextModel.Provider), nextModel.ID)
	}
	if as.config.Settings != nil {
		as.config.Settings.SetDefaultModelAndProvider(string(nextModel.Provider), nextModel.ID)
	}

	thinkingLevel := as.getThinkingLevelForModelSwitch(nil)
	as.SetThinkingLevel(thinkingLevel)

	as.emit(AgentSessionEvent{Type: "model_select", Data: map[string]interface{}{
		"model":    nextModel,
		"previous": previousModel,
		"source":   "cycle",
	}})

	return &ModelCycleResult{
		Model:         nextModel,
		ThinkingLevel: as.thinkingLevel,
		IsScoped:      false,
	}
}

// SwitchProvider changes the active provider.
func (as *AgentSession) SwitchProvider(providerName string) error {
	as.mu.Lock()
	as.activeModel.Provider = ai.Provider(providerName)
	as.activeModel.API = providerToAPI(ai.Provider(providerName))
	as.mu.Unlock()
	as.config.Agent.SetModel(as.activeModel)
	as.emit(AgentSessionEvent{Type: "provider_switch", Data: providerName})
	return nil
}

// ============================================================================
// Thinking Level Management
// ============================================================================

// SetThinkingLevel changes the thinking level, clamping to model capabilities.
func (as *AgentSession) SetThinkingLevel(level string) {
	availableLevels := as.GetAvailableThinkingLevels()
	effectiveLevel := level
	if !containsString(availableLevels, level) {
		effectiveLevel = as.clampThinkingLevel(level, availableLevels)
	}

	isChanging := effectiveLevel != as.thinkingLevel
	as.mu.Lock()
	as.thinkingLevel = effectiveLevel
	as.mu.Unlock()
	as.config.Agent.SetThinkingLevel(ai.ThinkingLevel(effectiveLevel))

	if isChanging {
		if as.config.SessionManager != nil {
			as.config.SessionManager.AppendThinkingLevelChange(effectiveLevel)
		}
		if as.config.Settings != nil {
			if as.SupportsThinking() || effectiveLevel != ThinkingOff {
				as.config.Settings.SetDefaultThinkingLevel(effectiveLevel)
			}
		}
	}
}

// CycleThinkingLevel cycles to the next thinking level.
func (as *AgentSession) CycleThinkingLevel() *string {
	if !as.SupportsThinking() {
		return nil
	}
	levels := as.GetAvailableThinkingLevels()
	currentIdx := 0
	for i, l := range levels {
		if l == as.thinkingLevel {
			currentIdx = i
			break
		}
	}
	nextIdx := (currentIdx + 1) % len(levels)
	nextLevel := levels[nextIdx]
	as.SetThinkingLevel(nextLevel)
	return &nextLevel
}

// GetAvailableThinkingLevels returns the thinking levels supported by the current model.
func (as *AgentSession) GetAvailableThinkingLevels() []string {
	if !as.SupportsThinking() {
		return []string{ThinkingOff}
	}
	if as.SupportsXhighThinking() {
		return ThinkingLevelsWithXHigh
	}
	return ThinkingLevels
}

// SupportsThinking checks if the current model supports reasoning/thinking.
func (as *AgentSession) SupportsThinking() bool {
	return as.activeModel.Reasoning
}

// SupportsXhighThinking checks if the current model supports xhigh thinking.
func (as *AgentSession) SupportsXhighThinking() bool {
	return ai.SupportsXHigh(as.activeModel)
}

// ThinkingLevel returns the current thinking level.
func (as *AgentSession) ThinkingLevel() string {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.thinkingLevel
}

// Model returns the current active model.
func (as *AgentSession) Model() ai.ModelInfo {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.activeModel
}

// IsStreaming returns whether the agent is currently streaming.
func (as *AgentSession) IsStreaming() bool {
	return as.config.Agent.IsStreaming()
}

// getThinkingLevelForModelSwitch determines the appropriate thinking level
// when switching models.
func (as *AgentSession) getThinkingLevelForModelSwitch(explicitLevel *string) string {
	if explicitLevel != nil {
		return *explicitLevel
	}
	if !as.SupportsThinking() {
		if as.config.Settings != nil {
			if level := as.config.Settings.GetDefaultThinkingLevel(); level != "" {
				return level
			}
		}
		return ThinkingOff
	}
	return as.thinkingLevel
}

func (as *AgentSession) clampThinkingLevel(level string, available []string) string {
	ordered := ThinkingLevelsWithXHigh
	availableSet := make(map[string]bool)
	for _, l := range available {
		availableSet[l] = true
	}

	requestedIdx := -1
	for i, l := range ordered {
		if l == level {
			requestedIdx = i
			break
		}
	}
	if requestedIdx == -1 {
		if len(available) > 0 {
			return available[0]
		}
		return ThinkingOff
	}

	// Try upward first
	for i := requestedIdx; i < len(ordered); i++ {
		if availableSet[ordered[i]] {
			return ordered[i]
		}
	}
	// Then downward
	for i := requestedIdx - 1; i >= 0; i-- {
		if availableSet[ordered[i]] {
			return ordered[i]
		}
	}
	if len(available) > 0 {
		return available[0]
	}
	return ThinkingOff
}

// ============================================================================
// Compaction
// ============================================================================

// Compact manually triggers context compaction.
func (as *AgentSession) Compact(ctx context.Context) error {
	if as.config.Compactor == nil {
		return fmt.Errorf("compactor not available")
	}

	as.emit(AgentSessionEvent{Type: "compaction_start", Data: CompactionManual})

	compactCtx, cancel := context.WithCancel(ctx)
	as.mu.Lock()
	as.compactionAbortController = cancel
	as.mu.Unlock()

	defer func() {
		as.mu.Lock()
		as.compactionAbortController = nil
		as.mu.Unlock()
	}()

	messages := as.config.Agent.Messages()
	compacted, err := as.config.Compactor.Compact(compactCtx, messages)
	if err != nil {
		as.emit(AgentSessionEvent{
			Type:  "compaction_end",
			Data:  CompactionManual,
			Error: err,
		})
		return err
	}

	as.config.Agent.SetMessages(compacted)
	as.mu.Lock()
	as.mu.Unlock()

	// Persist compaction to session
	if as.config.SessionManager != nil {
		as.config.SessionManager.AppendCompaction("Manual compaction", "", 0, nil, nil)
	}

	as.emit(AgentSessionEvent{Type: "compaction_end", Data: CompactionManual})
	return nil
}

// AbortCompaction cancels in-progress compaction.
func (as *AgentSession) AbortCompaction() {
	as.mu.Lock()
	if as.compactionAbortController != nil {
		as.compactionAbortController()
		as.compactionAbortController = nil
	}
	if as.autoCompactionAbortController != nil {
		as.autoCompactionAbortController()
		as.autoCompactionAbortController = nil
	}
	as.mu.Unlock()
}

// IsCompacting returns whether compaction is currently running.
func (as *AgentSession) IsCompacting() bool {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.compactionAbortController != nil || as.autoCompactionAbortController != nil || as.branchSummaryAbortController != nil
}

// checkCompaction checks if compaction is needed and runs it.
func (as *AgentSession) checkCompaction(assistantMessage ai.AssistantMessage) {
	compactionSettings := as.config.Settings.GetCompactionSettings()
	if !compactionSettings.Enabled {
		return
	}
	if assistantMessage.StopReason == ai.StopReasonAborted {
		return
	}

	contextWindow := as.activeModel.ContextWindow

	// Check overflow
	sameModel := as.activeModel.Provider == assistantMessage.Provider && as.activeModel.ID == assistantMessage.Model
	if sameModel && ai.IsContextOverflow(assistantMessage, contextWindow) {
		as.mu.Lock()
		if as.overflowRecoveryAttempted {
			as.mu.Unlock()
			as.emit(AgentSessionEvent{
				Type: "compaction_end",
				Data: CompactionOverflow,
				Error: fmt.Errorf("context overflow recovery failed after one compact-and-retry attempt"),
			})
			return
		}
		as.overflowRecoveryAttempted = true
		as.mu.Unlock()

		// Remove error message from agent state
		messages := as.config.Agent.Messages()
		if len(messages) > 0 {
			if last, ok := messages[len(messages)-1].(ai.AssistantMessage); ok && last.GetRole() == ai.RoleAssistant {
				as.config.Agent.SetMessages(messages[:len(messages)-1])
			}
		}

		as.runAutoCompaction(CompactionOverflow, true)
		return
	}

	// Check threshold
	var contextTokens int
	if assistantMessage.StopReason == ai.StopReasonError {
		estimate := ai.EstimateContextTokens(as.config.Agent.Messages())
		if estimate.LastUsageIndex == nil {
			return
		}
		contextTokens = estimate.Tokens
	} else {
		contextTokens = ai.CalculateContextTokens(assistantMessage.Usage)
	}

	if shouldCompact(contextTokens, contextWindow, compactionSettings.Threshold) {
		as.runAutoCompaction(CompactionThreshold, false)
	}
}

func (as *AgentSession) runAutoCompaction(reason CompactionReason, willRetry bool) {
	as.emit(AgentSessionEvent{Type: "compaction_start", Data: reason})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	as.mu.Lock()
	as.autoCompactionAbortController = cancel
	as.mu.Unlock()

	go func() {
		defer func() {
			as.mu.Lock()
			as.autoCompactionAbortController = nil
			as.mu.Unlock()
		}()

		if as.config.Compactor == nil {
			as.emit(AgentSessionEvent{Type: "compaction_end", Data: reason})
			return
		}

		messages := as.config.Agent.Messages()
		compacted, err := as.config.Compactor.Compact(ctx, messages)
		if err != nil {
			as.emit(AgentSessionEvent{
				Type:  "compaction_end",
				Data:  reason,
				Error: err,
			})
			return
		}

		as.config.Agent.SetMessages(compacted)
		as.mu.Lock()
			as.mu.Unlock()

		// Persist
		if as.config.SessionManager != nil {
			summary := "Auto-compaction"
			if reason == CompactionOverflow {
				summary = "Overflow recovery compaction"
			}
			as.config.SessionManager.AppendCompaction(summary, "", 0, nil, nil)
		}

		as.emit(AgentSessionEvent{Type: "compaction_end", Data: map[string]interface{}{
			"reason":    reason,
			"willRetry": willRetry,
		}})

		if willRetry {
			// Remove the error message if it's still the last one
			msgs := as.config.Agent.Messages()
			if len(msgs) > 0 {
				if last, ok := msgs[len(msgs)-1].(ai.AssistantMessage); ok && last.StopReason == ai.StopReasonError {
					as.config.Agent.SetMessages(msgs[:len(msgs)-1])
				}
			}
			go func() {
				time.Sleep(100 * time.Millisecond)
				as.config.Agent.Continue()
			}()
		}
	}()
}

// SetAutoCompactionEnabled toggles auto-compaction.
func (as *AgentSession) SetAutoCompactionEnabled(enabled bool) {
	if as.config.Settings != nil {
		as.config.Settings.SetCompactionEnabled(enabled)
	}
}

// AutoCompactionEnabled returns whether auto-compaction is enabled.
func (as *AgentSession) AutoCompactionEnabled() bool {
	if as.config.Settings == nil {
		return true
	}
	return as.config.Settings.GetCompactionEnabled()
}

// ============================================================================
// Auto-Retry
// ============================================================================

// isRetryableError checks if the assistant message has a retryable error.
func (as *AgentSession) isRetryableError(message ai.AssistantMessage) bool {
	contextWindow := as.activeModel.ContextWindow
	return ai.IsRetryableError(message, contextWindow)
}

// handleRetryableError handles retryable errors with exponential backoff.
// Returns true if retry was initiated.
func (as *AgentSession) handleRetryableError(message ai.AssistantMessage) bool {
	retrySettings := as.config.Settings.GetRetrySettings()
	if !retrySettings.Enabled {
		as.resolveRetry()
		return false
	}

	as.retryMu.Lock()
	if as.retryDone == nil {
		as.retryDone = make(chan struct{})
	}
	as.retryMu.Unlock()

	as.retryAttempt++
	if as.retryAttempt > retrySettings.MaxRetries {
		// Max retries exceeded
		as.emit(AgentSessionEvent{
			Type: "auto_retry_end",
			Data: map[string]interface{}{
				"success":     false,
				"attempt":     as.retryAttempt - 1,
				"finalError":  message.ErrorMessage,
			},
		})
		as.retryAttempt = 0
		as.resolveRetry()
		return false
	}

	delayMs := retrySettings.BaseDelayMs * (1 << (as.retryAttempt - 1))
	errMsg := ""
	if message.ErrorMessage != nil {
		errMsg = *message.ErrorMessage
	}

	as.emit(AgentSessionEvent{
		Type: "auto_retry_start",
		Data: map[string]interface{}{
			"attempt":      as.retryAttempt,
			"maxAttempts":  retrySettings.MaxRetries,
			"delayMs":      delayMs,
			"errorMessage": errMsg,
		},
	})

	// Remove error message from agent state
	messages := as.config.Agent.Messages()
	if len(messages) > 0 {
		if last, ok := messages[len(messages)-1].(ai.AssistantMessage); ok && last.GetRole() == ai.RoleAssistant {
			as.config.Agent.SetMessages(messages[:len(messages)-1])
		}
	}

	// Wait with exponential backoff then retry
	go func() {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
		as.config.Agent.Continue()
	}()

	return true
}

// AbortRetry cancels in-progress retry.
func (as *AgentSession) AbortRetry() {
	as.retryMu.Lock()
	defer as.retryMu.Unlock()
	if as.retryAbortController != nil {
		as.retryAbortController()
		as.retryAbortController = nil
	}
	attempt := as.retryAttempt
	as.retryAttempt = 0
	as.emit(AgentSessionEvent{
		Type: "auto_retry_end",
		Data: map[string]interface{}{
			"success":     false,
			"attempt":     attempt,
			"finalError":  "Retry cancelled",
		},
	})
	as.resolveRetry()
}

func (as *AgentSession) resolveRetry() {
	as.retryMu.Lock()
	defer as.retryMu.Unlock()
	if as.retryDone != nil {
		close(as.retryDone)
		as.retryDone = nil
	}
}

// WaitForRetry waits for any in-progress retry to complete.
func (as *AgentSession) WaitForRetry() {
	as.retryMu.Lock()
	ch := as.retryDone
	as.retryMu.Unlock()
	if ch != nil {
		<-ch
	}
}

// IsRetrying returns whether auto-retry is in progress.
func (as *AgentSession) IsRetrying() bool {
	as.retryMu.Lock()
	defer as.retryMu.Unlock()
	return as.retryDone != nil
}

// RetryAttempt returns the current retry attempt number (0 if not retrying).
func (as *AgentSession) RetryAttempt() int {
	as.retryMu.Lock()
	defer as.retryMu.Unlock()
	return as.retryAttempt
}

// SetAutoRetryEnabled toggles auto-retry.
func (as *AgentSession) SetAutoRetryEnabled(enabled bool) {
	if as.config.Settings != nil {
		as.config.Settings.SetRetryEnabled(enabled)
	}
}

// AutoRetryEnabled returns whether auto-retry is enabled.
func (as *AgentSession) AutoRetryEnabled() bool {
	if as.config.Settings == nil {
		return true
	}
	return as.config.Settings.GetRetryEnabled()
}

// ============================================================================
// Bash Execution
// ============================================================================

// ExecuteBash executes a bash command and records the result.
func (as *AgentSession) ExecuteBash(ctx context.Context, command string) (string, error) {
	bashCtx, cancel := context.WithCancel(ctx)
	as.mu.Lock()
	as.bashAbortController = cancel
	as.mu.Unlock()

	defer func() {
		as.mu.Lock()
		as.bashAbortController = nil
		as.mu.Unlock()
	}()

	// Apply command prefix if configured
	prefix := ""
	if as.config.Settings != nil {
		prefix = as.config.Settings.GetShellCommandPrefix()
	}
	resolvedCommand := command
	if prefix != "" {
		resolvedCommand = prefix + "\n" + command
	}

	// Execute the command via the agent's bash tool
	result, err := as.config.Agent.ExecuteToolCall(bashCtx, "bash", map[string]interface{}{
		"command": resolvedCommand,
	})
	if err != nil {
		return "", err
	}

	// Extract text output
	var output strings.Builder
	if result != nil {
		for _, c := range result.Content {
			if tc, ok := c.(ai.TextContent); ok {
				output.WriteString(tc.Text)
			}
		}
	}

	return output.String(), nil
}

// AbortBash cancels a running bash command.
func (as *AgentSession) AbortBash() {
	as.mu.Lock()
	defer as.mu.Unlock()
	if as.bashAbortController != nil {
		as.bashAbortController()
		as.bashAbortController = nil
	}
}

// IsBashRunning returns whether a bash command is currently running.
func (as *AgentSession) IsBashRunning() bool {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.bashAbortController != nil
}

func (as *AgentSession) flushPendingBashMessages() {
	as.mu.Lock()
	pending := as.pendingBashMessages
	as.pendingBashMessages = nil
	as.mu.Unlock()

	if len(pending) == 0 {
		return
	}
	messages := as.config.Agent.Messages()
	messages = append(messages, pending...)
	as.config.Agent.SetMessages(messages)
	for _, msg := range pending {
		if as.config.SessionManager != nil {
			as.config.SessionManager.AppendMessage(msg)
		}
	}
}

// ============================================================================
// Session Management
// ============================================================================

// NewSession creates a fresh session, resetting conversation history.
func (as *AgentSession) NewSession() {
	as.config.Agent.SetMessages(nil)
	if as.config.SessionManager != nil {
		as.config.SessionManager = session.CreateSession(as.config.CWD, as.config.Settings.GetSessionDir())
	}
	as.mu.Lock()
	as.stats = SessionStats{}
	as.steeringMessages = nil
	as.followUpMessages = nil
	as.pendingNextTurnMessages = nil
	as.overflowRecoveryAttempted = false
	as.retryAttempt = 0
	as.turnIndex = 0
	as.lastAssistantMessage = nil
	as.mu.Unlock()
	as.emit(AgentSessionEvent{Type: "new_session"})
}

// SetSessionName sets a display name for the current session.
func (as *AgentSession) SetSessionName(name string) {
	if as.config.SessionManager != nil {
		as.config.SessionManager.AppendSessionInfo(name)
	}
}

// SessionFile returns the current session file path.
func (as *AgentSession) SessionFile() string {
	if as.config.SessionManager != nil {
		if f := as.config.SessionManager.GetSessionFile(); f != nil { return *f }; return ""
	}
	return ""
}

// SessionID returns the current session ID.
func (as *AgentSession) SessionID() string {
	if as.config.SessionManager != nil {
		return as.config.SessionManager.GetSessionID()
	}
	return ""
}

// SetScopedModels updates the scoped models for cycling.
func (as *AgentSession) SetScopedModels(models []ai.ModelInfo) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.config.ScopedModels = models
}

// ============================================================================
// Tree Navigation
// ============================================================================

// NavigateTree navigates to a different node in the session tree.
func (as *AgentSession) NavigateTree(targetID string, summarize bool) error {
	if as.config.SessionManager == nil {
		return fmt.Errorf("session manager not available")
	}

	oldLeafID := as.config.SessionManager.GetLeafID()
	if oldLeafID != nil && targetID == *oldLeafID {
		return nil // Already at target
	}

	// If summarizing, generate a branch summary
	if summarize {
		// TODO: Implement branch summarization with LLM
		// For now, create a simple text summary
		summary := "Branch summary (navigation)"
		targetIDPtr := &targetID; as.config.SessionManager.BranchWithSummary(targetIDPtr, summary, nil, nil)
	} else {
		// Navigate without summary
		if targetID == "" {
			as.config.SessionManager.ResetLeaf()
		} else {
			as.config.SessionManager.Branch(targetID)
		}
	}

	// Update agent state
	sessionContext := as.config.SessionManager.BuildSessionContext()
	if sessionContext != nil {
		as.config.Agent.SetMessages(sessionContext.Messages)
	}

	as.emit(AgentSessionEvent{Type: "session_tree", Data: map[string]interface{}{
		"newLeafId": func() string { if l := as.config.SessionManager.GetLeafID(); l != nil { return *l }; return "" }(),
		"oldLeafId": oldLeafID,
	}})
	return nil
}

// GetUserMessagesForForking returns all user messages for fork selection.
func (as *AgentSession) GetUserMessagesForForking() []struct {
	EntryID string
	Text    string
} {
	if as.config.SessionManager == nil {
		return nil
	}
	// TODO: Implement with session manager getEntries
	return nil
}

// ============================================================================
// Reload
// ============================================================================

// Reload refreshes skills, settings, and system prompt.
func (as *AgentSession) Reload() error {
	if as.config.Settings != nil {
		as.config.Settings.Reload()
	}
	if as.config.SkillLoader != nil {
		as.config.SkillLoader.ClearCache()
	}

	// Rebuild system prompt
	as.rebuildSystemPrompt()
	as.emit(AgentSessionEvent{Type: "reload"})
	return nil
}

func (as *AgentSession) rebuildSystemPrompt() {
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
	as.baseSystemPrompt = prompt
	as.config.Agent.SetSystemPrompt(prompt)
}

// ============================================================================
// Read-only State Access
// ============================================================================

// Stats returns the current session statistics.
func (as *AgentSession) Stats() SessionStats {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.stats
}

// GetSessionStats computes full session statistics from agent state.
func (as *AgentSession) GetSessionStats() SessionStats {
	messages := as.config.Agent.Messages()
	userMsgs := 0
	assistantMsgs := 0
	toolCalls := 0
	toolResults := 0
	var totalInput, totalOutput, totalCacheRead, totalCacheWrite int
	var totalCost float64

	for _, msg := range messages {
		switch msg.GetRole() {
		case ai.RoleUser:
			userMsgs++
		case ai.RoleAssistant:
			assistantMsgs++
			if am, ok := msg.(ai.AssistantMessage); ok {
				for _, c := range am.Content {
					if _, ok := c.(ai.ToolCall); ok {
						toolCalls++
					}
				}
				totalInput += am.Usage.Input
				totalOutput += am.Usage.Output
				totalCacheRead += am.Usage.CacheRead
				totalCacheWrite += am.Usage.CacheWrite
				totalCost += am.Usage.Cost.Total
			}
		case ai.RoleTool:
			toolResults++
		}
	}

	return SessionStats{
		SessionFile:      as.SessionFile(),
		SessionID:        as.SessionID(),
		UserMessages:     userMsgs,
		AssistantMessages: assistantMsgs,
		ToolCalls:        toolCalls,
		ToolResults:      toolResults,
		TotalMessages:    len(messages),
		TokensInput:      totalInput,
		TokensOutput:     totalOutput,
		TokensCacheRead:  totalCacheRead,
		TokensCacheWrite: totalCacheWrite,
		TokensTotal:      totalInput + totalOutput + totalCacheRead + totalCacheWrite,
		Cost:             totalCost,
		ContextUsage:     as.GetContextUsage(),
	}
}

// GetContextUsage returns context window utilization info.
func (as *AgentSession) GetContextUsage() *ContextUsage {
	model := as.activeModel
	contextWindow := model.ContextWindow
	if contextWindow <= 0 {
		return nil
	}

	estimate := ai.EstimateContextTokens(as.config.Agent.Messages())
	if estimate.LastUsageIndex == nil {
		return &ContextUsage{
			Tokens:        nil,
			ContextWindow: contextWindow,
			Percent:       nil,
		}
	}

	tokens := estimate.Tokens
	percent := (float64(tokens) / float64(contextWindow)) * 100
	return &ContextUsage{
		Tokens:        &tokens,
		ContextWindow: contextWindow,
		Percent:       &percent,
	}
}

// Session returns the session manager.
func (as *AgentSession) Session() *session.SessionManager {
	return as.config.SessionManager
}

// Agent returns the underlying agent.
func (as *AgentSession) Agent() *agent.Agent {
	return as.config.Agent
}

// Abort aborts current operation and waits for agent to become idle.
func (as *AgentSession) Abort() {
	as.AbortRetry()
	as.config.Agent.Abort()
}

// Dispose removes all listeners and disconnects from agent.


// ExportToHTML exports the current session as an HTML file.
// If outputPath is empty, generates a timestamped file in the CWD.
func (as *AgentSession) ExportToHTML(outputPath string) (string, error) {
	if outputPath == "" {
		timestamp := time.Now().Format("2006-01-02-15-04-05")
		outputPath = filepath.Join(as.config.CWD, fmt.Sprintf("pi-session-%s.html", timestamp))
	}

	messages := as.config.Agent.Messages()
	title := "Pi Session"
	if id := as.SessionID(); id != "" {
		title = fmt.Sprintf("Pi Session - %s", id)
	}

	theme := "dark"
	if as.config.Settings != nil {
		theme = as.config.Settings.GetTheme()
	}

	options := export.ExportHTMLOptions{
		Title: title,
		Theme: theme,
	}

	if err := export.ExportHTML(messages, outputPath, options); err != nil {
		return "", fmt.Errorf("failed to export HTML: %w", err)
	}

	return outputPath, nil
}

func (as *AgentSession) Dispose() {
	if as.unsubscribeAgent != nil {
		as.unsubscribeAgent()
		as.unsubscribeAgent = nil
	}
	as.mu.Lock()
	as.listeners = nil
	as.mu.Unlock()
}

// GetActiveToolNames returns the names of currently active tools.
func (as *AgentSession) GetActiveToolNames() []string {
	var names []string
	for _, t := range as.config.Agent.Tools() {
		names = append(names, t.Name)
	}
	return names
}

// LastAssistantText returns text content of the last assistant message.
func (as *AgentSession) LastAssistantText() string {
	messages := as.config.Agent.Messages()
	for i := len(messages) - 1; i >= 0; i-- {
		if am, ok := messages[i].(ai.AssistantMessage); ok {
			if am.StopReason == ai.StopReasonAborted && len(am.Content) == 0 {
				continue
			}
			var text string
			for _, c := range am.Content {
				if tc, ok := c.(ai.TextContent); ok {
					text += tc.Text
				}
			}
			return strings.TrimSpace(text)
		}
	}
	return ""
}

// ============================================================================
// Skill Block Parsing
// ============================================================================

var skillBlockRegex = regexp.MustCompile(`^<skill name="([^"]+)" location="([^"]+)">\n([\s\S]*?)\n</skill>(?:\n\n([\s\S]+))?$`)

// ParseSkillBlock parses a skill block from message text.
// Returns nil if the text doesn't contain a skill block.
func ParseSkillBlock(text string) *ParsedSkillBlock {
	match := skillBlockRegex.FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	result := &ParsedSkillBlock{
		Name:     match[1],
		Location: match[2],
		Content:  match[3],
	}
	if match[4] != "" {
		result.UserMessage = strings.TrimSpace(match[4])
	}
	return result
}

// ============================================================================
// Internal Helpers
// ============================================================================

func (as *AgentSession) findLastAssistantMessage() *ai.AssistantMessage {
	messages := as.config.Agent.Messages()
	for i := len(messages) - 1; i >= 0; i-- {
		if am, ok := messages[i].(ai.AssistantMessage); ok {
			return &am
		}
	}
	return nil
}

func (as *AgentSession) findLastAssistantInMessages(messages []ai.Message) *ai.AssistantMessage {
	for i := len(messages) - 1; i >= 0; i-- {
		if am, ok := messages[i].(ai.AssistantMessage); ok {
			return &am
		}
	}
	return nil
}

func (as *AgentSession) getUserMessageText(message ai.Message) string {
	if um, ok := message.(ai.UserMessage); ok {
		var parts []string
		for _, c := range um.Content {
			if tc, ok := c.(ai.TextContent); ok {
				parts = append(parts, tc.Text)
			}
		}
		return strings.Join(parts, "")
	}
	return ""
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

// shouldCompact checks if compaction should be triggered based on threshold.
func shouldCompact(contextTokens, contextWindow int, threshold float64) bool {
	if contextWindow <= 0 {
		return false
	}
	if threshold <= 0 {
		threshold = 0.8
	}
	return float64(contextTokens)/float64(contextWindow) >= threshold
}

// containsString checks if a string slice contains a value.
func containsString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

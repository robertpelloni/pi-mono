package agent

import (
	"context"

	"github.com/badlogic/pi-mono/pkg/ai"
)

// ToolExecutionMode specifies how tool calls should be executed.
type ToolExecutionMode string

const (
	ToolExecutionSequential ToolExecutionMode = "sequential"
	ToolExecutionParallel   ToolExecutionMode = "parallel"
)

// AgentContext represents a snapshot passed into the low-level agent loop.
type AgentContext struct {
	SystemPrompt string
	Messages     []ai.Message
	Tools        []AgentTool
}

// BeforeToolCallContext provides contextual information to BeforeToolCall hooks.
type BeforeToolCallContext struct {
	AssistantMessage ai.AssistantMessage
	ToolCall         ai.ToolCall
	Args             map[string]any
	AgentContext     AgentContext
}

type BeforeToolCallResult struct {
	Block bool
}

// AfterToolCallContext provides contextual information to AfterToolCall hooks.
type AfterToolCallContext struct {
	AssistantMessage ai.AssistantMessage
	ToolCall         ai.ToolCall
	Args             map[string]any
	Result           AgentToolResult
	IsError          bool
	AgentContext     AgentContext
}

type AfterToolCallResult struct {
	Content []ai.Content
	Details any
	IsError *bool
}

// AgentLoopConfig specifies configuration options for the underlying execution loop.
type AgentLoopConfig struct {
	ai.SimpleStreamOptions
	Model ai.ModelInfo

	// ConvertToLlm translates higher-level application messages into pure LLM messages
	ConvertToLlm func(messages []ai.Message) ([]ai.Message, error)

	// TransformContext modifies the message array (e.g. context window pruning)
	TransformContext func(ctx context.Context, messages []ai.Message) ([]ai.Message, error)

	// GetApiKey fetches dynamic API keys if needed (e.g. short-lived tokens)
	GetApiKey func(provider ai.Provider) (string, error)

	// GetSteeringMessages injects messages mid-run after a tool execution turn
	GetSteeringMessages func() ([]ai.Message, error)

	// GetFollowUpMessages injects messages when the agent would otherwise stop
	GetFollowUpMessages func() ([]ai.Message, error)

	ToolExecution ToolExecutionMode

	BeforeToolCall func(ctx context.Context, callCtx BeforeToolCallContext) (*BeforeToolCallResult, error)
	AfterToolCall  func(ctx context.Context, callCtx AfterToolCallContext) (*AfterToolCallResult, error)
}

// AgentState represents the public properties and configuration of the Agent instance.
type AgentState interface {
	SystemPrompt() string
	SetSystemPrompt(string)

	Model() ai.ModelInfo
	SetModel(ai.ModelInfo)

	ThinkingLevel() ai.ThinkingLevel
	SetThinkingLevel(ai.ThinkingLevel)

	Tools() []AgentTool
	SetTools([]AgentTool)

	Messages() []ai.Message
	SetMessages([]ai.Message)

	IsStreaming() bool
	StreamingMessage() *ai.AssistantMessage
	PendingToolCalls() map[string]struct{}
	ErrorMessage() string
}

// Event Types mapped from TypeScript definitions
type EventType string

const (
	EventAgentStart          EventType = "agent_start"
	EventAgentEnd            EventType = "agent_end"
	EventTurnStart           EventType = "turn_start"
	EventTurnEnd             EventType = "turn_end"
	EventMessageStart        EventType = "message_start"
	EventMessageUpdate       EventType = "message_update"
	EventMessageEnd          EventType = "message_end"
	EventToolExecutionStart  EventType = "tool_execution_start"
	EventToolExecutionUpdate EventType = "tool_execution_update"
	EventToolExecutionEnd    EventType = "tool_execution_end"
)

// AgentEvent represents UI updates dispatched by the Agent.
type AgentEvent struct {
	Type EventType

	Messages              []ai.Message              // Used by EventAgentEnd
	Message               ai.Message                // Used by Turn/Message events
	ToolResults           []ai.ToolResultMessage    // Used by EventTurnEnd
	AssistantMessageEvent *ai.AssistantMessageEvent // Used by EventMessageUpdate

	ToolCallID    string         // Used by ToolExecution events
	ToolName      string         // Used by ToolExecution events
	Args          map[string]any // Used by ToolExecution events
	PartialResult any            // Used by EventToolExecutionUpdate
	Result        any            // Used by EventToolExecutionEnd
	IsError       bool           // Used by EventToolExecutionEnd
}

// AgentEventListener handles AgentEvent dispatches.
type AgentEventListener func(event AgentEvent)

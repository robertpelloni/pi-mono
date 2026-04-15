package ai

// AssistantMessageEventType represents the type of an event in the stream.
type AssistantMessageEventType string

const (
	EventStart         AssistantMessageEventType = "start"
	EventTextStart     AssistantMessageEventType = "text_start"
	EventTextDelta     AssistantMessageEventType = "text_delta"
	EventTextEnd       AssistantMessageEventType = "text_end"
	EventThinkingStart AssistantMessageEventType = "thinking_start"
	EventThinkingDelta AssistantMessageEventType = "thinking_delta"
	EventThinkingEnd   AssistantMessageEventType = "thinking_end"
	EventToolCallStart AssistantMessageEventType = "toolcall_start"
	EventToolCallDelta AssistantMessageEventType = "toolcall_delta"
	EventToolCallEnd   AssistantMessageEventType = "toolcall_end"
	EventDone          AssistantMessageEventType = "done"
	EventError         AssistantMessageEventType = "error"
)

// AssistantMessageEvent represents a single event emitted during a streaming API call.
type AssistantMessageEvent struct {
	Type         AssistantMessageEventType `json:"type"`
	ContentIndex *int                      `json:"contentIndex,omitempty"`
	Delta        *string                   `json:"delta,omitempty"`
	Content      *string                   `json:"content,omitempty"`
	Partial      *AssistantMessage         `json:"partial,omitempty"`
	ToolCall     *ToolCall                 `json:"toolCall,omitempty"`
	Reason       *StopReason               `json:"reason,omitempty"`
	Message      *AssistantMessage         `json:"message,omitempty"`
	Error        *AssistantMessage         `json:"error,omitempty"`
}

// AssistantMessageEventStream represents a stream of AssistantMessageEvents.
// In Go, this is typically handled via a channel that emits events and can be closed.
type AssistantMessageEventStream <-chan AssistantMessageEvent

// StreamFunction represents the generic function signature for all AI provider streaming calls.
// TOptions would typically be an interface in Go, but we pass any and rely on the provider implementation
// to assert the correct options type (e.g., OpenAIResponsesOptions).
type StreamFunction func(model ModelInfo, context Context, options any) AssistantMessageEventStream

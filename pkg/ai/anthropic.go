package ai

type AnthropicEffort string

const (
	AnthropicEffortLow    AnthropicEffort = "low"
	AnthropicEffortMedium AnthropicEffort = "medium"
	AnthropicEffortHigh   AnthropicEffort = "high"
	AnthropicEffortMax    AnthropicEffort = "max"
)

// AnthropicToolChoice defines the choice of tool usage for Anthropic models.
// It can be "auto", "any", "none", or a struct for a specific tool.
type AnthropicToolChoice struct {
	Type *string `json:"type,omitempty"` // e.g. "auto", "any", "none", "tool"
	Name *string `json:"name,omitempty"` // used if Type == "tool"
}

// AnthropicOptions represents the specific options available when calling the Anthropic Messages API.
// It embeds the generic StreamOptions.
type AnthropicOptions struct {
	StreamOptions
	// Enable extended thinking.
	ThinkingEnabled *bool `json:"thinkingEnabled,omitempty"`
	// Token budget for extended thinking (older models only).
	ThinkingBudgetTokens *int `json:"thinkingBudgetTokens,omitempty"`
	// Effort level for adaptive thinking (Opus 4.6 and Sonnet 4.6).
	Effort              *AnthropicEffort     `json:"effort,omitempty"`
	InterleavedThinking *bool                `json:"interleavedThinking,omitempty"`
	ToolChoice          *AnthropicToolChoice `json:"toolChoice,omitempty"`
	// Note: Pre-built Anthropic client instance is skipped in Go as it would rely on a Go-specific SDK client.
}

// StreamAnthropic is the StreamFunction implementation for the Anthropic Messages API.
func StreamAnthropic(model ModelInfo, context Context, options any) AssistantMessageEventStream {
	// Cast options to AnthropicOptions, falling back to empty if not provided or wrong type
	_, ok := options.(AnthropicOptions)
	if !ok {
		_ = AnthropicOptions{}
	}

	stream := make(chan AssistantMessageEvent)

	// TODO: Implement the actual Anthropic streaming logic here.
	// For now, this is a placeholder stub to define the architecture.
	go func() {
		defer close(stream)

		errMsg := "Anthropic provider streaming is not yet fully implemented in Go port."
		reason := StopReasonError
		stream <- AssistantMessageEvent{
			Type:   EventError,
			Reason: &reason,
			Error: &AssistantMessage{
				API:          model.API,
				Provider:     model.Provider,
				Model:        model.ID,
				StopReason:   reason,
				ErrorMessage: &errMsg,
			},
		}
	}()

	return stream
}

package ai

type GoogleThinkingLevel string

const (
	GoogleThinkingUnspecified GoogleThinkingLevel = "THINKING_LEVEL_UNSPECIFIED"
	GoogleThinkingMinimal     GoogleThinkingLevel = "MINIMAL"
	GoogleThinkingLow         GoogleThinkingLevel = "LOW"
	GoogleThinkingMedium      GoogleThinkingLevel = "MEDIUM"
	GoogleThinkingHigh        GoogleThinkingLevel = "HIGH"
)

type GoogleThinkingConfig struct {
	Enabled      bool                 `json:"enabled"`
	BudgetTokens *int                 `json:"budgetTokens,omitempty"` // -1 for dynamic, 0 to disable
	Level        *GoogleThinkingLevel `json:"level,omitempty"`
}

type GoogleToolChoice string

const (
	GoogleToolChoiceAuto GoogleToolChoice = "auto"
	GoogleToolChoiceNone GoogleToolChoice = "none"
	GoogleToolChoiceAny  GoogleToolChoice = "any"
)

// GoogleOptions represents the options available when calling the public Google Generative AI API.
type GoogleOptions struct {
	StreamOptions
	ToolChoice *GoogleToolChoice     `json:"toolChoice,omitempty"`
	Thinking   *GoogleThinkingConfig `json:"thinking,omitempty"`
}

// GoogleGeminiCliOptions represents the options available when calling the Google Gemini internal CLI API.
type GoogleGeminiCliOptions struct {
	StreamOptions
	ToolChoice *GoogleToolChoice     `json:"toolChoice,omitempty"`
	Thinking   *GoogleThinkingConfig `json:"thinking,omitempty"`
	ProjectID  *string               `json:"projectId,omitempty"`
}

// GoogleVertexOptions represents the options available when calling the Google Vertex AI API.
type GoogleVertexOptions struct {
	StreamOptions
	ToolChoice *GoogleToolChoice     `json:"toolChoice,omitempty"`
	Thinking   *GoogleThinkingConfig `json:"thinking,omitempty"`
	Project    *string               `json:"project,omitempty"`
	Location   *string               `json:"location,omitempty"`
}

// StreamGoogle is the StreamFunction implementation for the Google Generative AI API.
func StreamGoogle(model ModelInfo, context Context, options any) AssistantMessageEventStream {
	// Validate options type
	_, ok := options.(GoogleOptions)
	if !ok && options != nil {
		// Log or handle invalid options type if necessary in future implementations
	}

	stream := make(chan AssistantMessageEvent)

	// TODO: Implement the actual Google GenAI streaming logic here.
	go func() {
		defer close(stream)

		errMsg := "Google GenAI provider streaming is not yet fully implemented in Go port."
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

// StreamGoogleGeminiCli is the StreamFunction implementation for the Google Gemini internal CLI API.
func StreamGoogleGeminiCli(model ModelInfo, context Context, options any) AssistantMessageEventStream {
	// Validate options type
	_, ok := options.(GoogleGeminiCliOptions)
	if !ok && options != nil {
		// Log or handle invalid options type if necessary in future implementations
	}

	stream := make(chan AssistantMessageEvent)

	// TODO: Implement the actual Google GenAI CLI streaming logic here.
	go func() {
		defer close(stream)

		errMsg := "Google Gemini CLI provider streaming is not yet fully implemented in Go port."
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

// StreamGoogleVertex is the StreamFunction implementation for the Google Vertex AI API.
func StreamGoogleVertex(model ModelInfo, context Context, options any) AssistantMessageEventStream {
	// Validate options type
	_, ok := options.(GoogleVertexOptions)
	if !ok && options != nil {
		// Log or handle invalid options type if necessary in future implementations
	}

	stream := make(chan AssistantMessageEvent)

	// TODO: Implement the actual Google Vertex AI streaming logic here.
	go func() {
		defer close(stream)

		errMsg := "Google Vertex provider streaming is not yet fully implemented in Go port."
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

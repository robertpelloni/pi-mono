package ai

type OpenAIReasoningEffort string

const (
	OpenAIReasoningMinimal OpenAIReasoningEffort = "minimal"
	OpenAIReasoningLow     OpenAIReasoningEffort = "low"
	OpenAIReasoningMedium  OpenAIReasoningEffort = "medium"
	OpenAIReasoningHigh    OpenAIReasoningEffort = "high"
	OpenAIReasoningXHigh   OpenAIReasoningEffort = "xhigh"
)

type OpenAIServiceTier string

const (
	OpenAIServiceTierAuto    OpenAIServiceTier = "auto"
	OpenAIServiceTierDefault OpenAIServiceTier = "default"
)

type OpenAIReasoningSummary string

const (
	OpenAIReasoningSummaryAuto     OpenAIReasoningSummary = "auto"
	OpenAIReasoningSummaryDetailed OpenAIReasoningSummary = "detailed"
	OpenAIReasoningSummaryConcise  OpenAIReasoningSummary = "concise"
)

// OpenAIResponsesOptions represents the specific options available when calling the OpenAI Responses API.
// It embeds the generic StreamOptions.
type OpenAIResponsesOptions struct {
	StreamOptions
	ReasoningEffort  *OpenAIReasoningEffort  `json:"reasoningEffort,omitempty"`
	ReasoningSummary *OpenAIReasoningSummary `json:"reasoningSummary,omitempty"`
	ServiceTier      *OpenAIServiceTier      `json:"serviceTier,omitempty"`
}

// StreamOpenAIResponses is the StreamFunction implementation for the OpenAI Responses API.
func StreamOpenAIResponses(model ModelInfo, context Context, options any) AssistantMessageEventStream {
	// Validate options type
	_, ok := options.(OpenAIResponsesOptions)
	if !ok && options != nil {
		// Log or handle invalid options type if necessary in future implementations
	}

	stream := make(chan AssistantMessageEvent)

	// TODO: Implement the actual OpenAI streaming logic here.
	// For now, this is a placeholder stub to define the architecture.
	go func() {
		defer close(stream)

		// In a real implementation, we would create the client, send the request,
		// and parse the Server-Sent Events (SSE) into our AssistantMessageEvent types.
		// For now, we simulate an error to show it's unimplemented.

		errMsg := "OpenAI provider streaming is not yet fully implemented in Go port."
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

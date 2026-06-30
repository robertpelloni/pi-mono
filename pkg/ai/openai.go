package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// resolveBaseURL returns the API base URL for the given model.
// If the model has a custom BaseURL set, that is used. Otherwise,
// the default URL is based on the provider.
func resolveBaseURL(model ModelInfo) string {
	if model.BaseURL != "" {
		return model.BaseURL
	}
	switch model.Provider {
	case ProviderOllama:
		return "https://ollama.com/v1"
	case ProviderOpenRouter:
		return "https://openrouter.ai/api/v1"
	case ProviderGroq:
		return "https://api.groq.com/openai/v1"
	case ProviderCerebras:
		return "https://api.cerebras.ai/v1"
	case ProviderXAI:
		return "https://api.x.ai/v1"
	case ProviderMistral:
		return "https://api.mistral.ai/v1"
	case ProviderHuggingFace:
		return "https://api-inference.huggingface.co/v1"
	case ProviderMinimax:
		return "https://api.minimax.chat/v1"
	case ProviderMinimaxCN:
		return "https://api.minimax.chat/v1"
	default:
		return "https://api.openai.com/v1"
	}
}

// resolveAPIKey resolves the API key for a model by checking
// environment variables in order of specificity.
func resolveAPIKey(model ModelInfo) string {
	// 1. Check model-specific env var: <PROVIDER>_API_KEY
	providerKey := GetEnvAPIKey(model.Provider)
	if providerKey != "" {
		return providerKey
	}
	// 2. Check for Ollama-specific key
	if model.Provider == ProviderOllama {
		if key := os.Getenv("OLLAMA_API_KEY"); key != "" {
			return key
		}
	}
	// 3. Fallback to OpenAI key for OpenAI-compatible providers
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return key
	}
	return ""
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type string `json:"type"`
	Function struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Parameters  any    `json:"parameters"`
	} `json:"function"`
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Tools    []openAITool    `json:"tools,omitempty"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Role      string `json:"role,omitempty"`
			Content   string `json:"content,omitempty"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func StreamOpenAIResponses(ctx context.Context, model ModelInfo, aiCtx Context, options any) AssistantMessageEventStream {
	stream := make(chan AssistantMessageEvent)

	go func() {
		defer close(stream)

		reqCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		sendEvent := func(e AssistantMessageEvent) bool {
			select {
			case <-reqCtx.Done():
				return false
			case stream <- e:
				return true
			}
		}

		apiKey := resolveAPIKey(model)
		if apiKey == "" {
			errMsg := fmt.Sprintf("missing API key for provider %s", model.Provider)
			reason := StopReasonError
			if !sendEvent(AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}) {
				return
			}
			return
		}

		var reqMessages []openAIMessage

		// 1. Map System Prompt
		if aiCtx.SystemPrompt != nil && *aiCtx.SystemPrompt != "" {
			reqMessages = append(reqMessages, openAIMessage{Role: "system", Content: *aiCtx.SystemPrompt})
		}

		// 2. Map Messages
		for _, genericMsg := range aiCtx.Messages {
			switch msg := genericMsg.(type) {
			case UserMessage:
				content := ""
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						content += txt.Text
					}
				}
				reqMessages = append(reqMessages, openAIMessage{Role: "user", Content: content})

			case AssistantMessage:
				content := ""
				var toolCalls []openAIToolCall
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						content += txt.Text
					} else if tc, ok := pt.(ToolCall); ok {
						argBytes, _ := json.Marshal(tc.Arguments)
						toolCalls = append(toolCalls, openAIToolCall{
							ID:   tc.ID,
							Type: "function",
							Function: openAIFunctionCall{
								Name:      tc.Name,
								Arguments: string(argBytes),
							},
						})
					}
				}
				reqMessages = append(reqMessages, openAIMessage{
					Role:      "assistant",
					Content:   content,
					ToolCalls: toolCalls,
				})

			case ToolResultMessage:
				content := ""
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						content += txt.Text
					}
				}
				reqMessages = append(reqMessages, openAIMessage{
					Role:       "tool",
					Content:    content,
					ToolCallID: msg.ToolCallID,
				})
			}
		}

		var reqTools []openAITool
		for _, t := range aiCtx.Tools {
			reqTools = append(reqTools, openAITool{
				Type: "function",
				Function: struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					Parameters  any    `json:"parameters"`
				}{Name: t.Name, Description: t.Description, Parameters: t.Parameters},
			})
		}

		reqBody := openAIRequest{
			Model:    model.ID,
			Messages: reqMessages,
			Stream:   true,
			Tools:    reqTools,
		}

		reqBytes, err := json.Marshal(reqBody)
		if err != nil {
			errMsg := err.Error()
			reason := StopReasonError
			if !sendEvent(AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}) {
				return
			}
			return
		}

		// Implement Cancel Context
		var opt StreamOptions
		if o, ok := options.(StreamOptions); ok {
			opt = o
		} else if o, ok := options.(SimpleStreamOptions); ok {
			opt = o.StreamOptions
		}
		if opt.AbortSignal != nil {
			go func() {
				select {
				case <-opt.AbortSignal:
					cancel()
				case <-reqCtx.Done():
				}
			}()
		}

		baseURL := resolveBaseURL(model)
	req, err := http.NewRequestWithContext(reqCtx, "POST", baseURL+"/chat/completions", bytes.NewBuffer(reqBytes))
		if err != nil {
			errMsg := err.Error()
			reason := StopReasonError
			if !sendEvent(AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}) {
				return
			}
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			// Check if cancelled
			if reqCtx.Err() == context.Canceled {
				reason := StopReasonAborted
				if !sendEvent(AssistantMessageEvent{Type: EventDone, Reason: &reason}) {
					return
				}
				return
			}
			errMsg := fmt.Sprintf("OpenAI API request failed: %v", err)
			reason := StopReasonError
			if !sendEvent(AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}) {
				return
			}
			return
		}
		defer resp.Body.Close()

		// Handle non-200 responses — report error with parsed body.
		// Retry logic is handled at the AgentSession layer, not here.
		if resp.StatusCode != 200 {
			errMsg := formatProviderError("OpenAI", resp)
			reason := StopReasonError
			if !sendEvent(AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}) {
				return
			}
			return
		}

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk openAIStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if len(chunk.Choices) > 0 {
					delta := chunk.Choices[0].Delta.Content
					role := chunk.Choices[0].Delta.Role

					if role != "" {
						if !sendEvent(AssistantMessageEvent{
							Type: EventStart,
						}) {
							return
						}
					}

					if delta != "" {
						if !sendEvent(AssistantMessageEvent{
							Type:  EventTextDelta,
							Delta: &delta,
						}) {
							return
						}
					}

					if len(chunk.Choices[0].Delta.ToolCalls) > 0 {
						tc := chunk.Choices[0].Delta.ToolCalls[0]

						// Start
						if tc.ID != "" && tc.Function.Name != "" {
							name := tc.Function.Name
							id := tc.ID
							if !sendEvent(AssistantMessageEvent{
								Type: EventToolCallStart,
								ToolCall: &ToolCall{
									ID:   id,
									Name: name,
								},
							}) {
								return
							}
						}

						// Delta
						if tc.Function.Arguments != "" {
							args := tc.Function.Arguments
							if !sendEvent(AssistantMessageEvent{
								Type:  EventToolCallDelta,
								Delta: &args,
							}) {
								return
							}
						}
					}

					if chunk.Choices[0].FinishReason != nil {
						reason := StopReasonStop
						if *chunk.Choices[0].FinishReason == "tool_calls" {
							reason = StopReasonToolUse
							if !sendEvent(AssistantMessageEvent{Type: EventToolCallEnd}) {
								return
							}
						} else if *chunk.Choices[0].FinishReason == "length" {
							reason = StopReasonLength
						}
						if !sendEvent(AssistantMessageEvent{Type: EventDone, Reason: &reason}) {
							return
						}
						return
					}
				}
			}
		}

		reason := StopReasonStop
		if !sendEvent(AssistantMessageEvent{Type: EventDone, Reason: &reason}) {
			return
		}
	}()

	return stream
}

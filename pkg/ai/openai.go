package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

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
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string `json:"type"`
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

func StreamOpenAIResponses(model ModelInfo, aiCtx Context, options any) AssistantMessageEventStream {
	stream := make(chan AssistantMessageEvent)

	go func() {
		defer close(stream)

		apiKey := GetEnvAPIKey(ProviderOpenAI)
		if apiKey == "" {
			errMsg := "missing OPENAI_API_KEY"
			reason := StopReasonError
			stream <- AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}
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
						// OpenAI needs stringified JSON for arguments
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

				// OpenAI API strictness: if we have tool calls, content should ideally not be omitted, but can be empty string.
				// However, if we use omitempty on string, it disappears.
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
			stream <- AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}
			return
		}

		// Implement Cancel Context
		reqCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

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

		req, err := http.NewRequestWithContext(reqCtx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBytes))
		if err != nil {
			errMsg := err.Error()
			reason := StopReasonError
			stream <- AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{}

		// Basic Retry Logic
		var resp *http.Response
		maxRetries := 3

		for i := 0; i < maxRetries; i++ {
			// Recreate request inside loop to avoid consumed body errors
			req, err = http.NewRequestWithContext(reqCtx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBytes))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+apiKey)
			}

			resp, err = client.Do(req)
			if err != nil {
				if reqCtx.Err() == context.Canceled {
					reason := StopReasonAborted
					stream <- AssistantMessageEvent{Type: EventDone, Reason: &reason}
					return
				}
				break
			}
			if resp.StatusCode == 429 || resp.StatusCode >= 500 {
				resp.Body.Close()
				time.Sleep(time.Duration(2<<i) * time.Second)
				continue
			}
			break
		}

		if err != nil {
			errMsg := err.Error()
			reason := StopReasonError
			stream <- AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			errMsg := fmt.Sprintf("OpenAI API error: status %d", resp.StatusCode)
			reason := StopReasonError
			stream <- AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}
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
						stream <- AssistantMessageEvent{
							Type: EventStart,
						}
					}

					if delta != "" {
						stream <- AssistantMessageEvent{
							Type:  EventTextDelta,
							Delta: &delta,
						}
					}

					if len(chunk.Choices[0].Delta.ToolCalls) > 0 {
						tc := chunk.Choices[0].Delta.ToolCalls[0]

						// Start
						if tc.ID != "" && tc.Function.Name != "" {
							name := tc.Function.Name
							id := tc.ID
							stream <- AssistantMessageEvent{
								Type: EventToolCallStart,
								ToolCall: &ToolCall{
									ID:   id,
									Name: name,
								},
							}
						}

						// Delta
						if tc.Function.Arguments != "" {
							args := tc.Function.Arguments
							stream <- AssistantMessageEvent{
								Type:  EventToolCallDelta,
								Delta: &args,
							}
						}

						// OpenAI specifies that when index increments or finish reason drops, the tool call is done,
						// but practically we usually just wait for finish_reason == "tool_calls"
					}

					if chunk.Choices[0].FinishReason != nil {
						reasonStr := *chunk.Choices[0].FinishReason
						if reasonStr == "tool_calls" {
							stream <- AssistantMessageEvent{Type: EventToolCallEnd}
						}
					}

					if chunk.Choices[0].FinishReason != nil {
						reason := StopReasonStop
						stream <- AssistantMessageEvent{Type: EventDone, Reason: &reason}
						return
					}
				}
			}
		}

		reason := StopReasonStop
		stream <- AssistantMessageEvent{Type: EventDone, Reason: &reason}
	}()

	return stream
}

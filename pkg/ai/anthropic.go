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

type anthropicContent struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	ToolUseID string         `json:"tool_use_id,omitempty"`
	Content   string         `json:"content,omitempty"`
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	Stream    bool               `json:"stream"`
	MaxTokens int                `json:"max_tokens"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicStreamChunk struct {
	Type    string `json:"type"`
	Message *struct {
		StopReason string `json:"stop_reason,omitempty"`
	} `json:"message,omitempty"`
	Delta struct {
		StopReason  string `json:"stop_reason,omitempty"`
		Type        string `json:"type"`
		Text        string `json:"text"`
		PartialJson string `json:"partial_json,omitempty"`
	} `json:"delta"`
	ContentBlock *struct {
		Type  string          `json:"type"`
		ID    string          `json:"id,omitempty"`
		Name  string          `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
	} `json:"content_block,omitempty"`
}

func StreamAnthropic(ctx context.Context, model ModelInfo, aiCtx Context, options any) AssistantMessageEventStream {
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

		apiKey := GetEnvAPIKey(ProviderAnthropic)
		if apiKey == "" {
			errMsg := "missing ANTHROPIC_API_KEY"
			reason := StopReasonError
			if !sendEvent(AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}) {
				return
			}
			return
		}

		var reqMessages []anthropicMessage
		var systemPrompt string

		if aiCtx.SystemPrompt != nil {
			systemPrompt = *aiCtx.SystemPrompt
		}

		for _, genericMsg := range aiCtx.Messages {
			var blocks []anthropicContent

			switch msg := genericMsg.(type) {
			case UserMessage:
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						blocks = append(blocks, anthropicContent{Type: "text", Text: txt.Text})
					}
				}
				reqMessages = append(reqMessages, anthropicMessage{Role: "user", Content: blocks})

			case AssistantMessage:
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						blocks = append(blocks, anthropicContent{Type: "text", Text: txt.Text})
					} else if tc, ok := pt.(ToolCall); ok {
						blocks = append(blocks, anthropicContent{
							Type:  "tool_use",
							ID:    tc.ID,
							Name:  tc.Name,
							Input: tc.Arguments,
						})
					}
				}
				reqMessages = append(reqMessages, anthropicMessage{Role: "assistant", Content: blocks})

			case ToolResultMessage:
				// Anthropic sends tool results wrapped inside "user" messages
				content := ""
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						content += txt.Text
					}
				}

				blocks = append(blocks, anthropicContent{
					Type:      "tool_result",
					ToolUseID: msg.ToolCallID,
					Content:   content,
				})

				reqMessages = append(reqMessages, anthropicMessage{Role: "user", Content: blocks})
			}
		}

		var reqTools []anthropicTool
		for _, t := range aiCtx.Tools {
			reqTools = append(reqTools, anthropicTool{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.Parameters,
			})
		}

		reqBody := anthropicRequest{
			Model:     model.ID,
			Messages:  reqMessages,
			System:    strings.TrimSpace(systemPrompt),
			Stream:    true,
			MaxTokens: 4096,
			Tools:     reqTools,
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

		var opt StreamOptions
		if o, ok := options.(StreamOptions); ok {
			opt = o
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

		req, err := http.NewRequestWithContext(reqCtx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBytes))
		if err != nil {
			errMsg := err.Error()
			reason := StopReasonError
			if !sendEvent(AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}) {
				return
			}
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		client := &http.Client{}

		var resp *http.Response
		maxRetries := 3

		for i := 0; i < maxRetries; i++ {
			// Recreate request inside loop to avoid consumed body errors
			req, err = http.NewRequestWithContext(reqCtx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBytes))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("x-api-key", apiKey)
				req.Header.Set("anthropic-version", "2023-06-01")
			}

			resp, err = client.Do(req)
			if err != nil {
				if reqCtx.Err() == context.Canceled {
					reason := StopReasonAborted
					if !sendEvent(AssistantMessageEvent{Type: EventDone, Reason: &reason}) {
						return
					}
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
			if !sendEvent(AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}) {
				return
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			errMsg := fmt.Sprintf("Anthropic API error: status %d", resp.StatusCode)
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

			var chunk anthropicStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if chunk.Type == "content_block_delta" {
					if chunk.Delta.Text != "" {
						if !sendEvent(AssistantMessageEvent{
							Type:  EventTextDelta,
							Delta: &chunk.Delta.Text,
						}) {
							return
						}
					}
				} else if chunk.Type == "content_block_start" && chunk.ContentBlock != nil && chunk.ContentBlock.Type == "tool_use" {
					name := chunk.ContentBlock.Name
					id := chunk.ContentBlock.ID
					if !sendEvent(AssistantMessageEvent{
						Type: EventToolCallStart,
						ToolCall: &ToolCall{
							ID:   id,
							Name: name,
						},
					}) {
						return
					}
				} else if chunk.Type == "content_block_delta" && chunk.Delta.Type == "input_json_delta" {
					if chunk.Delta.PartialJson != "" {
						if !sendEvent(AssistantMessageEvent{
							Type:  EventToolCallDelta,
							Delta: &chunk.Delta.PartialJson,
						}) {
							return
						}
					}
				} else if chunk.Type == "content_block_stop" {
					if !sendEvent(AssistantMessageEvent{Type: EventToolCallEnd}) {
						return
					}
				} else if chunk.Type == "message_delta" && chunk.Delta.StopReason != "" {
					reason := StopReasonStop
					if chunk.Delta.StopReason == "tool_use" {
						reason = StopReasonToolUse
					} else if chunk.Delta.StopReason == "max_tokens" {
						reason = StopReasonLength
					}
					if !sendEvent(AssistantMessageEvent{Type: EventDone, Reason: &reason}) {
						return
					}
					return
				} else if chunk.Type == "message_stop" {
					// Done event already emitted by message_delta if stop_reason was there
					// But we fall back here just in case
					reason := StopReasonStop
					if !sendEvent(AssistantMessageEvent{Type: EventDone, Reason: &reason}) {
						return
					}
					return
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

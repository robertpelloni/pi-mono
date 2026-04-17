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

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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
	Type  string `json:"type"`
	Delta struct {
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

func StreamAnthropic(model ModelInfo, aiCtx Context, options any) AssistantMessageEventStream {
	stream := make(chan AssistantMessageEvent)

	go func() {
		defer close(stream)

		apiKey := GetEnvAPIKey(ProviderAnthropic)
		if apiKey == "" {
			errMsg := "missing ANTHROPIC_API_KEY"
			reason := StopReasonError
			stream <- AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}
			return
		}

		var reqMessages []anthropicMessage
		var systemPrompt string

		if aiCtx.SystemPrompt != nil {
			systemPrompt = *aiCtx.SystemPrompt
		}

		for _, genericMsg := range aiCtx.Messages {
			content := ""
			switch msg := genericMsg.(type) {
			case UserMessage:
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						content += txt.Text
					}
				}
				reqMessages = append(reqMessages, anthropicMessage{Role: "user", Content: content})
			case AssistantMessage:
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						content += txt.Text
					}
				}
				reqMessages = append(reqMessages, anthropicMessage{Role: "assistant", Content: content})
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
			stream <- AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}
			return
		}

		reqCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

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
			stream <- AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		client := &http.Client{}

		var resp *http.Response
		maxRetries := 3
		for i := 0; i < maxRetries; i++ {
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
			errMsg := fmt.Sprintf("Anthropic API error: status %d", resp.StatusCode)
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

			var chunk anthropicStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if chunk.Type == "content_block_delta" {
					if chunk.Delta.Text != "" {
						stream <- AssistantMessageEvent{
							Type:  EventTextDelta,
							Delta: &chunk.Delta.Text,
						}
					}
				} else if chunk.Type == "content_block_start" && chunk.ContentBlock != nil && chunk.ContentBlock.Type == "tool_use" {
					name := chunk.ContentBlock.Name
					id := chunk.ContentBlock.ID
					stream <- AssistantMessageEvent{
						Type: EventToolCallStart,
						ToolCall: &ToolCall{
							ID:   id,
							Name: name,
						},
					}
				} else if chunk.Type == "content_block_delta" && chunk.Delta.Type == "input_json_delta" {
					if chunk.Delta.PartialJson != "" {
						stream <- AssistantMessageEvent{
							Type:  EventToolCallDelta,
							Delta: &chunk.Delta.PartialJson,
						}
					}
				} else if chunk.Type == "message_stop" {
					reason := StopReasonStop
					stream <- AssistantMessageEvent{Type: EventDone, Reason: &reason}
					return
				}
			}
		}

		reason := StopReasonStop
		stream <- AssistantMessageEvent{Type: EventDone, Reason: &reason}
	}()

	return stream
}

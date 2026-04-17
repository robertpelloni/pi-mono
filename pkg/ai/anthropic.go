package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	Stream    bool               `json:"stream"`
	MaxTokens int                `json:"max_tokens"`
}

type anthropicStreamChunk struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}

// AnthropicOptions represents the specific options available when calling the Anthropic API.
type AnthropicOptions struct {
	StreamOptions
}

// StreamAnthropic is the StreamFunction implementation for the Anthropic API using SSE.
func StreamAnthropic(model ModelInfo, context Context, options any) AssistantMessageEventStream {
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

		for _, genericMsg := range context.Messages {
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

		reqBody := anthropicRequest{
			Model:     model.ID,
			Messages:  reqMessages,
			System:    strings.TrimSpace(systemPrompt),
			Stream:    true,
			MaxTokens: 4096, // required field by anthropic
		}

		reqBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		client := &http.Client{}
		resp, err := client.Do(req)
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

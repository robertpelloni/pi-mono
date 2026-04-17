package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Define basic JSON structures for OpenAI streaming
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// StreamOpenAIResponses implements standard net/http SSE logic for OpenAI
func StreamOpenAIResponses(model ModelInfo, context Context, options any) AssistantMessageEventStream {
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

		// Map internal context to OpenAI messages
		var reqMessages []openAIMessage
		for _, genericMsg := range context.Messages {
			content := ""
			role := "user"

			switch msg := genericMsg.(type) {
			case UserMessage:
				role = "user"
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						content += txt.Text
					}
				}
			case AssistantMessage:
				role = "assistant"
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						content += txt.Text
					}
				}
			}
			reqMessages = append(reqMessages, openAIMessage{Role: role, Content: content})
		}

		reqBody := openAIRequest{
			Model:    model.ID,
			Messages: reqMessages,
			Stream:   true,
		}

		reqBytes, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

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
					if delta != "" {
						stream <- AssistantMessageEvent{
							Type:  EventTextDelta,
							Delta: &delta,
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

package ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type googleMessagePart struct {
	Text string `json:"text"`
}

type googleMessage struct {
	Role  string              `json:"role"`
	Parts []googleMessagePart `json:"parts"`
}

type googleRequest struct {
	Contents []googleMessage `json:"contents"`
}

type googleStreamChunk struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason,omitempty"`
	} `json:"candidates"`
}

// GoogleOptions represents the specific options available when calling the Google API.
type GoogleOptions struct {
	StreamOptions
}

// StreamGoogle is the StreamFunction implementation for the Google API using SSE.
func StreamGoogle(model ModelInfo, context Context, options any) AssistantMessageEventStream {
	stream := make(chan AssistantMessageEvent)

	go func() {
		defer close(stream)

		apiKey := GetEnvAPIKey(ProviderGoogle)
		if apiKey == "" {
			errMsg := "missing GOOGLE_API_KEY"
			reason := StopReasonError
			stream <- AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}
			return
		}

		var reqMessages []googleMessage
		for _, genericMsg := range context.Messages {
			content := ""

			switch msg := genericMsg.(type) {
			case UserMessage:
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						content += txt.Text
					}
				}
				reqMessages = append(reqMessages, googleMessage{
					Role:  "user",
					Parts: []googleMessagePart{{Text: content}},
				})
			case AssistantMessage:
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						content += txt.Text
					}
				}
				reqMessages = append(reqMessages, googleMessage{
					Role:  "model",
					Parts: []googleMessagePart{{Text: content}},
				})
			}
		}

		reqBody := googleRequest{
			Contents: reqMessages,
		}

		reqBytes, _ := json.Marshal(reqBody)
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", model.ID, apiKey)

		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBytes))
		req.Header.Set("Content-Type", "application/json")

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
			errMsg := fmt.Sprintf("Google API error: status %d", resp.StatusCode)
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

			var chunk googleStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if len(chunk.Candidates) > 0 {
					if len(chunk.Candidates[0].Content.Parts) > 0 {
						delta := chunk.Candidates[0].Content.Parts[0].Text
						if delta != "" {
							stream <- AssistantMessageEvent{
								Type:  EventTextDelta,
								Delta: &delta,
							}
						}
					}

					if chunk.Candidates[0].FinishReason != "" && chunk.Candidates[0].FinishReason != "FINISH_REASON_UNSPECIFIED" {
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

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

type googleMessagePart struct {
	Text string `json:"text"`
}

type googleMessage struct {
	Role  string              `json:"role"`
	Parts []googleMessagePart `json:"parts"`
}

type googleSystemInstruction struct {
	Parts []googleMessagePart `json:"parts"`
}

type googleFunctionDeclaration struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type googleTool struct {
	FunctionDeclarations []googleFunctionDeclaration `json:"functionDeclarations"`
}

type googleRequest struct {
	SystemInstruction *googleSystemInstruction `json:"systemInstruction,omitempty"`
	Contents          []googleMessage          `json:"contents"`
	Tools             []googleTool             `json:"tools,omitempty"`
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

func StreamGoogle(model ModelInfo, aiCtx Context, options any) AssistantMessageEventStream {
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
		var sysInst *googleSystemInstruction

		if aiCtx.SystemPrompt != nil && *aiCtx.SystemPrompt != "" {
			sysInst = &googleSystemInstruction{
				Parts: []googleMessagePart{{Text: *aiCtx.SystemPrompt}},
			}
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

		var reqFuncs []googleFunctionDeclaration
		for _, t := range aiCtx.Tools {
			reqFuncs = append(reqFuncs, googleFunctionDeclaration{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			})
		}
		var reqTools []googleTool
		if len(reqFuncs) > 0 {
			reqTools = append(reqTools, googleTool{FunctionDeclarations: reqFuncs})
		}

		reqBody := googleRequest{
			SystemInstruction: sysInst,
			Contents:          reqMessages,
			Tools:             reqTools,
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
		} else if o, ok := options.(GoogleOptions); ok {
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

		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", model.ID, apiKey)
		req, err := http.NewRequestWithContext(reqCtx, "POST", url, bytes.NewBuffer(reqBytes))
		if err != nil {
			errMsg := err.Error()
			reason := StopReasonError
			stream <- AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}
			return
		}

		req.Header.Set("Content-Type", "application/json")

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

// GoogleOptions represents the specific options available when calling the Google API.
type GoogleOptions struct {
	StreamOptions
}

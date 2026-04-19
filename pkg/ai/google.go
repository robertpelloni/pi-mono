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
	Text         string `json:"text,omitempty"`
	FunctionCall *struct {
		Name string         `json:"name"`
		Args map[string]any `json:"args"`
	} `json:"functionCall,omitempty"`
	FunctionResponse *struct {
		Name     string         `json:"name"`
		Response map[string]any `json:"response"`
	} `json:"functionResponse,omitempty"`
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
				Text         string `json:"text,omitempty"`
				FunctionCall *struct {
					Name string         `json:"name"`
					Args map[string]any `json:"args"`
				} `json:"functionCall,omitempty"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason,omitempty"`
	} `json:"candidates"`
}

func StreamGoogle(ctx context.Context, model ModelInfo, aiCtx Context, options any) AssistantMessageEventStream {
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

		apiKey := GetEnvAPIKey(ProviderGoogle)
		if apiKey == "" {
			errMsg := "missing GOOGLE_API_KEY"
			reason := StopReasonError
			if !sendEvent(AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}) {
				return
			}
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
			switch msg := genericMsg.(type) {
			case UserMessage:
				var parts []googleMessagePart
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						parts = append(parts, googleMessagePart{Text: txt.Text})
					}
				}
				reqMessages = append(reqMessages, googleMessage{Role: "user", Parts: parts})

			case AssistantMessage:
				var parts []googleMessagePart
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						parts = append(parts, googleMessagePart{Text: txt.Text})
					} else if tc, ok := pt.(ToolCall); ok {
						parts = append(parts, googleMessagePart{
							FunctionCall: &struct {
								Name string         `json:"name"`
								Args map[string]any `json:"args"`
							}{Name: tc.Name, Args: tc.Arguments},
						})
					}
				}
				reqMessages = append(reqMessages, googleMessage{Role: "model", Parts: parts})

			case ToolResultMessage:
				// Map text content payload over into the function response map
				content := ""
				for _, pt := range msg.Content {
					if txt, ok := pt.(TextContent); ok {
						content += txt.Text
					}
				}

				respMap := map[string]any{
					"content": content,
				}

				reqMessages = append(reqMessages, googleMessage{
					Role: "function", // Google uses 'function' role occasionally, or 'user' with FunctionResponse. The latter is standard for V1beta.
					Parts: []googleMessagePart{{
						FunctionResponse: &struct {
							Name     string         `json:"name"`
							Response map[string]any `json:"response"`
						}{Name: msg.ToolName, Response: respMap},
					}},
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
			if !sendEvent(AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}) {
				return
			}
			return
		}

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
			if !sendEvent(AssistantMessageEvent{Type: EventError, Reason: &reason, Error: &AssistantMessage{ErrorMessage: &errMsg}}) {
				return
			}
			return
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}

		var resp *http.Response
		maxRetries := 3

		for i := 0; i < maxRetries; i++ {
			// Recreate request inside loop to avoid consumed body errors
			req, err = http.NewRequestWithContext(reqCtx, "POST", url, bytes.NewBuffer(reqBytes))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
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
			errMsg := fmt.Sprintf("Google API error: status %d", resp.StatusCode)
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

			var chunk googleStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if len(chunk.Candidates) > 0 {
					if len(chunk.Candidates[0].Content.Parts) > 0 {
						delta := chunk.Candidates[0].Content.Parts[0].Text
						if chunk.Candidates[0].Content.Parts[0].FunctionCall != nil {
							fc := chunk.Candidates[0].Content.Parts[0].FunctionCall
							// Google returns whole blobs in stream usually, so we trigger start and delta together
							if !sendEvent(AssistantMessageEvent{
								Type: EventToolCallStart,
								ToolCall: &ToolCall{
									ID:   "call_google_generated", // Google doesn't use Call IDs consistently natively
									Name: fc.Name,
								},
							}) {
								return
							}

							if argBytes, err := json.Marshal(fc.Args); err == nil {
								argStr := string(argBytes)
								if !sendEvent(AssistantMessageEvent{
									Type:  EventToolCallDelta,
									Delta: &argStr,
								}) {
									return
								}

								if !sendEvent(AssistantMessageEvent{Type: EventToolCallEnd}) {
									return
								}
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
					}

					if chunk.Candidates[0].FinishReason != "" && chunk.Candidates[0].FinishReason != "FINISH_REASON_UNSPECIFIED" {
						reason := StopReasonStop
						if chunk.Candidates[0].FinishReason == "MAX_TOKENS" {
							reason = StopReasonLength
						}
						// Google doesn't have an explicit tool_calls finish reason, it usually emits STOP
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

// GoogleOptions represents the specific options available when calling the Google API.
type GoogleOptions struct {
	StreamOptions
}

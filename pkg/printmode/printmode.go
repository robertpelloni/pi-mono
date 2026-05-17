package printmode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/sdk"
)

// PrintModeOptions configures single-shot execution.
type PrintModeOptions struct {
	// Output mode: "text" for final response only, "json" for all events
	Mode string // "text" or "json"
	// Additional prompts to send after initialMessage
	Messages []string
	// First message to send (may contain @file content)
	InitialMessage string
	// Images to attach to the initial message
	InitialImages []ai.ImageContent
	// Writer for output (defaults to os.Stdout)
	Writer io.Writer
}

// RunPrintMode executes the agent in single-shot (print) mode.
func RunPrintMode(ctx context.Context, sessionResult *sdk.CreateAgentSessionResult, options PrintModeOptions) int {
	w := options.Writer
	if w == nil {
		w = os.Stdout
	}

	mode := options.Mode
	if mode == "" {
		mode = "text"
	}

	exitCode := 0
	agentLoop := sessionResult.Agent

	// Subscribe to events for JSON mode
	if mode == "json" {
		agentLoop.Subscribe(func(event agent.AgentEvent) {
			data, err := json.Marshal(event)
			if err == nil {
				fmt.Fprintf(w, "%s\n", data)
			}
		})
	}

	// Send initial message
	if options.InitialMessage != "" {
		userMsg := ai.UserMessage{
			Content: []ai.Content{
				ai.TextContent{Text: options.InitialMessage},
			},
		}
		for _, img := range options.InitialImages {
			userMsg.Content = append(userMsg.Content, img)
		}

		err := agentLoop.Prompt(ctx, userMsg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
	}

	// Send additional messages
	for _, msg := range options.Messages {
		userMsg := ai.UserMessage{
			Content: []ai.Content{
				ai.TextContent{Text: msg},
			},
		}
		err := agentLoop.Prompt(ctx, userMsg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
	}

	// In text mode, output the last assistant message
	if mode == "text" {
		messages := agentLoop.Messages()
		if len(messages) > 0 {
			lastMsg := messages[len(messages)-1]
			if lastMsg.GetRole() == ai.RoleAssistant {
				assistantMsg := lastMsg.(ai.AssistantMessage)
				if assistantMsg.StopReason == ai.StopReasonError || assistantMsg.StopReason == ai.StopReasonAborted {
					if assistantMsg.ErrorMessage != nil {
						fmt.Fprintf(os.Stderr, "%s\n", *assistantMsg.ErrorMessage)
					}
					exitCode = 1
				} else {
					for _, content := range assistantMsg.Content {
						if tc, ok := content.(ai.TextContent); ok {
							fmt.Fprintf(w, "%s\n", tc.Text)
						}
					}
				}
			}
		}
	}

	return exitCode
}

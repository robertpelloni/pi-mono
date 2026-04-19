package tui

import (
	"fmt"
	"strings"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// The TypeScript @mariozechner/pi-tui relies heavily on a custom differential renderer for the terminal.
// However, since we are moving to Go, we can eventually utilize a more standard Go TUI library like
// `charmbracelet/bubbletea`. For this structural scaffolding phase, we will implement a basic renderer
// that consumes `agent.AgentEvent` items and handles the terminal readout identically to the TS `InteractiveAgent`.

// Renderer defines the interface for rendering agent events to a UI.
type Renderer interface {
	RenderEvent(event agent.AgentEvent)
}

// BasicTerminalRenderer provides a simple stdout implementation for the TUI.
type BasicTerminalRenderer struct {
	ActiveToolCallID string
}

func NewBasicTerminalRenderer() *BasicTerminalRenderer {
	return &BasicTerminalRenderer{}
}

func (r *BasicTerminalRenderer) RenderEvent(event agent.AgentEvent) {
	switch event.Type {
	case agent.EventAgentStart:
		fmt.Println("\n[Agent] Starting new run...")

	case agent.EventMessageStart:
		if event.Message != nil {
			if event.Message.GetRole() == ai.RoleAssistant {
				fmt.Print("\n[Assistant]: ")
			}
		}

	case agent.EventMessageUpdate:
		if event.AssistantMessageEvent != nil {
			if event.AssistantMessageEvent.Type == ai.EventTextDelta && event.AssistantMessageEvent.Delta != nil {
				fmt.Print(*event.AssistantMessageEvent.Delta)
			}
		}

	case agent.EventMessageEnd:
		fmt.Println() // Add newline after assistant finishes streaming message

	case agent.EventToolExecutionStart:
		fmt.Printf("\n  [Tool Execution] %s(args: %v)...\n", event.ToolName, event.Args)
		r.ActiveToolCallID = event.ToolCallID

	case agent.EventToolExecutionEnd:
		if r.ActiveToolCallID == event.ToolCallID {
			status := "SUCCESS"
			if event.IsError {
				status = "ERROR"
			}

			// We format the raw output slightly
			contentStr := ""
			if res, ok := event.Result.(agent.AgentToolResult); ok {
				for _, c := range res.Content {
					if txt, ok := c.(ai.TextContent); ok {
						contentStr += txt.Text
					}
				}
			}

			// Truncate long tool outputs for terminal readability
			displayStr := contentStr
			if len(displayStr) > 200 {
				displayStr = displayStr[:197] + "..."
			}

			fmt.Printf("  [Tool Finished] %s -> %s\n    %s\n", event.ToolName, status, strings.ReplaceAll(displayStr, "\n", "\n    "))
			r.ActiveToolCallID = ""
		}

	case agent.EventTurnEnd:
		// Turn has completed

	case agent.EventAgentEnd:
		fmt.Println("\n[Agent] Run completed.")
	}
}

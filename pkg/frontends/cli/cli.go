package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

type CLIRenderer struct {
	agent *agent.Agent
}

func NewCLIRenderer(ag *agent.Agent) *CLIRenderer {
	return &CLIRenderer{agent: ag}
}

func (r *CLIRenderer) RenderEvent(event agent.AgentEvent) {
	switch event.Type {
	case agent.EventAgentStart:
		fmt.Println("\n--- [Agent Started] ---")

	case agent.EventMessageStart:
		if event.Message != nil && event.Message.GetRole() == ai.RoleAssistant {
			fmt.Print("\n[Assistant]: ")
		}

	case agent.EventMessageUpdate:
		if event.AssistantMessageEvent != nil && event.AssistantMessageEvent.Type == ai.EventTextDelta {
			if event.AssistantMessageEvent.Delta != nil {
				fmt.Print(*event.AssistantMessageEvent.Delta)
			}
		}

	case agent.EventMessageEnd:
		fmt.Println()

	case agent.EventToolExecutionStart:
		argsStr := fmt.Sprintf("%v", event.Args)
		fmt.Printf("\n[Running Tool] %s(%s)...\n", event.ToolName, argsStr)

	case agent.EventToolExecutionEnd:
		status := "SUCCESS"
		if event.IsError {
			status = "ERROR"
		}

		contentStr := ""
		if res, ok := event.Result.(agent.AgentToolResult); ok {
			for _, c := range res.Content {
				if txt, ok := c.(ai.TextContent); ok {
					contentStr += txt.Text
				}
			}
		}

		if len(contentStr) > 200 {
			contentStr = contentStr[:197] + "..."
		}
		fmt.Printf("[Tool Finished] %s -> %s\n  %s\n", event.ToolName, status, strings.ReplaceAll(contentStr, "\n", "\n  "))

	case agent.EventAgentEnd:
		fmt.Println("\n--- [Agent Finished] ---")
	}
}

func (r *CLIRenderer) Start() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Pi CLI Agent (Plain Text Mode). Type 'exit' to quit.")

	for {
		fmt.Print("\n> ")
		text, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		text = strings.TrimSpace(text)
		if text == "exit" || text == "quit" {
			break
		}
		if text == "" {
			continue
		}

		userMsg := ai.UserMessage{
			Content: []ai.Content{
				ai.TextContent{Text: text},
			},
			Timestamp: time.Now().UnixMilli(),
		}

		err = r.agent.Prompt(context.Background(), userMsg)
		if err != nil {
			fmt.Printf("Execution error: %v\n", err)
		}
	}
	return nil
}

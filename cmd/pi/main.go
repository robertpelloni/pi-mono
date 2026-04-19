package main

import (
	"context"
	"fmt"
	"os"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/tools"
	"github.com/badlogic/pi-mono/pkg/tui"
)

// main is a PoC entrypoint connecting the agent framework directly to a basic CLI renderer.
func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting cwd:", err)
		return
	}

	modelInfo := ai.ModelInfo{
		ID:       "gpt-4o",
		Provider: ai.ProviderOpenAI,
		API:      ai.ApiOpenAIResponses,
	}

	toolList := []agent.AgentTool{
		tools.ReadTool(cwd),
		tools.BashTool(cwd),
		tools.WriteTool(cwd),
	}

	// This connects our newly implemented StreamFunction signatures from pkg/ai
	// Since we haven't wired up the dynamic model-registry layer entirely yet in cmd/pi,
	// we'll pass the explicit OpenAI streaming implementation from pkg/ai
	agentLoop := agent.NewAgent(modelInfo, toolList, ai.StreamOpenAIResponses, agent.AgentLoopConfig{
		ToolExecution: agent.ToolExecutionParallel,
	})

	// Add our TUI hook
	renderer := tui.NewBasicTerminalRenderer()
	agentLoop.Subscribe(renderer.RenderEvent)

	// Add user prompt to context
	userMsg := ai.UserMessage{
		Content: []ai.Content{
			ai.TextContent{Text: "Read README.md and summarize it using bash commands."},
		},
		Timestamp: 1234567890,
	}

	agentLoop.SetSystemPrompt("You are an autonomous coding agent. Use tools whenever possible.")

	fmt.Println("Dispatching Prompt...")
	err = agentLoop.Prompt(context.Background(), userMsg)
	if err != nil {
		fmt.Printf("Execution failed: %v\n", err)
	}
}

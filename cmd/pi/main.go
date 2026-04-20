package main

import (
	"fmt"
	"os"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/tools"
	"github.com/badlogic/pi-mono/pkg/tui"
)

// main connects the agent framework to the interactive Bubbletea CLI interface.
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
		tools.LsTool(cwd),
		tools.GrepTool(cwd),
		tools.FindTool(cwd),
	}

	agentLoop := agent.NewAgent(modelInfo, toolList, ai.StreamOpenAIResponses, agent.AgentLoopConfig{
		ToolExecution: agent.ToolExecutionParallel,
	})

	agentLoop.SetSystemPrompt("You are an autonomous coding agent. Use tools whenever possible.")

	renderer := tui.NewInteractiveRenderer(agentLoop)
	agentLoop.Subscribe(renderer.RenderEvent)

	// Start the TUI, which will now handle user prompt input interactively.
	if err := renderer.Start(); err != nil {
		fmt.Printf("UI execution failed: %v\n", err)
	}
}

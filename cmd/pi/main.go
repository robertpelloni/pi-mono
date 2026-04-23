package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/frontends/bubbletea"
	"github.com/badlogic/pi-mono/pkg/frontends/cli"
	"github.com/badlogic/pi-mono/pkg/tools"
)

type Renderer interface {
	Start() error
	RenderEvent(event agent.AgentEvent)
}

func main() {
	frontendType := flag.String("frontend", "bubbletea", "Select frontend UI (bubbletea, cli)")
	flag.Parse()

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

	var renderer Renderer

	switch *frontendType {
	case "cli":
		renderer = cli.NewCLIRenderer(agentLoop)
	case "bubbletea":
		renderer = bubbletea.NewInteractiveRenderer(agentLoop)
	default:
		fmt.Println("Invalid frontend type specified. Falling back to bubbletea.")
		renderer = bubbletea.NewInteractiveRenderer(agentLoop)
	}

	agentLoop.Subscribe(renderer.RenderEvent)

	if err := renderer.Start(); err != nil {
		fmt.Printf("UI execution failed: %v\n", err)
	}
}

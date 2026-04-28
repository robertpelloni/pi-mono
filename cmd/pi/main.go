package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/frontends/bubbletea"
	"github.com/badlogic/pi-mono/pkg/frontends/cli"
	"github.com/badlogic/pi-mono/pkg/server"
	"github.com/badlogic/pi-mono/pkg/tools"
	"net/http"
)

type Renderer interface {
	Start() error
	RenderEvent(event agent.AgentEvent)
}

func main() {
	// Define CLI flags
	frontendType := flag.String("frontend", "bubbletea", "Select frontend UI (bubbletea, cli, web)")
	webPort := flag.Int("port", 8080, "Port for web UI")
	modelID := flag.String("model", "gpt-4o", "Select the AI model ID to use (e.g., gpt-4o, claude-3-5-sonnet-20240620, gemini-1.5-pro)")
	providerName := flag.String("provider", "openai", "Select the AI provider (openai, anthropic, google)")
	systemPrompt := flag.String("prompt", "You are an autonomous coding agent. Use tools whenever possible.", "Override the default system prompt")
	dir := flag.String("dir", "", "Set the working directory (defaults to current directory)")

	flag.Parse()

	// Determine working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting cwd:", err)
		return
	}
	if *dir != "" {
		cwd = *dir
		err = os.Chdir(cwd)
		if err != nil {
			fmt.Printf("Error changing to directory %s: %v\n", cwd, err)
			return
		}
	}

	// Map Provider
	var provider ai.Provider
	var streamFunc ai.StreamFunction

	switch *providerName {
	case "anthropic":
		provider = ai.ProviderAnthropic
		streamFunc = ai.StreamAnthropic
	case "google", "gemini":
		provider = ai.ProviderGoogle
		streamFunc = ai.StreamGoogle
	case "openai":
		fallthrough
	default:
		provider = ai.ProviderOpenAI
		streamFunc = ai.StreamOpenAIResponses
	}

	modelInfo := ai.ModelInfo{
		ID:       *modelID,
		Provider: provider,
		API:      ai.ApiOpenAIResponses, // Simplified for now
	}

	// Initialize Tools
	toolList := []agent.AgentTool{
		tools.ReadTool(cwd),
		tools.BashTool(cwd),
		tools.WriteTool(cwd),
		tools.LsTool(cwd),
		tools.GrepTool(cwd),
		tools.FindTool(cwd),
	}

	// Initialize Agent
	agentLoop := agent.NewAgent(modelInfo, toolList, streamFunc, agent.AgentLoopConfig{
		ToolExecution: agent.ToolExecutionParallel,
	})

	agentLoop.SetSystemPrompt(*systemPrompt)

	// Initialize Frontend Renderer
	var renderer Renderer

	if *frontendType == "web" {
		srv := server.NewServer("", agentLoop)
		fmt.Printf("Starting web UI on :%d\n", *webPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *webPort), srv); err != nil {
			fmt.Printf("Web server failed: %v\n", err)
		}
		return
	}

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

	// Start Application
	if err := renderer.Start(); err != nil {
		fmt.Printf("UI execution failed: %v\n", err)
	}
}

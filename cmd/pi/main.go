package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/extensions/acp_adapter"
	"github.com/badlogic/pi-mono/pkg/extensions/babysitter"
	"github.com/badlogic/pi-mono/pkg/extensions/plannotator"
	"github.com/badlogic/pi-mono/pkg/extensions/react_fallback"
	"github.com/badlogic/pi-mono/pkg/extensions/worktrees"
	"github.com/badlogic/pi-mono/pkg/frontends/bubbletea"
	"github.com/badlogic/pi-mono/pkg/frontends/cli"
	"github.com/badlogic/pi-mono/pkg/server"
	"github.com/badlogic/pi-mono/pkg/tools"
	pkgversion "github.com/badlogic/pi-mono/pkg"
)

// Version is set at build time via -ldflags
var Version = pkgversion.Version

type Renderer interface {
	Start() error
	RenderEvent(event agent.AgentEvent)
}

func main() {
	// Define CLI flags
	versionFlag := flag.Bool("version", false, "Print version and exit")
	frontendType := flag.String("frontend", "bubbletea", "Select frontend UI (bubbletea, cli, web)")
	webPort := flag.Int("port", 8080, "Port for web UI")
	modelID := flag.String("model", "", "Select the AI model ID (e.g., gpt-4o, claude-sonnet-4-20250514, gemini-2.5-pro)")
	providerName := flag.String("provider", "", "Select the AI provider (openai, anthropic, google)")
	systemPrompt := flag.String("prompt", "", "Override the default system prompt")
	dir := flag.String("dir", "", "Set the working directory (defaults to current directory)")
	thinkingLevel := flag.String("thinking", "", "Set thinking/reasoning level (low, medium, high, off)")
	toolMode := flag.String("tools", "parallel", "Tool execution mode (parallel, sequential)")
	apiKeyFlag := flag.String("api-key", "", "API key for the selected provider (overrides env var)")
	noTools := flag.Bool("no-tools", false, "Disable all built-in tools")
	offline := flag.Bool("offline", false, "Run in offline mode (disables version checks)")

	flag.Parse()

	// Version
	if *versionFlag {
		fmt.Printf("pi-go version %s\n", Version)
		os.Exit(0)
	}

	// Offline mode
	if *offline {
		os.Setenv("PI_OFFLINE", "1")
		os.Setenv("PI_SKIP_VERSION_CHECK", "1")
	}

	// Determine working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting cwd:", err)
		os.Exit(1)
	}
	if *dir != "" {
		cwd = *dir
		err = os.Chdir(cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error changing to directory %s: %v\n", cwd, err)
			os.Exit(1)
		}
	}

	// Auto-detect provider from model ID if not specified
	if *providerName == "" {
		*providerName = detectProvider(*modelID)
	}

	// Default model if not specified
	if *modelID == "" {
		switch *providerName {
		case "anthropic":
			*modelID = "claude-sonnet-4-20250514"
		case "google", "gemini":
			*modelID = "gemini-2.5-pro"
		default:
			*modelID = "gpt-4o"
		}
	}

	// Map provider to stream function and API type
	var provider ai.Provider
	var streamFunc ai.StreamFunction
	var apiType ai.Api

	switch *providerName {
	case "anthropic":
		provider = ai.ProviderAnthropic
		streamFunc = ai.StreamAnthropic
		apiType = ai.ApiAnthropicMessages
	case "google", "gemini":
		provider = ai.ProviderGoogle
		streamFunc = ai.StreamGoogle
		apiType = ai.ApiGoogleGenerativeAI
	case "openai":
		fallthrough
	default:
		provider = ai.ProviderOpenAI
		streamFunc = ai.StreamOpenAIResponses
		apiType = ai.ApiOpenAIResponses
	}

	// Override API key from CLI flag
	if *apiKeyFlag != "" {
		os.Setenv(providerAPIKeyEnv(provider), *apiKeyFlag)
	}

	modelInfo := ai.ModelInfo{
		ID:       *modelID,
		Provider: provider,
		API:      apiType,
	}

	// Default system prompt
	defaultSystemPrompt := "You are an autonomous coding agent. Use tools whenever possible. " +
		"When reading files, use the read tool. When running commands, use the bash tool. " +
		"When writing files, use the write tool. When editing files, use the edit tool. " +
		"When listing directories, use the ls tool. When searching, use grep or find."

	effectiveSystemPrompt := defaultSystemPrompt
	if *systemPrompt != "" {
		effectiveSystemPrompt = *systemPrompt
	}

	// Initialize Tools
	var toolList []agent.AgentTool
	if !*noTools {
		toolList = []agent.AgentTool{
			tools.ReadTool(cwd),
			tools.BashTool(cwd),
			tools.WriteTool(cwd),
			tools.EditTool(cwd),
			tools.LsTool(cwd),
			tools.GrepTool(cwd),
			tools.FindTool(cwd),
		}
	}

	// Initialize Community Plugins
	worktreePlugin := worktrees.NewWorktreePlugin()
	plannotatorPlugin := plannotator.NewPlannotatorPlugin()
	reactFallbackPlugin := react_fallback.NewReActFallbackPlugin()
	babysitterPlugin := babysitter.NewBabysitterPlugin()
	acpAdapterPlugin := acp_adapter.NewACPAdapterPlugin()

	// Optionally inject tools from plugins
	if !*noTools {
		toolList = worktreePlugin.AddTools(toolList)
		toolList = plannotatorPlugin.AddTools(toolList)
		toolList = babysitterPlugin.AddTools(toolList)
		toolList = acpAdapterPlugin.AddTools(toolList)
	}

	// Tool execution mode
	toolExecution := agent.ToolExecutionParallel
	if *toolMode == "sequential" {
		toolExecution = agent.ToolExecutionSequential
	}

	// Initialize Agent
	agentLoop := agent.NewAgent(modelInfo, toolList, streamFunc, agent.AgentLoopConfig{
		ToolExecution: toolExecution,
		AfterToolCall: reactFallbackPlugin.InterceptAfterToolCall,
	})
	agentLoop.SetSystemPrompt(effectiveSystemPrompt)

	// Set thinking level
	if *thinkingLevel != "" {
		agentLoop.SetThinkingLevel(ai.ThinkingLevel(*thinkingLevel))
	}

	// Print startup info
	fmt.Fprintf(os.Stderr, "pi-go v%s | model: %s | provider: %s | tools: %d | frontend: %s\n",
		Version, *modelID, *providerName, len(toolList), *frontendType)

	// Check for API key
	apiKey := ai.GetEnvAPIKey(provider)
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "\nWarning: No API key set for %s. Set %s environment variable or use --api-key flag.\n",
			*providerName, providerAPIKeyEnv(provider))
	}

	// Initialize Frontend Renderer
	if *frontendType == "web" {
		srv := server.NewServer("", agentLoop)
		fmt.Fprintf(os.Stderr, "Starting web UI on :%d\n", *webPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *webPort), srv); err != nil {
			fmt.Fprintf(os.Stderr, "Web server failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	var renderer Renderer
	switch *frontendType {
	case "cli":
		renderer = cli.NewCLIRenderer(agentLoop)
	case "bubbletea":
		renderer = bubbletea.NewInteractiveRenderer(agentLoop)
	default:
		fmt.Fprintf(os.Stderr, "Invalid frontend type %q. Falling back to bubbletea.\n", *frontendType)
		renderer = bubbletea.NewInteractiveRenderer(agentLoop)
	}

	agentLoop.Subscribe(renderer.RenderEvent)

	// Start Application
	if err := renderer.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "UI execution failed: %v\n", err)
		os.Exit(1)
	}
}

// detectProvider attempts to infer the provider from a model ID string.
func detectProvider(modelID string) string {
	if modelID == "" {
		return "openai"
	}
	// Anthropic patterns
	if len(modelID) >= 6 && modelID[:6] == "claude" {
		return "anthropic"
	}
	// Google patterns
	if len(modelID) >= 6 && (modelID[:6] == "gemini" || modelID[:6] == "gemma-") {
		return "google"
	}
	// OpenAI patterns
	if len(modelID) >= 3 && (modelID[:3] == "gpt" || modelID[:3] == "o1-" || modelID[:3] == "o3-") {
		return "openai"
	}
	if len(modelID) >= 4 && modelID[:4] == "codex" {
		return "openai"
	}
	// Default to OpenAI
	return "openai"
}

// providerAPIKeyEnv returns the environment variable name for a provider's API key.
func providerAPIKeyEnv(provider ai.Provider) string {
	switch provider {
	case ai.ProviderAnthropic:
		return "ANTHROPIC_API_KEY"
	case ai.ProviderGoogle:
		return "GOOGLE_API_KEY"
	case ai.ProviderOpenAI:
		return "OPENAI_API_KEY"
	default:
		return "API_KEY"
	}
}

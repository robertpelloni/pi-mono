package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/compaction"
	"github.com/badlogic/pi-mono/pkg/extensions/acp_adapter"
	"github.com/badlogic/pi-mono/pkg/extensions/babysitter"
	"github.com/badlogic/pi-mono/pkg/extensions/plannotator"
	"github.com/badlogic/pi-mono/pkg/extensions/react_fallback"
	"github.com/badlogic/pi-mono/pkg/extensions/worktrees"
	"github.com/badlogic/pi-mono/pkg/frontends/bubbletea"
	"github.com/badlogic/pi-mono/pkg/frontends/cli"
	"github.com/badlogic/pi-mono/pkg/modelresolver"
	"github.com/badlogic/pi-mono/pkg/outputguard"
	"github.com/badlogic/pi-mono/pkg/server"
	"github.com/badlogic/pi-mono/pkg/session"
	"github.com/badlogic/pi-mono/pkg/settings"
	"github.com/badlogic/pi-mono/pkg/skills"
	"github.com/badlogic/pi-mono/pkg/slashcommands"
	"github.com/badlogic/pi-mono/pkg/systemprompt"
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
	systemPromptFlag := flag.String("prompt", "", "Override the default system prompt")
	dir := flag.String("dir", "", "Set the working directory (defaults to current directory)")
	thinkingLevel := flag.String("thinking", "", "Set thinking/reasoning level (low, medium, high, off)")
	toolMode := flag.String("tools", "parallel", "Tool execution mode (parallel, sequential)")
	apiKeyFlag := flag.String("api-key", "", "API key for the selected provider (overrides env var)")
	noTools := flag.Bool("no-tools", false, "Disable all built-in tools")
	offline := flag.Bool("offline", false, "Run in offline mode (disables version checks)")
	continueSession := flag.Bool("continue", false, "Continue the most recent session")
	resumeSession := flag.Bool("resume", false, "Interactively select a session to resume")
	sessionID := flag.String("session", "", "Open a specific session by ID prefix")
	noSession := flag.Bool("no-session", false, "Run without persisting the session to disk")
	forkSession := flag.String("fork", "", "Fork from an existing session")
	listModels := flag.String("list-models", "", "List available models (optional: search pattern)")
	messageFlag := flag.String("message", "", "Send a single message and exit (non-interactive)")
	noSkills := flag.Bool("no-skills", false, "Disable loading of skills from ~/.pi/skills and .pi/skills")
	noGuard := flag.Bool("no-guard", false, "Disable output guard (secret redaction, safety checks)")
	compactThreshold := flag.Int("compact-threshold", 100000, "Token threshold for auto-compaction (0 to disable)")

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

	// ─── Initialize Settings ───
	agentDir, err := settings.InitAgentDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not init agent dir: %v\n", err)
		agentDir = settings.AgentDir()
	}
	settingsManager := settings.Create(cwd, agentDir)

	// Drain any non-fatal settings errors
	for _, se := range settingsManager.DrainErrors() {
		fmt.Fprintf(os.Stderr, "Warning: settings %s: %v\n", se.Scope, se.Error)
	}

	// ─── Initialize Model Registry ───
	modelRegistry := modelresolver.NewModelRegistryWithDefaults()

	// ─── List Models ───
	for _, arg := range os.Args[1:] {
		if arg == "--list-models" || strings.HasPrefix(arg, "--list-models=") {
			pattern := *listModels
			if strings.HasPrefix(arg, "--list-models=") {
				pattern = strings.TrimPrefix(arg, "--list-models=")
			}
			models := modelRegistry.AllModels()
			if pattern != "" {
				models = modelRegistry.Search(pattern)
			}
			fmt.Fprintf(os.Stderr, "Available models (%d):\n", len(models))
			for _, m := range models {
				reasoning := ""
				if m.Reasoning {
					reasoning = " [reasoning]"
				}
				fmt.Fprintf(os.Stderr, "  %-40s %-15s %s%s\n", m.ID, m.Provider, m.Name, reasoning)
			}
			os.Exit(0)
		}
	}

	// ─── Resolve Model & Provider ───
	if *providerName == "" && *modelID != "" {
		*providerName = detectProvider(*modelID)
	}
	if *providerName == "" {
		*providerName = settingsManager.GetDefaultProvider()
	}
	if *providerName == "" {
		*providerName = "openai"
	}

	if *modelID == "" {
		*modelID = settingsManager.GetDefaultModel()
	}
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

	var modelInfo ai.ModelInfo
	if resolved := modelRegistry.Find(*providerName, *modelID); resolved != nil {
		modelInfo = *resolved
	} else {
		modelInfo = ai.ModelInfo{
			ID:       *modelID,
			Provider: ai.Provider(*providerName),
			API:      providerToAPI(ai.Provider(*providerName)),
		}
	}

	// ─── Stream Function ───
	var streamFunc ai.StreamFunction
	switch *providerName {
	case "anthropic":
		streamFunc = ai.StreamAnthropic
	case "google", "gemini":
		streamFunc = ai.StreamGoogle
	default:
		streamFunc = ai.StreamOpenAIResponses
	}

	// Override API key from CLI flag
	if *apiKeyFlag != "" {
		os.Setenv(providerAPIKeyEnv(modelInfo.Provider), *apiKeyFlag)
	}

	// ─── Initialize Tools ───
	var toolList []agent.AgentTool
	if !*noTools {
		toolList = tools.CreateAllTools(cwd)
	}

	// Community Plugins
	worktreePlugin := worktrees.NewWorktreePlugin()
	plannotatorPlugin := plannotator.NewPlannotatorPlugin()
	reactFallbackPlugin := react_fallback.NewReActFallbackPlugin()
	babysitterPlugin := babysitter.NewBabysitterPlugin()
	acpAdapterPlugin := acp_adapter.NewACPAdapterPlugin()

	if !*noTools {
		toolList = worktreePlugin.AddTools(toolList)
		toolList = plannotatorPlugin.AddTools(toolList)
		toolList = babysitterPlugin.AddTools(toolList)
		toolList = acpAdapterPlugin.AddTools(toolList)
	}

	// ─── Load Skills ───
	var loadedSkills []skills.Skill
	if !*noSkills {
		skillLoader := skills.NewSkillLoader()
		skillResult := skillLoader.LoadSkills(agentDir, cwd)
		loadedSkills = skillResult.Skills
		for _, diag := range skillResult.Diagnostics {
			fmt.Fprintf(os.Stderr, "Warning: skill %s: %s\n", diag.Path, diag.Message)
		}
		if len(loadedSkills) > 0 {
			fmt.Fprintf(os.Stderr, "Loaded %d skills\n", len(loadedSkills))
		}
	}

	// ─── Build System Prompt ───
	var selectedToolNames []string
	for _, t := range toolList {
		selectedToolNames = append(selectedToolNames, t.Name)
	}

	toolSnippets := systemprompt.DefaultToolSnippets()
	customSnippets := systemprompt.BuildToolSnippetsFromAgentTools(toolList)
	for k, v := range customSnippets {
		toolSnippets[k] = v
	}

	contextFiles := systemprompt.LoadProjectContextFiles(cwd)
	skillRefs := skills.ToSkillRefs(loadedSkills)

	effectiveSystemPrompt := systemprompt.BuildSystemPrompt(systemprompt.BuildSystemPromptOptions{
		CustomPrompt:   *systemPromptFlag,
		SelectedTools:  selectedToolNames,
		ToolSnippets:   toolSnippets,
		ContextFiles:   contextFiles,
		Skills:         skillRefs,
		CWD:            cwd,
	})

	// ─── Output Guard ───
	var beforeToolCallHooks []func(ctx context.Context, callCtx agent.BeforeToolCallContext) (*agent.BeforeToolCallResult, error)
	var afterToolCallHooks []func(ctx context.Context, callCtx agent.AfterToolCallContext) (*agent.AfterToolCallResult, error)

	if !*noGuard {
		beforeToolCallHooks = append(beforeToolCallHooks, outputguard.InitBeforeHook(cwd))
		afterToolCallHooks = append(afterToolCallHooks, outputguard.InitAfterHook(cwd))
	}
	afterToolCallHooks = append(afterToolCallHooks, reactFallbackPlugin.InterceptAfterToolCall)

	// Compose BeforeToolCall from all hooks
	composedBeforeToolCall := func(ctx context.Context, callCtx agent.BeforeToolCallContext) (*agent.BeforeToolCallResult, error) {
		for _, hook := range beforeToolCallHooks {
			result, err := hook(ctx, callCtx)
			if err != nil {
				return result, err
			}
			if result != nil && result.Block {
				return result, nil
			}
		}
		return nil, nil
	}

	// Compose AfterToolCall from all hooks
	composedAfterToolCall := func(ctx context.Context, callCtx agent.AfterToolCallContext) (*agent.AfterToolCallResult, error) {
		for _, hook := range afterToolCallHooks {
			result, err := hook(ctx, callCtx)
			if err != nil {
				return result, err
			}
			if result != nil {
				return result, nil
			}
		}
		return nil, nil
	}

	// ─── Tool Execution Mode ───
	toolExecution := agent.ToolExecutionParallel
	if *toolMode == "sequential" {
		toolExecution = agent.ToolExecutionSequential
	}

	// ─── Initialize Agent ───
	agentConfig := agent.AgentLoopConfig{
		ToolExecution:  toolExecution,
		BeforeToolCall: composedBeforeToolCall,
		AfterToolCall:  composedAfterToolCall,
	}

	// ─── Compaction ───
	if *compactThreshold > 0 {
		compactor := compaction.NewCompactor(compaction.CompactionConfig{
			MaxTokens: *compactThreshold,
			Strategy:  compaction.StrategySummarize,
			KeepLastN: 6,
		})
		agentConfig.TransformContext = func(ctx context.Context, messages []ai.Message) ([]ai.Message, error) {
			if compactor.ShouldCompact(messages) {
				fmt.Fprintf(os.Stderr, "[Compaction] Context exceeds %d tokens, compacting...\n", *compactThreshold)
				return compactor.Compact(ctx, messages)
			}
			return messages, nil
		}
	}

	agentLoop := agent.NewAgent(modelInfo, toolList, streamFunc, agentConfig)
	agentLoop.SetSystemPrompt(effectiveSystemPrompt)

	if *thinkingLevel != "" {
		agentLoop.SetThinkingLevel(ai.ThinkingLevel(*thinkingLevel))
	}

	// ─── Slash Commands ───
	slashRegistry := slashcommands.NewRegistry()

	// Register dynamic model/provider switching
	slashRegistry.Register(slashcommands.SlashCommandInfo{
		Name:        "model",
		Description: "Switch the active model (e.g., /model gpt-4o)",
		Source:      slashcommands.SourceBuiltin,
	}, func(args string) (slashcommands.SlashCommandResult, error) {
		if args == "" {
			return slashcommands.SlashCommandResult{Info: "Usage: /model <model-id>"}, nil
		}
		return slashcommands.SlashCommandResult{SwitchModel: args}, nil
	})

	// ─── Initialize Session ───
	sessionDir := settingsManager.GetSessionDir()
	var sess *session.Session

	if *noSession {
		sess = session.InMemorySession()
	} else if *forkSession != "" {
		sess, err = session.ForkFrom(*forkSession, cwd, sessionDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error forking session: %v\n", err)
			os.Exit(1)
		}
	} else if *sessionID != "" {
		sessions, _ := session.ListSessions(cwd, sessionDir)
		for _, si := range sessions {
			if strings.HasPrefix(si.ID, *sessionID) {
				sess, err = session.OpenSession(si.Path, cwd)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error opening session %s: %v\n", si.ID, err)
					os.Exit(1)
				}
				break
			}
		}
		if sess == nil {
			fmt.Fprintf(os.Stderr, "No session found matching '%s'\n", *sessionID)
			os.Exit(1)
		}
	} else if *continueSession {
		sess, err = session.ContinueRecent(cwd, sessionDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error continuing session: %v\n", err)
			sess = session.NewSession(cwd, sessionDir)
		}
	} else if *resumeSession {
		sessions, _ := session.ListSessions(cwd, sessionDir)
		if len(sessions) == 0 {
			fmt.Fprintln(os.Stderr, "No sessions found")
			sess = session.NewSession(cwd, sessionDir)
		} else {
			fmt.Fprintf(os.Stderr, "Available sessions:\n")
			for i, si := range sessions {
				fmt.Fprintf(os.Stderr, "  %d. %s (updated %s, %d messages)\n",
					i+1, si.ID, si.Updated.Format("2006-01-02 15:04"), si.MessageCount)
			}
			sess, err = session.OpenSession(sessions[0].Path, cwd)
			if err != nil {
				sess = session.NewSession(cwd, sessionDir)
			}
		}
	} else {
		sess = session.NewSession(cwd, sessionDir)
	}

	// Restore session messages
	if len(sess.Messages()) > 0 {
		agentLoop.SetMessages(sess.Messages())
	}

	// Persist messages to session
	agentLoop.Subscribe(func(event agent.AgentEvent) {
		if event.Type == agent.EventMessageEnd && event.Message != nil {
			sess.AppendMessage(event.Message)
		}
	})

	// ─── Startup Banner ───
	fmt.Fprintf(os.Stderr, "pi-go v%s | model: %s | provider: %s | tools: %d | frontend: %s | session: %s\n",
		Version, *modelID, *providerName, len(toolList), *frontendType, sess.ID)

	if len(contextFiles) > 0 {
		fmt.Fprintf(os.Stderr, "  context: %s\n", contextFileNames(contextFiles))
	}
	if len(loadedSkills) > 0 {
		fmt.Fprintf(os.Stderr, "  skills: %s\n", skillNames(loadedSkills))
	}

	apiKey := ai.GetEnvAPIKey(modelInfo.Provider)
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "\n⚠ No API key set for %s. Set %s or use --api-key.\n",
			*providerName, providerAPIKeyEnv(modelInfo.Provider))
	}

	// ─── Single Message Mode ───
	if *messageFlag != "" {
		userMsg := ai.UserMessage{
			Content:   []ai.Content{ai.TextContent{Text: *messageFlag}},
		}
		renderer := cli.NewCLIRenderer(agentLoop)
		agentLoop.Subscribe(renderer.RenderEvent)

		err := agentLoop.Prompt(context.Background(), userMsg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// ─── Web Frontend ───
	if *frontendType == "web" {
		srv := server.NewServer("", agentLoop)
		fmt.Fprintf(os.Stderr, "Starting web UI on :%d\n", *webPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *webPort), srv); err != nil {
			fmt.Fprintf(os.Stderr, "Web server failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// ─── Interactive Frontend ───
	var renderer Renderer
	switch *frontendType {
	case "cli":
		renderer = cli.NewCLIRenderer(agentLoop)
	case "bubbletea":
		renderer = bubbletea.NewInteractiveRendererWithSlashCommands(agentLoop, slashRegistry)
	default:
		fmt.Fprintf(os.Stderr, "Invalid frontend %q. Falling back to bubbletea.\n", *frontendType)
		renderer = bubbletea.NewInteractiveRendererWithSlashCommands(agentLoop, slashRegistry)
	}

	agentLoop.Subscribe(renderer.RenderEvent)

	// Start the UI
	if err := renderer.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "UI execution failed: %v\n", err)
		os.Exit(1)
	}

	// Slash commands are handled inside the frontend's input loop
}

// ─── Helpers ───

func detectProvider(modelID string) string {
	m := strings.ToLower(modelID)
	if strings.HasPrefix(m, "claude") {
		return "anthropic"
	}
	if strings.HasPrefix(m, "gemini") || strings.HasPrefix(m, "gemma-") {
		return "google"
	}
	if strings.HasPrefix(m, "gpt") || strings.HasPrefix(m, "o1-") || strings.HasPrefix(m, "o3-") || strings.HasPrefix(m, "o4-") || strings.HasPrefix(m, "codex") {
		return "openai"
	}
	return "openai"
}

func providerAPIKeyEnv(provider ai.Provider) string {
	switch provider {
	case ai.ProviderAnthropic:
		return "ANTHROPIC_API_KEY"
	case ai.ProviderGoogle:
		return "GEMINI_API_KEY"
	case ai.ProviderOpenAI:
		return "OPENAI_API_KEY"
	default:
		return "API_KEY"
	}
}

func providerToAPI(provider ai.Provider) ai.Api {
	switch provider {
	case ai.ProviderAnthropic:
		return ai.ApiAnthropicMessages
	case ai.ProviderGoogle:
		return ai.ApiGoogleGenerativeAI
	case ai.ProviderOpenAI:
		return ai.ApiOpenAIResponses
	case ai.ProviderAzureOpenAI:
		return ai.ApiAzureOpenAIResponses
	case ai.ProviderAmazonBedrock:
		return ai.ApiBedrockConverseStream
	case ai.ProviderMistral:
		return ai.ApiMistralConversations
	default:
		return ai.ApiOpenAICompletions
	}
}

func contextFileNames(files []systemprompt.ContextFile) string {
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.Path
	}
	return strings.Join(names, ", ")
}

func skillNames(s []skills.Skill) string {
	names := make([]string, len(s))
	for i, sk := range s {
		names[i] = sk.Name
	}
	return strings.Join(names, ", ")
}

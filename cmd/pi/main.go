package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/agentsession"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/auth"
	"github.com/badlogic/pi-mono/pkg/compaction"
	"github.com/badlogic/pi-mono/pkg/extensions/acp_adapter"
	"github.com/badlogic/pi-mono/pkg/extensions/babysitter"
	"github.com/badlogic/pi-mono/pkg/extensions/plannotator"
	"github.com/badlogic/pi-mono/pkg/extensions/worktrees"
	"github.com/badlogic/pi-mono/pkg/fileprocessor"
	"github.com/badlogic/pi-mono/pkg/frontends/bubbletea"
	"github.com/badlogic/pi-mono/pkg/frontends/cli"
	cliflags "github.com/badlogic/pi-mono/pkg/listmodels"
	"path/filepath"

	"github.com/badlogic/pi-mono/pkg/modelregistry"
	"github.com/badlogic/pi-mono/pkg/modelresolver"
	"github.com/badlogic/pi-mono/pkg/outputguard"
	"github.com/badlogic/pi-mono/pkg/server"
	"github.com/badlogic/pi-mono/pkg/session"
	"github.com/badlogic/pi-mono/pkg/sessionruntime"
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
	helpFlag := flag.Bool("help", false, "Show help")
	frontendType := flag.String("frontend", "bubbletea", "Select frontend UI (bubbletea, cli, web)")
	webPort := flag.Int("port", 8080, "Port for web UI")
	modelID := flag.String("model", "", "Select the AI model ID")
	providerName := flag.String("provider", "", "Select the AI provider")
	systemPromptFlag := flag.String("prompt", "", "Override the default system prompt")
	appendSystemPromptFlag := flag.String("append-prompt", "", "Append to the default system prompt")
	dir := flag.String("dir", "", "Set the working directory")
	thinkingLevelFlag := flag.String("thinking", "", "Set thinking/reasoning level (off, minimal, low, medium, high, xhigh)")
	toolMode := flag.String("tools", "parallel", "Tool execution mode (parallel, sequential)")
	apiKeyFlag := flag.String("api-key", "", "API key for the selected provider")
	noTools := flag.Bool("no-tools", false, "Disable all built-in tools")
	offline := flag.Bool("offline", false, "Run in offline mode")
	continueSession := flag.Bool("continue", false, "Continue the most recent session")
	resumeSession := flag.Bool("resume", false, "Select a session to resume")
	sessionID := flag.String("session", "", "Open a specific session by ID")
	noSession := flag.Bool("no-session", false, "Run without persisting the session")
	forkSession := flag.String("fork", "", "Fork from an existing session")
	listModelsFlag := flag.Bool("list-models", false, "List available models")
	listModelsPattern := flag.String("list-models-pattern", "", "Filter model list by pattern")
	messageFlag := flag.String("message", "", "Send a single message and exit (non-interactive)")
	printFlag := flag.Bool("print", false, "Non-interactive mode: process prompt and exit")
	noSkills := flag.Bool("no-skills", false, "Disable loading of skills")
	noGuard := flag.Bool("no-guard", false, "Disable output guard")
	compactThreshold := flag.Int("compact-threshold", 100000, "Token threshold for auto-compaction (0 to disable)")
	noExtensions := flag.Bool("no-extensions", false, "Disable extension discovery")
	enablePlannotator := flag.Bool("plannotator", false, "Enable the interactive pi-plannotator plan review tool")

	flag.Parse()

	// Help
	if *helpFlag {
		fmt.Fprintf(os.Stderr, "pi-go - AI coding assistant with read, bash, edit, write tools\n\n")
		fmt.Fprintf(os.Stderr, "Usage: pi-go [options] [@files...] [messages...]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		os.Exit(0)
	}

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
		if err := os.Chdir(cwd); err != nil {
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
	for _, se := range settingsManager.DrainErrors() {
		fmt.Fprintf(os.Stderr, "Warning: settings %s: %v\n", se.Scope, se.Error)
	}

	// ─── Initialize Auth Storage ───
	authPath := agentDir + string(os.PathSeparator) + "auth.json"
	authStorage := auth.NewAuthStorage(authPath)
	if err := authStorage.DrainErrors(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: auth storage error: %v\n", err)
	}

	// Override API key from CLI flag
	if *apiKeyFlag != "" {
		authStorage.SetRuntimeAPIKey(*providerName, *apiKeyFlag)
	}

	// ─── Initialize Model Registry ───
	modelRegistry := modelresolver.NewModelRegistryWithDefaults()

	// Load models.json if it exists
	modelsJSONPath := modelresolver.GetModelsJSONPath(agentDir)
	if err := modelRegistry.LoadFromModelsJSON(modelsJSONPath, authStorage); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: models.json error: %v\n", err)
	}

	// Load provider-cache.json if it exists (adds dynamic models like ollama-cloud)
	providerCachePath := filepath.Join(agentDir, "provider-cache.json")
	if _, err := os.Stat(providerCachePath); err == nil {
		if cacheModels, err := modelregistry.LoadProviderCacheModels(providerCachePath); err == nil {
			for _, m := range cacheModels {
				modelRegistry.Register(m)
			}
		}
	}

	// ─── List Models ───
	if *listModelsFlag {
		available := modelRegistry.GetAvailable(authStorage)
		cliflags.ListModels(available, *listModelsPattern, os.Stdout)
		os.Exit(0)
	}

	// ─── Resolve Model & Provider ───
	if *providerName == "" && *modelID != "" {
		*providerName = modelresolver.DetectProvider(*modelID)
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

	// ─── Resolve Thinking Level ───
	thinkingLevel := ai.ThinkingLevel("medium")
	if *thinkingLevelFlag != "" {
		thinkingLevel = ai.ThinkingLevel(*thinkingLevelFlag)
	}
	if !modelInfo.Reasoning {
		thinkingLevel = ai.ThinkingLevel("off")
	}

	// ─── Process @file arguments ───
	var fileText string
	var fileImages []ai.ImageContent
	fileArgs := flag.Args()
	filePaths := extractFileArgs(fileArgs)
	messages := extractMessages(fileArgs)

	if len(filePaths) > 0 {
		fileContent, err := fileprocessor.ProcessFileArgs(cwd, filePaths)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: file processing error: %v\n", err)
		} else {
			fileText = fileContent.Text
			for _, img := range fileContent.Images {
				fileImages = append(fileImages, ai.ImageContent{
					Data:     img.Data,
					MimeType: img.MimeType,
				})
			}
		}
	}

	// Build initial message from stdin, @files, and CLI messages
	stdinContent := readStdinIfPiped()
	initialMessage := fileprocessor.BuildInitialMessage(stdinContent, fileText, messages)

	// ─── Initialize Tools ───
	var toolList []agent.AgentTool
	if !*noTools {
		toolList = tools.CreateAllTools(cwd)
	}

	// Community Plugins
	if !*noExtensions {
		worktreePlugin := worktrees.NewWorktreePlugin()
		plannotatorPlugin := plannotator.NewPlannotatorPlugin()
		if *enablePlannotator {
			plannotatorPlugin.Enabled = true
		}
		babysitterPlugin := babysitter.NewBabysitterPlugin()
		acpAdapterPlugin := acp_adapter.NewACPAdapterPlugin()

		if !*noTools {
			toolList = worktreePlugin.AddTools(toolList)
			toolList = plannotatorPlugin.AddTools(toolList)
			toolList = babysitterPlugin.AddTools(toolList)
			toolList = acpAdapterPlugin.AddTools(toolList)
		}
	}

	// ─── Load Skills ───
	var loadedSkills []skills.Skill
	var skillLoader *skills.SkillLoader
	if !*noSkills {
		skillLoader = skills.NewSkillLoader()
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
		CustomPrompt:        *systemPromptFlag,
		AppendSystemPrompt:  *appendSystemPromptFlag,
		SelectedTools:       selectedToolNames,
		ToolSnippets:        toolSnippets,
		ContextFiles:        contextFiles,
		Skills:              skillRefs,
		CWD:                 cwd,
	})

	// ─── Output Guard ───
	var beforeToolCallHooks []func(ctx context.Context, callCtx agent.BeforeToolCallContext) (*agent.BeforeToolCallResult, error)
	var afterToolCallHooks []func(ctx context.Context, callCtx agent.AfterToolCallContext) (*agent.AfterToolCallResult, error)
	if !*noGuard {
		beforeToolCallHooks = append(beforeToolCallHooks, outputguard.InitBeforeHook(cwd))
		afterToolCallHooks = append(afterToolCallHooks, outputguard.InitAfterHook(cwd))
	}

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
		ToolExecution:   toolExecution,
		BeforeToolCall:  composedBeforeToolCall,
		AfterToolCall:   composedAfterToolCall,
	}

	// ─── Compaction ───
	var compactor *compaction.Compactor
	if *compactThreshold > 0 {
		compactor = compaction.NewCompactor(compaction.CompactionConfig{
			MaxTokens: *compactThreshold,
			Strategy: compaction.StrategySummarize,
			KeepLastN: 6,
			SummarizeFn: func(ctx context.Context, messages []ai.Message) (string, error) {
				prompt := compaction.PrepareSummarizationPrompt(messages, nil, nil)
				_ = prompt // Will be sent to LLM for summarization
				result := compaction.GenerateDefaultSummary(messages, nil, nil, compaction.NewFileOps())
				return result.Summary, nil
			},
		})
		agentConfig.TransformContext = func(ctx context.Context, messages []ai.Message) ([]ai.Message, error) {
			if compactor.ShouldCompact(messages) {
				fmt.Fprintf(os.Stderr, "[Compaction] Context exceeds %d tokens, compacting.\n", *compactThreshold)
				return compactor.Compact(ctx, messages)
			}
			return messages, nil
		}
	}

	agentLoop := agent.NewAgent(modelInfo, toolList, streamFunc, agentConfig)
	agentLoop.SetSystemPrompt(effectiveSystemPrompt)
	agentLoop.SetThinkingLevel(thinkingLevel)

	// ─── Slash Commands ───
	slashRegistry := slashcommands.NewRegistry()
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
	var sess *session.SessionManager
	if *noSession {
		sess = session.InMemorySession(cwd)
	} else if *forkSession != "" {
		sess = session.ForkFrom(*forkSession, cwd, sessionDir)
	} else if *sessionID != "" {
		sessions, _ := session.ListSessions(cwd, sessionDir)
		for _, si := range sessions {
			if strings.HasPrefix(si.ID, *sessionID) {
				sess = session.OpenSession(si.Path, sessionDir, nil)
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
		sess = session.ContinueRecent(cwd, sessionDir)
		if sess == nil {
			sess = session.CreateSession(cwd, sessionDir)
		}
	} else if *resumeSession {
		sessions, _ := session.ListSessions(cwd, sessionDir)
		if len(sessions) == 0 {
			sess = session.CreateSession(cwd, sessionDir)
		} else {
			fmt.Fprintf(os.Stderr, "Available sessions:\n")
			for i, si := range sessions {
				fmt.Fprintf(os.Stderr, " %d. %s (updated %s, %d messages)\n", i+1, si.ID, si.Modified.Format("2006-01-02 15:04"), si.MessageCount)
			}
			sess = session.OpenSession(sessions[0].Path, sessionDir, nil)
			if err != nil {
				sess = session.CreateSession(cwd, sessionDir)
			}
		}
	} else {
		sess = session.CreateSession(cwd, sessionDir)
	}

	if ctx := sess.BuildSessionContext(); len(ctx.Messages) > 0 {
		agentLoop.SetMessages(ctx.Messages)
	}

	agentLoop.Subscribe(func(event agent.AgentEvent) {
		if event.Type == agent.EventMessageEnd && event.Message != nil {
			sess.AppendMessage(event.Message)
		}
	})

	// ─── Create AgentSession (wraps Agent + Session + Retry + Compaction) ───
	var agentSess *agentsession.AgentSession
	agentSess = agentsession.NewAgentSession(agentsession.AgentSessionConfig{
		Agent:          agentLoop,
		SessionManager: sess,
		Settings:       settingsManager,
		ModelRegistry:  modelRegistry,
		SkillLoader:    skillLoader,
		Compactor:      compactor,
		SlashCommands:  slashRegistry,
		CWD:            cwd,
		AgentDir:       agentDir,
	})

	// ─── Create Session Runtime ───
	var runtime *sessionruntime.AgentSessionRuntime
	runtime, runtimeErr := sessionruntime.CreateAgentSessionRuntime(
		sessionruntime.DefaultCreateRuntime,
		sessionruntime.CreateAgentSessionRuntimeOptions{
			CWD:             cwd,
			AgentDir:        agentDir,
			SessionManager:  sess,
			SettingsManager: settingsManager,
			ModelRegistry:   modelRegistry,
			SlashRegistry:   slashRegistry,
			Model:           &modelInfo,
			StreamFn:        streamFunc,
			Tools:           toolList,
		},
	)
	_ = runtime
	if runtimeErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: session runtime init failed: %v\n", runtimeErr)
	}

	// ─── Startup Banner ───
	fmt.Fprintf(os.Stderr, "pi-go v%s | model: %s | provider: %s | tools: %d | frontend: %s | session: %s\n",
		Version, *modelID, *providerName, len(toolList), *frontendType, sess.GetSessionID())
	if len(contextFiles) > 0 {
		fmt.Fprintf(os.Stderr, " context: %s\n", contextFileNames(contextFiles))
	}
	if len(loadedSkills) > 0 {
		fmt.Fprintf(os.Stderr, " skills: %s\n", skillNames(loadedSkills))
	}

	apiKey := authStorage.GetAPIKey(*providerName)
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "\n⚠ No API key set for %s. Set %s or use --api-key.\n",
			*providerName, providerAPIKeyEnv(ai.Provider(*providerName)))
	}

	// ─── Print/Non-interactive Mode ───
	if *printFlag || *messageFlag != "" {
		msg := *messageFlag
		if msg == "" {
			msg = initialMessage
		}
		if msg == "" && stdinContent != "" {
			msg = stdinContent
		}

		userMsg := ai.UserMessage{
			Content: []ai.Content{ai.TextContent{Text: msg}},
		}
		for _, img := range fileImages {
			userMsg.Content = append(userMsg.Content, img)
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
		if agentSess != nil {
			renderer = bubbletea.NewInteractiveRendererWithAgentSession(agentSess, slashRegistry)
		} else {
			renderer = bubbletea.NewInteractiveRendererWithSlashCommands(agentLoop, slashRegistry)
		}
	default:
		fmt.Fprintf(os.Stderr, "Invalid frontend %q. Falling back to bubbletea.\n", *frontendType)
		if agentSess != nil {
			renderer = bubbletea.NewInteractiveRendererWithAgentSession(agentSess, slashRegistry)
		} else {
			renderer = bubbletea.NewInteractiveRendererWithSlashCommands(agentLoop, slashRegistry)
		}
	}
	agentLoop.Subscribe(renderer.RenderEvent)

	if err := renderer.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "UI execution failed: %v\n", err)
		os.Exit(1)
	}
}

// ─── Helpers ───

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

// extractFileArgs extracts @file arguments from the CLI args.
func extractFileArgs(args []string) []string {
	var fileArgs []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "@") {
			fileArgs = append(fileArgs, arg[1:])
		}
	}
	return fileArgs
}

// extractMessages extracts non-file arguments (plain messages).
func extractMessages(args []string) []string {
	var messages []string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "@") && !strings.HasPrefix(arg, "-") {
			messages = append(messages, arg)
		}
	}
	return messages
}

// readStdinIfPiped reads stdin if it's piped (not a terminal).
func readStdinIfPiped() string {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "" // Terminal, not piped
	}
	buf := make([]byte, 1024*1024) // 1MB max
	n, _ := os.Stdin.Read(buf)
	if n > 0 {
		return string(buf[:n])
	}
	return ""
}

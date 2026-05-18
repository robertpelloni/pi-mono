package sdk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/compaction"
	"github.com/badlogic/pi-mono/pkg/modelresolver"
	"github.com/badlogic/pi-mono/pkg/resourceloader"
	"github.com/badlogic/pi-mono/pkg/session"
	"github.com/badlogic/pi-mono/pkg/settings"
	"github.com/badlogic/pi-mono/pkg/skills"
	"github.com/badlogic/pi-mono/pkg/systemprompt"
	"github.com/badlogic/pi-mono/pkg/tools"
)

const defaultThinkingLevel = "medium"

// CreateAgentSessionOptions configures session creation.
type CreateAgentSessionOptions struct {
	CWD           string
	AgentDir      string
	ModelID       string
	Provider      string
	ThinkingLevel string
	NoTools       bool
	NoSkills      bool
	NoSession     bool
	Continue      bool
	Resume        bool
	SessionID     string
	ForkSession   string
	SystemPrompt  string
	APIKey        string
	CompactThreshold int
}

// CreateAgentSessionResult holds the result of creating a session.
type CreateAgentSessionResult struct {
	Agent              *agent.Agent
	Session            *session.SessionManager
	SettingsManager    *settings.SettingsManager
	ModelRegistry      *modelresolver.ModelRegistry
	ResourceLoader     *resourceloader.ResourceLoader
	LoadedSkills       []skills.Skill
	ContextFiles       []systemprompt.ContextFile
	ModelFallbackMsg   string
	Diagnostics        []Diagnostic
}

// Diagnostic represents a non-fatal setup issue.
type Diagnostic struct {
	Type    string `json:"type"` // "info", "warning", "error"
	Message string `json:"message"`
}

// CreateAgentSession creates a fully configured agent session from options.
// This is the Go equivalent of the TypeScript SDK's createAgentSession().
func CreateAgentSession(ctx context.Context, opts CreateAgentSessionOptions) (*CreateAgentSessionResult, error) {
	result := &CreateAgentSessionResult{}
	var diagnostics []Diagnostic
	var err error

	// ─── Determine CWD ───
	cwd := opts.CWD
	if cwd == "" {
			cwd, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("error getting cwd: %w", err)
		}
	}
	if opts.CWD != "" && opts.CWD != cwd {
		cwd = opts.CWD
		if err := os.Chdir(cwd); err != nil {
			return nil, fmt.Errorf("error changing to directory %s: %w", cwd, err)
		}
	}

	// ─── Initialize Settings ───
	agentDir := opts.AgentDir
	if agentDir == "" {
			agentDir, err = settings.InitAgentDir()
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{Type: "warning", Message: fmt.Sprintf("could not init agent dir: %v", err)})
			agentDir = settings.AgentDir()
		}
	}
	settingsManager := settings.Create(cwd, agentDir)
	for _, se := range settingsManager.DrainErrors() {
		diagnostics = append(diagnostics, Diagnostic{Type: "warning", Message: fmt.Sprintf("settings %s: %v", se.Scope, se.Error)})
	}

	// ─── Initialize Model Registry ───
	modelRegistry := modelresolver.NewModelRegistryWithDefaults()

	// ─── Resolve Model & Provider ───
	provider := opts.Provider
	if provider == "" && opts.ModelID != "" {
		provider = detectProvider(opts.ModelID)
	}
	if provider == "" {
		provider = settingsManager.GetDefaultProvider()
	}
	if provider == "" {
		provider = "openai"
	}

	modelID := opts.ModelID
	if modelID == "" {
		modelID = settingsManager.GetDefaultModel()
	}
	if modelID == "" {
		switch provider {
		case "anthropic":
			modelID = "claude-sonnet-4-20250514"
		case "google", "gemini":
			modelID = "gemini-2.5-pro"
		default:
			modelID = "gpt-4o"
		}
	}

	var modelInfo ai.ModelInfo
	if resolved := modelRegistry.Find(provider, modelID); resolved != nil {
		modelInfo = *resolved
	} else {
		modelInfo = ai.ModelInfo{
			ID:       modelID,
			Provider: ai.Provider(provider),
			API:      providerToAPI(ai.Provider(provider)),
		}
	}

	// ─── Stream Function ───
	streamFunc := ai.StreamOpenAIResponses
	switch provider {
	case "anthropic":
		streamFunc = ai.StreamAnthropic
	case "google", "gemini":
		streamFunc = ai.StreamGoogle
	}

	// Override API key from CLI flag
	if opts.APIKey != "" {
		os.Setenv(providerAPIKeyEnv(modelInfo.Provider), opts.APIKey)
	}

	// ─── Initialize Tools ───
	var toolList []agent.AgentTool
	if !opts.NoTools {
		toolList = tools.CreateAllTools(cwd)
	}

	// ─── Load Skills ───
	var loadedSkills []skills.Skill
	if !opts.NoSkills {
		skillLoader := skills.NewSkillLoader()
		skillResult := skillLoader.LoadSkills(agentDir, cwd)
		loadedSkills = skillResult.Skills
		for _, diag := range skillResult.Diagnostics {
			diagnostics = append(diagnostics, Diagnostic{Type: "warning", Message: fmt.Sprintf("skill %s: %s", diag.Path, diag.Message)})
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
		CustomPrompt:  opts.SystemPrompt,
		SelectedTools: selectedToolNames,
		ToolSnippets:  toolSnippets,
		ContextFiles:  contextFiles,
		Skills:        skillRefs,
		CWD:           cwd,
	})

	// ─── Initialize Agent ───
	agentConfig := agent.AgentLoopConfig{
		ToolExecution: agent.ToolExecutionParallel,
	}

	// ─── Compaction ───
	if opts.CompactThreshold > 0 {
		compactor := compaction.NewCompactor(compaction.CompactionConfig{
			MaxTokens: opts.CompactThreshold,
			Strategy:  compaction.StrategySummarize,
			KeepLastN: 6,
		})
		agentConfig.TransformContext = func(ctx context.Context, messages []ai.Message) ([]ai.Message, error) {
			if compactor.ShouldCompact(messages) {
				fmt.Fprintf(os.Stderr, "[Compaction] Context exceeds %d tokens, compacting...\n", opts.CompactThreshold)
				return compactor.Compact(ctx, messages)
			}
			return messages, nil
		}
	}

	agentLoop := agent.NewAgent(modelInfo, toolList, streamFunc, agentConfig)
	agentLoop.SetSystemPrompt(effectiveSystemPrompt)

	thinkingLevel := opts.ThinkingLevel
	if thinkingLevel == "" {
		thinkingLevel = defaultThinkingLevel
	}
	agentLoop.SetThinkingLevel(ai.ThinkingLevel(thinkingLevel))

	// ─── Initialize Session ───
	sessionDir := settingsManager.GetSessionDir()
	var sess *session.SessionManager

	if opts.NoSession {
		sess = session.InMemorySession(cwd)
	} else if opts.ForkSession != "" {
		sess = session.ForkFrom(opts.ForkSession, cwd, sessionDir)
	} else if opts.SessionID != "" {
		sessions, _ := session.ListSessions(cwd, sessionDir)
		for _, si := range sessions {
			if strings.HasPrefix(si.ID, opts.SessionID) {
				sess = session.OpenSession(si.Path, sessionDir, nil)
				break
			}
		}
		if sess == nil {
			return nil, fmt.Errorf("no session found matching '%s'", opts.SessionID)
		}
	} else if opts.Continue {
		sess = session.ContinueRecent(cwd, sessionDir)
		if sess == nil {
			sess = session.CreateSession(cwd, sessionDir)
		}
	} else {
		sess = session.CreateSession(cwd, sessionDir)
	}

	// Restore session messages
	if ctx := sess.BuildSessionContext(); len(ctx.Messages) > 0 {
		agentLoop.SetMessages(ctx.Messages)
	}

	// Persist messages to session
	agentLoop.Subscribe(func(event agent.AgentEvent) {
		if event.Type == agent.EventMessageEnd && event.Message != nil {
			sess.AppendMessage(event.Message)
		}
	})

	// Check API key availability
	apiKey := ai.GetEnvAPIKey(modelInfo.Provider)
	if apiKey == "" {
		diagnostics = append(diagnostics, Diagnostic{
			Type:    "warning",
			Message: fmt.Sprintf("No API key set for %s. Set %s or use --api-key.", provider, providerAPIKeyEnv(modelInfo.Provider)),
		})
	}

	result.Agent = agentLoop
	result.Session = sess
	result.SettingsManager = settingsManager
	result.ModelRegistry = modelRegistry
	result.LoadedSkills = loadedSkills
	result.ContextFiles = contextFiles
	result.Diagnostics = diagnostics

	return result, nil
}

// Helper functions

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

// InitAgentDir initializes and returns the agent directory path.
func InitAgentDir() (string, error) {
	dir, err := settings.InitAgentDir()
	if err != nil {
		return dir, err
	}
	return dir, nil
}

// Ensure filepath is used
var _ = filepath.Join

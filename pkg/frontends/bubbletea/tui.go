package bubbletea

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/agentsession"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/agentregistry"
	"github.com/badlogic/pi-mono/pkg/slashcommands"
	"github.com/badlogic/pi-mono/pkg/util"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// AgentUIModel represents the Bubbletea state for the interactive AI agent interface.
type AgentUIModel struct {
	eventsChan    chan agent.AgentEvent
	sessionEvents chan agentsession.AgentSessionEvent
	conversation  strings.Builder
	viewport      viewport.Model
	textarea      textarea.Model
	agent         *agent.Agent
	agentSession  *agentsession.AgentSession
	slashRegistry *slashcommands.Registry
	activeTool    string
	isGenerating  bool
	err           error
	quitting      bool
	statusLine    string
	modelInfo     string
	spinner       spinner.Model
	subagentActive bool
	cronjobCount   int

	// Autocompletion state
	showCompletions bool
	completions     []string
	completionIndex int
	completionPrefix string // text before the cursor that triggered completion
}

// EventMsg is a wrapper to send AgentEvent instances into the Bubbletea Update loop.
type EventMsg agent.AgentEvent

// SessionEventMsg wraps AgentSessionEvent for the tea loop.
type SessionEventMsg agentsession.AgentSessionEvent

// ExecutionDoneMsg signals that an agent generation loop has returned
type ExecutionDoneMsg struct {
	Err error
}

// SlashResultMsg wraps a slash command result for the tea loop.
type SlashResultMsg struct {
	Result slashcommands.SlashCommandResult
}

// InitialModel creates the initial Bubbletea model using the raw Agent.
func InitialModel(ag *agent.Agent, eventsChan chan agent.AgentEvent, slashReg *slashcommands.Registry) *AgentUIModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = StyleAssistant

	ta := textarea.New()
	ta.Placeholder = "Type a message... (Ctrl+S to send, / for commands)"
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 20000
	ta.SetWidth(80)
	ta.SetHeight(3)

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to the Pi Go Agent CLI! Type your prompt below.")

	return &AgentUIModel{
		eventsChan:    eventsChan,
		viewport:      vp,
		textarea:      ta,
		agent:         ag,
		slashRegistry: slashReg,
		spinner:       s,
	}
}

// InitialModelWithSession creates the initial Bubbletea model using an AgentSession.
func InitialModelWithSession(as *agentsession.AgentSession, eventsChan chan agent.AgentEvent, sessionEvents chan agentsession.AgentSessionEvent, slashReg *slashcommands.Registry) *AgentUIModel {
	model := InitialModel(as.Agent(), eventsChan, slashReg)
	model.agentSession = as
	model.sessionEvents = sessionEvents
	if m := as.Model(); m.ID != "" {
		model.modelInfo = fmt.Sprintf("%s/%s", m.Provider, m.ID)
	}
	return model
}

// Init establishes the initial state and begins listening to the channel.
func (m *AgentUIModel) Init() tea.Cmd {
	cmds := []tea.Cmd{textarea.Blink, m.listenForEvents(), m.spinner.Tick}
	if m.sessionEvents != nil {
		cmds = append(cmds, m.listenForSessionEvents())
	}
	return tea.Batch(cmds...)
}

// listenForEvents reads from the channels and dispatches messages to the tea loop.
func (m *AgentUIModel) listenForEvents() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.eventsChan
		if !ok {
			return nil
		}
		return EventMsg(event)
	}
}

// listenForSessionEvents reads from the session event channel.
func (m *AgentUIModel) listenForSessionEvents() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.sessionEvents
		if !ok {
			return nil
		}
		return SessionEventMsg(event)
	}
}

// Update processes incoming messages and updates the model state.
func (m *AgentUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	if m.isGenerating {
		m.spinner, spCmd = m.spinner.Update(msg)
	}

	if m.showCompletions {
		if kmsg, ok := msg.(tea.KeyMsg); ok {
			switch kmsg.Type {
			case tea.KeyUp, tea.KeyTab:
				m.completionIndex--
				if m.completionIndex < 0 {
					m.completionIndex = len(m.completions) - 1
				}
				m.textarea, tiCmd = m.textarea.Update(msg)
				m.viewport, vpCmd = m.viewport.Update(msg)
				return m, tea.Batch(tiCmd, vpCmd, spCmd)
			case tea.KeyDown:
				m.completionIndex++
				if m.completionIndex >= len(m.completions) {
					m.completionIndex = 0
				}
				m.textarea, tiCmd = m.textarea.Update(msg)
				m.viewport, vpCmd = m.viewport.Update(msg)
				return m, tea.Batch(tiCmd, vpCmd, spCmd)
			case tea.KeyEnter:
				m.applyCompletion()
				m.textarea, tiCmd = m.textarea.Update(msg)
				m.viewport, vpCmd = m.viewport.Update(msg)
				return m, tea.Batch(tiCmd, vpCmd, spCmd)
			case tea.KeyEsc:
				m.showCompletions = false
				m.textarea, tiCmd = m.textarea.Update(msg)
				m.viewport, vpCmd = m.viewport.Update(msg)
				return m, tea.Batch(tiCmd, vpCmd, spCmd)
			}
		}
	}

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case spinner.TickMsg:
		// Handled above via spinner.Update
	case tea.KeyMsg:
		// Trigger completion
		m.updateCompletions()

		switch msg.Type {
		case tea.KeyCtrlP:
			if m.agentSession != nil {
				res := m.agentSession.CycleModel("forward")
				if res != nil {
					m.conversation.WriteString(StyleSystem.Render(fmt.Sprintf("\n[System] Cycled model to: %s/%s\n", res.Model.Provider, res.Model.ID)))
					m.modelInfo = fmt.Sprintf("%s/%s", res.Model.Provider, res.Model.ID)
					m.viewport.SetContent(m.conversation.String())
					m.viewport.GotoBottom()
				}
			}
		case tea.KeyCtrlN:
			if m.agentSession != nil {
				m.agentSession.NewSession()
			}
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyCtrlS:
			if !m.isGenerating && m.textarea.Value() != "" {
				userText := m.textarea.Value()
				m.textarea.Reset()

				// Check if it's a slash command
				if strings.HasPrefix(strings.TrimSpace(userText), "/") && m.slashRegistry != nil {
					result, isCommand, err := m.slashRegistry.Execute(userText)
					if isCommand {
						if err != nil {
							m.conversation.WriteString(StyleError.Render(fmt.Sprintf("\n[Error] %v\n", err)))
						} else {
							m.handleSlashResult(result)
						}
						m.viewport.SetContent(m.conversation.String())
						m.viewport.GotoBottom()
						return m, tea.Batch(tiCmd, vpCmd)
					}
				}

				// Regular user message
				m.isGenerating = true
				m.conversation.WriteString(StyleUser.Render(fmt.Sprintf("\n[User]: %s\n", userText)))
				m.viewport.SetContent(m.conversation.String())
				m.viewport.GotoBottom()

				// Use AgentSession.Prompt if available, otherwise use Agent.Prompt
				if m.agentSession != nil {
					return m, tea.Batch(func() tea.Msg {
						err := m.agentSession.Prompt(context.Background(), userText)
						return ExecutionDoneMsg{Err: err}
					})
				}

				return m, tea.Batch(func() tea.Msg {
					userMsg := ai.UserMessage{
						Content:   []ai.Content{ai.TextContent{Text: userText}},
						Timestamp: time.Now().UnixMilli(),
					}
					err := m.agent.Prompt(context.Background(), userMsg)
					return ExecutionDoneMsg{Err: err}
				})
			}
		case tea.KeyEscape:
			// Abort current generation
			if m.isGenerating && m.agentSession != nil {
				m.agentSession.Abort()
			}
		}

	case ExecutionDoneMsg:
		m.isGenerating = false
		if msg.Err != nil {
			m.conversation.WriteString(StyleError.Render(fmt.Sprintf("\n[Error]: %v\n", msg.Err)))
		}
		m.conversation.WriteString("\n")
		m.viewport.SetContent(m.conversation.String())
		m.viewport.GotoBottom()

	case SlashResultMsg:
		m.handleSlashResult(msg.Result)
		m.viewport.SetContent(m.conversation.String())
		m.viewport.GotoBottom()

	case SessionEventMsg:
		event := agentsession.AgentSessionEvent(msg)
		switch event.Type {
		case "compaction_start":
			m.conversation.WriteString(StyleCompaction.Render("\n[Compaction] Starting...\n"))
		case "compaction_end":
			m.conversation.WriteString(StyleCompaction.Render("[Compaction] Complete.\n"))
		case "auto_retry_start":
			if data, ok := event.Data.(map[string]interface{}); ok {
				m.conversation.WriteString(StyleRetry.Render(fmt.Sprintf(
					"\n[Retry] Attempt %v/%v (delay: %vms)\n",
					data["attempt"], data["maxAttempts"], data["delayMs"],
				)))
			}
		case "auto_retry_end":
			if data, ok := event.Data.(map[string]interface{}); ok {
				success := data["success"]
				m.conversation.WriteString(StyleRetry.Render(fmt.Sprintf(
					"[Retry] Done (success=%v, attempt=%v)\n",
					success, data["attempt"],
				)))
			}
		case "model_select":
			if m.agentSession != nil {
				model := m.agentSession.Model()
				m.modelInfo = fmt.Sprintf("%s/%s", model.Provider, model.ID)
			}
		case "new_session":
			m.conversation = strings.Builder{}
			m.viewport.SetContent("New session started.")
		case "reload":
			m.conversation.WriteString(StyleSystem.Render("\n[System] Reloaded.\n"))
		case "queue_update":
			// Could show queue count in status
		}
		m.viewport.SetContent(m.conversation.String())
		m.viewport.GotoBottom()
		evCmd = m.listenForSessionEvents()

	case EventMsg:
		event := agent.AgentEvent(msg)
		// Update background state
		if agentregistry.GlobalScheduler != nil {
			m.cronjobCount = len(agentregistry.GlobalScheduler.ListTasks())
		}

		switch event.Type {
		case agent.EventAgentStart:
			m.conversation.WriteString(StyleSystem.Render("\n[Agent] Starting...\n"))
			m.statusLine = "Generating..."
		case agent.EventMessageStart:
			if event.Message != nil {
				if event.Message.GetRole() == ai.RoleAssistant {
					m.conversation.WriteString(StyleAssistant.Render("\n[Assistant]: "))
				}
			}
		case agent.EventMessageUpdate:
			if event.AssistantMessageEvent != nil {
				switch event.AssistantMessageEvent.Type {
				case ai.EventTextDelta:
					if event.AssistantMessageEvent.Delta != nil {
						text := *event.AssistantMessageEvent.Delta
						if strings.HasPrefix(text, "Thought:") || strings.Contains(m.conversation.String(), "Thought:") {
							m.statusLine = "Reasoning (ReAct)..."
						}
						m.conversation.WriteString(text)
					}
				case ai.EventThinkingStart:
					m.conversation.WriteString(StyleThinking.Render("\n[Thinking] "))
				case ai.EventThinkingDelta:
					if event.AssistantMessageEvent.Delta != nil {
						m.conversation.WriteString(StyleThinking.Render(*event.AssistantMessageEvent.Delta))
					}
				}
			}
		case agent.EventMessageEnd:
			m.conversation.WriteString("\n")
		case agent.EventToolExecutionStart:
			m.activeTool = event.ToolName
			if m.activeTool == "delegate_task" {
				m.subagentActive = true
			}
			argsStr := formatArgs(event.Args)
			m.conversation.WriteString(StyleToolPending.Render(fmt.Sprintf("\n  [Running Tool] %s(%s)...", m.activeTool, argsStr)))
			m.statusLine = fmt.Sprintf("Running: %s", m.activeTool)
		case agent.EventToolExecutionEnd:
			status := "✓"
			style := StyleToolSuccess
			if event.IsError {
				status = "✗"
				style = StyleToolError
			}
			if m.activeTool == "delegate_task" {
				m.subagentActive = false
			}
			contentStr := extractContent(event.Result)
			displayStr := contentStr

			// Special handling for diffs
			if m.activeTool == "patch" || m.activeTool == "edit" || (strings.Contains(displayStr, "---") && strings.Contains(displayStr, "+++")) {
				displayStr = RenderDiff(displayStr)
			} else if len(displayStr) > 1000 {
				displayStr = displayStr[:997] + "..."
			}

			displayStr = strings.ReplaceAll(displayStr, "\n", "\n  ")
			m.conversation.WriteString(style.Render(fmt.Sprintf(" %s\n  %s\n", status, displayStr)))
			m.activeTool = ""
			m.statusLine = ""
		case agent.EventTurnStart:
			m.statusLine = "Generating response..."
		case agent.EventTurnEnd:
			m.statusLine = ""
		case agent.EventAgentEnd:
			m.conversation.WriteString(StyleSystem.Render("\n[Agent] Done.\n"))
			m.statusLine = "Ready"
		}
		m.viewport.SetContent(m.conversation.String())
		m.viewport.GotoBottom()
		evCmd = m.listenForEvents()
	}

	return m, tea.Batch(tiCmd, vpCmd, spCmd, evCmd)
}

func (m *AgentUIModel) updateCompletions() {
	val := m.textarea.Value()
	cursorPos := m.getCursorIndex()
	if cursorPos == 0 {
		m.showCompletions = false
		return
	}

	// Simple trigger: '/' at start
	if strings.HasPrefix(val, "/") && !strings.Contains(val[:cursorPos], " ") {
		m.showCompletions = true
		prefix := val[1:cursorPos]
		m.completionPrefix = "/"
		m.completions = []string{}
		if m.slashRegistry != nil {
			for _, cmd := range m.slashRegistry.ListCommands() {
				if strings.HasPrefix(cmd, prefix) {
					m.completions = append(m.completions, "/"+cmd)
				}
			}
		}
		if len(m.completions) == 0 {
			m.showCompletions = false
		} else if m.completionIndex >= len(m.completions) {
			m.completionIndex = 0
		}
		return
	}

	// Trigger: '@' for files
	lastAt := strings.LastIndex(val[:cursorPos], "@")
	if lastAt != -1 && (lastAt == 0 || val[lastAt-1] == ' ') {
		prefix := val[lastAt+1 : cursorPos]
		m.showCompletions = true
		m.completionPrefix = "@"
		// In a real app we'd debounce this or use a cache
		files := util.ListFilesRecursively(".", 100) // limit to 100
		m.completions = []string{}
		for _, f := range files {
			if strings.Contains(strings.ToLower(f), strings.ToLower(prefix)) {
				m.completions = append(m.completions, "@"+f)
			}
		}
		if len(m.completions) == 0 {
			m.showCompletions = false
		} else if m.completionIndex >= len(m.completions) {
			m.completionIndex = 0
		}
		return
	}

	m.showCompletions = false
}

func (m *AgentUIModel) getCursorIndex() int {
	val := m.textarea.Value()
	lines := strings.Split(val, "\n")
	line := m.textarea.Line()
	col := m.textarea.LineInfo().ColumnOffset

	count := 0
	for i := 0; i < line && i < len(lines); i++ {
		count += len(lines[i]) + 1
	}
	count += col
	return count
}

func (m *AgentUIModel) applyCompletion() {
	if !m.showCompletions || m.completionIndex < 0 || m.completionIndex >= len(m.completions) {
		return
	}

	val := m.textarea.Value()
	cursorPos := m.getCursorIndex()
	completion := m.completions[m.completionIndex]

	var newVal string
	if m.completionPrefix == "/" {
		newVal = completion + " " + val[cursorPos:]
	} else if m.completionPrefix == "@" {
		lastAt := strings.LastIndex(val[:cursorPos], "@")
		newVal = val[:lastAt] + completion + " " + val[cursorPos:]
	}

	m.textarea.SetValue(newVal)
	m.textarea.SetCursor(len(newVal))
	m.showCompletions = false
}

// View renders the current state of the model.
func (m *AgentUIModel) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	status := m.statusLine
	if m.isGenerating {
		status = m.spinner.View() + " " + status
	}
	if m.subagentActive {
		status = "SUBAGENT ACTIVE │ " + status
	}

	header := ""
	if m.modelInfo != "" {
		header = StyleHeader.Render(fmt.Sprintf(" %s │ %s ", m.modelInfo, status))
	} else if status != "" {
		header = StyleHeader.Render(fmt.Sprintf(" %s ", status))
	}

	footer := ""
	if m.agentSession != nil {
		stats := m.agentSession.GetSessionStats()
		footer = StyleSystem.Render(fmt.Sprintf(
			" IN: %d │ OUT: %d │ COST: $%.4f │ CTX: %s │ CRON: %d ",
			stats.TokensInput, stats.TokensOutput, stats.Cost,
			formatContextUsage(stats.ContextUsage), m.cronjobCount,
		))
	}

	content := fmt.Sprintf(
		"%s\n%s\n%s\n\n%s",
		header,
		m.viewport.View(),
		footer,
		m.textarea.View(),
	)

	if m.showCompletions {
		var b strings.Builder
		b.WriteString("\n" + StyleCompletionHeader.Render(" Completions: ") + "\n")
		for i, c := range m.completions {
			if i == m.completionIndex {
				b.WriteString(StyleCompletionSelected.Render("> " + c) + "\n")
			} else {
				b.WriteString(StyleCompletionItem.Render("  " + c) + "\n")
			}
			if i > 10 {
				b.WriteString(StyleSystem.Render("  ...") + "\n")
				break
			}
		}
		content += b.String()
	}

	return content
}

func formatContextUsage(u *agentsession.ContextUsage) string {
	if u == nil || u.Tokens == nil || u.Percent == nil {
		return "N/A"
	}
	return fmt.Sprintf("%d/%d (%.1f%%)", *u.Tokens, u.ContextWindow, *u.Percent)
}

// handleSlashResult processes a slash command result.
func (m *AgentUIModel) handleSlashResult(result slashcommands.SlashCommandResult) {
	if result.Error != "" {
		m.conversation.WriteString(StyleError.Render(fmt.Sprintf("\n[Error] %s\n", result.Error)))
	}
	if result.Info != "" {
		m.conversation.WriteString(StyleSlashInfo.Render(fmt.Sprintf("\n%s\n", result.Info)))
	}
	if result.Message != "" {
		m.isGenerating = true
		m.conversation.WriteString(StyleUser.Render(fmt.Sprintf("\n[User]: %s\n", result.Message)))
		go func() {
			if m.agentSession != nil {
				m.agentSession.Prompt(context.Background(), result.Message)
			} else {
				userMsg := ai.UserMessage{
					Content:   []ai.Content{ai.TextContent{Text: result.Message}},
					Timestamp: time.Now().UnixMilli(),
				}
				m.agent.Prompt(context.Background(), userMsg)
			}
		}()
	}
	if result.Quit {
		m.quitting = true
	}
	if result.SwitchModel != "" {
		m.conversation.WriteString(StyleSystem.Render(fmt.Sprintf("\n[System] Switching model to: %s\n", result.SwitchModel)))
		if m.agentSession != nil {
			m.agentSession.SwitchModel(result.SwitchModel)
		}
	}
	if result.SwitchProvider != "" {
		m.conversation.WriteString(StyleSystem.Render(fmt.Sprintf("\n[System] Switching provider to: %s\n", result.SwitchProvider)))
		if m.agentSession != nil {
			m.agentSession.SwitchProvider(result.SwitchProvider)
		}
	}
	if result.SwitchSession != "" {
		m.conversation.WriteString(StyleSystem.Render(fmt.Sprintf("\n[System] Switching to session: %s\n", result.SwitchSession)))
		if m.agentSession != nil {
			// In a real app we would load the session here
			m.conversation.WriteString(StyleSlashInfo.Render(" (Session data loading not implemented in TUI yet, starting fresh history for ID)\n"))
			m.conversation = strings.Builder{}
			m.viewport.SetContent("Switched to session " + result.SwitchSession)
		}
	}
	if result.Compact {
		m.conversation.WriteString(StyleSystem.Render("\n[System] Compaction requested\n"))
		if m.agentSession != nil {
			go m.agentSession.Compact(context.Background())
		}
	}
	if result.Export != "" || result.Export == "" && len(result.Export) >= 0 {
		// Export command triggered
		if m.agentSession != nil {
			go func() {
				path, err := m.agentSession.ExportToHTML(result.Export)
				if err != nil {
					m.conversation.WriteString(StyleError.Render(fmt.Sprintf("\n[Error] Export failed: %v\n", err)))
				} else {
					m.conversation.WriteString(StyleSlashInfo.Render(fmt.Sprintf("\nSession exported to: %s\n", path)))
				}
			}()
		}
	}
	if result.ThinkingLevel != "" {
		m.conversation.WriteString(StyleSystem.Render(fmt.Sprintf("\n[System] Setting thinking level: %s\n", result.ThinkingLevel)))
		if m.agentSession != nil {
			m.agentSession.SetThinkingLevel(result.ThinkingLevel)
		}
	}
	if result.NewSession {
		m.conversation = strings.Builder{}
		m.viewport.SetContent("New session started.")
	}
	if result.Reload {
		if m.agentSession != nil {
			m.agentSession.Reload()
		}
	}
}

// formatArgs creates a short display string for tool arguments.
func formatArgs(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	parts := make([]string, 0, len(args))
	for k, v := range args {
		s := fmt.Sprintf("%v", v)
		if len(s) > 50 {
			s = s[:47] + "..."
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, s))
	}
	result := strings.Join(parts, ", ")
	if len(result) > 100 {
		result = result[:97] + "..."
	}
	return result
}

// extractContent pulls text content from a tool result.
func extractContent(result any) string {
	if result == nil {
		return ""
	}
	if tr, ok := result.(agent.AgentToolResult); ok {
		var sb strings.Builder
		for _, c := range tr.Content {
			if txt, ok := c.(ai.TextContent); ok {
				sb.WriteString(txt.Text)
			}
		}
		return sb.String()
	}
	return fmt.Sprintf("%v", result)
}

// BubbleteaRenderer adapts the Agent setup to the interactive UI model.
type BubbleteaRenderer struct {
	eventsChan    chan agent.AgentEvent
	sessionEvents chan agentsession.AgentSessionEvent
	program       *tea.Program
}

// NewInteractiveRenderer creates a renderer using the raw Agent.
func NewInteractiveRenderer(ag *agent.Agent) *BubbleteaRenderer {
	eventsChan := make(chan agent.AgentEvent, 100)
	model := InitialModel(ag, eventsChan, nil)
	p := tea.NewProgram(model, tea.WithAltScreen())
	return &BubbleteaRenderer{
		eventsChan: eventsChan,
		program:    p,
	}
}

// NewInteractiveRendererWithSession creates a renderer using AgentSession.
func NewInteractiveRendererWithSession(as *agentsession.AgentSession) *BubbleteaRenderer {
	eventsChan := make(chan agent.AgentEvent, 100)
	sessionEvents := make(chan agentsession.AgentSessionEvent, 100)

	// Subscribe to agent session events
	as.Subscribe(func(event agentsession.AgentSessionEvent) {
		sessionEvents <- event
	})

	model := InitialModelWithSession(as, eventsChan, sessionEvents, nil)
	p := tea.NewProgram(model, tea.WithAltScreen())
	return &BubbleteaRenderer{
		eventsChan:    eventsChan,
		sessionEvents: sessionEvents,
		program:       p,
	}
}

// NewInteractiveRendererWithAgentSession creates a renderer using AgentSession with slash command support.
func NewInteractiveRendererWithAgentSession(as *agentsession.AgentSession, slashReg *slashcommands.Registry) *BubbleteaRenderer {
	eventsChan := make(chan agent.AgentEvent, 100)
	sessionEvents := make(chan agentsession.AgentSessionEvent, 100)

	// Subscribe to agent session events
	as.Subscribe(func(event agentsession.AgentSessionEvent) {
		select {
		case sessionEvents <- event:
		default:
			// Drop event if channel is full
		}
	})

	model := InitialModelWithSession(as, eventsChan, sessionEvents, slashReg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	return &BubbleteaRenderer{
		eventsChan:    eventsChan,
		sessionEvents: sessionEvents,
		program:       p,
	}
}

// NewInteractiveRendererWithSlashCommands creates a renderer with slash command support.
func NewInteractiveRendererWithSlashCommands(ag *agent.Agent, slashReg *slashcommands.Registry) *BubbleteaRenderer {
	eventsChan := make(chan agent.AgentEvent, 100)
	model := InitialModel(ag, eventsChan, slashReg)
	p := tea.NewProgram(model, tea.WithAltScreen())
	return &BubbleteaRenderer{
		eventsChan: eventsChan,
		program:    p,
	}
}

func (r *BubbleteaRenderer) Start() error {
	_, err := r.program.Run()
	return err
}

func (r *BubbleteaRenderer) RenderEvent(event agent.AgentEvent) {
	r.eventsChan <- event
}

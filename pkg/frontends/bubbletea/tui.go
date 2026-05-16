package bubbletea

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/slashcommands"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	styleUser      = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	styleAssistant = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	styleTool      = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleError     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleSystem    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleThinking  = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Italic(true)
	styleSlashInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("120"))
	styleSlashErr  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleHeader    = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
)

// AgentUIModel represents the Bubbletea state for the interactive AI agent interface.
type AgentUIModel struct {
	eventsChan    chan agent.AgentEvent
	conversation  strings.Builder
	viewport      viewport.Model
	textarea      textarea.Model
	agent         *agent.Agent
	slashRegistry *slashcommands.Registry
	activeTool    string
	isGenerating  bool
	err           error
	quitting      bool
	statusLine    string
}

// EventMsg is a wrapper to send AgentEvent instances into the Bubbletea Update loop.
type EventMsg agent.AgentEvent

// ExecutionDoneMsg signals that an agent generation loop has returned
type ExecutionDoneMsg struct {
	Err error
}

// SlashResultMsg wraps a slash command result for the tea loop.
type SlashResultMsg struct {
	Result slashcommands.SlashCommandResult
}

func InitialModel(ag *agent.Agent, eventsChan chan agent.AgentEvent, slashReg *slashcommands.Registry) *AgentUIModel {
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
	}
}

// Init establishes the initial state and begins listening to the channel.
func (m *AgentUIModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.listenForEvents())
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

// Update processes incoming messages and updates the model state.
func (m *AgentUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
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
							m.conversation.WriteString(styleSlashErr.Render(fmt.Sprintf("\n[Error] %v\n", err)))
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
				m.conversation.WriteString(styleUser.Render(fmt.Sprintf("\n[User]: %s\n", userText)))
				m.viewport.SetContent(m.conversation.String())
				m.viewport.GotoBottom()

				// Launch agent prompt in background
				return m, tea.Batch(
					func() tea.Msg {
						userMsg := ai.UserMessage{
							Content:   []ai.Content{ai.TextContent{Text: userText}},
							Timestamp: time.Now().UnixMilli(),
						}
						err := m.agent.Prompt(context.Background(), userMsg)
						return ExecutionDoneMsg{Err: err}
					},
				)
			}
		case tea.KeyEscape:
			// Could cancel current generation in the future
		}
	case ExecutionDoneMsg:
		m.isGenerating = false
		if msg.Err != nil {
			m.conversation.WriteString(styleError.Render(fmt.Sprintf("\n[Error]: %v\n", msg.Err)))
		}
		m.conversation.WriteString("\n")
		m.viewport.SetContent(m.conversation.String())
		m.viewport.GotoBottom()
	case SlashResultMsg:
		m.handleSlashResult(msg.Result)
		m.viewport.SetContent(m.conversation.String())
		m.viewport.GotoBottom()
	case EventMsg:
		event := agent.AgentEvent(msg)
		switch event.Type {
		case agent.EventAgentStart:
			m.conversation.WriteString(styleSystem.Render("\n[Agent] Starting...\n"))
			m.statusLine = "Generating..."
		case agent.EventMessageStart:
			if event.Message != nil {
				if event.Message.GetRole() == ai.RoleAssistant {
					m.conversation.WriteString(styleAssistant.Render("\n[Assistant]: "))
				}
			}
		case agent.EventMessageUpdate:
			if event.AssistantMessageEvent != nil {
				switch event.AssistantMessageEvent.Type {
				case ai.EventTextDelta:
					if event.AssistantMessageEvent.Delta != nil {
						m.conversation.WriteString(*event.AssistantMessageEvent.Delta)
					}
				case ai.EventThinkingStart:
					m.conversation.WriteString(styleThinking.Render("\n[Thinking] "))
				case ai.EventThinkingDelta:
					if event.AssistantMessageEvent.Delta != nil {
						m.conversation.WriteString(styleThinking.Render(*event.AssistantMessageEvent.Delta))
					}
				}
			}
		case agent.EventMessageEnd:
			m.conversation.WriteString("\n")
		case agent.EventToolExecutionStart:
			m.activeTool = event.ToolName
			argsStr := formatArgs(event.Args)
			m.conversation.WriteString(styleTool.Render(fmt.Sprintf("\n  [Running Tool] %s(%s)...", m.activeTool, argsStr)))
			m.statusLine = fmt.Sprintf("Running: %s", m.activeTool)
		case agent.EventToolExecutionUpdate:
			// Stream partial tool results if available
		case agent.EventToolExecutionEnd:
			status := "✓"
			if event.IsError {
				status = "✗"
			}
			contentStr := extractContent(event.Result)
			displayStr := contentStr
			if len(displayStr) > 200 {
				displayStr = displayStr[:197] + "..."
			}
			displayStr = strings.ReplaceAll(displayStr, "\n", "\n  ")

			if event.IsError {
				m.conversation.WriteString(styleError.Render(fmt.Sprintf(" %s\n  %s\n", status, displayStr)))
			} else {
				m.conversation.WriteString(styleTool.Render(fmt.Sprintf(" %s\n  %s\n", status, displayStr)))
			}
			m.activeTool = ""
			m.statusLine = ""
		case agent.EventTurnStart:
			m.statusLine = "Generating response..."
		case agent.EventTurnEnd:
			m.statusLine = ""
		case agent.EventAgentEnd:
			m.conversation.WriteString(styleSystem.Render("\n[Agent] Done.\n"))
			m.statusLine = "Ready"
		}
		m.viewport.SetContent(m.conversation.String())
		m.viewport.GotoBottom()
		return m, m.listenForEvents()
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

// View renders the current state of the model.
func (m *AgentUIModel) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	header := ""
	if m.statusLine != "" {
		header = styleHeader.Render(fmt.Sprintf("  %s", m.statusLine))
	}

	return fmt.Sprintf(
		"%s\n%s\n\n%s",
		header,
		m.viewport.View(),
		m.textarea.View(),
	)
}

// handleSlashResult processes a slash command result.
func (m *AgentUIModel) handleSlashResult(result slashcommands.SlashCommandResult) {
	if result.Error != "" {
		m.conversation.WriteString(styleSlashErr.Render(fmt.Sprintf("\n[Error] %s\n", result.Error)))
	}
	if result.Info != "" {
		m.conversation.WriteString(styleSlashInfo.Render(fmt.Sprintf("\n%s\n", result.Info)))
	}
	if result.Message != "" {
		// The slash command wants to send a message as if the user typed it
		m.isGenerating = true
		m.conversation.WriteString(styleUser.Render(fmt.Sprintf("\n[User]: %s\n", result.Message)))
		go func() {
			userMsg := ai.UserMessage{
				Content:   []ai.Content{ai.TextContent{Text: result.Message}},
				Timestamp: time.Now().UnixMilli(),
			}
			m.agent.Prompt(context.Background(), userMsg)
		}()
	}
	if result.Quit {
		m.quitting = true
	}
	if result.SwitchModel != "" {
		m.conversation.WriteString(styleSystem.Render(fmt.Sprintf("\n[System] Switching model to: %s\n", result.SwitchModel)))
		// Model switching would be handled by the application layer
	}
	if result.SwitchProvider != "" {
		m.conversation.WriteString(styleSystem.Render(fmt.Sprintf("\n[System] Switching provider to: %s\n", result.SwitchProvider)))
	}
	if result.Compact {
		m.conversation.WriteString(styleSystem.Render("\n[System] Compaction requested\n"))
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
	eventsChan chan agent.AgentEvent
	program    *tea.Program
}

func NewInteractiveRenderer(ag *agent.Agent) *BubbleteaRenderer {
	eventsChan := make(chan agent.AgentEvent, 100)
	model := InitialModel(ag, eventsChan, nil)
	p := tea.NewProgram(model, tea.WithAltScreen())
	return &BubbleteaRenderer{
		eventsChan: eventsChan,
		program:    p,
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

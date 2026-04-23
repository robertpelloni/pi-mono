package bubbletea

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
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
)

// AgentUIModel represents the Bubbletea state for the interactive AI agent interface.
type AgentUIModel struct {
	eventsChan    chan agent.AgentEvent
	conversation  strings.Builder
	viewport      viewport.Model
	textarea      textarea.Model
	agent         *agent.Agent
	activeTool    string
	isGenerating  bool
	err           error
	quitting      bool
}

// EventMsg is a wrapper to send AgentEvent instances into the Bubbletea Update loop.
type EventMsg agent.AgentEvent

// ExecutionDoneMsg signals that an agent generation loop has returned
type ExecutionDoneMsg struct {
	Err error
}

func InitialModel(ag *agent.Agent, eventsChan chan agent.AgentEvent) *AgentUIModel {
	ta := textarea.New()
	ta.Placeholder = "Type a message... (Ctrl+S to send)"
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 20000
	ta.SetWidth(80)
	ta.SetHeight(3)

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to the Pi Go Agent CLI! Type your prompt below.")

	return &AgentUIModel{
		eventsChan: eventsChan,
		viewport:   vp,
		textarea:   ta,
		agent:      ag,
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

		case tea.KeyCtrlS: // Using Ctrl+S to submit so Enter can be used for newlines
			if !m.isGenerating && m.textarea.Value() != "" {
				userText := m.textarea.Value()
				m.textarea.Reset()
				m.isGenerating = true

				m.conversation.WriteString(styleUser.Render(fmt.Sprintf("\n[User]: %s\n", userText)))
				m.viewport.SetContent(m.conversation.String())
				m.viewport.GotoBottom()

				// Launch agent prompt in background
				return m, tea.Batch(
					func() tea.Msg {
						userMsg := ai.UserMessage{
							Content: []ai.Content{
								ai.TextContent{Text: userText},
							},
							Timestamp: time.Now().UnixMilli(),
						}
						err := m.agent.Prompt(context.Background(), userMsg)
						return ExecutionDoneMsg{Err: err}
					},
				)
			}
		}

	case ExecutionDoneMsg:
		m.isGenerating = false
		if msg.Err != nil {
			m.conversation.WriteString(styleError.Render(fmt.Sprintf("\n[Error]: %v\n", msg.Err)))
		}
		m.conversation.WriteString("\n")
		m.viewport.SetContent(m.conversation.String())
		m.viewport.GotoBottom()

	case EventMsg:
		event := agent.AgentEvent(msg)
		switch event.Type {
		case agent.EventAgentStart:
			m.conversation.WriteString(styleSystem.Render("\n[Agent] Starting...\n"))

		case agent.EventMessageStart:
			if event.Message != nil {
				if event.Message.GetRole() == ai.RoleAssistant {
					m.conversation.WriteString(styleAssistant.Render("\n[Assistant]: "))
				}
			}

		case agent.EventMessageUpdate:
			if event.AssistantMessageEvent != nil && event.AssistantMessageEvent.Type == ai.EventTextDelta {
				if event.AssistantMessageEvent.Delta != nil {
					m.conversation.WriteString(*event.AssistantMessageEvent.Delta)
				}
			}

		case agent.EventMessageEnd:
			m.conversation.WriteString("\n")

		case agent.EventToolExecutionStart:
			m.activeTool = event.ToolName
			argsStr := fmt.Sprintf("%v", event.Args)
			m.conversation.WriteString(styleTool.Render(fmt.Sprintf("\n  [Running Tool] %s(%s)...", m.activeTool, argsStr)))

		case agent.EventToolExecutionEnd:
			status := "SUCCESS"
			if event.IsError {
				status = "ERROR"
			}

			contentStr := ""
			if res, ok := event.Result.(agent.AgentToolResult); ok {
				for _, c := range res.Content {
					if txt, ok := c.(ai.TextContent); ok {
						contentStr += txt.Text
					}
				}
			}

			displayStr := contentStr
			if len(displayStr) > 100 {
				displayStr = displayStr[:97] + "..."
			}

			if event.IsError {
				m.conversation.WriteString(styleError.Render(fmt.Sprintf(" -> %s\n    %s\n", status, strings.ReplaceAll(displayStr, "\n", "\n    "))))
			} else {
				m.conversation.WriteString(styleTool.Render(fmt.Sprintf(" -> %s\n    %s\n", status, strings.ReplaceAll(displayStr, "\n", "\n    "))))
			}
			m.activeTool = ""

		case agent.EventAgentEnd:
			m.conversation.WriteString(styleSystem.Render("\n[Agent] Done.\n"))
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

	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	)
}

// BubbleteaRenderer adapts the Agent setup to the interactive UI model.
type BubbleteaRenderer struct {
	eventsChan chan agent.AgentEvent
	program    *tea.Program
}

func NewInteractiveRenderer(ag *agent.Agent) *BubbleteaRenderer {
	eventsChan := make(chan agent.AgentEvent, 100)

	model := InitialModel(ag, eventsChan)
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

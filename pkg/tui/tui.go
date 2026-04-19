package tui

import (
	"fmt"
	"strings"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
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

// AgentUIModel represents the Bubbletea state for the AI agent interface.
type AgentUIModel struct {
	eventsChan    chan agent.AgentEvent
	conversation  strings.Builder
	activeTool    string
	toolArguments string
	err           error
	quitting      bool
}

// EventMsg is a wrapper to send AgentEvent instances into the Bubbletea Update loop.
type EventMsg agent.AgentEvent

// Init establishes the initial state and begins listening to the channel.
func (m *AgentUIModel) Init() tea.Cmd {
	return m.listenForEvents()
}

// listenForEvents reads from the channels and dispatches messages to the tea loop.
func (m *AgentUIModel) listenForEvents() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.eventsChan
		if !ok {
			// Channel closed
			return nil
		}
		return EventMsg(event)
	}
}

// Update processes incoming messages and updates the model state.
func (m *AgentUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}

	case EventMsg:
		event := agent.AgentEvent(msg)
		switch event.Type {
		case agent.EventAgentStart:
			m.conversation.WriteString(styleSystem.Render("\n[Agent] Starting new run...\n"))

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
			m.conversation.WriteString(styleTool.Render(fmt.Sprintf("\n  [Running Tool] %s(%s)...\n", m.activeTool, argsStr)))

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

			// Truncate long tool outputs for terminal readability
			displayStr := contentStr
			if len(displayStr) > 200 {
				displayStr = displayStr[:197] + "..."
			}

			if event.IsError {
				m.conversation.WriteString(styleError.Render(fmt.Sprintf("  [Tool Finished] %s -> %s\n    %s\n", event.ToolName, status, strings.ReplaceAll(displayStr, "\n", "\n    "))))
			} else {
				m.conversation.WriteString(styleTool.Render(fmt.Sprintf("  [Tool Finished] %s -> %s\n    %s\n", event.ToolName, status, strings.ReplaceAll(displayStr, "\n", "\n    "))))
			}
			m.activeTool = ""

		case agent.EventTurnEnd:
			// Turn has completed

		case agent.EventAgentEnd:
			m.conversation.WriteString(styleSystem.Render("\n[Agent] Run completed.\n"))
			m.quitting = true
			return m, tea.Quit
		}

		// Continue listening for the next event
		return m, m.listenForEvents()
	}

	return m, nil
}

// View renders the current state of the model.
func (m *AgentUIModel) View() string {
	if m.err != nil {
		return styleError.Render(fmt.Sprintf("Error: %v\n", m.err))
	}

	out := m.conversation.String()

	if !m.quitting {
		out += "\n\n" + styleSystem.Render("(Press 'q' or ctrl+c to quit)")
	}

	return out
}

// BubbleteaRenderer adapts the Bubbletea Model to the agent.Renderer interface.
type BubbleteaRenderer struct {
	eventsChan chan agent.AgentEvent
	program    *tea.Program
}

func NewBubbleteaRenderer() *BubbleteaRenderer {
	eventsChan := make(chan agent.AgentEvent, 100)

	model := &AgentUIModel{
		eventsChan: eventsChan,
	}

	p := tea.NewProgram(model)

	// We run the program asynchronously since the Agent execution blocks
	go func() {
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running TUI: %v", err)
		}
	}()

	return &BubbleteaRenderer{
		eventsChan: eventsChan,
		program:    p,
	}
}

func (r *BubbleteaRenderer) RenderEvent(event agent.AgentEvent) {
	r.eventsChan <- event
}

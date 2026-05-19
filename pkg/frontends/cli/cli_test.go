package cli

import (
	"fmt"
	"testing"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestNewCLIRenderer(t *testing.T) {
	renderer := NewCLIRenderer(nil)
	if renderer == nil {
		t.Fatal("Expected non-nil renderer")
	}
}

func TestCLIRenderer_RenderEvent_AgentStart(t *testing.T) {
	renderer := &CLIRenderer{agent: nil}
	// We can't easily redirect fmt.Println, but we can test the event handling
	// by verifying it doesn't panic
	renderer.RenderEvent(agent.AgentEvent{Type: agent.EventAgentStart})
}

func TestCLIRenderer_RenderEvent_MessageStart(t *testing.T) {
	renderer := &CLIRenderer{}
	msg := ai.AssistantMessage{
		Content: []ai.Content{ai.TextContent{Text: "hello"}},
	}
	renderer.RenderEvent(agent.AgentEvent{
		Type:    agent.EventMessageStart,
		Message: msg,
	})
}

func TestCLIRenderer_RenderEvent_TextDelta(t *testing.T) {
	renderer := &CLIRenderer{}
	delta := "Hello world"
	renderer.RenderEvent(agent.AgentEvent{
		Type: agent.EventMessageUpdate,
		AssistantMessageEvent: &ai.AssistantMessageEvent{
			Type:  ai.EventTextDelta,
			Delta: &delta,
		},
	})
}

func TestCLIRenderer_RenderEvent_ToolStart(t *testing.T) {
	renderer := &CLIRenderer{}
	renderer.RenderEvent(agent.AgentEvent{
		Type:     agent.EventToolExecutionStart,
		ToolName: "read",
		Args:     map[string]any{"path": "/tmp/test.txt"},
	})
}

func TestCLIRenderer_RenderEvent_ToolEnd(t *testing.T) {
	renderer := &CLIRenderer{}
	result := agent.AgentToolResult{
		Content: []ai.Content{ai.TextContent{Text: "file contents here"}},
	}
	renderer.RenderEvent(agent.AgentEvent{
		Type:     agent.EventToolExecutionEnd,
		ToolName: "read",
		Result:   result,
	})
}

func TestCLIRenderer_RenderEvent_ToolEndError(t *testing.T) {
	renderer := &CLIRenderer{}
	result := agent.AgentToolResult{
		Content: []ai.Content{ai.TextContent{Text: "file not found"}},
	}
	renderer.RenderEvent(agent.AgentEvent{
		Type:     agent.EventToolExecutionEnd,
		ToolName: "read",
		IsError:  true,
		Result:   result,
	})
}

func TestCLIRenderer_RenderEvent_MessageEnd(t *testing.T) {
	renderer := &CLIRenderer{}
	renderer.RenderEvent(agent.AgentEvent{Type: agent.EventMessageEnd})
}

func TestCLIRenderer_RenderEvent_AgentEnd(t *testing.T) {
	renderer := &CLIRenderer{}
	renderer.RenderEvent(agent.AgentEvent{Type: agent.EventAgentEnd})
}

func TestCLIRenderer_RenderEvent_UnknownEvent(t *testing.T) {
	renderer := &CLIRenderer{}
	// Should not panic on unknown events
	renderer.RenderEvent(agent.AgentEvent{Type: "unknown_event"})
}

func TestCLIRenderer_RenderEvent_NilMessage(t *testing.T) {
	renderer := &CLIRenderer{}
	// Message start with nil message should not panic
	renderer.RenderEvent(agent.AgentEvent{
		Type:    agent.EventMessageStart,
		Message: nil,
	})
}

func TestCLIRenderer_RenderEvent_NilDelta(t *testing.T) {
	renderer := &CLIRenderer{}
	renderer.RenderEvent(agent.AgentEvent{
		Type: agent.EventMessageUpdate,
		AssistantMessageEvent: &ai.AssistantMessageEvent{
			Type:  ai.EventTextDelta,
			Delta: nil,
		},
	})
}

func TestFmtSprintf(t *testing.T) {
	// Quick test that args formatting works
	args := map[string]any{"path": "/tmp/test.txt", "limit": 10}
	result := fmt.Sprintf("%v", args)
	_ = result
}

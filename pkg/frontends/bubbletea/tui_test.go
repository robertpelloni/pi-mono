package bubbletea

import (
    "strings"
    "testing"

    "github.com/badlogic/pi-mono/pkg/agent"
    "github.com/badlogic/pi-mono/pkg/ai"
)

func TestAssistantMarkdown_Render(t *testing.T) {
    // Initialize a model without an actual agent (nil) and no slash registry.
    model := InitialModel(nil, nil, nil)

    // Simulate assistant message start.
    startEvt := EventMsg(agent.AgentEvent{Type: agent.EventMessageStart, Message: ai.AssistantMessage{}})
    model.Update(startEvt)

    // Simulate a text delta containing markdown.
    mdText := "**bold**"
    updateEvt := EventMsg(agent.AgentEvent{Type: agent.EventMessageUpdate, AssistantMessageEvent: &ai.AssistantMessageEvent{Type: ai.EventTextDelta, Delta: &mdText}})
    model.Update(updateEvt)

    // End the assistant message.
    endEvt := EventMsg(agent.AgentEvent{Type: agent.EventMessageEnd})
    model.Update(endEvt)

    // Render the view.
    out := model.View()
    // Glamour renders bold with ANSI escape code \x1b[1m.
    if !strings.Contains(strings.ToLower(out), "bold") {
        t.Fatalf("expected rendered markdown containing 'bold' in view output, got: %s", out)
    }
}

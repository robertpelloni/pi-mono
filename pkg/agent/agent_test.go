package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/badlogic/pi-mono/pkg/ai"
)

func mockStreamOpenAI(ctx context.Context, model ai.ModelInfo, aiCtx ai.Context, options any) ai.AssistantMessageEventStream {
	stream := make(chan ai.AssistantMessageEvent, 10)
	go func() {
		defer close(stream)

		// Send some simulated chunks
		select {
		case <-ctx.Done():
			return
		case stream <- ai.AssistantMessageEvent{Type: ai.EventStart}:
		}

		select {
		case <-ctx.Done():
			return
		case stream <- ai.AssistantMessageEvent{
			Type:  ai.EventTextDelta,
			Delta: func() *string { s := "Hello "; return &s }(),
		}:
		}

		select {
		case <-ctx.Done():
			return
		case stream <- ai.AssistantMessageEvent{
			Type:  ai.EventTextDelta,
			Delta: func() *string { s := "World!"; return &s }(),
		}:
		}

		select {
		case <-ctx.Done():
			return
		case stream <- ai.AssistantMessageEvent{
			Type:   ai.EventDone,
			Reason: func() *ai.StopReason { r := ai.StopReasonStop; return &r }(),
		}:
		}
	}()
	return stream
}

func mockTool() AgentTool {
	return AgentTool{
		Name: "mock_tool",
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate AgentToolUpdateCallback) (AgentToolResult, error) {
			return AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: "tool executed"},
				},
			}, nil
		},
	}
}

func TestAgentPrompt(t *testing.T) {
	modelInfo := ai.ModelInfo{
		ID:       "mock-model",
		Provider: ai.ProviderOpenAI,
	}

	agent := NewAgent(modelInfo, []AgentTool{mockTool()}, mockStreamOpenAI, AgentLoopConfig{})

	eventsCh := make(chan AgentEvent, 100)
	unsub := agent.Subscribe(func(e AgentEvent) {
		eventsCh <- e
	})
	defer unsub()

	userMsg := ai.UserMessage{
		Content: []ai.Content{
			ai.TextContent{Text: "Hi"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := agent.Prompt(ctx, userMsg)
	if err != nil {
		t.Fatalf("agent.Prompt failed: %v", err)
	}

	// Verify events
	close(eventsCh)
	var events []AgentEvent
	for e := range eventsCh {
		events = append(events, e)
	}

	hasStart := false
	hasEnd := false
	hasTurnEnd := false
	for _, e := range events {
		if e.Type == EventAgentStart {
			hasStart = true
		}
		if e.Type == EventAgentEnd {
			hasEnd = true
		}
		if e.Type == EventTurnEnd {
			hasTurnEnd = true
			if am, ok := e.Message.(ai.AssistantMessage); ok {
				if !strings.Contains(am.Content[0].(ai.TextContent).Text, "Hello World!") {
					t.Errorf("Expected final message content 'Hello World!', got '%v'", am.Content[0])
				}
			} else {
				t.Errorf("Expected AssistantMessage at TurnEnd")
			}
		}
	}

	if !hasStart {
		t.Errorf("Expected EventAgentStart")
	}
	if !hasEnd {
		t.Errorf("Expected EventAgentEnd")
	}
	if !hasTurnEnd {
		t.Errorf("Expected EventTurnEnd")
	}
}

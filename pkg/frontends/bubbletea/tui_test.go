package bubbletea

import (
	"testing"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/slashcommands"
)

func TestFormatArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		expected string
	}{
		{"empty", map[string]any{}, ""},
		{"single", map[string]any{"path": "/tmp/test.txt"}, "path=/tmp/test.txt"},
		{"multiple", map[string]any{"a": "1", "b": "2"}, ""},
		{"long value", map[string]any{"data": string(make([]byte, 100))}, "data="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatArgs(tt.args)
			if tt.name == "empty" && result != "" {
				t.Errorf("Expected empty for empty args, got %q", result)
			}
			if tt.name == "single" && result == "" {
				t.Error("Expected non-empty result for single arg")
			}
		})
	}
}

func TestFormatArgs_NilArgs(t *testing.T) {
	result := formatArgs(nil)
	if result != "" {
		t.Errorf("Expected empty for nil args, got %q", result)
	}
}

func TestExtractContent_Nil(t *testing.T) {
	result := extractContent(nil)
	if result != "" {
		t.Errorf("Expected empty for nil, got %q", result)
	}
}

func TestExtractContent_ToolResult(t *testing.T) {
	result := agent.AgentToolResult{
		Content: []ai.Content{
			ai.TextContent{Text: "hello world"},
		},
	}
	content := extractContent(result)
	if content != "hello world" {
		t.Errorf("Expected 'hello world', got %q", content)
	}
}

func TestExtractContent_String(t *testing.T) {
	content := extractContent("some string")
	if content == "" {
		t.Error("Expected non-empty content for string input")
	}
}

func TestInitialModel(t *testing.T) {
	eventsChan := make(chan agent.AgentEvent, 10)
	slashReg := slashcommands.NewRegistry()
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	
	model := InitialModel(ag, eventsChan, slashReg)
	if model == nil {
		t.Fatal("Expected non-nil model")
	}
	if model.agent == nil {
		t.Error("Expected agent to be set")
	}
	if model.eventsChan == nil {
		t.Error("Expected eventsChan to be set")
	}
}

func TestInitialModel_NilSlashReg(t *testing.T) {
	eventsChan := make(chan agent.AgentEvent, 10)
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	
	model := InitialModel(ag, eventsChan, nil)
	if model == nil {
		t.Fatal("Expected non-nil model")
	}
}

func TestAgentUIModel_View(t *testing.T) {
	eventsChan := make(chan agent.AgentEvent, 10)
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	
	model := InitialModel(ag, eventsChan, nil)
	view := model.View()
	if view == "" {
		t.Error("Expected non-empty view")
	}
}

func TestAgentUIModel_View_Quitting(t *testing.T) {
	eventsChan := make(chan agent.AgentEvent, 10)
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	
	model := InitialModel(ag, eventsChan, nil)
	model.quitting = true
	view := model.View()
	if !containsStr(view, "Goodbye") {
		t.Errorf("Expected 'Goodbye' in quitting view, got %q", view)
	}
}

func TestAgentUIModel_Styles(t *testing.T) {
	// Test that styles are defined
	_ = styleUser.String()
	_ = styleAssistant.String()
	_ = styleTool.String()
	_ = styleError.String()
	_ = styleSystem.String()
	_ = styleThinking.String()
	_ = styleSlashInfo.String()
	_ = styleSlashErr.String()
	_ = styleHeader.String()
	_ = styleCompaction.String()
	_ = styleRetry.String()
}

func TestNewInteractiveRenderer(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	renderer := NewInteractiveRenderer(ag)
	if renderer == nil {
		t.Fatal("Expected non-nil renderer")
	}
	if renderer.eventsChan == nil {
		t.Error("Expected eventsChan to be set")
	}
}

func TestNewInteractiveRendererWithSlashCommands(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	slashReg := slashcommands.NewRegistry()
	renderer := NewInteractiveRendererWithSlashCommands(ag, slashReg)
	if renderer == nil {
		t.Fatal("Expected non-nil renderer")
	}
}

func TestBubbleteaRenderer_RenderEvent(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	renderer := NewInteractiveRenderer(ag)
	
	// Send an event - should not block
	renderer.RenderEvent(agent.AgentEvent{Type: agent.EventAgentStart})
}

func TestEventMsg_Type(t *testing.T) {
	event := EventMsg{Type: agent.EventAgentStart}
	if event.Type != agent.EventAgentStart {
		t.Error("Event type mismatch")
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || 
		(len(s) > 0 && len(sub) > 0 && findSubstr(s, sub)))
}

func findSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

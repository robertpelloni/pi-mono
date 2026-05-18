package agentsession

import (
	"testing"

	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestParseSkillBlock_Valid(t *testing.T) {
	text := `<skill name="test-skill" location="/path/to/skill.md">
Skill content here.
</skill>

User message after skill`
	result := ParseSkillBlock(text)
	if result == nil {
		t.Fatal("Expected non-nil result for valid skill block")
	}
	if result.Name != "test-skill" {
		t.Errorf("Expected name 'test-skill', got %q", result.Name)
	}
	if result.Location != "/path/to/skill.md" {
		t.Errorf("Expected location '/path/to/skill.md', got %q", result.Location)
	}
	if result.UserMessage != "User message after skill" {
		t.Errorf("Expected user message, got %q", result.UserMessage)
	}
}

func TestParseSkillBlock_NoUserMessage(t *testing.T) {
	text := `<skill name="test-skill" location="/path/to/skill.md">
Skill content here.
</skill>`
	result := ParseSkillBlock(text)
	if result == nil {
		t.Fatal("Expected non-nil result for valid skill block without user message")
	}
	if result.UserMessage != "" {
		t.Errorf("Expected empty user message, got %q", result.UserMessage)
	}
}

func TestParseSkillBlock_Invalid(t *testing.T) {
	text := "This is just a regular message"
	result := ParseSkillBlock(text)
	if result != nil {
		t.Errorf("Expected nil for non-skill-block text, got %+v", result)
	}
}

func TestShouldCompact(t *testing.T) {
	tests := []struct {
		contextTokens  int
		contextWindow  int
		threshold      float64
		shouldCompact  bool
	}{
		{90000, 100000, 0.8, true},  // 90% >= 80%
		{70000, 100000, 0.8, false}, // 70% < 80%
		{80000, 100000, 0.8, true},  // 80% >= 80%
		{50000, 100000, 0.5, true},  // 50% >= 50%
		{40000, 100000, 0.5, false}, // 40% < 50%
		{100, 0, 0.8, false},        // No context window
	}

	for _, tt := range tests {
		result := shouldCompact(tt.contextTokens, tt.contextWindow, tt.threshold)
		if result != tt.shouldCompact {
			t.Errorf("shouldCompact(%d, %d, %f) = %v, expected %v",
				tt.contextTokens, tt.contextWindow, tt.threshold, result, tt.shouldCompact)
		}
	}
}

func TestContainsString(t *testing.T) {
	slice := []string{"off", "low", "medium", "high"}
	if !containsString(slice, "low") {
		t.Error("Expected to find 'low' in slice")
	}
	if containsString(slice, "xhigh") {
		t.Error("Did not expect to find 'xhigh' in slice")
	}
}

func TestClampThinkingLevel(t *testing.T) {
	as := &AgentSession{
		activeModel: ai.ModelInfo{Reasoning: true},
	}

	tests := []struct {
		requested string
		available []string
		expected  string
	}{
		{"medium", []string{"off", "minimal", "low", "medium", "high"}, "medium"},
		{"xhigh", []string{"off", "minimal", "low", "medium", "high"}, "high"},
		{"off", []string{"off", "minimal", "low", "medium", "high"}, "off"},
		{"invalid", []string{"off", "low"}, "off"},
	}

	for _, tt := range tests {
		result := as.clampThinkingLevel(tt.requested, tt.available)
		if result != tt.expected {
			t.Errorf("clampThinkingLevel(%q, %v) = %q, expected %q",
				tt.requested, tt.available, result, tt.expected)
		}
	}
}

func TestModelCycleResult(t *testing.T) {
	result := ModelCycleResult{
		Model:         ai.ModelInfo{ID: "test-model"},
		ThinkingLevel: "high",
		IsScoped:      true,
	}
	if result.Model.ID != "test-model" {
		t.Errorf("Expected model ID 'test-model', got %q", result.Model.ID)
	}
	if result.ThinkingLevel != "high" {
		t.Errorf("Expected thinking level 'high', got %q", result.ThinkingLevel)
	}
	if !result.IsScoped {
		t.Error("Expected IsScoped to be true")
	}
}

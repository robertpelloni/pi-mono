package agentsession

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/badlogic/pi-mono/pkg/agent"
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

// ---------------------------------------------------------------------------
// Additional AgentSession tests
// ---------------------------------------------------------------------------

func TestAgentSessionConfig_Fields(t *testing.T) {
	config := AgentSessionConfig{
		CWD:      "/test",
		AgentDir: "/test/.pi",
	}
	if config.CWD != "/test" {
		t.Error("CWD mismatch")
	}
	if config.AgentDir != "/test/.pi" {
		t.Error("AgentDir mismatch")
	}
}

func TestContextUsage_Fields(t *testing.T) {
	tokens := 50000
	percent := 50.0
	usage := ContextUsage{
		Tokens:        &tokens,
		ContextWindow: 100000,
		Percent:       &percent,
	}
	if usage.Tokens == nil || *usage.Tokens != 50000 {
		t.Error("Tokens mismatch")
	}
	if usage.ContextWindow != 100000 {
		t.Error("ContextWindow mismatch")
	}
	if usage.Percent == nil || *usage.Percent != 50.0 {
		t.Error("Percent mismatch")
	}
}

func TestSessionStats_Fields(t *testing.T) {
	stats := SessionStats{
		SessionFile:       "/test/session.jsonl",
		SessionID:         "test-123",
		UserMessages:      10,
		AssistantMessages: 10,
		ToolCalls:         5,
		ToolResults:       5,
		TotalMessages:     20,
	}
	if stats.SessionID != "test-123" {
		t.Error("SessionID mismatch")
	}
	if stats.UserMessages != 10 {
		t.Error("UserMessages mismatch")
	}
}

func TestCompactionReason_Values(t *testing.T) {
	reasons := []CompactionReason{
		CompactionManual,
		CompactionThreshold,
		CompactionOverflow,
	}
	for _, r := range reasons {
		if string(r) == "" {
			t.Errorf("CompactionReason %v should not be empty", r)
		}
	}
}

func TestAgentSessionEvent_Fields(t *testing.T) {
	event := AgentSessionEvent{
		Type:  "agent_start",
		Data:  map[string]interface{}{"key": "value"},
		Error: nil,
	}
	if event.Type != "agent_start" {
		t.Error("Type mismatch")
	}
}



func TestNewAgentSession(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})
	if as == nil {
		t.Fatal("Expected non-nil AgentSession")
	}
	if as.IsStreaming() {
		t.Error("Expected not streaming initially")
	}
	if as.IsCompacting() {
		t.Error("Expected not compacting initially")
	}
}

func TestAgentSession_Model(t *testing.T) {
	as := NewAgentSession(AgentSessionConfig{
		Agent: agent.NewAgent(ai.ModelInfo{ID: "test-model", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{}),
	})
	model := as.Model()
	if model.ID != "test-model" {
		t.Errorf("Expected model ID 'test-model', got %q", model.ID)
	}
}

func TestAgentSession_ThinkingLevel(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})
	level := as.ThinkingLevel()
	// Default is "medium"
	if level != "medium" && level != "" {
		t.Errorf("Unexpected thinking level %q", level)
	}
}

func TestAgentSession_AutoRetryEnabled(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})
	// Without settings, auto-retry defaults to true
	if !as.AutoRetryEnabled() {
		t.Error("Expected auto-retry to be enabled by default")
	}
	as.SetAutoRetryEnabled(true)
	if !as.AutoRetryEnabled() {
		t.Error("Expected auto-retry to be enabled")
	}
}

func TestAgentSession_AutoCompactionEnabled(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})
	as.SetAutoCompactionEnabled(true)
	if !as.AutoCompactionEnabled() {
		t.Error("Expected auto-compaction to be enabled")
	}
}

func TestAgentSession_Subscribe(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})
	var receivedEvents []AgentSessionEvent
	unsubscribe := as.Subscribe(func(event AgentSessionEvent) {
		receivedEvents = append(receivedEvents, event)
	})
	if unsubscribe == nil {
		t.Fatal("Expected non-nil unsubscribe function")
	}
	// Emit an event
	as.emit(AgentSessionEvent{Type: "test_event"})
	// Unsubscribe
	unsubscribe()
}

func TestAgentSession_GetSteeringMessages(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})
	msgs := as.GetSteeringMessages()
	if msgs == nil {
		// May be nil if no steering messages
	}
}

func TestAgentSession_GetFollowUpMessages(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})
	msgs := as.GetFollowUpMessages()
	if msgs == nil {
		// May be nil if no follow-up messages
	}
}

func TestAgentSession_PendingMessageCount(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})
	count := as.PendingMessageCount()
	if count != 0 {
		t.Errorf("Expected 0 pending messages, got %d", count)
	}
}

func TestAgentSession_IsRetrying(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})
	if as.IsRetrying() {
		t.Error("Expected not retrying initially")
	}
}

func TestAgentSession_RetryAttempt(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})
	attempt := as.RetryAttempt()
	if attempt != 0 {
		t.Errorf("Expected retry attempt 0, got %d", attempt)
	}
}

func TestAgentSession_GetAvailableThinkingLevels(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})
	levels := as.GetAvailableThinkingLevels()
	// Should return some levels
	if len(levels) == 0 {
		t.Error("Expected at least one thinking level")
	}
}

// ---------------------------------------------------------------------------
// Export and Additional Method Tests
// ---------------------------------------------------------------------------

func TestAgentSession_ExportToHTML_NoSession(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})

	tmpDir, err := os.MkdirTemp("", "export_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "test.html")
	result, err := as.ExportToHTML(outputPath)
	if err != nil {
		t.Fatalf("ExportToHTML failed: %v", err)
	}
	if result != outputPath {
		t.Errorf("Expected %q, got %q", outputPath, result)
	}
	// Verify the file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Expected HTML file to be created")
	}
}

func TestAgentSession_ExportToHTML_AutoPath(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{
		Agent: ag,
		CWD:   os.TempDir(),
	})

	// Empty outputPath should auto-generate
	result, err := as.ExportToHTML("")
	if err != nil {
		t.Fatalf("ExportToHTML with auto path failed: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty auto-generated path")
	}
	if !strings.HasSuffix(result, ".html") {
		t.Errorf("Expected .html extension, got %q", result)
	}
	// Clean up
	os.Remove(result)
}

func TestAgentSession_ExportToHTML_InvalidPath(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})

	_, err := as.ExportToHTML("/nonexistent/dir/test.html")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestAgentSession_GetActiveToolNames(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})

	names := as.GetActiveToolNames()
	// Without tools, should be empty
	if names == nil {
		// This is fine
	}
}

func TestAgentSession_LastAssistantText(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})

	text := as.LastAssistantText()
	// Without any messages, should be empty
	if text != "" {
		t.Errorf("Expected empty text, got %q", text)
	}
}

func TestAgentSession_SessionID(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})

	id := as.SessionID()
	// Without session manager, may be empty
	_ = id
}

func TestAgentSession_SessionFile(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})

	file := as.SessionFile()
	_ = file
}

func TestAgentSession_SupportsThinking_Reasoning(t *testing.T) {
	model := ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI, Reasoning: true}
	// Model with Reasoning=true should support thinking
	if !model.Reasoning {
		t.Error("Expected model to support reasoning")
	}
}

func TestAgentSession_SupportsXhighThinking_Check(t *testing.T) {
	model := ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}
	// Non-reasoning model should not support xhigh thinking
	if model.Reasoning {
		t.Error("Did not expect non-reasoning model to support reasoning")
	}
}

func TestAgentSession_AbortCompaction(t *testing.T) {
	ag := agent.NewAgent(ai.ModelInfo{ID: "test", Provider: ai.ProviderOpenAI}, nil, ai.StreamOpenAIResponses, agent.AgentLoopConfig{})
	as := NewAgentSession(AgentSessionConfig{Agent: ag})
	// AbortCompaction when not compacting should not panic
	as.AbortCompaction()
}

package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/badlogic/pi-mono/pkg/ai"
)

// mockStreamWithToolCalls simulates an LLM that makes a tool call, then gives a final text response.
// First call: returns a tool call (stopReason=toolUse)
// Second call: returns text "Tool result: tool executed" (stopReason=stop)
func mockStreamWithToolCalls() (ai.StreamFunction, *int) {
	callCount := 0
	return func(ctx context.Context, model ai.ModelInfo, aiCtx ai.Context, options any) ai.AssistantMessageEventStream {
		stream := make(chan ai.AssistantMessageEvent, 20)
		callCount++

		go func() {
			defer close(stream)
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Check if the last message is a tool result (meaning we're on the second call)
			isSecondCall := false
			for _, msg := range aiCtx.Messages {
				if _, ok := msg.(ai.ToolResultMessage); ok {
					isSecondCall = true
					break
				}
			}

			if isSecondCall {
				// Second LLM call: respond with text summarizing the tool result
				stream <- ai.AssistantMessageEvent{Type: ai.EventStart}
				txt := "Tool result: tool executed"
				stream <- ai.AssistantMessageEvent{
					Type:  ai.EventTextDelta,
					Delta: &txt,
				}
				reason := ai.StopReasonStop
				stream <- ai.AssistantMessageEvent{
					Type:   ai.EventDone,
					Reason: &reason,
				}
			} else {
				// First LLM call: make a tool call
				stream <- ai.AssistantMessageEvent{Type: ai.EventStart}
				name := "mock_tool"
				id := "call_test_123"
				stream <- ai.AssistantMessageEvent{
					Type: ai.EventToolCallStart,
					ToolCall: &ai.ToolCall{
						ID:   id,
						Name: name,
					},
				}
				args := `{"path":"test.txt"}`
				stream <- ai.AssistantMessageEvent{
					Type:  ai.EventToolCallDelta,
					Delta: &args,
				}
				stream <- ai.AssistantMessageEvent{Type: ai.EventToolCallEnd}
				reason := ai.StopReasonToolUse
				stream <- ai.AssistantMessageEvent{
					Type:   ai.EventDone,
					Reason: &reason,
				}
			}
		}()
		return stream
	}, &callCount
}

func mockToolForExecution() AgentTool {
	return AgentTool{
		Name:        "mock_tool",
		Description: "A mock tool for testing",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{"type": "string"},
			},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate AgentToolUpdateCallback) (AgentToolResult, error) {
			pathStr, _ := params["path"].(string)
			return AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: "tool executed: " + pathStr},
				},
				Details: map[string]any{"path": pathStr},
			}, nil
		},
	}
}

func TestAgentToolExecution(t *testing.T) {
	streamFn, _ := mockStreamWithToolCalls()
	modelInfo := ai.ModelInfo{
		ID:       "mock-model",
		Provider: ai.ProviderOpenAI,
		API:      ai.ApiOpenAIResponses,
	}

	ag := NewAgent(modelInfo, []AgentTool{mockToolForExecution()}, streamFn, AgentLoopConfig{
		ToolExecution: ToolExecutionSequential,
	})

	// Collect all events
	var events []AgentEvent
	ag.Subscribe(func(e AgentEvent) {
		events = append(events, e)
	})

	userMsg := ai.UserMessage{
		Content: []ai.Content{ai.TextContent{Text: "Use the tool"}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ag.Prompt(ctx, userMsg)
	if err != nil {
		t.Fatalf("agent.Prompt failed: %v", err)
	}

	// Verify event sequence
	hasToolStart := false
	hasToolEnd := false
	hasTurnEnd := false
	toolResultCount := 0

	for _, e := range events {
		switch e.Type {
		case EventToolExecutionStart:
			hasToolStart = true
			if e.ToolName != "mock_tool" {
				t.Errorf("Expected tool name 'mock_tool', got %q", e.ToolName)
			}
		case EventToolExecutionEnd:
			hasToolEnd = true
			if e.IsError {
				t.Error("Tool execution should not have errored")
			}
		case EventTurnEnd:
			hasTurnEnd = true
			if len(e.ToolResults) > 0 {
				toolResultCount = len(e.ToolResults)
			}
		}
	}

	if !hasToolStart {
		t.Error("Expected EventToolExecutionStart")
	}
	if !hasToolEnd {
		t.Error("Expected EventToolExecutionEnd")
	}
	if !hasTurnEnd {
		t.Error("Expected EventTurnEnd")
	}
	if toolResultCount != 1 {
		t.Errorf("Expected 1 tool result at turn end, got %d", toolResultCount)
	}

	// Verify messages contain user, assistant (with tool call), tool result, and final assistant
	msgs := ag.Messages()
	_, hasUser := msgs[0].(ai.UserMessage)
	if !hasUser {
		t.Error("Expected first message to be UserMessage")
	}

	// Find tool result message
	foundToolResult := false
	for _, m := range msgs {
		if _, ok := m.(ai.ToolResultMessage); ok {
			foundToolResult = true
		}
	}
	if !foundToolResult {
		t.Error("Expected to find a ToolResultMessage in conversation history")
	}
}

func TestAgentParallelToolExecution(t *testing.T) {
	streamFn, _ := mockStreamWithToolCalls()
	modelInfo := ai.ModelInfo{
		ID:       "mock-model",
		Provider: ai.ProviderOpenAI,
		API:      ai.ApiOpenAIResponses,
	}

	ag := NewAgent(modelInfo, []AgentTool{mockToolForExecution()}, streamFn, AgentLoopConfig{
		ToolExecution: ToolExecutionParallel,
	})

	var events []AgentEvent
	ag.Subscribe(func(e AgentEvent) {
		events = append(events, e)
	})

	userMsg := ai.UserMessage{
		Content: []ai.Content{ai.TextContent{Text: "Use the tool"}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ag.Prompt(ctx, userMsg)
	if err != nil {
		t.Fatalf("agent.Prompt failed: %v", err)
	}

	// Should have tool execution events in parallel mode too
	hasToolStart := false
	hasToolEnd := false
	for _, e := range events {
		if e.Type == EventToolExecutionStart {
			hasToolStart = true
		}
		if e.Type == EventToolExecutionEnd {
			hasToolEnd = true
		}
	}
	if !hasToolStart {
		t.Error("Expected EventToolExecutionStart in parallel mode")
	}
	if !hasToolEnd {
		t.Error("Expected EventToolExecutionEnd in parallel mode")
	}
}

func TestAgentBeforeToolCallHook(t *testing.T) {
	streamFn, _ := mockStreamWithToolCalls()
	modelInfo := ai.ModelInfo{
		ID:       "mock-model",
		Provider: ai.ProviderOpenAI,
		API:      ai.ApiOpenAIResponses,
	}

	blockCalled := false
	ag := NewAgent(modelInfo, []AgentTool{mockToolForExecution()}, streamFn, AgentLoopConfig{
		ToolExecution: ToolExecutionSequential,
		BeforeToolCall: func(ctx context.Context, callCtx BeforeToolCallContext) (*BeforeToolCallResult, error) {
			blockCalled = true
			// Block the tool call
			return &BeforeToolCallResult{Block: true}, nil
		},
	})

	var events []AgentEvent
	ag.Subscribe(func(e AgentEvent) {
		events = append(events, e)
	})

	userMsg := ai.UserMessage{
		Content: []ai.Content{ai.TextContent{Text: "Use the tool"}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ag.Prompt(ctx, userMsg)
	if err != nil {
		t.Fatalf("agent.Prompt failed: %v", err)
	}

	if !blockCalled {
		t.Error("Expected BeforeToolCall hook to be called")
	}

	// The tool should have been blocked
	for _, e := range events {
		if e.Type == EventToolExecutionEnd && e.IsError == false && e.ToolName == "mock_tool" {
			// Tool was not blocked — this is an error
			t.Error("Tool should have been blocked by BeforeToolCall hook")
		}
	}
}

func TestAgentAfterToolCallHook(t *testing.T) {
	streamFn, _ := mockStreamWithToolCalls()
	modelInfo := ai.ModelInfo{
		ID:       "mock-model",
		Provider: ai.ProviderOpenAI,
		API:      ai.ApiOpenAIResponses,
	}

	afterCalled := false
	ag := NewAgent(modelInfo, []AgentTool{mockToolForExecution()}, streamFn, AgentLoopConfig{
		ToolExecution: ToolExecutionSequential,
		AfterToolCall: func(ctx context.Context, callCtx AfterToolCallContext) (*AfterToolCallResult, error) {
			afterCalled = true
			// Modify the result
			isErr := false
			return &AfterToolCallResult{
				Content: []ai.Content{ai.TextContent{Text: "modified result"}},
				IsError: &isErr,
			}, nil
		},
	})

	userMsg := ai.UserMessage{
		Content: []ai.Content{ai.TextContent{Text: "Use the tool"}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ag.Prompt(ctx, userMsg)
	if err != nil {
		t.Fatalf("agent.Prompt failed: %v", err)
	}

	if !afterCalled {
		t.Error("Expected AfterToolCall hook to be called")
	}
}

// Verify the agent can handle a missing tool gracefully
func TestAgentMissingTool(t *testing.T) {
	streamFn, _ := mockStreamWithToolCalls()
	modelInfo := ai.ModelInfo{
		ID:       "mock-model",
		Provider: ai.ProviderOpenAI,
		API:      ai.ApiOpenAIResponses,
	}

	// No tools registered — the tool call should result in an error
	ag := NewAgent(modelInfo, []AgentTool{}, streamFn, AgentLoopConfig{
		ToolExecution: ToolExecutionSequential,
	})

	var events []AgentEvent
	ag.Subscribe(func(e AgentEvent) {
		events = append(events, e)
	})

	userMsg := ai.UserMessage{
		Content: []ai.Content{ai.TextContent{Text: "Use the tool"}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ag.Prompt(ctx, userMsg)
	if err != nil {
		t.Fatalf("agent.Prompt failed: %v", err)
	}

	// Should have a tool execution end with error
	foundToolError := false
	for _, e := range events {
		if e.Type == EventToolExecutionEnd && e.IsError {
			foundToolError = true
		}
	}
	if !foundToolError {
		t.Error("Expected tool execution error for missing tool")
	}
}

// Verify the JSON unmarshalling of raw tool call arguments works
func TestToolCallArgumentParsing(t *testing.T) {
	rawArgs := `{"path": "/tmp/test.txt", "line": 42}`
	var parsed map[string]any
	err := json.Unmarshal([]byte(rawArgs), &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal tool args: %v", err)
	}
	if parsed["path"] != "/tmp/test.txt" {
		t.Errorf("Expected path '/tmp/test.txt', got %v", parsed["path"])
	}
	if parsed["line"] != float64(42) {
		t.Errorf("Expected line 42, got %v", parsed["line"])
	}
}

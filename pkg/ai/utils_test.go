package ai

import (
	"testing"
)

func TestIsContextOverflow_ErrorBased(t *testing.T) {
	errMsg := "prompt is too long: 213462 tokens > 200000 maximum"
	msg := AssistantMessage{
		StopReason:  StopReasonError,
		ErrorMessage: &errMsg,
	}
	if !IsContextOverflow(msg, 200000) {
		t.Error("Expected context overflow detection for Anthropic-style error")
	}
}

func TestIsContextOverflow_NonOverflow(t *testing.T) {
	errMsg := "rate limit exceeded"
	msg := AssistantMessage{
		StopReason:  StopReasonError,
		ErrorMessage: &errMsg,
	}
	if IsContextOverflow(msg, 200000) {
		t.Error("Rate limit errors should not be detected as overflow")
	}
}

func TestIsContextOverflow_SilentOverflow(t *testing.T) {
	msg := AssistantMessage{
		StopReason: StopReasonStop,
		Usage: Usage{
			Input:     150000,
			CacheRead: 60000,
		},
	}
	if !IsContextOverflow(msg, 200000) {
		t.Error("Expected silent overflow detection when usage exceeds context window")
	}
}

func TestIsContextOverflow_NoOverflow(t *testing.T) {
	msg := AssistantMessage{
		StopReason: StopReasonStop,
		Usage: Usage{
			Input:     50000,
			CacheRead: 10000,
		},
	}
	if IsContextOverflow(msg, 200000) {
		t.Error("Should not detect overflow when within context window")
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		errMsg    string
		retryable bool
	}{
		{"overloaded error", "overloaded_error", true},
		{"rate limit", "Rate limit exceeded", true},
		{"500 error", "500 Internal Server Error", true},
		{"503 error", "503 Service Unavailable", true},
		{"context overflow", "prompt is too long: 300000 tokens", false},
		{"normal stop", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := AssistantMessage{
				StopReason:  StopReasonError,
				ErrorMessage: &tt.errMsg,
			}
			result := IsRetryableError(msg, 200000)
			if result != tt.retryable {
				t.Errorf("Expected retryable=%v for %q, got %v", tt.retryable, tt.errMsg, result)
			}
		})
	}
}

func TestCalculateContextTokens(t *testing.T) {
	usage := Usage{
		Input:      1000,
		Output:     500,
		CacheRead:  200,
		CacheWrite: 100,
	}
	result := CalculateContextTokens(usage)
	expected := 1800
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestEstimateContextTokens(t *testing.T) {
	messages := []Message{
		UserMessage{Timestamp: 1},
		AssistantMessage{
			Usage:      Usage{Input: 1000, Output: 500, CacheRead: 200, CacheWrite: 100},
			StopReason: StopReasonStop,
			Timestamp:  2,
		},
	}
	result := EstimateContextTokens(messages)
	if result.Tokens != 1800 {
		t.Errorf("Expected 1800 tokens, got %d", result.Tokens)
	}
	if result.LastUsageIndex == nil || *result.LastUsageIndex != 1 {
		t.Error("Expected LastUsageIndex to be 1")
	}
}

func TestOverflowPatterns(t *testing.T) {
	patterns := []struct {
		errMsg   string
		overflow bool
	}{
		{"prompt is too long: 300000 tokens > 200000 maximum", true},
		{"request_too_large", true},
		{"Your input exceeds the context window of this model", true},
		{"input token count (1196265) exceeds the maximum number of tokens allowed (1048575)", true},
		{"This model's maximum prompt length is 131072 but the request contains 537812 tokens", true},
		{"Please reduce the length of the messages or completion", true},
		{"This endpoint's maximum context length is 128000 tokens. However, you requested about 140000 tokens", true},
		{"Throttling error: Too many tokens, please wait before trying again", false},
		{"rate limit exceeded for provider", false},
		{"too many requests, slow down", false},
	}

	for _, tt := range patterns {
		overflow := isOverflowError(tt.errMsg)
		nonOverflow := isNonOverflowError(tt.errMsg)
		result := overflow && !nonOverflow
		if result != tt.overflow {
			t.Errorf("Expected overflow=%v for %q, got %v (overflow=%v nonOverflow=%v)",
				tt.overflow, tt.errMsg, result, overflow, nonOverflow)
		}
	}
}

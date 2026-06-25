package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIsContextOverflow_ErrorBased(t *testing.T) {
	errMsg := "prompt is too long: 213462 tokens > 200000 maximum"
	msg := AssistantMessage{
		StopReason:   StopReasonError,
		ErrorMessage: &errMsg,
	}
	if !IsContextOverflow(msg, 200000) {
		t.Error("Expected context overflow detection for Anthropic-style error")
	}
}

func TestIsContextOverflow_NonOverflow(t *testing.T) {
	errMsg := "rate limit exceeded"
	msg := AssistantMessage{
		StopReason:   StopReasonError,
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
				StopReason:   StopReasonError,
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
			Usage:       Usage{Input: 1000, Output: 500, CacheRead: 200, CacheWrite: 100},
			StopReason:  StopReasonStop,
			Timestamp:   2,
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
			t.Errorf("Expected overflow=%v for %q, got %v (overflow=%v nonOverflow=%v)", tt.overflow, tt.errMsg, result, overflow, nonOverflow)
		}
	}
}

// ---------------------------------------------------------------------------
// Provider error parsing tests
// ---------------------------------------------------------------------------

func TestParseOpenAIErrorBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "structured error",
			body:     `{"error":{"message":"Rate limit reached","type":"rate_limit_error","code":"429"}}`,
			expected: "[rate_limit_error] (code: 429) Rate limit reached",
		},
		{
			name:     "message only",
			body:     `{"error":{"message":"Something went wrong"}}`,
			expected: "Something went wrong",
		},
		{
			name:     "invalid JSON",
			body:     "not json at all",
			expected: "not json at all",
		},
		{
			name:     "empty body",
			body:     "",
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOpenAIErrorBody(tt.body)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseAnthropicErrorBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "overloaded error",
			body:     `{"error":{"type":"overloaded_error","message":"Overloaded"}}`,
			expected: "[overloaded_error] Overloaded",
		},
		{
			name:     "message only",
			body:     `{"error":{"message":"API error"}}`,
			expected: "API error",
		},
		{
			name:     "invalid JSON",
			body:     "plain text error",
			expected: "plain text error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAnthropicErrorBody(tt.body)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseGoogleErrorBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "resource exhausted",
			body:     `{"error":{"code":429,"message":"Resource exhausted","status":"RESOURCE_EXHAUSTED"}}`,
			expected: "[RESOURCE_EXHAUSTED] Resource exhausted",
		},
		{
			name:     "message only",
			body:     `{"error":{"message":"API error"}}`,
			expected: "API error",
		},
		{
			name:     "invalid JSON",
			body:     "some error text",
			expected: "some error text",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGoogleErrorBody(tt.body)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected time.Duration
	}{
		{"empty header", "", 0},
		{"numeric seconds", "30", 30 * time.Second},
		{"float seconds", "1.5", 1500 * time.Millisecond},
		{"zero", "0", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{Header: http.Header{}}
			if tt.header != "" {
				resp.Header.Set("Retry-After", tt.header)
			}
			result := extractRetryAfter(resp)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFormatProviderError(t *testing.T) {
	// Test with OpenAI error
	openAIBody := `{"error":{"message":"Rate limit reached","type":"rate_limit_error","code":"429"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "20")
		w.WriteHeader(429)
		w.Write([]byte(openAIBody))
	}))
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL, nil)
	client := &http.Client{}
	httpResp, _ := client.Do(req)
	defer httpResp.Body.Close()

	errMsg := formatProviderError("OpenAI", httpResp)
	if !strings.Contains(errMsg, "429") {
		t.Errorf("Expected status 429 in error message, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "Rate limit reached") {
		t.Errorf("Expected parsed message in error, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "retry after") {
		t.Errorf("Expected retry-after info in error, got: %s", errMsg)
	}
}

func TestReadErrorBody(t *testing.T) {
	// Normal body
	result := readErrorBody(strings.NewReader("hello world"))
	if result != "hello world" {
		t.Errorf("Expected 'hello world', got %q", result)
	}

	// Large body should be truncated
	bigBody := strings.Repeat("x", 10000)
	result = readErrorBody(strings.NewReader(bigBody))
	if len(result) > 4100 {
		t.Errorf("Body should be truncated to ~4KB, got %d bytes", len(result))
	}
}

func TestOpenAIErrorRespUnmarshal(t *testing.T) {
	raw := `{"error":{"message":"You exceeded your current quota","type":"insufficient_quota","code":"429"}}`
	var resp openAIErrorResp
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error == nil {
		t.Fatal("Expected error to be non-nil")
	}
	if resp.Error.Message != "You exceeded your current quota" {
		t.Errorf("Unexpected message: %s", resp.Error.Message)
	}
	if resp.Error.Type != "insufficient_quota" {
		t.Errorf("Unexpected type: %s", resp.Error.Type)
	}
	if resp.Error.Code != "429" {
		t.Errorf("Unexpected code: %s", resp.Error.Code)
	}
}

func TestAnthropicErrorRespUnmarshal(t *testing.T) {
	raw := `{"error":{"type":"overloaded_error","message":"Anthropic's API is temporarily overloaded."}}`
	var resp anthropicErrorResp
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error == nil {
		t.Fatal("Expected error to be non-nil")
	}
	if resp.Error.Type != "overloaded_error" {
		t.Errorf("Unexpected type: %s", resp.Error.Type)
	}
}

func TestGoogleErrorRespUnmarshal(t *testing.T) {
	raw := `{"error":{"code":429,"message":"Resource has been exhausted","status":"RESOURCE_EXHAUSTED"}}`
	var resp googleErrorResp
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error == nil {
		t.Fatal("Expected error to be non-nil")
	}
	if resp.Error.Code != 429 {
		t.Errorf("Unexpected code: %d", resp.Error.Code)
	}
	if resp.Error.Status != "RESOURCE_EXHAUSTED" {
		t.Errorf("Unexpected status: %s", resp.Error.Status)
	}
}

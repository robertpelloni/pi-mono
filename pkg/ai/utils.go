package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// IsContextOverflow checks if an assistant message represents a context overflow error.
// It handles two cases:
// 1. Error-based overflow: Most providers return stopReason "error" with a specific error message pattern.
// 2. Silent overflow: Some providers accept overflow requests and return successfully
//    but with usage.input exceeding the context window.
func IsContextOverflow(message AssistantMessage, contextWindow int) bool {
	// Case 1: Check error message patterns
	if message.StopReason == StopReasonError && message.ErrorMessage != nil {
		errMsg := *message.ErrorMessage
		// Skip messages matching known non-overflow patterns (e.g. throttling / rate-limit)
		if !isNonOverflowError(errMsg) && isOverflowError(errMsg) {
			return true
		}
	}
	// Case 2: Silent overflow (z.ai style) - successful but usage exceeds context
	if contextWindow > 0 && message.StopReason == StopReasonStop {
		inputTokens := message.Usage.Input + message.Usage.CacheRead
		if inputTokens > contextWindow {
			return true
		}
	}
	return false
}

// overflowPatterns matches error messages from different providers indicating context overflow.
var overflowPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)prompt is too long`),                   // Anthropic token overflow
	regexp.MustCompile(`(?i)request_too_large`),                    // Anthropic request byte-size overflow
	regexp.MustCompile(`(?i)input is too long for requested model`), // Amazon Bedrock
	regexp.MustCompile(`(?i)exceeds the context window`),           // OpenAI
	regexp.MustCompile(`(?i)input token count.*exceeds the maximum`), // Google
	regexp.MustCompile(`(?i)maximum prompt length is \d+`),         // xAI
	regexp.MustCompile(`(?i)reduce the length of the messages`),    // Groq
	regexp.MustCompile(`(?i)maximum context length is \d+ tokens`), // OpenRouter
	regexp.MustCompile(`(?i)exceeds the limit of \d+`),            // GitHub Copilot
	regexp.MustCompile(`(?i)exceeds the available context size`),   // llama.cpp
	regexp.MustCompile(`(?i)greater than the context length`),      // LM Studio
	regexp.MustCompile(`(?i)context window exceeds limit`),         // MiniMax
	regexp.MustCompile(`(?i)exceeded model token limit`),           // Kimi
	regexp.MustCompile(`(?i)too large for model with \d+ maximum`), // Mistral
	regexp.MustCompile(`(?i)model_context_window_exceeded`),        // z.ai
	regexp.MustCompile(`(?i)prompt too long; exceeded (?:max )?context length`), // Ollama
	regexp.MustCompile(`(?i)context[_ ]length[_ ]exceeded`),        // Generic
	regexp.MustCompile(`(?i)too many tokens`),                      // Generic
	regexp.MustCompile(`(?i)token limit exceeded`),                 // Generic
	regexp.MustCompile(`(?i)^4(?:00|13)\s*(?:status code)?\s*\(no body\)`), // Cerebras
}

// nonOverflowPatterns matches error messages that look like overflow but aren't.
var nonOverflowPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^(Throttling error|Service unavailable):`), // AWS Bedrock non-overflow
	regexp.MustCompile(`(?i)rate limit`),     // Generic rate limiting
	regexp.MustCompile(`(?i)too many requests`), // HTTP 429 style
}

func isOverflowError(errMsg string) bool {
	for _, p := range overflowPatterns {
		if p.MatchString(errMsg) {
			return true
		}
	}
	return false
}

func isNonOverflowError(errMsg string) bool {
	for _, p := range nonOverflowPatterns {
		if p.MatchString(errMsg) {
			return true
		}
	}
	return false
}

// IsRetryableError checks if an assistant message error is retryable
// (overloaded, rate limit, server errors, network errors).
// Context overflow errors are NOT retryable (handled by compaction instead).
func IsRetryableError(message AssistantMessage, contextWindow int) bool {
	if message.StopReason != StopReasonError || message.ErrorMessage == nil {
		return false
	}
	// Context overflow is handled by compaction, not retry
	if IsContextOverflow(message, contextWindow) {
		return false
	}
	errMsg := *message.ErrorMessage
	errLower := strings.ToLower(errMsg)
	// Match: overloaded, rate limit, 429, 500s, network errors, etc.
	retryablePatterns := []string{
		"overloaded",
		"provider returned error",
		"rate limit",
		"too many requests",
		"429",
		"500",
		"502",
		"503",
		"504",
		"service unavailable",
		"server error",
		"internal error",
		"network error",
		"connection error",
		"connection refused",
		"other side closed",
		"fetch failed",
		"upstream connect",
		"reset before headers",
		"socket hang up",
		"ended without",
		"timed out",
		"timeout",
		"terminated",
		"retry delay",
	}
	for _, pattern := range retryablePatterns {
		if strings.Contains(errLower, pattern) {
			return true
		}
	}
	return false
}

// CalculateContextTokens estimates the total context tokens from an assistant message's usage.
func CalculateContextTokens(usage Usage) int {
	return usage.Input + usage.Output + usage.CacheRead + usage.CacheWrite
}

// EstimateContextTokens estimates total context tokens from a list of messages.
// Uses the most recent assistant usage as a reference point.
func EstimateContextTokens(messages []Message) EstimateResult {
	var lastUsageIndex *int
	var lastUsage Usage
	for i := len(messages) - 1; i >= 0; i-- {
		if am, ok := messages[i].(AssistantMessage); ok {
			if am.StopReason != StopReasonError && am.StopReason != StopReasonAborted {
				idx := i
				lastUsageIndex = &idx
				lastUsage = am.Usage
				break
			}
		}
	}
	if lastUsageIndex == nil {
		return EstimateResult{Tokens: 0, LastUsageIndex: nil}
	}
	return EstimateResult{
		Tokens:          CalculateContextTokens(lastUsage),
		LastUsageIndex:  lastUsageIndex,
	}
}

// EstimateResult holds the result of context token estimation.
type EstimateResult struct {
	Tokens          int
	LastUsageIndex  *int
}

// ---------------------------------------------------------------------------
// Shared provider error handling utilities
// ---------------------------------------------------------------------------

// readErrorBody reads up to 4KB of an error response body for debugging.
// Returns the raw text, or "(could not read body)" on failure.
func readErrorBody(body io.Reader) string {
	data, err := io.ReadAll(io.LimitReader(body, 4096))
	if err != nil {
		return "(could not read body)"
	}
	return string(data)
}

// openAIErrorResp represents OpenAI's structured error JSON.
// Format: { "error": { "message": "...", "type": "...", "code": "..." } }
type openAIErrorResp struct {
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// anthropicErrorResp represents Anthropic's structured error JSON.
// Format: { "error": { "type": "...", "message": "..." } }
type anthropicErrorResp struct {
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// googleErrorResp represents Google's structured error JSON.
// Format: { "error": { "code": 429, "message": "...", "status": "RESOURCE_EXHAUSTED" } }
type googleErrorResp struct {
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// parseOpenAIErrorBody attempts to parse OpenAI's structured error JSON.
// Falls back to raw body text if parsing fails.
func parseOpenAIErrorBody(bodyText string) string {
	var errResp openAIErrorResp
	if err := json.Unmarshal([]byte(bodyText), &errResp); err == nil && errResp.Error != nil {
		// Build prefix in order: [type] (code: code) message
		var prefix []string
		if errResp.Error.Type != "" {
			prefix = append(prefix, fmt.Sprintf("[%s]", errResp.Error.Type))
		}
		if errResp.Error.Code != "" {
			prefix = append(prefix, fmt.Sprintf("(code: %s)", errResp.Error.Code))
		}
		prefix = append(prefix, errResp.Error.Message)
		return strings.Join(prefix, " ")
	}
	return bodyText
}

// parseAnthropicErrorBody attempts to parse Anthropic's structured error JSON.
func parseAnthropicErrorBody(bodyText string) string {
	var errResp anthropicErrorResp
	if err := json.Unmarshal([]byte(bodyText), &errResp); err == nil && errResp.Error != nil {
		if errResp.Error.Type != "" {
			return fmt.Sprintf("[%s] %s", errResp.Error.Type, errResp.Error.Message)
		}
		return errResp.Error.Message
	}
	return bodyText
}

// parseGoogleErrorBody attempts to parse Google's structured error JSON.
func parseGoogleErrorBody(bodyText string) string {
	var errResp googleErrorResp
	if err := json.Unmarshal([]byte(bodyText), &errResp); err == nil && errResp.Error != nil {
		if errResp.Error.Status != "" {
			return fmt.Sprintf("[%s] %s", errResp.Error.Status, errResp.Error.Message)
		}
		return errResp.Error.Message
	}
	return bodyText
}

// extractRetryAfter extracts the Retry-After header value as a duration.
// Returns 0 if the header is absent or cannot be parsed.
func extractRetryAfter(resp *http.Response) time.Duration {
	val := resp.Header.Get("Retry-After")
	if val == "" {
		return 0
	}
	// Try numeric seconds first
	var secs float64
	if _, err := fmt.Sscanf(val, "%f", &secs); err == nil && secs > 0 {
		return time.Duration(secs * float64(time.Second))
	}
	// Try HTTP-date format
	if t, err := http.ParseTime(val); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

// formatProviderError creates a standardized error message for a provider.
// It includes the status code, parsed error body, and Retry-After info.
// The caller must NOT close resp.Body before calling this — it reads the body.
func formatProviderError(providerName string, resp *http.Response) string {
	bodyText := readErrorBody(resp.Body)

	var parsedMsg string
	switch providerName {
	case "OpenAI":
		parsedMsg = parseOpenAIErrorBody(bodyText)
	case "Anthropic":
		parsedMsg = parseAnthropicErrorBody(bodyText)
	case "Google":
		parsedMsg = parseGoogleErrorBody(bodyText)
	default:
		parsedMsg = bodyText
	}

	// Trim overly long messages
	if len(parsedMsg) > 500 {
		parsedMsg = parsedMsg[:497] + "..."
	}

	errMsg := fmt.Sprintf("%s API error: status %d: %s", providerName, resp.StatusCode, parsedMsg)

	// Include Retry-After info if available
	if retryAfter := extractRetryAfter(resp); retryAfter > 0 {
		errMsg += fmt.Sprintf(" (retry after %s)", retryAfter.Round(time.Second))
	}

	return errMsg
}

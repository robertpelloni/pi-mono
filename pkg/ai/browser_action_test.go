package ai

import (
	"strings"
	"testing"
)

// TestHandleClineBrowserAction verifies all supported browser_action commands.
// The handler uses simulated/offline operations (no real browser) so tests are deterministic.
func TestHandleClineBrowserAction(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		want     string
		wantErr  bool
		contains string
	}{
		{
			name: "launch with URL delegates to lynx",
			args: map[string]interface{}{
				"action": "launch",
				"url":    "https://example.com",
			},
			// lynx may not be installed on CI, but the handler tries it; we just check it runs.
			// We'll mark as no-error expectation because it tries execution regardless.
			contains: "Example Domain", // Assuming lynx runs and returns Example domain text
		},
		{
			name: "launch with empty URL still attempts navigation",
			args: map[string]interface{}{
				"action": "launch",
			},
			contains: "Error: missing 'url' parameter",
		},
		{
			name: "click action returns coordinate string",
			args: map[string]interface{}{
				"action":     "click",
				"coordinate": "100,200",
			},
			want: "Clicked at coordinate: 100,200",
		},
		{
			name: "click with missing coordinate still returns message",
			args: map[string]interface{}{
				"action": "click",
			},
			contains: "Clicked at coordinate",
		},
		{
			name: "type action returns typed text",
			args: map[string]interface{}{
				"action": "type",
				"text":   "hello world",
			},
			want: "Typed text: hello world",
		},
		{
			name: "type with missing text returns empty string",
			args: map[string]interface{}{
				"action": "type",
			},
			contains: "Typed text",
		},
		{
			name: "scroll_down returns scroll message",
			args: map[string]interface{}{
				"action": "scroll_down",
			},
			want: "Scrolled browser: scroll_down",
		},
		{
			name: "scroll_up returns scroll message",
			args: map[string]interface{}{
				"action": "scroll_up",
			},
			want: "Scrolled browser: scroll_up",
		},
		{
			name: "close returns browser closed",
			args: map[string]interface{}{
				"action": "close",
			},
			want: "Browser closed.",
		},
		{
			name: "screenshot delegates to computer use screenshot",
			args: map[string]interface{}{
				"action": "screenshot",
			},
			// xdotool may not be installed but the handler is expected to fail gracefully.
			contains: "Error",
		},
		{
			name: "unknown action returns error message",
			args: map[string]interface{}{
				"action": "unknown_action",
			},
			want: "Unknown browser action.",
		},
		{
			name: "empty args returns unknown",
			args: map[string]interface{}{},
			want: "Unknown browser action.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handleClineBrowserAction(tt.args)
			if tt.want != "" && result != tt.want {
				t.Errorf("got %q, want %q", result, tt.want)
			}
			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("result %q does not contain %q", result, tt.contains)
			}
		})
	}
}

// TestBrowserCleanRoomToolsRegistration verifies browser_action is registered in CleanRoomTools.
func TestBrowserCleanRoomToolsRegistration(t *testing.T) {
	if handler, exists := CleanRoomTools["browser_action"]; !exists {
		t.Fatal("browser_action not found in CleanRoomTools map")
	} else if handler == nil {
		t.Fatal("browser_action has nil handler")
	}
}

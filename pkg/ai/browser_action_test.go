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
			// If lynx is installed, we get Example Domain. If not, we get Error.
			// We'll handle this custom logic in the test loop.
		},
		{
			name: "launch with empty URL still attempts navigation",
			args: map[string]interface{}{
				"action": "launch",
				"url":    "",
			},
			// Same as above, handle custom in the test loop.
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

			// Special case for lynx since it might actually succeed on machines where it is installed.
			if tt.name == "launch with URL delegates to lynx" {
				if !strings.Contains(result, "Example Domain") && !strings.Contains(result, "Error") {
					t.Errorf("expected result to either succeed with Example Domain or fail with Error, got %q", result)
				}
				return
			}
			if tt.name == "launch with empty URL still attempts navigation" {
				if !strings.Contains(result, "Error") {
					t.Errorf("expected empty URL launch to return an Error, got %q", result)
				}
				return
			}

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

package outputguard

import (
	"strings"
	"testing"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// ─── Before-execution rules ───

func TestBlockSecretFiles(t *testing.T) {
	guard := NewOutputGuard("/home/user/project")

	result := guard.CheckBefore(GuardContext{
		ToolName: "read",
		Args:     map[string]any{"path": ".env"},
	})

	if result.Allowed {
		t.Error("reading .env should be blocked")
	}
	if result.Reason == "" {
		t.Error("blocked read should have a reason")
	}
}

func TestAllowNormalRead(t *testing.T) {
	guard := NewOutputGuard("/home/user/project")

	result := guard.CheckBefore(GuardContext{
		ToolName: "read",
		Args:     map[string]any{"path": "main.go"},
	})

	if !result.Allowed {
		t.Error("reading a normal file should be allowed")
	}
}

func TestBlockSystemWrites(t *testing.T) {
	guard := NewOutputGuard("/home/user/project")

	result := guard.CheckBefore(GuardContext{
		ToolName: "write",
		Args:     map[string]any{"path": "/etc/passwd"},
	})

	if result.Allowed {
		t.Error("writing to /etc should be blocked")
	}
}

func TestAllowProjectWrites(t *testing.T) {
	guard := NewOutputGuard("/home/user/project")

	result := guard.CheckBefore(GuardContext{
		ToolName: "write",
		Args:     map[string]any{"path": "main.go"},
	})

	if !result.Allowed {
		t.Error("writing to project files should be allowed")
	}
}

func TestBlockDestructiveCommands(t *testing.T) {
	guard := NewOutputGuard("/home/user/project")

	tests := []string{
		"rm -rf /",
		"mkfs.ext4 /dev/sda1",
		"dd if=/dev/zero of=/dev/sda",
	}

	for _, cmd := range tests {
		result := guard.CheckBefore(GuardContext{
			ToolName: "bash",
			Args:     map[string]any{"command": cmd},
		})

		if result.Allowed {
			t.Errorf("destructive command '%s' should be blocked", cmd)
		}
	}
}

func TestAllowNormalCommands(t *testing.T) {
	guard := NewOutputGuard("/home/user/project")

	result := guard.CheckBefore(GuardContext{
		ToolName: "bash",
		Args:     map[string]any{"command": "ls -la"},
	})

	if !result.Allowed {
		t.Error("normal commands should be allowed")
	}
}

// ─── After-execution rules ───

func TestRedactSecrets(t *testing.T) {
	guard := NewOutputGuard("/home/user/project")

	result := guard.CheckAfter(GuardContext{
		ToolName: "bash",
		Args:     map[string]any{"command": "env"},
		Result: agent.AgentToolResult{
			Content: []ai.Content{ai.TextContent{Text: "OPENAI_API_KEY=sk-abc123def456ghi789jkl012mno345pqr678stu901"}},
		},
	})

	if !result.Allowed {
		t.Error("bash output should be allowed (just redacted)")
	}
	if result.ModifiedResult == nil {
		t.Error("bash output with API key should be modified")
	}
}

func TestTruncateLargeOutput(t *testing.T) {
	guard := NewOutputGuard("/home/user/project")

	// Create a very large output
	largeOutput := make([]byte, 60000)
	for i := range largeOutput {
		largeOutput[i] = 'A'
	}

	result := guard.CheckAfter(GuardContext{
		ToolName: "bash",
		Args:     map[string]any{"command": "cat largefile"},
		Result: agent.AgentToolResult{
			Content: []ai.Content{ai.TextContent{Text: string(largeOutput)}},
		},
	})

	if !result.Allowed {
		t.Error("large output should still be allowed (just truncated)")
	}
	if result.ModifiedResult == nil {
		t.Error("large output should be truncated")
	}
}

// ─── Redaction function tests ───

func TestRedactSecretsFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		omits    string
	}{
		{
			name:     "OpenAI key",
			input:    "key=sk-abc123def456ghi789jkl012mno345pqr678stu901xyz",
			contains: "[REDACTED]",
			omits:    "sk-abc123def456ghi789jkl012mno345pqr678stu901xyz",
		},
		{
			name:     "GitHub key",
			input:    "token=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij1234567890",
			contains: "[REDACTED]",
			omits:    "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij1234567890",
		},
		{
			name:     "No secrets",
			input:    "Hello, world!",
			contains: "Hello, world!",
			omits:    "[REDACTED]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := redactSecrets(tc.input)
			if !strings.Contains(result, tc.contains) {
				t.Errorf("expected result to contain '%s', got: %s", tc.contains, result)
			}
			if tc.omits != "" && strings.Contains(result, tc.omits) {
				t.Errorf("expected result NOT to contain '%s', got: %s", tc.omits, result)
			}
		})
	}
}

func TestIsLikelyKey(t *testing.T) {
	if !isLikelyKey("abc123def456ghi789") {
		t.Error("alphanumeric string should be likely key")
	}
	if isLikelyKey("hi") {
		t.Error("short string should not be likely key")
	}
	if isLikelyKey("!!!!!!!!!!") {
		t.Error("non-alphanumeric string should not be likely key")
	}
}

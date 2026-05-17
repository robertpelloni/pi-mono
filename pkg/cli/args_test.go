package cli

import (
	"testing"
)

func TestParseArgs_Empty(t *testing.T) {
	args := ParseArgs([]string{})
	if args.Help {
		t.Error("Help should be false")
	}
	if args.Version {
		t.Error("Version should be false")
	}
	if len(args.Messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(args.Messages))
	}
}

func TestParseArgs_Help(t *testing.T) {
	args := ParseArgs([]string{"--help"})
	if !args.Help {
		t.Error("Help should be true")
	}
	args = ParseArgs([]string{"-h"})
	if !args.Help {
		t.Error("Help should be true with -h")
	}
}

func TestParseArgs_Model(t *testing.T) {
	args := ParseArgs([]string{"--model", "gpt-4o"})
	if args.Model != "gpt-4o" {
		t.Errorf("Expected model gpt-4o, got %s", args.Model)
	}
}

func TestParseArgs_Provider(t *testing.T) {
	args := ParseArgs([]string{"--provider", "anthropic"})
	if args.Provider != "anthropic" {
		t.Errorf("Expected provider anthropic, got %s", args.Provider)
	}
}

func TestParseArgs_Thinking(t *testing.T) {
	args := ParseArgs([]string{"--thinking", "high"})
	if args.Thinking != "high" {
		t.Errorf("Expected thinking high, got %s", args.Thinking)
	}
}

func TestParseArgs_InvalidThinking(t *testing.T) {
	args := ParseArgs([]string{"--thinking", "invalid"})
	if len(args.Diagnostics) == 0 {
		t.Error("Expected diagnostic for invalid thinking level")
	}
}

func TestParseArgs_Continue(t *testing.T) {
	args := ParseArgs([]string{"--continue"})
	if !args.Continue {
		t.Error("Continue should be true")
	}
	args = ParseArgs([]string{"-c"})
	if !args.Continue {
		t.Error("Continue should be true with -c")
	}
}

func TestParseArgs_Messages(t *testing.T) {
	args := ParseArgs([]string{"hello", "world"})
	if len(args.Messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(args.Messages))
	}
	if args.Messages[0] != "hello" {
		t.Errorf("Expected first message 'hello', got %s", args.Messages[0])
	}
}

func TestParseArgs_FileArgs(t *testing.T) {
	args := ParseArgs([]string{"@prompt.md"})
	if len(args.FileArgs) != 1 {
		t.Fatalf("Expected 1 file arg, got %d", len(args.FileArgs))
	}
	if args.FileArgs[0] != "prompt.md" {
		t.Errorf("Expected 'prompt.md', got %s", args.FileArgs[0])
	}
}

func TestParseArgs_ListModels(t *testing.T) {
	args := ParseArgs([]string{"--list-models"})
	if args.ListModels != "true" {
		t.Errorf("Expected ListModels='true', got %s", args.ListModels)
	}

	args = ParseArgs([]string{"--list-models", "sonnet"})
	if args.ListModels != "sonnet" {
		t.Errorf("Expected ListModels='sonnet', got %s", args.ListModels)
	}
}

func TestIsValidThinkingLevel(t *testing.T) {
	tests := []struct {
		level   string
		isValid bool
	}{
		{"off", true},
		{"minimal", true},
		{"low", true},
		{"medium", true},
		{"high", true},
		{"xhigh", true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		if IsValidThinkingLevel(tt.level) != tt.isValid {
			t.Errorf("IsValidThinkingLevel(%q) = %v, want %v", tt.level, !tt.isValid, tt.isValid)
		}
	}
}

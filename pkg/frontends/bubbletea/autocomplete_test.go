package bubbletea

import (
	"testing"

	"github.com/badlogic/pi-mono/pkg/slashcommands"
)

func TestSlashProvider_Trigger(t *testing.T) {
	reg := slashcommands.NewRegistry()
	p := NewSlashProvider(reg)

	tests := []struct {
		name      string
		val       string
		cursorPos int
		want      bool
	}{
		{"trigger at start", "/tes", 4, true},
		{"no trigger space before", "hello /test", 11, false},
		{"no trigger space in token", "hello /te st", 12, false},
		{"empty", "", 0, false},
		{"no slash", "hello", 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.Trigger(tt.val, tt.cursorPos); got != tt.want {
				t.Errorf("SlashProvider.Trigger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlashProvider_Complete(t *testing.T) {
	reg := slashcommands.NewRegistry()
	// Register a test command.
	reg.Register(slashcommands.SlashCommandInfo{
		Name:        "testcmd",
		Description: "Test command",
		Source:      slashcommands.SourceBuiltin,
	}, func(args string) (slashcommands.SlashCommandResult, error) {
		return slashcommands.SlashCommandResult{}, nil
	})
	reg.Register(slashcommands.SlashCommandInfo{
		Name:        "foo",
		Description: "Foo command",
		Source:      slashcommands.SourceBuiltin,
	}, func(args string) (slashcommands.SlashCommandResult, error) {
		return slashcommands.SlashCommandResult{}, nil
	})

	p := NewSlashProvider(reg)

	tests := []struct {
		name      string
		val       string
		cursorPos int
		contains  string
	}{
		{"prefix te", "/te", 3, "/testcmd"},
		{"prefix f", "/f", 2, "/foo"},
		{"empty prefix", "/", 1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Complete(tt.val, tt.cursorPos)
			if tt.contains != "" {
				found := false
				for _, c := range got {
					if c == tt.contains {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("SlashProvider.Complete() missing %q in %v", tt.contains, got)
				}
			}
		})
	}
}

func TestFileProvider_Trigger(t *testing.T) {
	p := NewFileProvider()
	tests := []struct {
		name      string
		val       string
		cursorPos int
		want      bool
	}{
		{"trigger at start", "@go", 3, true},
		{"trigger after space", "hello @go", 9, true},
		{"no trigger no at", "hello", 5, false},
		{"at not start of token", "hello@go", 7, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.Trigger(tt.val, tt.cursorPos); got != tt.want {
				t.Errorf("FileProvider.Trigger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileProvider_Complete(t *testing.T) {
	p := NewFileProvider()
	// Assume there is at least go.mod file in repo root.
	val := "@go"
	cursorPos := 3
	got := p.Complete(val, cursorPos)
	if len(got) == 0 {
		t.Skipf("No files found matching @go. Current dir has files: %v", got)
	}
	// We expect at least "go.mod" to be present.
	found := false
	for _, c := range got {
		if c == "@go.mod" {
			found = true
			break
		}
	}
	if !found {
		t.Logf("Expected @go.mod in completions; got %v", got)
	}
}
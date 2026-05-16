package slashcommands

import (
	"strings"
	"testing"
)

func TestExecuteHelp(t *testing.T) {
	r := NewRegistry()
	result, isCommand, err := r.Execute("/help")

	if !isCommand {
		t.Error("/help should be recognized as a slash command")
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Info == "" {
		t.Error("/help should return info")
	}
	if !strings.Contains(result.Info, "/help") {
		t.Error("/help output should list commands")
	}
}

func TestExecuteQuit(t *testing.T) {
	r := NewRegistry()
	result, isCommand, _ := r.Execute("/quit")

	if !isCommand {
		t.Error("/quit should be recognized")
	}
	if !result.Quit {
		t.Error("/quit should set Quit=true")
	}
}

func TestExecuteExit(t *testing.T) {
	r := NewRegistry()
	result, isCommand, _ := r.Execute("/exit")

	if !isCommand {
		t.Error("/exit should be recognized")
	}
	if !result.Quit {
		t.Error("/exit should set Quit=true")
	}
}

func TestExecuteNew(t *testing.T) {
	r := NewRegistry()
	result, isCommand, _ := r.Execute("/new")

	if !isCommand {
		t.Error("/new should be recognized")
	}
	if !result.NewSession {
		t.Error("/new should set NewSession=true")
	}
}

func TestExecuteCompact(t *testing.T) {
	r := NewRegistry()
	result, isCommand, _ := r.Execute("/compact")

	if !isCommand {
		t.Error("/compact should be recognized")
	}
	if !result.Compact {
		t.Error("/compact should set Compact=true")
	}
}

func TestExecuteModel(t *testing.T) {
	r := NewRegistry()

	// Without args
	result, isCommand, _ := r.Execute("/model")
	if !isCommand {
		t.Error("/model should be recognized")
	}
	if !strings.Contains(result.Info, "Usage") {
		t.Error("/model without args should show usage")
	}

	// With args
	result, isCommand, _ = r.Execute("/model gpt-4o")
	if !isCommand {
		t.Error("/model with args should be recognized")
	}
	if result.SwitchModel != "gpt-4o" {
		t.Errorf("expected SwitchModel='gpt-4o', got '%s'", result.SwitchModel)
	}
}

func TestExecuteProvider(t *testing.T) {
	r := NewRegistry()
	result, isCommand, _ := r.Execute("/provider anthropic")

	if !isCommand {
		t.Error("/provider should be recognized")
	}
	if result.SwitchProvider != "anthropic" {
		t.Errorf("expected SwitchProvider='anthropic', got '%s'", result.SwitchProvider)
	}
}

func TestExecuteUnknown(t *testing.T) {
	r := NewRegistry()
	result, isCommand, _ := r.Execute("/nonexistent")

	if !isCommand {
		t.Error("unknown commands should still be recognized as slash commands")
	}
	if result.Error == "" {
		t.Error("unknown command should return an error message")
	}
}

func TestExecuteNonCommand(t *testing.T) {
	r := NewRegistry()
	_, isCommand, _ := r.Execute("hello world")

	if isCommand {
		t.Error("non-slash input should not be treated as a command")
	}
}

func TestList(t *testing.T) {
	r := NewRegistry()
	list := r.List()

	if len(list) == 0 {
		t.Error("should have built-in commands")
	}

	// Check a few required commands
	required := []string{"help", "quit", "new", "compact", "model"}
	for _, name := range required {
		found := false
		for _, cmd := range list {
			if cmd.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing required command: /%s", name)
		}
	}
}

func TestRegisterCustom(t *testing.T) {
	r := NewRegistry()
	r.Register(SlashCommandInfo{
		Name:        "custom-test",
		Description: "A custom test command",
		Source:      SourceExtension,
	}, func(args string) (SlashCommandResult, error) {
		return SlashCommandResult{Message: "custom executed: " + args}, nil
	})

	result, isCommand, _ := r.Execute("/custom-test hello")
	if !isCommand {
		t.Error("custom command should be recognized")
	}
	if result.Message != "custom executed: hello" {
		t.Errorf("unexpected result: %s", result.Message)
	}
}

func TestHotkeys(t *testing.T) {
	r := NewRegistry()
	result, isCommand, _ := r.Execute("/hotkeys")

	if !isCommand {
		t.Error("/hotkeys should be recognized")
	}
	if !strings.Contains(result.Info, "Ctrl+S") {
		t.Error("/hotkeys should list Ctrl+S shortcut")
	}
}

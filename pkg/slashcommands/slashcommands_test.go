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

// ---------------------------------------------------------------------------
// New Slash Command Tests
// ---------------------------------------------------------------------------

func TestRegistry_ExportCommand(t *testing.T) {
	r := NewRegistry()
	result, isCommand, err := r.Execute("/export test.html")
	if !isCommand {
		t.Error("Expected /export to be a recognized command")
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.Export != "test.html" {
		t.Errorf("Expected Export='test.html', got %q", result.Export)
	}
}

func TestRegistry_ExportCommand_Empty(t *testing.T) {
	r := NewRegistry()
	result, isCommand, _ := r.Execute("/export")
	if !isCommand {
		t.Error("Expected /export to be recognized")
	}
	// Empty args means auto-path
	if result.Export != "" {
		// This is fine - empty means auto-generate
	}
}

func TestRegistry_ThinkingCommand(t *testing.T) {
	r := NewRegistry()
	result, isCommand, err := r.Execute("/thinking high")
	if !isCommand {
		t.Error("Expected /thinking to be recognized")
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ThinkingLevel != "high" {
		t.Errorf("Expected ThinkingLevel='high', got %q", result.ThinkingLevel)
	}
}

func TestRegistry_ThinkingCommand_NoArgs(t *testing.T) {
	r := NewRegistry()
	result, isCommand, _ := r.Execute("/thinking")
	if !isCommand {
		t.Error("Expected /thinking to be recognized")
	}
	if result.Info == "" {
		t.Error("Expected usage info when no args provided")
	}
}

func TestRegistry_TreeCommand(t *testing.T) {
	r := NewRegistry()
	result, isCommand, err := r.Execute("/tree entry-123")
	if !isCommand {
		t.Error("Expected /tree to be recognized")
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.Tree != "entry-123" {
		t.Errorf("Expected Tree='entry-123', got %q", result.Tree)
	}
}

func TestRegistry_TreeCommand_NoArgs(t *testing.T) {
	r := NewRegistry()
	result, isCommand, _ := r.Execute("/tree")
	if !isCommand {
		t.Error("Expected /tree to be recognized")
	}
	if result.Info == "" {
		t.Error("Expected usage info")
	}
}

func TestRegistry_ForkCommand(t *testing.T) {
	r := NewRegistry()
	result, isCommand, err := r.Execute("/fork entry-456")
	if !isCommand {
		t.Error("Expected /fork to be recognized")
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.Fork != "entry-456" {
		t.Errorf("Expected Fork='entry-456', got %q", result.Fork)
	}
}

func TestRegistry_ForkCommand_NoArgs(t *testing.T) {
	r := NewRegistry()
	result, isCommand, _ := r.Execute("/fork")
	if !isCommand {
		t.Error("Expected /fork to be recognized")
	}
	if result.Info == "" {
		t.Error("Expected usage info")
	}
}

func TestRegistry_ImportCommand(t *testing.T) {
	r := NewRegistry()
	result, isCommand, err := r.Execute("/import /path/to/session.jsonl")
	if !isCommand {
		t.Error("Expected /import to be recognized")
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ImportSession != "/path/to/session.jsonl" {
		t.Errorf("Expected ImportSession='/path/to/session.jsonl', got %q", result.ImportSession)
	}
}

func TestRegistry_LoginCommand(t *testing.T) {
	r := NewRegistry()
	result, isCommand, err := r.Execute("/login anthropic")
	if !isCommand {
		t.Error("Expected /login to be recognized")
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.Login != "anthropic" {
		t.Errorf("Expected Login='anthropic', got %q", result.Login)
	}
}

func TestRegistry_ShareCommand(t *testing.T) {
	r := NewRegistry()
	result, isCommand, err := r.Execute("/share")
	if !isCommand {
		t.Error("Expected /share to be recognized")
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !result.Share {
		t.Error("Expected Share=true")
	}
}

func TestRegistry_ScopedModelsCommand(t *testing.T) {
	r := NewRegistry()
	result, isCommand, _ := r.Execute("/scoped-models")
	if !isCommand {
		t.Error("Expected /scoped-models to be recognized")
	}
	if result.Info == "" {
		t.Error("Expected info for /scoped-models")
	}
}

func TestRegistry_LogoutCommand(t *testing.T) {
	r := NewRegistry()
	result, isCommand, _ := r.Execute("/logout")
	if !isCommand {
		t.Error("Expected /logout to be recognized")
	}
	if result.Info == "" {
		t.Error("Expected info for /logout")
	}
}

func TestSlashCommandResult_NewFields(t *testing.T) {
	result := SlashCommandResult{
		Export:         "test.html",
		Fork:          "entry-123",
		ThinkingLevel: "high",
		Tree:          "entry-456",
		Login:         "anthropic",
		ImportSession: "/path/session.jsonl",
		Share:         true,
	}
	if result.Export != "test.html" {
		t.Error("Export mismatch")
	}
	if result.Fork != "entry-123" {
		t.Error("Fork mismatch")
	}
	if result.ThinkingLevel != "high" {
		t.Error("ThinkingLevel mismatch")
	}
	if result.Tree != "entry-456" {
		t.Error("Tree mismatch")
	}
	if result.Login != "anthropic" {
		t.Error("Login mismatch")
	}
	if result.ImportSession != "/path/session.jsonl" {
		t.Error("ImportSession mismatch")
	}
	if !result.Share {
		t.Error("Share mismatch")
	}
}

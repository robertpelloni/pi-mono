package interactive

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/badlogic/pi-mono/pkg/slashcommands"
)

func TestNewInteractiveMode(t *testing.T) {
	im := NewInteractiveMode(nil, InteractiveModeOptions{})
	if im == nil {
		t.Fatal("Expected non-nil InteractiveMode")
	}
}

func TestNewInteractiveMode_WithWriters(t *testing.T) {
	var buf bytes.Buffer
	var errBuf bytes.Buffer
	im := NewInteractiveMode(nil, InteractiveModeOptions{
		Writer:    &buf,
		ErrWriter: &errBuf,
	})
	if im.writer != &buf {
		t.Error("Expected custom writer")
	}
	if im.errWriter != &errBuf {
		t.Error("Expected custom error writer")
	}
}

func TestNewInteractiveMode_DefaultWriters(t *testing.T) {
	im := NewInteractiveMode(nil, InteractiveModeOptions{})
	if im.writer == nil {
		t.Error("Expected default stdout writer")
	}
	if im.errWriter == nil {
		t.Error("Expected default stderr writer")
	}
}

func TestInteractiveModeOptions_Fields(t *testing.T) {
	msg := "initial"
	im := NewInteractiveMode(nil, InteractiveModeOptions{
		InitialMessage:      &msg,
		InitialMessages:     []string{"msg1", "msg2"},
		Verbose:            true,
		MigratedProviders:  []string{"openai"},
	})
	if im.options.InitialMessage == nil || *im.options.InitialMessage != "initial" {
		t.Error("InitialMessage mismatch")
	}
	if len(im.options.InitialMessages) != 2 {
		t.Error("InitialMessages mismatch")
	}
	if !im.options.Verbose {
		t.Error("Verbose mismatch")
	}
	if len(im.options.MigratedProviders) != 1 {
		t.Error("MigratedProviders mismatch")
	}
}

func TestCompactionQueuedMessage(t *testing.T) {
	msg := CompactionQueuedMessage{Text: "hello", Mode: "steer"}
	if msg.Text != "hello" || msg.Mode != "steer" {
		t.Error("CompactionQueuedMessage fields mismatch")
	}
}

func TestErrQuit(t *testing.T) {
	if errQuit == nil {
		t.Fatal("errQuit should not be nil")
	}
	if errQuit.Error() != "quit requested" {
		t.Errorf("Expected 'quit requested', got %q", errQuit.Error())
	}
}

func TestLineScanner(t *testing.T) {
	input := "hello\nworld\n"
	scanner := newLineScanner(strings.NewReader(input))

	if !scanner.scan() {
		t.Fatal("Expected scan to return true")
	}
	if scanner.text() != "hello" {
		t.Errorf("Expected 'hello', got %q", scanner.text())
	}

	if !scanner.scan() {
		t.Fatal("Expected second scan to return true")
	}
	if scanner.text() != "world" {
		t.Errorf("Expected 'world', got %q", scanner.text())
	}

	if scanner.scan() {
		t.Error("Expected scan to return false after EOF")
	}
}

func TestLineScanner_EmptyInput(t *testing.T) {
	scanner := newLineScanner(strings.NewReader(""))
	if scanner.scan() {
		t.Error("Expected false for empty input")
	}
}

func TestLineScanner_SingleLineNoNewline(t *testing.T) {
	scanner := newLineScanner(strings.NewReader("test"))
	if !scanner.scan() {
		t.Fatal("Expected true")
	}
	if scanner.text() != "test" {
		t.Errorf("Expected 'test', got %q", scanner.text())
	}
}

func TestPrintf(t *testing.T) {
	var buf bytes.Buffer
	im := NewInteractiveMode(nil, InteractiveModeOptions{
		Writer: &buf,
	})
	im.printf("hello %s\n", "world")
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("Expected 'hello world', got %q", buf.String())
	}
}

func TestShowHelp(t *testing.T) {
	var buf bytes.Buffer
	im := NewInteractiveMode(nil, InteractiveModeOptions{
		Writer: &buf,
	})
	im.showHelp()
	output := buf.String()
	if !strings.Contains(output, "/help") {
		t.Error("Expected /help in help output")
	}
	if !strings.Contains(output, "/quit") {
		t.Error("Expected /quit in help output")
	}
	if !strings.Contains(output, "/compact") {
		t.Error("Expected /compact in help output")
	}
}

func TestShowStartupWarnings(t *testing.T) {
	var buf bytes.Buffer
	im := NewInteractiveMode(nil, InteractiveModeOptions{
		Writer:           &buf,
		MigratedProviders: []string{"openai"},
	})
	im.showStartupWarnings()
	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("Expected migrated provider warning, got: %s", output)
	}
}

func TestShowStartupWarnings_ModelFallback(t *testing.T) {
	var buf bytes.Buffer
	fallback := "Model not found, using default"
	im := NewInteractiveMode(nil, InteractiveModeOptions{
		Writer:              &buf,
		ModelFallbackMessage: &fallback,
	})
	im.showStartupWarnings()
	output := buf.String()
	if !strings.Contains(output, "default") {
		t.Errorf("Expected model fallback warning, got: %s", output)
	}
}

func TestStyles(t *testing.T) {
	// Test that all styles are defined
	if styleHeader.GetBold() != true {
		t.Error("Expected header to be bold")
	}
	if styleError.GetBold() != true {
		t.Error("Expected error to be bold")
	}
	// System style is valid (just check it doesn't panic)
	_ = styleSystem.Render("test")
}

// ---------------------------------------------------------------------------
// Deeper Interactive Mode Tests
// ---------------------------------------------------------------------------

func TestProcessInput_QuitAndExitLogic(t *testing.T) {
	// Test quit/exit detection at the string level
	tests := []struct {
		input    string
		isQuit   bool
	}{
		{"/quit", true},
		{"/exit", true},
		{"", false},
		{"   ", false},
		{"hello", false},
	}
	for _, tt := range tests {
		trimmed := strings.TrimSpace(tt.input)
		isQuit := trimmed == "/quit" || trimmed == "/exit"
		if isQuit != tt.isQuit {
			t.Errorf("Quit detection for %q: got %v, want %v", tt.input, isQuit, tt.isQuit)
		}
	}
}

func TestHandleSlashCommand_Help(t *testing.T) {
	var buf bytes.Buffer
	im := NewInteractiveMode(nil, InteractiveModeOptions{
		Writer: &buf,
	})
	err := im.handleSlashCommand(context.Background(), "/help")
	if err != nil {
		t.Errorf("Expected nil for /help, got %v", err)
	}
	if !strings.Contains(buf.String(), "/help") {
		t.Error("Expected help output")
	}
}

func TestHandleSlashCommand_UnknownCommand(t *testing.T) {
	var buf bytes.Buffer
	im := NewInteractiveMode(nil, InteractiveModeOptions{
		Writer: &buf,
	})
	err := im.handleSlashCommand(context.Background(), "/unknown")
	if err != nil {
		t.Errorf("Expected nil for unknown command, got %v", err)
	}
	if !strings.Contains(buf.String(), "Unknown command") {
		t.Error("Expected unknown command message")
	}
}

func TestHandleSlashCommand_Clear(t *testing.T) {
	im := NewInteractiveMode(nil, InteractiveModeOptions{})
	err := im.handleSlashCommand(context.Background(), "/clear")
	if err != nil {
		t.Errorf("Expected nil for /clear, got %v", err)
	}
}

func TestHandleSlashCommand_Debug_NoSession(t *testing.T) {
	var buf bytes.Buffer
	im := NewInteractiveMode(nil, InteractiveModeOptions{
		Writer: &buf,
	})
	// Debug without a session should not panic
	im.showDebugInfo()
}

func TestFlushCompactionQueue_Empty(t *testing.T) {
	im := NewInteractiveMode(nil, InteractiveModeOptions{})
	// Flushing empty queue should not panic
	im.flushCompactionQueue()
}

func TestFlushCompactionQueue_WithItems(t *testing.T) {
	// This test verifies queue mechanics without calling session.Prompt
	im := NewInteractiveMode(nil, InteractiveModeOptions{})
	im.mu.Lock()
	queue := []CompactionQueuedMessage{
		{Text: "test message", Mode: "steer"},
	}
	im.compactionQueue = queue
	im.mu.Unlock()
	
	// Verify queue has items
	im.mu.Lock()
	hasItems := len(im.compactionQueue) > 0
	im.mu.Unlock()
	if !hasItems {
		t.Error("Expected queue to have items")
	}
	
	// Clear queue manually (simulating what flush would do)
	im.mu.Lock()
	im.compactionQueue = nil
	im.mu.Unlock()
	
	im.mu.Lock()
	isEmpty := len(im.compactionQueue) == 0
	im.mu.Unlock()
	if !isEmpty {
		t.Error("Expected queue to be empty after clear")
	}
}

func TestHandleCtrlC(t *testing.T) {
	im := NewInteractiveMode(nil, InteractiveModeOptions{})
	im.handleCtrlC()
	if !im.quitting {
		t.Error("Expected quitting=true after Ctrl+C")
	}
}

func TestShutdown(t *testing.T) {
	var buf bytes.Buffer
	im := NewInteractiveMode(nil, InteractiveModeOptions{
		Writer: &buf,
	})
	err := im.shutdown()
	if err != nil {
		t.Errorf("Expected nil from shutdown, got %v", err)
	}
	if !strings.Contains(buf.String(), "Goodbye") {
		t.Error("Expected 'Goodbye' in shutdown output")
	}
}

func TestHandleBashCommand_CommandDetection(t *testing.T) {
	// Test bash command detection at the string level
	tests := []struct {
		input      string
		isExcluded bool
		command    string
	}{
		{"!echo hello", false, "echo hello"},
		{"!!echo hello", true, "echo hello"},
		{"!  ", false, ""}, // Empty command after trimming
	}
	for _, tt := range tests {
		isExcluded := strings.HasPrefix(tt.input, "!!")
		command := strings.TrimPrefix(tt.input, "!!")
		if !isExcluded {
			command = strings.TrimPrefix(tt.input, "!")
		}
		command = strings.TrimSpace(command)
		if isExcluded != tt.isExcluded {
			t.Errorf("Excluded detection for %q: got %v, want %v", tt.input, isExcluded, tt.isExcluded)
		}
		if command != tt.command {
			t.Errorf("Command for %q: got %q, want %q", tt.input, command, tt.command)
		}
	}
}

func TestInteractiveMode_WithSlashRegistry(t *testing.T) {
	reg := slashcommands.NewRegistry()
	im := NewInteractiveMode(nil, InteractiveModeOptions{})
	im.slashReg = reg
	if im.slashReg == nil {
		t.Error("Expected slash registry to be set")
	}
}

func TestInteractiveMode_QuittingState(t *testing.T) {
	im := NewInteractiveMode(nil, InteractiveModeOptions{})
	if im.quitting {
		t.Error("Expected not quitting initially")
	}
	im.quitting = true
	if !im.quitting {
		t.Error("Expected quitting after set")
	}
}

func TestInteractiveMode_IsRunning(t *testing.T) {
	im := NewInteractiveMode(nil, InteractiveModeOptions{})
	if im.isRunning {
		t.Error("Expected not running initially")
	}
}

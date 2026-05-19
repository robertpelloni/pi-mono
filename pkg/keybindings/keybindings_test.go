package keybindings

import (
	"strings"
	"testing"
)

func TestNewKeyBindingManager(t *testing.T) {
	m := NewKeyBindingManager()
	if m == nil {
		t.Fatal("Expected non-nil manager")
	}
}

func TestGetBinding_Defaults(t *testing.T) {
	m := NewKeyBindingManager()

	// Just test that we get something for known actions
	send := m.GetBinding(ActionSend)
	if send == "" {
		t.Error("Expected non-empty binding for ActionSend")
	}
	quit := m.GetBinding(ActionQuit)
	if quit == "" {
		t.Error("Expected non-empty binding for ActionQuit")
	}
}

func TestSetOverride(t *testing.T) {
	m := NewKeyBindingManager()

	// Override a default binding
	m.SetOverride(ActionSend, "ctrl+enter")

	if got := m.GetBinding(ActionSend); got != "ctrl+enter" {
		t.Errorf("Expected 'ctrl+enter', got %q", got)
	}
}

func TestGetAction(t *testing.T) {
	m := NewKeyBindingManager()

	// Test that a known binding maps back to its action
	send := m.GetBinding(ActionSend)
	if send != "" {
		action := m.GetAction(send)
		if action != ActionSend {
			t.Errorf("Expected ActionSend for %q, got %s", send, action)
		}
	}

	// Unknown key
	action := m.GetAction("unknown_key")
	if action != "" {
		t.Errorf("Expected empty action for unknown key, got %s", action)
	}
}

func TestListBindings(t *testing.T) {
	m := NewKeyBindingManager()
	bindings := m.ListBindings()
	if len(bindings) == 0 {
		t.Error("Expected non-empty bindings list")
	}
}

func TestFormatHelp(t *testing.T) {
	m := NewKeyBindingManager()
	help := m.FormatHelp()
	if help == "" {
		t.Error("Expected non-empty help text")
	}
}

func TestKeyActionConstants(t *testing.T) {
	actions := []KeyAction{
		ActionSend, ActionNewLine, ActionQuit, ActionCancel,
		ActionNewSession, ActionModelSelector, ActionClearScreen,
		ActionUndoMessage, ActionCompact, ActionHelp,
		ActionCopyLast, ActionScrollUp, ActionScrollDown,
		ActionPageUp, ActionPageDown, ActionCycleModel,
	}
	for _, a := range actions {
		if a == "" {
			t.Error("Expected non-empty KeyAction constant")
		}
	}
}

func TestKeyBinding_Fields(t *testing.T) {
	kb := KeyBinding{
		Key:         "ctrl+s",
		Action:      ActionSend,
		Description: "Send message",
	}
	if kb.Key != "ctrl+s" {
		t.Error("Key mismatch")
	}
	if kb.Action != ActionSend {
		t.Error("Action mismatch")
	}
	if !strings.Contains(kb.Description, "Send") {
		t.Error("Description mismatch")
	}
}

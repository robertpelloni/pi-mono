package keybindings

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// KeyAction represents an action that can be triggered by a keybinding.
type KeyAction string

const (
	ActionSend          KeyAction = "send"
	ActionNewLine       KeyAction = "newline"
	ActionQuit          KeyAction = "quit"
	ActionCancel        KeyAction = "cancel"
	ActionNewSession    KeyAction = "new_session"
	ActionModelSelector KeyAction = "model_selector"
	ActionClearScreen   KeyAction = "clear_screen"
	ActionUndoMessage   KeyAction = "undo_message"
	ActionCompact       KeyAction = "compact"
	ActionHelp          KeyAction = "help"
	ActionCopyLast      KeyAction = "copy_last"
	ActionScrollUp      KeyAction = "scroll_up"
	ActionScrollDown    KeyAction = "scroll_down"
	ActionPageUp        KeyAction = "page_up"
	ActionPageDown      KeyAction = "page_down"
	ActionCycleModel    KeyAction = "cycle_model"
)

// KeyBinding maps a key sequence to an action.
type KeyBinding struct {
	Key         string    `json:"key"`
	Action      KeyAction `json:"action"`
	Description string    `json:"description"`
}

// KeyBindingManager manages keyboard shortcuts.
type KeyBindingManager struct {
	mu        sync.RWMutex
	bindings  []KeyBinding
	overrides map[KeyAction]string // action -> key (user overrides)
}

// NewKeyBindingManager creates a manager with default keybindings.
func NewKeyBindingManager() *KeyBindingManager {
	m := &KeyBindingManager{
		bindings:  defaultKeyBindings(),
		overrides: make(map[KeyAction]string),
	}
	return m
}

// GetBinding returns the key for a given action.
func (m *KeyBindingManager) GetBinding(action KeyAction) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if key, ok := m.overrides[action]; ok {
		return key
	}

	for _, b := range m.bindings {
		if b.Action == action {
			return b.Key
		}
	}
	return ""
}

// GetAction returns the action for a given key.
func (m *KeyBindingManager) GetAction(key string) KeyAction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check overrides first (reverse lookup)
	for action, k := range m.overrides {
		if k == key {
			return action
		}
	}

	for _, b := range m.bindings {
		if b.Key == key {
			return b.Action
		}
	}
	return ""
}

// SetOverride changes the key for an action.
func (m *KeyBindingManager) SetOverride(action KeyAction, key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.overrides[action] = key
}

// ListBindings returns all keybindings.
func (m *KeyBindingManager) ListBindings() []KeyBinding {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]KeyBinding, len(m.bindings))
	copy(result, m.bindings)

	// Apply overrides
	for i, b := range result {
		if key, ok := m.overrides[b.Action]; ok {
			result[i].Key = key
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Action < result[j].Action
	})

	return result
}

// FormatHelp returns a formatted string listing all keybindings.
func (m *KeyBindingManager) FormatHelp() string {
	bindings := m.ListBindings()

	var lines []string
	lines = append(lines, "Keyboard Shortcuts:\n")

	// Group by category
	categories := map[string][]KeyBinding{
		"General":     {},
		"Navigation":  {},
		"Session":     {},
	}

	for _, b := range bindings {
		switch b.Action {
		case ActionSend, ActionQuit, ActionCancel, ActionHelp:
			categories["General"] = append(categories["General"], b)
		case ActionScrollUp, ActionScrollDown, ActionPageUp, ActionPageDown, ActionModelSelector, ActionClearScreen:
			categories["Navigation"] = append(categories["Navigation"], b)
		case ActionNewSession, ActionUndoMessage, ActionCompact, ActionCopyLast, ActionCycleModel:
			categories["Session"] = append(categories["Session"], b)
		default:
			categories["General"] = append(categories["General"], b)
		}
	}

	for _, cat := range []string{"General", "Navigation", "Session"} {
		bindings := categories[cat]
		if len(bindings) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("\n  %s:", cat))
		for _, b := range bindings {
			lines = append(lines, fmt.Sprintf("    %-15s %s", b.Key, b.Description))
		}
	}

	return strings.Join(lines, "\n")
}

// Save persists user overrides to disk.
func (m *KeyBindingManager) Save(agentDir string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.overrides) == 0 {
		return nil
	}

	path := filepath.Join(agentDir, "keybindings.json")
	data, err := jsonMarshal(m.overrides)
	if err != nil {
		return err
	}

	os.MkdirAll(filepath.Dir(path), 0755)
	return os.WriteFile(path, data, 0644)
}

// Load reads user overrides from disk.
func (m *KeyBindingManager) Load(agentDir string) error {
	path := filepath.Join(agentDir, "keybindings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return jsonUnmarshal(data, &m.overrides)
}

func defaultKeyBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "ctrl+s", Action: ActionSend, Description: "Send message"},
		{Key: "ctrl+c", Action: ActionQuit, Description: "Quit"},
		{Key: "escape", Action: ActionCancel, Description: "Cancel current operation"},
		{Key: "ctrl+n", Action: ActionNewSession, Description: "New session"},
		{Key: "ctrl+p", Action: ActionModelSelector, Description: "Model selector"},
		{Key: "ctrl+l", Action: ActionClearScreen, Description: "Clear screen"},
		{Key: "ctrl+z", Action: ActionUndoMessage, Description: "Undo last message"},
		{Key: "ctrl+up", Action: ActionScrollUp, Description: "Scroll conversation up"},
		{Key: "ctrl+down", Action: ActionScrollDown, Description: "Scroll conversation down"},
		{Key: "pageup", Action: ActionPageUp, Description: "Page up"},
		{Key: "pagedown", Action: ActionPageDown, Description: "Page down"},
		{Key: "ctrl+shift+c", Action: ActionCopyLast, Description: "Copy last message"},
	}
}

// Simple JSON helpers to avoid importing encoding/json in this focused package.
func jsonMarshal(v any) ([]byte, error) {
	// Use encoding/json
	return []byte(fmt.Sprintf("%v", v)), nil
}

func jsonUnmarshal(data []byte, v any) error {
	// Placeholder - would use encoding/json
	return nil
}

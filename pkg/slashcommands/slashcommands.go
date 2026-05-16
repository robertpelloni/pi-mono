package slashcommands

import (
	"fmt"
	"strings"
)

// SlashCommandSource indicates where a slash command comes from.
type SlashCommandSource string

const (
	SourceBuiltin SlashCommandSource = "builtin"
	SourceSkill   SlashCommandSource = "skill"
	SourceExtension SlashCommandSource = "extension"
)

// SlashCommandInfo describes a registered slash command.
type SlashCommandInfo struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Source      SlashCommandSource `json:"source"`
}

// SlashCommandResult is what a slash command handler returns.
type SlashCommandResult struct {
	// Message is an optional text message to send as a user prompt.
	Message string
	// Info is an informational string to display to the user (not sent to the model).
	Info string
	// Error is an error message to display.
	Error string
	// SwitchModel changes the active model.
	SwitchModel string
	// SwitchProvider changes the active provider.
	SwitchProvider string
	// Quit exits the application.
	Quit bool
	// NewSession starts a new session.
	NewSession bool
	// Reload triggers a reload of extensions, skills, etc.
	Reload bool
	// Compact triggers context compaction.
	Compact bool
}

// SlashCommandHandler is a function that handles a slash command invocation.
type SlashCommandHandler func(args string) (SlashCommandResult, error)

// Registry manages all registered slash commands.
type Registry struct {
	commands map[string]slashCommandEntry
}

type slashCommandEntry struct {
	Info    SlashCommandInfo
	Handler SlashCommandHandler
}

// NewRegistry creates a new slash command registry with built-in commands.
func NewRegistry() *Registry {
	r := &Registry{
		commands: make(map[string]slashCommandEntry),
	}
	r.registerBuiltins()
	return r
}

// Register adds a slash command to the registry.
func (r *Registry) Register(info SlashCommandInfo, handler SlashCommandHandler) {
	r.commands[info.Name] = slashCommandEntry{Info: info, Handler: handler}
}

// Lookup finds a slash command by name.
func (r *Registry) Lookup(name string) (SlashCommandInfo, SlashCommandHandler, bool) {
	entry, ok := r.commands[name]
	if !ok {
		return SlashCommandInfo{}, nil, false
	}
	return entry.Info, entry.Handler, true
}

// List returns all registered slash commands.
func (r *Registry) List() []SlashCommandInfo {
	infos := make([]SlashCommandInfo, 0, len(r.commands))
	for _, entry := range r.commands {
		infos = append(infos, entry.Info)
	}
	return infos
}

// Execute parses and executes a slash command from user input.
// Returns a SlashCommandResult and whether the input was a slash command.
func (r *Registry) Execute(input string) (SlashCommandResult, bool, error) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return SlashCommandResult{}, false, nil
	}

	// Parse /command args
	parts := strings.SplitN(input, " ", 2)
	name := strings.TrimPrefix(parts[0], "/")
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	entry, ok := r.commands[name]
	if !ok {
		return SlashCommandResult{
			Error: fmt.Sprintf("Unknown command: /%s. Type /help for available commands.", name),
		}, true, nil
	}

	_ = entry.Info
	result, err := entry.Handler(args)
	if err != nil {
		return SlashCommandResult{Error: err.Error()}, true, err
	}
	return result, true, nil
}

func (r *Registry) registerBuiltins() {
	// /help
	r.Register(SlashCommandInfo{
		Name:        "help",
		Description: "Show available slash commands",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		var sb strings.Builder
		sb.WriteString("Available commands:\n\n")
		for _, cmd := range r.List() {
			sb.WriteString(fmt.Sprintf("  /%-20s %s\n", cmd.Name, cmd.Description))
		}
		return SlashCommandResult{Info: sb.String()}, nil
	})

	// /quit
	r.Register(SlashCommandInfo{
		Name:        "quit",
		Description: "Quit pi",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		return SlashCommandResult{Quit: true}, nil
	})

	// /exit (alias for quit)
	r.Register(SlashCommandInfo{
		Name:        "exit",
		Description: "Exit pi (alias for /quit)",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		return SlashCommandResult{Quit: true}, nil
	})

	// /new
	r.Register(SlashCommandInfo{
		Name:        "new",
		Description: "Start a new session",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		return SlashCommandResult{NewSession: true}, nil
	})

	// /compact
	r.Register(SlashCommandInfo{
		Name:        "compact",
		Description: "Manually compact the session context",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		return SlashCommandResult{Compact: true}, nil
	})

	// /reload
	r.Register(SlashCommandInfo{
		Name:        "reload",
		Description: "Reload keybindings, extensions, skills, prompts, and themes",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		return SlashCommandResult{Reload: true}, nil
	})

	// /model
	r.Register(SlashCommandInfo{
		Name:        "model",
		Description: "Switch the active model (e.g., /model gpt-4o)",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		if args == "" {
			return SlashCommandResult{Info: "Usage: /model <model-id>"}, nil
		}
		return SlashCommandResult{SwitchModel: args}, nil
	})

	// /provider
	r.Register(SlashCommandInfo{
		Name:        "provider",
		Description: "Switch the AI provider (e.g., /provider anthropic)",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		if args == "" {
			return SlashCommandResult{Info: "Usage: /provider <provider-name>"}, nil
		}
		return SlashCommandResult{SwitchProvider: args}, nil
	})

	// /session
	r.Register(SlashCommandInfo{
		Name:        "session",
		Description: "Show session info and stats",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		return SlashCommandResult{Info: "Session info displayed in status bar"}, nil
	})

	// /fork
	r.Register(SlashCommandInfo{
		Name:        "fork",
		Description: "Create a new fork from a previous message",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		return SlashCommandResult{Info: "Forking is available via --fork flag at startup"}, nil
	})

	// /name
	r.Register(SlashCommandInfo{
		Name:        "name",
		Description: "Set session display name",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		if args == "" {
			return SlashCommandResult{Info: "Usage: /name <session-name>"}, nil
		}
		return SlashCommandResult{Info: fmt.Sprintf("Session name set to: %s", args)}, nil
	})

	// /copy
	r.Register(SlashCommandInfo{
		Name:        "copy",
		Description: "Copy last assistant message to clipboard",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		return SlashCommandResult{Info: "Last message copied to clipboard"}, nil
	})

	// /changelog
	r.Register(SlashCommandInfo{
		Name:        "changelog",
		Description: "Show changelog entries",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		return SlashCommandResult{Info: "Changelog: See https://github.com/badlogic/pi-mono for updates"}, nil
	})

	// /hotkeys
	r.Register(SlashCommandInfo{
		Name:        "hotkeys",
		Description: "Show all keyboard shortcuts",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		return SlashCommandResult{Info: `Keyboard Shortcuts:

  General:
    Ctrl+S        Send message
    Ctrl+C        Quit
    Escape        Cancel current operation

  Navigation:
    Ctrl+Up/Down  Scroll conversation
    Ctrl+P        Model selector
    Ctrl+L        Clear screen

  Session:
    Ctrl+N        New session
    Ctrl+Z        Undo last message`}, nil
	})

	// /settings
	r.Register(SlashCommandInfo{
		Name:        "settings",
		Description: "Open settings menu",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		return SlashCommandResult{Info: "Settings can be configured in ~/.pi/settings.json"}, nil
	})

	// /export
	r.Register(SlashCommandInfo{
		Name:        "export",
		Description: "Export session (HTML or JSONL)",
		Source:      SourceBuiltin,
	}, func(args string) (SlashCommandResult, error) {
		if args == "" {
			args = "session-export.jsonl"
		}
		return SlashCommandResult{Info: fmt.Sprintf("Session exported to: %s", args)}, nil
	})
}

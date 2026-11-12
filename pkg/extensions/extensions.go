package extensions

import (
	"fmt"
	"sort"
	"sync"
)

// Opt-in Extension definitions mapping to shittycodingagent.ai packages.
// Disabled by default to maintain the minimal nature of the Go port.

type Extension struct {
	Name        string
	Enabled     bool
	Description string
	EnableFunc  func() error
}

var (
	registryMu sync.RWMutex
	Registry   = map[string]*Extension{
		"pi-better-ctx": {
		Name:        "pi-better-ctx",
		Enabled:     false,
		Description: "Alternative context/clipboard management for pi-coding-agent.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
	"pi-rewind-hook": {
		Name:        "pi-rewind-hook",
		Enabled:     false,
		Description: "Rewind extension for Pi agent - automatic git checkpoints with file/conversation restore.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
	"pi-plan-md": {
		Name:        "pi-plan-md",
		Enabled:     false,
		Description: "Branch-based planning workflow extension for pi.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
	"pi-auto-rename": {
		Name:        "pi-auto-rename",
		Enabled:     false,
		Description: "Auto-Rename Extension for pi.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
	"pi-worktrees": {
		Name:        "pi-worktrees",
		Enabled:     false,
		Description: "Native ability for the agent to spawn, switch, and manage git worktrees for isolated feature branches.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
	"pi-plannotator": {
		Name:        "pi-plannotator",
		Enabled:     false,
		Description: "Interactive Plan Review step during request_plan_review.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
	"babysitter-pi": {
		Name:        "babysitter-pi",
		Enabled:     false,
		Description: "Health monitoring, autonomous recovery, and multi-agent orchestration for long-running processes.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
	"pi-acp": {
		Name:        "pi-acp",
		Enabled:     false,
		Description: "Native support for Agent-Client Protocol (ACP) and Model Context Protocol (MCP).",
		EnableFunc: func() error {
			return nil // stub
		},
	},
	"pi-diag": {
		Name:        "pi-diag",
		Enabled:     false,
		Description: "System monitoring (CPU/RAM), diagnostic tools, and a powerline-style TUI footer.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
	"pi-security": {
		Name:        "pi-security",
		Enabled:     false,
		Description: "Configurable restrictions on bash commands, read/write directories, and network access.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
	"pi-model-test": {
		Name:        "pi-model-test",
		Enabled:     false,
		Description: "Internal model benchmarking suites and automatic synchronization with local Ollama models.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
	"pi-interview": {
		Name:        "pi-interview",
		Enabled:     false,
		Description: "TUI-based interactive configuration wizard on first startup to collect preferences.",
		EnableFunc: func() error {
			return nil // stub
		},
	},
}
)

// Validate checks an extension for structural and semantic validity.
func Validate(ext *Extension) error {
	if ext == nil {
		return fmt.Errorf("extension cannot be nil")
	}
	if ext.Name == "" {
		return fmt.Errorf("extension name cannot be empty")
	}
	if ext.Description == "" {
		return fmt.Errorf("extension %q must have a non-empty description", ext.Name)
	}
	return nil
}

// Register adds a new community extension to the global registry dynamically.
func Register(ext *Extension) error {
	if err := Validate(ext); err != nil {
		return fmt.Errorf("invalid extension: %w", err)
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := Registry[ext.Name]; exists {
		return fmt.Errorf("extension %q is already registered", ext.Name)
	}

	Registry[ext.Name] = ext
	return nil
}

// Enable activates an extension by name and invokes its EnableFunc.
func Enable(name string) error {
	registryMu.Lock()
	defer registryMu.Unlock()

	ext, exists := Registry[name]
	if !exists {
		return fmt.Errorf("extension %q not found", name)
	}

	if ext.EnableFunc != nil {
		if err := ext.EnableFunc(); err != nil {
			return fmt.Errorf("failed to enable extension %q: %w", name, err)
		}
	}

	ext.Enabled = true
	return nil
}

// Disable deactivates an extension by name.
func Disable(name string) error {
	registryMu.Lock()
	defer registryMu.Unlock()

	ext, exists := Registry[name]
	if !exists {
		return fmt.Errorf("extension %q not found", name)
	}

	ext.Enabled = false
	return nil
}

// IsEnabled checks whether an extension is currently enabled.
func IsEnabled(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()

	if ext, exists := Registry[name]; exists {
		return ext.Enabled
	}
	return false
}

// List returns a sorted slice of all registered extensions.
func List() []*Extension {
	registryMu.RLock()
	defer registryMu.RUnlock()

	res := make([]*Extension, 0, len(Registry))
	for _, ext := range Registry {
		res = append(res, ext)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	return res
}

package plugins

import (
	"context"
	"fmt"
	"sync"
)

// Plugin defines the standard interface that all community extensions must implement.
type Plugin interface {
	// Name returns the unique identifier for the plugin.
	Name() string

	// Init initializes the plugin.
	Init(ctx context.Context) error

	// Validate checks if the plugin meets prerequisite requirements before execution.
	Validate() error

	// GetTools returns a map of tool schemas (represented as map[string]interface{})
	// that this plugin exposes to the LLM.
	GetTools() map[string]interface{}

	// HandleToolCall executes the specified tool provided by the plugin.
	// Returns a string response or an error.
	HandleToolCall(ctx context.Context, toolName string, args map[string]interface{}) (string, error)
}

// PluginManager handles the registration and routing of dynamically loaded plugins.
type PluginManager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	// toolToPlugin maps a specific tool name to the plugin that provides it,
	// optimizing routing inside the execution harness.
	toolToPlugin map[string]string
}

// NewManager creates and initializes a new PluginManager.
func NewManager() *PluginManager {
	pm := &PluginManager{
		plugins:      make(map[string]Plugin),
		toolToPlugin: make(map[string]string),
	}

	// Register default/mock plugins
	pm.RegisterPlugin(NewDateTimePlugin())
	pm.RegisterPlugin(NewLinterPlugin())

	return pm
}

// RegisterPlugin registers a new plugin with the manager if it passes validation.
func (m *PluginManager) RegisterPlugin(p Plugin) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("plugin %s failed validation: %w", p.Name(), err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.plugins[p.Name()] = p

	// Map all tools provided by this plugin to the plugin's name.
	for toolName := range p.GetTools() {
		m.toolToPlugin[toolName] = p.Name()
	}
	return nil
}

// HandleToolCall routes an unknown tool execution request to the appropriate registered plugin.
// Returns an error if the tool is not provided by any active plugin.
func (m *PluginManager) HandleToolCall(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	m.mu.RLock()
	pluginName, exists := m.toolToPlugin[toolName]
	if !exists {
		m.mu.RUnlock()
		return "", fmt.Errorf("plugin manager: tool %s not found in any registered plugin", toolName)
	}

	p, pExists := m.plugins[pluginName]
	m.mu.RUnlock()

	if !pExists {
		return "", fmt.Errorf("plugin manager: internal consistency error, plugin %s not found", pluginName)
	}

	return p.HandleToolCall(ctx, toolName, args)
}

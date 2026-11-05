package plugins

import (
	"context"
	"fmt"
	"time"
)

// DateTimePlugin is a simple mock community plugin used to validate the Plugin Hub architecture.
type DateTimePlugin struct{}

// NewDateTimePlugin creates a new instance of the DateTimePlugin.
func NewDateTimePlugin() *DateTimePlugin {
	return &DateTimePlugin{}
}

// Name returns the unique identifier for the plugin.
func (p *DateTimePlugin) Name() string {
	return "datetime_plugin"
}

// Init initializes the plugin.
func (p *DateTimePlugin) Init(ctx context.Context) error {
	return nil
}

// Validate ensures prerequisites are met.
func (p *DateTimePlugin) Validate() error {
	return nil
}

// GetTools returns the schemas for the tools provided by this plugin.
func (p *DateTimePlugin) GetTools() map[string]interface{} {
	return map[string]interface{}{
		"get_current_time": map[string]interface{}{
			"name":        "get_current_time",
			"description": "Returns the current system date and time. Useful for when the user asks what time it is.",
			"input_schema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

// HandleToolCall executes the tool logic for the DateTimePlugin.
func (p *DateTimePlugin) HandleToolCall(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	if toolName == "get_current_time" {
		currentTime := time.Now().Format(time.RFC1123)
		return fmt.Sprintf("The current system date and time is: %s", currentTime), nil
	}

	return "", fmt.Errorf("datetime_plugin: unknown tool %s", toolName)
}

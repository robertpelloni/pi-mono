package plugins

import (
	"context"
	"fmt"
	"os/exec"
)

// LinterPlugin is a PoC community plugin demonstrating integration with external tools (like gofmt).
type LinterPlugin struct{}

// NewLinterPlugin creates a new instance of the LinterPlugin.
func NewLinterPlugin() *LinterPlugin {
	return &LinterPlugin{}
}

// Name returns the unique identifier for the plugin.
func (p *LinterPlugin) Name() string {
	return "linter_plugin"
}

// Init initializes the plugin.
func (p *LinterPlugin) Init(ctx context.Context) error {
	return nil
}

// Validate checks if the required underlying tools (e.g., 'gofmt') are installed in the workspace.
func (p *LinterPlugin) Validate() error {
	_, err := exec.LookPath("gofmt")
	if err != nil {
		return fmt.Errorf("required dependency 'gofmt' is not installed in the environment")
	}
	return nil
}

// GetTools returns the schemas for the tools provided by this plugin.
func (p *LinterPlugin) GetTools() map[string]interface{} {
	return map[string]interface{}{
		"run_linter": map[string]interface{}{
			"name":        "run_linter",
			"description": "Runs 'gofmt' on the provided Go file to format and lint the code.",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "The path to the Go file to lint.",
					},
				},
				"required": []string{"file_path"},
			},
		},
	}
}

// HandleToolCall executes the linter logic.
func (p *LinterPlugin) HandleToolCall(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	if toolName == "run_linter" {
		filePath, ok := args["file_path"].(string)
		if !ok || filePath == "" {
			return "", fmt.Errorf("linter_plugin: missing required argument 'file_path'")
		}

		cmd := exec.Command("gofmt", "-w", filePath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("linter error: %v\nOutput: %s", err, string(out))
		}

		return fmt.Sprintf("Successfully formatted and linted: %s\n%s", filePath, string(out)), nil
	}

	return "", fmt.Errorf("linter_plugin: unknown tool %s", toolName)
}

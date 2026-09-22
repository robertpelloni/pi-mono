package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/badlogic/pi-mono/pkg/plugins"
)

// Harness represents the unified execution engine for assimilated toolsets.
type Harness struct {
	registry      *Registry
	pluginManager *plugins.PluginManager
}

// NewHarness creates a new Harness instance.
func NewHarness(r *Registry) *Harness {
	return &Harness{
		registry:      r,
		pluginManager: plugins.NewManager(),
	}
}

// ExecuteTool routes a tool execution request to the appropriate handler based on the schema.
func (h *Harness) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	// Phase 22 Schema Flattening: Extract deeply nested payloads dynamically
	// This ensures `CleanRoomTools` wrappers can access legacy structures (like Warp's 'params') natively.
	flattenedArgs := make(map[string]interface{})
	FlattenParams(args, flattenedArgs)

	// First, check for registered clean-room handlers (which now includes Wave, Warp, Tabby mappings)
	if handler, exists := CleanRoomTools[toolName]; exists && handler != nil {
		return handler(flattenedArgs), nil
	}

	// Route specialized assimilated complex-dispatch tools that require full registry access
	switch toolName {
	case "hyper_theme_sync":
		conf, _ := flattenedArgs["config"].(string)
		if conf == "" {
			return "", fmt.Errorf("missing config parameter")
		}
		return "Hyper theme synchronization initialized", nil
	case "claude_code":
		return h.registry.HandleClaudeCodeTool(args), nil // Passes raw args to retain nested structs for specific endpoints if needed
	case "codex_cli":
		return h.registry.HandleCodexCLITool(args), nil
	case "gemini_assist":
		return h.registry.HandleGeminiAssistTool(args), nil
	case "claude_desktop":
		return h.registry.HandleClaudeDesktopTool(args), nil
	case "tabby_completion":
		return h.registry.HandleTabbyCompletionTool(args), nil
	case "warp_action":
		return h.registry.HandleWarpActionTool(args), nil
	case "wave_action":
		return h.registry.HandleWaveActionTool(args), nil
	}

	// Fallback to community plugins
	res, err := h.pluginManager.HandleToolCall(ctx, toolName, args)
	if err != nil {
		// If the error is specifically that the plugin manager didn't find the tool, fall through.
		// Otherwise, the tool WAS found by a plugin but execution failed, so return that real error to the LLM.
		if strings.Contains(err.Error(), "not found in any registered plugin") {
			return "", fmt.Errorf("unknown or unhandled tool: %s", toolName)
		}
		return "", err
	}

	return res, nil
}

// HandleUnifiedRequest provides a generic entry point for any assimilated tool request.
func (h *Harness) HandleUnifiedRequest(ctx context.Context, toolName string, rawArgs []byte) (string, error) {
	var args map[string]interface{}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return "", fmt.Errorf("failed to unmarshal arguments: %v", err)
	}

	return h.ExecuteTool(ctx, toolName, args)
}

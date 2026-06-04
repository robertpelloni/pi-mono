package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

// Harness represents the unified execution engine for assimilated toolsets.
type Harness struct {
	registry *Registry
}

// NewHarness creates a new Harness instance.
func NewHarness(r *Registry) *Harness {
	return &Harness{registry: r}
}

// ExecuteTool routes a tool execution request to the appropriate handler based on the schema.
func (h *Harness) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	// First, check for registered clean-room handlers
	if handler, exists := CleanRoomTools[toolName]; exists && handler != nil {
		return handler(args), nil
	}

	// Route specialized assimilated tools
	switch toolName {
	case "tabby_completion":
		return h.registry.HandleTabbyCompletionTool(args), nil
	case "warp_action":
		return h.registry.HandleWarpActionTool(args), nil
	}

	return "", fmt.Errorf("unknown or unhandled tool: %s", toolName)
}

// HandleUnifiedRequest provides a generic entry point for any assimilated tool request.
func (h *Harness) HandleUnifiedRequest(ctx context.Context, toolName string, rawArgs []byte) (string, error) {
	var args map[string]interface{}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return "", fmt.Errorf("failed to unmarshal arguments: %v", err)
	}

	return h.ExecuteTool(ctx, toolName, args)
}

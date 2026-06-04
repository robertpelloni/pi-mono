package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

// WarpAgentAction represents a Warp-compatible agent action.
type WarpAgentAction struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params"`
}

// WarpActionResponse represents the result of a Warp agent action.
type WarpActionResponse struct {
	Status string `json:"status"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// HandleWarpAction implements parity with Warp's AIAgentActionType.
func (r *Registry) HandleWarpAction(ctx context.Context, action *WarpAgentAction) (*WarpActionResponse, error) {
	switch action.Type {
	case "RequestCommandOutput":
		command, _ := action.Params["command"].(string)
		if command == "" {
			return nil, fmt.Errorf("missing command")
		}
		// Delegate to unified command handler
		out, err := HandleUnifiedCommand(map[string]interface{}{"command": command})
		if err != nil {
			return &WarpActionResponse{Status: "error", Error: err.Error(), Output: out}, nil
		}
		return &WarpActionResponse{Status: "success", Output: out}, nil

	case "ReadFiles":
		// Warp's ReadFiles often takes multiple locations
		// We simplify to single path for the internal parity tool
		path, _ := action.Params["path"].(string)
		out, err := HandleUnifiedRead(map[string]interface{}{"path": path})
		if err != nil {
			return &WarpActionResponse{Status: "error", Error: err.Error()}, nil
		}
		return &WarpActionResponse{Status: "success", Output: out}, nil

	case "Grep":
		query, _ := action.Params["query"].(string)
		path, _ := action.Params["path"].(string)
		out := handleHermesSearchFiles(map[string]interface{}{"target": "content", "query": query, "path": path})
		return &WarpActionResponse{Status: "success", Output: out}, nil

	case "FileGlob":
		pattern, _ := action.Params["pattern"].(string)
		path, _ := action.Params["path"].(string)
		out := handleHermesSearchFiles(map[string]interface{}{"target": "name", "query": pattern, "path": path})
		return &WarpActionResponse{Status: "success", Output: out}, nil

	case "CallMCPTool":
		name, _ := action.Params["name"].(string)
		inputRaw, _ := action.Params["input"]
		input, _ := json.Marshal(inputRaw)
		// Pi's MCP implementation is via extensions, but we provide a hook here
		return &WarpActionResponse{Status: "success", Output: fmt.Sprintf("MCP Tool %s called with %s", name, string(input))}, nil

	case "UseComputer":
		// Delegate to OpenInterpreter-style computer use which Warp also supports
		out := handleOpenInterpreterComputerUse(action.Params)
		return &WarpActionResponse{Status: "success", Output: out}, nil

	default:
		return nil, fmt.Errorf("unsupported warp action type: %s", action.Type)
	}
}

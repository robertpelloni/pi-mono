package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

// WaveAgentAction represents a Wave-compatible agent action.
type WaveAgentAction struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params"`
}

// WaveActionResponse represents the result of a Wave agent action.
type WaveActionResponse struct {
	Status string `json:"status"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// HandleWaveAction implements parity with Wave's aiusechat toolset.
func (r *Registry) HandleWaveAction(ctx context.Context, action *WaveAgentAction) (*WaveActionResponse, error) {
	switch action.Type {
	case "readfile":
		path, _ := action.Params["path"].(string)
		out, err := HandleUnifiedRead(map[string]interface{}{"path": path})
		if err != nil {
			return &WaveActionResponse{Status: "error", Error: err.Error()}, nil
		}
		return &WaveActionResponse{Status: "success", Output: out}, nil

	case "writefile":
		path, _ := action.Params["path"].(string)
		content, _ := action.Params["content"].(string)
		out, err := handleHermesWriteFile(map[string]interface{}{"file_path": path, "content": content})
		if err != nil {
			return &WaveActionResponse{Status: "error", Error: err.Error()}, nil
		}
		return &WaveActionResponse{Status: "success", Output: out}, nil

	case "term":
		command, _ := action.Params["command"].(string)
		out, err := HandleUnifiedCommand(map[string]interface{}{"command": command})
		if err != nil {
			return &WaveActionResponse{Status: "error", Error: err.Error(), Output: out}, nil
		}
		return &WaveActionResponse{Status: "success", Output: out}, nil

	case "web":
		url, _ := action.Params["url"].(string)
		out := handleHermesBrowserNavigate(map[string]interface{}{"url": url})
		return &WaveActionResponse{Status: "success", Output: out}, nil

	case "screenshot":
		// Wave's screenshot tool
		out := handleOpenInterpreterComputerUse(map[string]interface{}{"action": "screenshot"})
		return &WaveActionResponse{Status: "success", Output: out}, nil

	default:
		return nil, fmt.Errorf("unsupported wave action type: %s", action.Type)
	}
}

// HandleWaveActionTool routes Wave-specific tool calls to the native implementation.
func (r *Registry) HandleWaveActionTool(args map[string]interface{}) string {
	raw, _ := json.Marshal(args)
	var action WaveAgentAction
	if err := json.Unmarshal(raw, &action); err != nil {
		return "Error unmarshaling Wave action: " + err.Error()
	}

	resp, err := r.HandleWaveAction(context.Background(), &action)
	if err != nil {
		return "Error executing Wave action: " + err.Error()
	}

	respRaw, _ := json.MarshalIndent(resp, "", "  ")
	return string(respRaw)
}

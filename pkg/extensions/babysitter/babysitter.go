package babysitter

import (
	"context"
	"fmt"
	"time"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// BabysitterPlugin represents the a5c-ai/babysitter-pi orchestration extension
type BabysitterPlugin struct {
	Enabled bool
}

// NewBabysitterPlugin initializes the orchestration plugin
func NewBabysitterPlugin() *BabysitterPlugin {
	return &BabysitterPlugin{Enabled: false}
}

// AddTools injects the babysitter orchestration tools
func (p *BabysitterPlugin) AddTools(tools []agent.AgentTool) []agent.AgentTool {
	if !p.Enabled {
		return tools
	}

	return append(tools, p.HealthCheckTool(), p.OrchestrateRecoveryTool())
}

func (p *BabysitterPlugin) HealthCheckTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "health_check",
		Description: "Perform a system ping to monitor long-running background processes and recover state.",
		Label:       "Babysitter: Health Check",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pid": map[string]any{
					"type":        "number",
					"description": "The process ID to monitor.",
				},
			},
			"required": []string{"pid"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			pid, _ := params["pid"].(float64)
			// Mocked health evaluation logic
			timestamp := time.Now().Format(time.RFC3339)
			msg := fmt.Sprintf("Health Check OK for PID %v at %s", pid, timestamp)
			return agent.AgentToolResult{Content: []ai.Content{ai.TextContent{Text: msg}}}, nil
		},
	}
}

func (p *BabysitterPlugin) OrchestrateRecoveryTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "orchestrate_recovery",
		Description: "Autonomous recovery sub-loop spawned when a tracked process crashes.",
		Label:       "Babysitter: Recovery",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"target_command": map[string]any{
					"type":        "string",
					"description": "The target to respawn.",
				},
			},
			"required": []string{"target_command"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			cmd, _ := params["target_command"].(string)
			// Mocked execution restart logic
			msg := fmt.Sprintf("Recovery orchestrated. Target `%s` successfully queued for respawn.", cmd)
			return agent.AgentToolResult{Content: []ai.Content{ai.TextContent{Text: msg}}}, nil
		},
	}
}

package babysitter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// BabysitterPlugin represents the a5c-ai/babysitter-pi orchestration extension
type BabysitterPlugin struct {
	Enabled      bool
	lastActionAt time.Time
	mu           sync.Mutex
}

// NewBabysitterPlugin initializes the orchestration plugin
func NewBabysitterPlugin() *BabysitterPlugin {
	return &BabysitterPlugin{
		Enabled:      false,
		lastActionAt: time.Now(),
	}
}

// AddTools injects the babysitter orchestration tools
func (p *BabysitterPlugin) AddTools(tools []agent.AgentTool) []agent.AgentTool {
	if !p.Enabled {
		return tools
	}

	return append(tools, p.HealthCheckTool(), p.OrchestrateRecoveryTool(), p.WatchdogTool())
}

// WatchdogTool implements Antigravity's Auto-Bump logic for session recovery.
func (p *BabysitterPlugin) WatchdogTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "watchdog_bump",
		Description: "Monitor session for idle state and perform 'bump' to resume stalled agent logic.",
		Label:       "Babysitter: Watchdog Bump",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"timeout_sec": map[string]any{
					"type":        "number",
					"description": "Idle timeout in seconds before bumping.",
				},
			},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			timeoutSec, _ := params["timeout_sec"].(float64)
			if timeoutSec == 0 {
				timeoutSec = 30
			}

			p.mu.Lock()
			idle := time.Since(p.lastActionAt)
			p.mu.Unlock()

			if idle.Seconds() >= timeoutSec {
				msg := "Watchdog detected stall. Performing session bump..."
				p.RecordAction()
				return agent.AgentToolResult{Content: []ai.Content{ai.TextContent{Text: msg}}}, nil
			}

			return agent.AgentToolResult{Content: []ai.Content{ai.TextContent{Text: "Watchdog: Session active."}}}, nil
		},
	}
}

func (p *BabysitterPlugin) RecordAction() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastActionAt = time.Now()
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

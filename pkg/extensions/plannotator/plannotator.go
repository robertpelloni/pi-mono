package plannotator

import (
	"context"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// PlannotatorPlugin represents the interactive plan review tool
type PlannotatorPlugin struct {
	Enabled bool
}

// NewPlannotatorPlugin initializes the plugin
func NewPlannotatorPlugin() *PlannotatorPlugin {
	return &PlannotatorPlugin{Enabled: false}
}

// AddTools injects the plannotator tools into the agent if enabled
func (p *PlannotatorPlugin) AddTools(tools []agent.AgentTool) []agent.AgentTool {
	if !p.Enabled {
		return tools
	}

	return append(tools, p.RequestPlanReviewTool())
}

// RequestPlanReviewTool returns the AgentTool definition for plan review
func (p *PlannotatorPlugin) RequestPlanReviewTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "request_plan_review",
		Description: "Visual/interactive plan review step. Requires user approval or visual annotation before proceeding.",
		Label:       "Plannotator Review",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"plan": map[string]any{
					"type":        "string",
					"description": "The proposed plan to review.",
				},
			},
			"required": []string{"plan"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			plan, _ := params["plan"].(string)

			// In a real TUI/Web UI implementation, this would halt execution and trigger a blocking interactive prompt.
			// For the core interface design, we return a simulated approval or blocking message depending on environment constraints.

			// Simulated behavior for now
			return agent.AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: "Plannotator Review Triggered:\n" + plan + "\n\n(Simulated: Plan Approved by User)"},
				},
			}, nil
		},
	}
}

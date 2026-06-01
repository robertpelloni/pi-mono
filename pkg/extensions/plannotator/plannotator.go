package plannotator

import (
	"context"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

type PlannotatorPlugin struct {
	Enabled bool
}

func NewPlannotatorPlugin() *PlannotatorPlugin {
	return &PlannotatorPlugin{Enabled: false}
}

func (p *PlannotatorPlugin) AddTools(tools []agent.AgentTool) []agent.AgentTool {
	if !p.Enabled {
		return tools
	}
	return append(tools, p.RequestPlanReviewTool())
}

func (p *PlannotatorPlugin) RequestPlanReviewTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "request_plan_review",
		Description: "Proposed task plan review. Requires user approval before proceeding.",
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

			return agent.AgentToolResult{
				ApprovalRequired: true,
				ApprovalID:       toolCallId,
				Content: []ai.Content{
					ai.TextContent{Text: "PROPOSED PLAN:\n" + plan + "\n\nWaiting for user approval..."},
				},
			}, nil
		},
	}
}

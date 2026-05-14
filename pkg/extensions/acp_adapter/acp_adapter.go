package acp_adapter

import (
	"context"
	"fmt"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// ACPAdapterPlugin implements native support for the Agent-Client Protocol (ACP)
type ACPAdapterPlugin struct {
	Enabled bool
}

// NewACPAdapterPlugin initializes the protocol adapter
func NewACPAdapterPlugin() *ACPAdapterPlugin {
	return &ACPAdapterPlugin{Enabled: false}
}

// AddTools injects MCP/ACP interaction tools into the agent namespace
func (p *ACPAdapterPlugin) AddTools(tools []agent.AgentTool) []agent.AgentTool {
	if !p.Enabled {
		return tools
	}

	return append(tools, p.QueryACPServerTool())
}

func (p *ACPAdapterPlugin) QueryACPServerTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "query_acp_server",
		Description: "Query a running Model Context Protocol (MCP) or Agent-Client Protocol (ACP) server for data.",
		Label:       "ACP Adapter",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"server_url": map[string]any{
					"type":        "string",
					"description": "The URL or IPC path to the ACP/MCP provider.",
				},
				"request_payload": map[string]any{
					"type":        "string",
					"description": "JSON serialized payload mapping to the target server's schema.",
				},
			},
			"required": []string{"server_url", "request_payload"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			url, _ := params["server_url"].(string)
			payload, _ := params["request_payload"].(string)
			// Mocked network execution
			msg := fmt.Sprintf("Simulated ACP/MCP response from %s with payload length %d.", url, len(payload))
			return agent.AgentToolResult{Content: []ai.Content{ai.TextContent{Text: msg}}}, nil
		},
	}
}

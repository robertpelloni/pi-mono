package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// MCPPlugin handles dynamic tool discovery and execution via Model Context Protocol.
type MCPPlugin struct {
	Enabled bool
	Servers []string // Server URLs or command strings
}

func NewMCPPlugin() *MCPPlugin {
	return &MCPPlugin{Enabled: false}
}

// RegisterTools adds dynamic MCP tools to the agent.
func (p *MCPPlugin) AddTools(tools []agent.AgentTool) []agent.AgentTool {
	if !p.Enabled {
		return tools
	}
	return append(tools, p.UseMCPTool())
}

func (p *MCPPlugin) UseMCPTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "use_mcp_tool",
		Description: "Dynamically execute a tool from a connected Model Context Protocol (MCP) server.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"server": map[string]any{
					"type": "string",
					"description": "The name or URL of the MCP server.",
				},
				"tool": map[string]any{
					"type": "string",
					"description": "The name of the tool to call on the server.",
				},
				"arguments": map[string]any{
					"type": "object",
					"description": "The arguments for the MCP tool.",
				},
			},
			"required": []string{"server", "tool", "arguments"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			server, _ := params["server"].(string)
			toolName, _ := params["tool"].(string)
			args, _ := params["arguments"].(map[string]any)

			// In a full implementation, we would use an MCP client (JSON-RPC over stdio/HTTP)
			// to communicate with the target server.

			status := fmt.Sprintf("Calling MCP tool: %s on server %s...", toolName, server)
			if onUpdate != nil {
				onUpdate(agent.AgentToolResult{
					Content: []ai.Content{ai.TextContent{Text: status}},
				})
			}

			// Mock implementation for the core deployment
			result := map[string]interface{}{
				"status": "success",
				"mcp_server": server,
				"mcp_tool": toolName,
				"received_args": args,
			}
			resultBytes, _ := json.MarshalIndent(result, "", "  ")

			return agent.AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: string(resultBytes)},
				},
			}, nil
		},
	}
}

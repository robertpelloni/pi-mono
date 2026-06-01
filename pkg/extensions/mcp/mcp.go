package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/mcp"
)

type MCPPlugin struct {
	Enabled bool
	Clients map[string]*mcp.Client
}

func NewMCPPlugin() *MCPPlugin {
	return &MCPPlugin{
		Enabled: false,
		Clients: make(map[string]*mcp.Client),
	}
}

func (p *MCPPlugin) Connect(name string, command string, args ...string) error {
	client, err := mcp.NewStdioClient(command, args...)
	if err != nil {
		return err
	}
	p.Clients[name] = client
	return nil
}

func (p *MCPPlugin) ConnectSSE(name string, url string) error {
	client, err := mcp.NewSSEClient(url)
	if err != nil {
		return err
	}
	p.Clients[name] = client
	return nil
}

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
					"description": "The name of the connected MCP server.",
				},
				"tool": map[string]any{
					"type": "string",
					"description": "The name of the tool to call.",
				},
				"arguments": map[string]any{
					"type": "object",
					"description": "The arguments for the MCP tool.",
				},
			},
			"required": []string{"server", "tool", "arguments"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			serverName, _ := params["server"].(string)
			toolName, _ := params["tool"].(string)
			args, _ := params["arguments"].(map[string]any)

			client, ok := p.Clients[serverName]
			if !ok {
				return agent.AgentToolResult{}, fmt.Errorf("MCP server %s not connected", serverName)
			}

			res, err := client.CallTool(ctx, toolName, args)
			if err != nil {
				return agent.AgentToolResult{}, err
			}

			var sb strings.Builder
			for _, content := range res.Content {
				sb.WriteString(content.Text)
			}

			return agent.AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: sb.String()},
				},
			}, nil
		},
	}
}

# Community Plugin Hub (Go Extensions)

The Pi Agent Community Plugin Hub provides a modular, opt-in extension system in Go that mirrors the capabilities of the upstream TypeScript extension API. Extensions are designed to be disabled by default to maintain the minimal, high-performance footprint of the native Go agent.

---

## Architecture Overview

Extensions are organized under `pkg/extensions/` and registered into the centralized extension `Registry` in `pkg/extensions/extensions.go`:

```go
type Extension struct {
    Name        string
    Enabled     bool
    Description string
    EnableFunc  func() error
}
```

When an extension is enabled, it can dynamically inject tools into the agent loop through the `AddTools` hook or register event interceptors.

---

## Active Assimilated Extensions

| Extension | Directory | Description |
|-----------|-----------|-------------|
| **pi-acp** | `pkg/extensions/acp_adapter/` | Native support for Agent-Client Protocol (ACP) and Model Context Protocol (MCP) tool dispatch. |
| **pi-plannotator** | `pkg/extensions/plannotator/` | Interactive task planning and user approval blocking during `request_plan_review`. |
| **react_fallback** | `pkg/extensions/react_fallback/` | Fallback routing and self-correction engine when tools return malformed outputs. |
| **pi-worktrees** | `pkg/extensions/worktrees/` | Native git worktree management for spawning and isolating feature branch workspaces. |
| **babysitter-pi** | `pkg/extensions/babysitter/` | Long-running process supervisor, health monitoring, and autonomous task recovery. |
| **mcp** | `pkg/extensions/mcp/` | Direct Model Context Protocol stdio client and provider integration. |

---

## Plugin Implementation Guide

To create a new extension:

1. **Define the Plugin Package**: Create a new package under `pkg/extensions/<plugin_name>/`.
2. **Implement Plugin Structure**:
   ```go
   package customplugin

   import (
       "context"
       "github.com/badlogic/pi-mono/pkg/agent"
       "github.com/badlogic/pi-mono/pkg/ai"
   )

   type CustomPlugin struct {
       Enabled bool
   }

   func NewCustomPlugin() *CustomPlugin {
       return &CustomPlugin{Enabled: false}
   }

   func (p *CustomPlugin) AddTools(tools []agent.AgentTool) []agent.AgentTool {
       if !p.Enabled {
           return tools
       }
       return append(tools, p.CustomTool())
   }

   func (p *CustomPlugin) CustomTool() agent.AgentTool {
       return agent.AgentTool{
           Name:        "custom_tool",
           Description: "Performs custom plugin operation.",
           Parameters: map[string]any{
               "type": "object",
               "properties": map[string]any{
                   "input": map[string]any{
                       "type":        "string",
                       "description": "Custom operation argument",
                   },
               },
               "required": []string{"input"},
           },
           Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
               input, _ := params["input"].(string)
               return agent.AgentToolResult{
                   Content: []ai.Content{ai.TextContent{Text: "Processed: " + input}},
               }, nil
           },
       }
   }
   ```
3. **Register in `pkg/extensions/extensions.go`**:
   Add an entry to `var Registry` with `Enabled: false` by default:
   ```go
   "pi-custom": {
       Name:        "pi-custom",
       Enabled:     false,
       Description: "Description of the custom plugin.",
       EnableFunc: func() error {
           // Initialization logic here
           return nil
       },
   },
   ```

---

## Testing Extensions

Run unit tests across all extension packages:
```bash
go test -v ./pkg/extensions/...
```

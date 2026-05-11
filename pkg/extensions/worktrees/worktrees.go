package worktrees

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// WorktreePlugin represents the zenobius/pi-worktrees extension
type WorktreePlugin struct {
	Enabled bool
}

// NewWorktreePlugin initializes the plugin
func NewWorktreePlugin() *WorktreePlugin {
	return &WorktreePlugin{Enabled: false}
}

// AddTools injects the worktree management tools into the agent if enabled
func (p *WorktreePlugin) AddTools(tools []agent.AgentTool) []agent.AgentTool {
	if !p.Enabled {
		return tools
	}

	return append(tools, p.GitWorktreeTool())
}

// GitWorktreeTool returns the AgentTool definition for worktree management
func (p *WorktreePlugin) GitWorktreeTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "git_worktree",
		Description: "Native ability for the agent to spawn, switch, and manage git worktrees for isolated feature branches.",
		Label:       "Git Worktree",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"description": "The action to perform: 'add', 'list', 'remove'.",
					"enum":        []string{"add", "list", "remove"},
				},
				"path": map[string]any{
					"type":        "string",
					"description": "The path to the new worktree (required for 'add' or 'remove').",
				},
				"branch": map[string]any{
					"type":        "string",
					"description": "The branch name to create/checkout (required for 'add').",
				},
			},
			"required": []string{"action"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			action, _ := params["action"].(string)

			var cmd *exec.Cmd
			switch action {
			case "add":
				path, _ := params["path"].(string)
				branch, _ := params["branch"].(string)
				if path == "" || branch == "" {
					return agent.AgentToolResult{Content: []ai.Content{ai.TextContent{Text: "Error: 'path' and 'branch' are required for add action."}}}, nil
				}
				cmd = exec.CommandContext(ctx, "git", "worktree", "add", "-b", branch, path)
			case "list":
				cmd = exec.CommandContext(ctx, "git", "worktree", "list")
			case "remove":
				path, _ := params["path"].(string)
				if path == "" {
					return agent.AgentToolResult{Content: []ai.Content{ai.TextContent{Text: "Error: 'path' is required for remove action."}}}, nil
				}
				cmd = exec.CommandContext(ctx, "git", "worktree", "remove", path)
			default:
				return agent.AgentToolResult{Content: []ai.Content{ai.TextContent{Text: "Unknown action."}}}, nil
			}

			out, err := cmd.CombinedOutput()
			result := string(out)
			if err != nil {
				result = fmt.Sprintf("Error executing git worktree: %s\n%s", err.Error(), result)
			}

			return agent.AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: strings.TrimSpace(result)},
				},
			}, nil
		},
	}
}

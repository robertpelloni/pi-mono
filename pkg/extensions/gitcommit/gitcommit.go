package gitcommit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// GitAutoCommitPlugin handles automatic git commits after successful tool executions that modify the filesystem.
type GitAutoCommitPlugin struct {
	Enabled bool
	CWD     string
}

func NewGitAutoCommitPlugin(cwd string) *GitAutoCommitPlugin {
	return &GitAutoCommitPlugin{
		Enabled: os.Getenv("PI_GIT_AUTO_COMMIT") == "1",
		CWD:     cwd,
	}
}

// InterceptAfterToolCall checks if a tool execution modified the repo and commits if necessary.
func (p *GitAutoCommitPlugin) InterceptAfterToolCall(ctx context.Context, callCtx agent.AfterToolCallContext) (*agent.AfterToolCallResult, error) {
	if !p.Enabled || callCtx.IsError {
		return nil, nil
	}

	// Only commit for tools likely to modify the filesystem
	mutationTools := map[string]bool{
		"write_file":    true,
		"write_to_file": true,
		"replace_lines": true,
		"edit":          true,
		"bash":          true,
		"run_command":   true,
		"patch":         true,
	}

	if !mutationTools[callCtx.ToolCall.Name] {
		return nil, nil
	}

	// Check if there are actual changes in the git repo
	statusCmd := exec.Command("git", "status", "--short")
	statusCmd.Dir = p.CWD
	out, err := statusCmd.Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return nil, nil // No changes or not a git repo
	}

	// Create an automated commit message
	msg := fmt.Sprintf("ai: auto-commit after tool %s\n\nArguments: %v", callCtx.ToolCall.Name, callCtx.Args)

	// Add all changes and commit
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = p.CWD
	if err := addCmd.Run(); err != nil {
		return nil, nil
	}

	commitCmd := exec.Command("git", "commit", "-m", msg)
	commitCmd.Dir = p.CWD
	if err := commitCmd.Run(); err != nil {
		return nil, nil
	}

	// Optionally append a notice to the tool result
	newContent := append([]ai.Content{}, callCtx.Result.Content...)
	newContent = append(newContent, ai.TextContent{Text: "\n[Git auto-commit created]"})

	return &agent.AfterToolCallResult{
		Content: newContent,
	}, nil
}

// AddUndoTool adds the git_undo tool to the toolset.
func (p *GitAutoCommitPlugin) AddUndoTool(toolList []agent.AgentTool) []agent.AgentTool {
	undoTool := agent.AgentTool{
		Name:        "git_undo",
		Description: "Revert the last ai auto-commit. Use this if the agent made a mistake in the previous step.",
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			// Check if the last commit was an auto-commit
			logCmd := exec.Command("git", "log", "-1", "--pretty=%B")
			logCmd.Dir = p.CWD
			out, err := logCmd.Output()
			if err != nil {
				return agent.AgentToolResult{}, fmt.Errorf("failed to check git log: %v", err)
			}

			if !strings.HasPrefix(string(out), "ai: auto-commit") {
				return agent.AgentToolResult{
					Content: []ai.Content{ai.TextContent{Text: "The last commit was not an AI auto-commit. Manual undo required."}},
					IsError: true,
				}, nil
			}

			// Revert the last commit and keep changes staged (soft reset) or un-stage (mixed reset)
			resetCmd := exec.Command("git", "reset", "HEAD~1")
			resetCmd.Dir = p.CWD
			if err := resetCmd.Run(); err != nil {
				return agent.AgentToolResult{}, fmt.Errorf("git reset failed: %v", err)
			}

			return agent.AgentToolResult{
				Content: []ai.Content{ai.TextContent{Text: "Successfully reverted the last AI auto-commit. Files are now in their modified state before that commit."}},
			}, nil
		},
	}
	return append(toolList, undoTool)
}

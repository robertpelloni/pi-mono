package tools

import (
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// ReadTool returns the core AgentTool for reading files.
func ReadTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "read",
		Label:       "read",
		Description: "Read the contents of a file.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to read",
				},
			},
			"required": []string{"path"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			path, ok := params["path"].(string)
			if !ok {
				return agent.AgentToolResult{}, fmt.Errorf("invalid or missing path parameter")
			}

			// We resolve path relative to the provided cwd
			absPath := path
			if !filepath.IsAbs(path) {
				absPath = filepath.Join(cwd, path)
			}

			contentBytes, err := ioutil.ReadFile(absPath)
			if err != nil {
				return agent.AgentToolResult{}, err
			}

			return agent.AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: string(contentBytes)},
				},
				Details: map[string]interface{}{
					"path": absPath,
					"size": len(contentBytes),
				},
			}, nil
		},
	}
}

// BashTool returns the core AgentTool for executing terminal commands.
func BashTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "bash",
		Label:       "bash",
		Description: "Run a bash command in the terminal.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The command to run",
				},
			},
			"required": []string{"command"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			command, ok := params["command"].(string)
			if !ok {
				return agent.AgentToolResult{}, fmt.Errorf("invalid or missing command parameter")
			}

			// WARNING: The constraint states "Never execute commands that taskkill all node processes".
			if strings.Contains(command, "pkill node") || strings.Contains(command, "killall node") {
				return agent.AgentToolResult{}, fmt.Errorf("blocked: cannot taskkill node processes")
			}

			cmd := exec.CommandContext(ctx, "bash", "-c", command)
			cmd.Dir = cwd

			outBytes, err := cmd.CombinedOutput()

			outputStr := string(outBytes)
			isError := err != nil

			if isError && outputStr == "" {
				outputStr = err.Error()
			}

			return agent.AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: outputStr},
				},
				Details: map[string]interface{}{
					"command": command,
					"isError": isError,
					"exitCode": func() int {
						if cmd.ProcessState != nil {
							return cmd.ProcessState.ExitCode()
						}
						return -1
					}(),
				},
			}, nil
		},
	}
}

// WriteTool returns the core AgentTool for completely overwriting files.
func WriteTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "write",
		Label:       "write",
		Description: "Write content to a file, completely replacing existing content.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type": "string",
				},
				"content": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []string{"path", "content"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			path, ok1 := params["path"].(string)
			content, ok2 := params["content"].(string)
			if !ok1 || !ok2 {
				return agent.AgentToolResult{}, fmt.Errorf("missing path or content parameters")
			}

			absPath := path
			if !filepath.IsAbs(path) {
				absPath = filepath.Join(cwd, path)
			}

			_, err := agent.WithFileMutationQueue(absPath, func() (any, error) {
				return nil, ioutil.WriteFile(absPath, []byte(content), 0644)
			})

			if err != nil {
				return agent.AgentToolResult{}, err
			}

			return agent.AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: "File written successfully."},
				},
			}, nil
		},
	}
}

// EditTool returns the core AgentTool for partially updating files using a simple replace block.
func EditTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "edit",
		Label:       "edit",
		Description: "Targeted find-and-replace edits in files.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type": "string",
				},
				"find": map[string]interface{}{
					"type": "string",
				},
				"replace": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []string{"path", "find", "replace"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			path, ok1 := params["path"].(string)
			findStr, ok2 := params["find"].(string)
			replaceStr, ok3 := params["replace"].(string)
			if !ok1 || !ok2 || !ok3 {
				return agent.AgentToolResult{}, fmt.Errorf("missing parameters")
			}

			absPath := path
			if !filepath.IsAbs(path) {
				absPath = filepath.Join(cwd, path)
			}

			_, err := agent.WithFileMutationQueue(absPath, func() (any, error) {
				contentBytes, err := ioutil.ReadFile(absPath)
				if err != nil {
					return nil, err
				}

				contentStr := string(contentBytes)
				if !strings.Contains(contentStr, findStr) {
					return nil, fmt.Errorf("find string not found in file")
				}

				newContent := strings.Replace(contentStr, findStr, replaceStr, 1)
				return nil, ioutil.WriteFile(absPath, []byte(newContent), 0644)
			})

			if err != nil {
				return agent.AgentToolResult{}, err
			}

			return agent.AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: "Edit applied successfully."},
				},
			}, nil
		},
	}
}

// LsTool returns the core AgentTool for listing directory contents.
func LsTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "ls",
		Label:       "ls",
		Description: "List contents of a directory.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type": "string",
				},
			},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			path := cwd
			if p, ok := params["path"].(string); ok && p != "" {
				if !filepath.IsAbs(p) {
					path = filepath.Join(cwd, p)
				} else {
					path = p
				}
			}

			cmd := exec.CommandContext(ctx, "ls", "-a", "-l", "-F", "--group-directories-first", path)
			cmd.Dir = cwd

			outBytes, err := cmd.CombinedOutput()
			outputStr := string(outBytes)

			if err != nil && outputStr == "" {
				outputStr = err.Error()
			}

			return agent.AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: outputStr},
				},
			}, nil
		},
	}
}

// GrepTool returns the core AgentTool for searching file contents.
func GrepTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "grep",
		Label:       "grep",
		Description: "Search file contents.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type": "string",
				},
				"path": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []string{"pattern"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			pattern, ok := params["pattern"].(string)
			if !ok {
				return agent.AgentToolResult{}, fmt.Errorf("missing pattern parameter")
			}

			path := cwd
			if p, ok := params["path"].(string); ok && p != "" {
				if !filepath.IsAbs(p) {
					path = filepath.Join(cwd, p)
				} else {
					path = p
				}
			}

			cmd := exec.CommandContext(ctx, "grep", "-rnI", pattern, path)
			cmd.Dir = cwd

			outBytes, err := cmd.CombinedOutput()
			outputStr := string(outBytes)

			if err != nil {
				// grep returns exit code 1 if nothing is found, which isn't really an "error"
				if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
					outputStr = "No matches found."
				} else if outputStr == "" {
					outputStr = err.Error()
				}
			}

			return agent.AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: outputStr},
				},
			}, nil
		},
	}
}

// FindTool returns the core AgentTool for finding files by name.
func FindTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "find",
		Label:       "find",
		Description: "Find files by name.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
				"path": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []string{"name"},
		},
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			name, ok := params["name"].(string)
			if !ok {
				return agent.AgentToolResult{}, fmt.Errorf("missing name parameter")
			}

			path := cwd
			if p, ok := params["path"].(string); ok && p != "" {
				if !filepath.IsAbs(p) {
					path = filepath.Join(cwd, p)
				} else {
					path = p
				}
			}

			cmd := exec.CommandContext(ctx, "find", path, "-name", name)
			cmd.Dir = cwd

			outBytes, err := cmd.CombinedOutput()
			outputStr := string(outBytes)

			if err != nil && outputStr == "" {
				outputStr = err.Error()
			}

			if outputStr == "" {
				outputStr = "No files found."
			}

			return agent.AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: outputStr},
				},
			}, nil
		},
	}
}

// GooseDeveloperShellTool returns the core AgentTool alias for Goose developer__shell.
func GooseDeveloperShellTool(cwd string) agent.AgentTool {
	baseDef := BashTool(cwd)
	baseDef.Name = "developer__shell"
	baseDef.Label = "developer__shell"
	baseDef.Description = "Run shell commands (Goose format)"
	baseDef.Parameters = ai.GooseDeveloperShell.InputSchema
	return baseDef
}

// GooseFinalOutputTool returns the core AgentTool alias for Goose recipe__final_output.
func GooseFinalOutputTool() agent.AgentTool {
	return agent.AgentTool{
		Name:        "recipe__final_output",
		Label:       "recipe__final_output",
		Description: "Output the final result for the user (Goose format)",
		Parameters:  ai.GooseFinalOutput.InputSchema,
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			msg, _ := params["message"].(string)
			return agent.AgentToolResult{
				Content: []ai.Content{
					ai.TextContent{Text: msg},
				},
				Details: map[string]interface{}{
					"finalOutput": msg,
				},
			}, nil
		},
	}
}

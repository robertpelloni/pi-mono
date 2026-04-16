package ai

import (
	"encoding/json"
)

// CleanRoomToolSchema represents an exact 1:1 parity tool schema for internal model harnesses
// (e.g., Claude Code, Codex, Copilot CLI, Gemini CLI)
type CleanRoomToolSchema struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}

// These schemas map exact parameter names that models are pre-trained on to our internal unified implementations.

var ClaudeCodeRead = CleanRoomToolSchema{
	Name:        "read_file",
	Description: "Read the contents of a file.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "The absolute or relative path to the file to read.",
			},
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "Optional line number to start reading from.",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Optional number of lines to read.",
			},
		},
		"required": []string{"file_path"},
	},
}

var CopilotRead = CleanRoomToolSchema{
	Name:        "vscode_read",
	Description: "Read file contents",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"uri": map[string]interface{}{
				"type":        "string",
				"description": "The URI or path of the file to read",
			},
		},
		"required": []string{"uri"},
	},
}

var AiderRunCommand = CleanRoomToolSchema{
	Name:        "run_command",
	Description: "Run a command in the terminal",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"cmd": map[string]interface{}{
				"type":        "string",
				"description": "The command string to execute",
			},
		},
		"required": []string{"cmd"},
	},
}

var ClaudeCodeBash = CleanRoomToolSchema{
	Name:        "bash",
	Description: "Run a bash command",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The bash command to run.",
			},
		},
		"required": []string{"command"},
	},
}

var GeminiShell = CleanRoomToolSchema{
	Name:        "shell",
	Description: "Execute a shell script",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"script": map[string]interface{}{
				"type":        "string",
				"description": "The shell script or command to execute",
			},
		},
		"required": []string{"script"},
	},
}

// MapCleanRoomParams maps the various incoming parameter names back to a unified internal parameter set.
// For example, file_path, uri, filename -> path. command, cmd, script -> command.
func MapCleanRoomParams(toolName string, rawArgs []byte) (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, err
	}

	unified := make(map[string]interface{})

	// Copy over original args first
	for k, v := range args {
		unified[k] = v
	}

	// Read Normalization
	if path, ok := args["file_path"].(string); ok {
		unified["path"] = path
	} else if uri, ok := args["uri"].(string); ok {
		unified["path"] = uri
	} else if filename, ok := args["filename"].(string); ok {
		unified["path"] = filename
	}

	// Command Normalization
	if cmd, ok := args["cmd"].(string); ok {
		unified["command"] = cmd
	} else if script, ok := args["script"].(string); ok {
		unified["command"] = script
	}

	return unified, nil
}

// --- HERMES AGENT & II-AGENT PARITY SCHEMAS ---

// Hermes File Toolset Parity
var HermesPatch = CleanRoomToolSchema{
	Name:        "patch",
	Description: "Targeted find-and-replace edits in files.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to edit.",
			},
			"find": map[string]interface{}{
				"type":        "string",
				"description": "The exact string or regex to find.",
			},
			"replace": map[string]interface{}{
				"type":        "string",
				"description": "The replacement string.",
			},
		},
		"required": []string{"file_path", "find", "replace"},
	},
}

var HermesSearchFiles = CleanRoomToolSchema{
	Name:        "search_files",
	Description: "Search file contents or find files by name.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"target": map[string]interface{}{
				"type":        "string",
				"description": "Either 'content' or 'name'.",
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query or regex.",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The directory to search in.",
			},
		},
		"required": []string{"target", "query"},
	},
}

var HermesTerminal = CleanRoomToolSchema{
	Name:        "terminal",
	Description: "Execute shell commands on a Linux environment. Filesystem persists between calls.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The command string to execute",
			},
			"background": map[string]interface{}{
				"type":        "boolean",
				"description": "Run in background?",
			},
		},
		"required": []string{"command"},
	},
}

// --- EXTENDED HERMES TOOLS ---

var HermesBrowserNavigate = CleanRoomToolSchema{
	Name:        "browser_navigate",
	Description: "Navigate to a URL in the browser.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"url"},
	},
}

var HermesBrowserClick = CleanRoomToolSchema{
	Name:        "browser_click",
	Description: "Click on an element identified by its ref ID.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"ref_id": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"ref_id"},
	},
}

var HermesBrowserType = CleanRoomToolSchema{
	Name:        "browser_type",
	Description: "Type text into an input field identified by its ref ID.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"ref_id": map[string]interface{}{
				"type": "string",
			},
			"text": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"ref_id", "text"},
	},
}

var HermesBrowserSnapshot = CleanRoomToolSchema{
	Name:        "browser_snapshot",
	Description: "Get a text-based snapshot of the current page's accessibility tree.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"full": map[string]interface{}{
				"type": "boolean",
			},
		},
	},
}

var HermesClarify = CleanRoomToolSchema{
	Name:        "clarify",
	Description: "Ask the user a question when you need clarification.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"question": map[string]interface{}{
				"type": "string",
			},
			"choices": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
		},
		"required": []string{"question"},
	},
}

var HermesExecuteCode = CleanRoomToolSchema{
	Name:        "execute_code",
	Description: "Run a Python script that can call Hermes tools programmatically.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"code": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"code"},
	},
}

var HermesCronjob = CleanRoomToolSchema{
	Name:        "cronjob",
	Description: "Unified scheduled-task manager.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type": "string",
				"enum": []string{"create", "list", "update", "pause", "resume", "run", "remove"},
			},
			"schedule": map[string]interface{}{
				"type": "string",
			},
			"command": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"action"},
	},
}

var HermesDelegateTask = CleanRoomToolSchema{
	Name:        "delegate_task",
	Description: "Spawn one or more subagents to work on tasks in isolated contexts.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task": map[string]interface{}{
				"type": "string",
			},
			"context": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"task"},
	},
}

var HermesWriteFile = CleanRoomToolSchema{
	Name:        "write_file",
	Description: "Write content to a file, completely replacing existing content.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type": "string",
			},
			"content": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"file_path", "content"},
	},
}

var HermesHACallService = CleanRoomToolSchema{
	Name:        "ha_call_service",
	Description: "Call a Home Assistant service to control a device.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"domain":    map[string]interface{}{"type": "string"},
			"service":   map[string]interface{}{"type": "string"},
			"entity_id": map[string]interface{}{"type": "string"},
		},
		"required": []string{"domain", "service"},
	},
}

var HermesMemory = CleanRoomToolSchema{
	Name:        "memory",
	Description: "Save important information to persistent memory that survives across sessions.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"key":   map[string]interface{}{"type": "string"},
			"value": map[string]interface{}{"type": "string"},
		},
		"required": []string{"key", "value"},
	},
}

var HermesMOA = CleanRoomToolSchema{
	Name:        "mixture_of_agents",
	Description: "Route a hard problem through multiple frontier LLMs collaboratively.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"prompt": map[string]interface{}{"type": "string"},
		},
		"required": []string{"prompt"},
	},
}

var HermesSessionSearch = CleanRoomToolSchema{
	Name:        "session_search",
	Description: "Search your long-term memory of past conversations.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{"type": "string"},
		},
		"required": []string{"query"},
	},
}

var HermesSkillManage = CleanRoomToolSchema{
	Name:        "skill_manage",
	Description: "Manage skills (create, update, delete).",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action":  map[string]interface{}{"type": "string", "enum": []string{"create", "update", "delete"}},
			"name":    map[string]interface{}{"type": "string"},
			"content": map[string]interface{}{"type": "string"},
		},
		"required": []string{"action", "name"},
	},
}

var HermesWebSearch = CleanRoomToolSchema{
	Name:        "web_search",
	Description: "Search the web for information on any topic.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{"type": "string"},
		},
		"required": []string{"query"},
	},
}

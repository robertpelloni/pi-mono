package cleanroomschemas

// ToolParameterSchema defines a parameter for a tool schema.
type ToolParameterSchema struct {
	Type        string             `json:"type"`
	Description string             `json:"description,omitempty"`
	Enum        []string           `json:"enum,omitempty"`
	Items       *ToolParameterSchema `json:"items,omitempty"`
	Optional    bool               `json:"-"`
}

// ToolSchema defines the JSON schema for a tool's parameters.
type ToolSchema struct {
	Type       string                        `json:"type"`
	Properties map[string]ToolParameterSchema `json:"properties"`
	Required   []string                      `json:"required,omitempty"`
}

// --- READ ALIASES ---

// ClaudeCodeReadSchema is the schema for Claude Code's read tool.
var ClaudeCodeReadSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"file_path": {Type: "string", Description: "The absolute or relative path to the file to read."},
		"offset":    {Type: "number", Description: "Optional line number to start reading from.", Optional: true},
		"limit":     {Type: "number", Description: "Optional number of lines to read.", Optional: true},
	},
	Required: []string{"file_path"},
}

// CopilotReadSchema is the schema for Copilot's read tool.
var CopilotReadSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"uri": {Type: "string", Description: "The URI or path of the file to read"},
	},
	Required: []string{"uri"},
}

// --- BASH ALIASES ---

// ClaudeCodeBashSchema is the schema for Claude Code's bash tool.
var ClaudeCodeBashSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"command": {Type: "string", Description: "The bash command to run."},
	},
	Required: []string{"command"},
}

// GeminiShellSchema is the schema for Gemini's shell tool.
var GeminiShellSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"script": {Type: "string", Description: "The shell script or command to execute"},
	},
	Required: []string{"script"},
}

// AiderRunCommandSchema is the schema for Aider's run command tool.
var AiderRunCommandSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"cmd": {Type: "string", Description: "The command to run in the terminal"},
	},
	Required: []string{"cmd"},
}

// AiderReplaceLinesSchema is the schema for Aider's replace lines tool.
var AiderReplaceLinesSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"file_path":   {Type: "string", Description: "Path to the file to edit."},
		"start_line":  {Type: "number", Description: "The 1-indexed start line number to replace."},
		"end_line":    {Type: "number", Description: "The 1-indexed end line number to replace."},
		"replacement": {Type: "string", Description: "The replacement text string."},
	},
	Required: []string{"file_path", "start_line", "end_line", "replacement"},
}

// --- GREP ALIASES ---

// ClaudeCodeGrepSchema is the schema for Claude Code's grep tool.
var ClaudeCodeGrepSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"pattern":  {Type: "string", Description: "The regex pattern to search for"},
		"path":     {Type: "string", Description: "The directory or file path to search in", Optional: true},
		"include":  {Type: "string", Description: "File glob to include", Optional: true},
		"exclude":  {Type: "string", Description: "File glob to exclude", Optional: true},
	},
	Required: []string{"pattern"},
}

// OpenCodeSearchSchema is the schema for OpenCode's search tool.
var OpenCodeSearchSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"query":     {Type: "string", Description: "The search query"},
		"directory": {Type: "string", Description: "The directory to search", Optional: true},
	},
	Required: []string{"query"},
}

// --- HERMES AGENT SCHEMAS ---

// HermesPatchSchema is the schema for Hermes patch tool.
var HermesPatchSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"file_path": {Type: "string", Description: "Path to the file to edit."},
		"find":      {Type: "string", Description: "The exact string or regex to find."},
		"replace":   {Type: "string", Description: "The replacement string."},
	},
	Required: []string{"file_path", "find", "replace"},
}

// HermesTerminalSchema is the schema for Hermes terminal tool.
var HermesTerminalSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"command":   {Type: "string", Description: "The command string to execute"},
		"background": {Type: "boolean", Description: "Run in background?", Optional: true},
	},
	Required: []string{"command"},
}

// HermesWriteFileSchema is the schema for Hermes write file tool.
var HermesWriteFileSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"file_path": {Type: "string", Description: "Path to the file to write."},
		"content":   {Type: "string", Description: "The content to write."},
	},
	Required: []string{"file_path", "content"},
}

// HermesMemorySchema is the schema for Hermes memory tool.
var HermesMemorySchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"key":   {Type: "string", Description: "The memory key."},
		"value": {Type: "string", Description: "The memory value."},
	},
	Required: []string{"key", "value"},
}

// --- CLINE SCHEMAS ---

// ClineExecuteCommandSchema is the schema for Cline's execute command tool.
var ClineExecuteCommandSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"command":           {Type: "string", Description: "The CLI command to execute."},
		"requires_approval": {Type: "boolean", Description: "Whether explicit approval is needed."},
	},
	Required: []string{"command", "requires_approval"},
}

// ClineWriteToFileSchema is the schema for Cline's write file tool.
var ClineWriteToFileSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"path":    {Type: "string", Description: "The absolute path of the file."},
		"content": {Type: "string", Description: "The complete intended content of the file."},
	},
	Required: []string{"path", "content"},
}

// ClineAskFollowupSchema is the schema for Cline's ask follow-up question tool.
var ClineAskFollowupSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"question": {Type: "string", Description: "The question to ask the user."},
	},
	Required: []string{"question"},
}

// ClineListCodeDefinitionNamesSchema is the schema for Cline's list definitions tool.
var ClineListCodeDefinitionNamesSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"path": {Type: "string", Description: "The directory path."},
	},
	Required: []string{"path"},
}

// ClineBrowserActionSchema is the schema for Cline's browser action tool.
var ClineBrowserActionSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"action":     {Type: "string", Description: "launch, click, type, scroll_down, scroll_up, close"},
		"url":        {Type: "string", Description: "URL for launch action", Optional: true},
		"coordinate": {Type: "string", Description: "Coordinate for click action", Optional: true},
		"text":       {Type: "string", Description: "Text for type action", Optional: true},
	},
	Required: []string{"action"},
}

// --- OPEN INTERPRETER SCHEMAS ---

// OpenInterpreterComputerUseSchema is the schema for Open Interpreter's computer use tool.
var OpenInterpreterComputerUseSchema = ToolSchema{
	Type: "object",
	Properties: map[string]ToolParameterSchema{
		"action": {Type: "string", Description: "The computer action (key, type, mouse_move, left_click, etc.)"},
		"text":   {Type: "string", Description: "Text for key/type actions", Optional: true},
		"coordinate": {Type: "array", Description: "Coordinate for mouse actions", Optional: true,
			Items: &ToolParameterSchema{Type: "number"}},
	},
	Required: []string{"action"},
}

// AllCleanRoomToolSchemas returns all clean room tool schemas for registration.
func AllCleanRoomToolSchemas() map[string]ToolSchema {
	return map[string]ToolSchema{
		"claude_code_read":        ClaudeCodeReadSchema,
		"copilot_read":            CopilotReadSchema,
		"claude_code_bash":        ClaudeCodeBashSchema,
		"gemini_shell":            GeminiShellSchema,
		"aider_run_command":       AiderRunCommandSchema,
		"aider_replace_lines":    AiderReplaceLinesSchema,
		"claude_code_grep":        ClaudeCodeGrepSchema,
		"open_code_search":        OpenCodeSearchSchema,
		"hermes_patch":            HermesPatchSchema,
		"hermes_terminal":         HermesTerminalSchema,
		"hermes_write_file":       HermesWriteFileSchema,
		"hermes_memory":           HermesMemorySchema,
		"cline_execute_command":   ClineExecuteCommandSchema,
		"cline_write_to_file":     ClineWriteToFileSchema,
		"cline_ask_followup":      ClineAskFollowupSchema,
		"cline_list_definitions":  ClineListCodeDefinitionNamesSchema,
		"cline_browser_action":    ClineBrowserActionSchema,
		"open_interpreter_computer": OpenInterpreterComputerUseSchema,
	}
}

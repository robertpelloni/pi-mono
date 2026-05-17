package rpctypes

import "github.com/badlogic/pi-mono/pkg/ai"

// ============================================================================
// RPC Commands (stdin)
// ============================================================================

// RpcCommand is the union type for all RPC commands.
type RpcCommand struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`

	// Prompt fields
	Images             []ai.ImageContent `json:"images,omitempty"`
	StreamingBehavior  string            `json:"streamingBehavior,omitempty"`

	// Model fields
	Provider string `json:"provider,omitempty"`
	ModelID  string `json:"modelId,omitempty"`

	// Thinking
	Level string `json:"level,omitempty"`

	// Steering/follow-up
	Mode string `json:"mode,omitempty"`

	// Compaction
	CustomInstructions string `json:"customInstructions,omitempty"`
	Enabled            bool   `json:"enabled,omitempty"`

	// Bash
	Command string `json:"command,omitempty"`

	// Session
	SessionPath  string `json:"sessionPath,omitempty"`
	EntryID      string `json:"entryId,omitempty"`
	OutputPath   string `json:"outputPath,omitempty"`
	Name         string `json:"name,omitempty"`
	ParentSession string `json:"parentSession,omitempty"`
}

// ============================================================================
// RPC Session State
// ============================================================================

// RpcSessionState represents the current session state.
type RpcSessionState struct {
	Model                 ai.ModelInfo    `json:"model,omitempty"`
	ThinkingLevel         string          `json:"thinkingLevel"`
	IsStreaming           bool            `json:"isStreaming"`
	IsCompacting          bool            `json:"isCompacting"`
	SteeringMode          string          `json:"steeringMode"`
	FollowUpMode          string          `json:"followUpMode"`
	SessionFile           string          `json:"sessionFile,omitempty"`
	SessionID             string          `json:"sessionId"`
	SessionName           string          `json:"sessionName,omitempty"`
	AutoCompactionEnabled bool            `json:"autoCompactionEnabled"`
	MessageCount          int             `json:"messageCount"`
	PendingMessageCount   int             `json:"pendingMessageCount"`
}

// ============================================================================
// RPC Responses (stdout)
// ============================================================================

// RpcResponse is a response to an RPC command.
type RpcResponse struct {
	ID      string      `json:"id,omitempty"`
	Type    string      `json:"type"` // always "response"
	Command string      `json:"command"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// RpcError creates an error response.
func RpcError(id, command, errMsg string) RpcResponse {
	return RpcResponse{
		ID:      id,
		Type:    "response",
		Command: command,
		Success: false,
		Error:   errMsg,
	}
}

// RpcSuccess creates a success response.
func RpcSuccess(id, command string, data interface{}) RpcResponse {
	return RpcResponse{
		ID:      id,
		Type:    "response",
		Command: command,
		Success: true,
		Data:    data,
	}
}

// ============================================================================
// Extension UI Events
// ============================================================================

// RpcExtensionUIRequest is sent when an extension needs user input.
type RpcExtensionUIRequest struct {
	Type       string   `json:"type"` // "extension_ui_request"
	ID         string   `json:"id"`
	Method     string   `json:"method"` // "select", "confirm", "input", "editor", "notify", etc.
	Title      string   `json:"title,omitempty"`
	Message    string   `json:"message,omitempty"`
	Options    []string `json:"options,omitempty"`
	Timeout    int      `json:"timeout,omitempty"`
	Placeholder string  `json:"placeholder,omitempty"`
	NotifyType string   `json:"notifyType,omitempty"`
	StatusKey  string   `json:"statusKey,omitempty"`
	StatusText string   `json:"statusText,omitempty"`
	WidgetKey  string   `json:"widgetKey,omitempty"`
	WidgetLines []string `json:"widgetLines,omitempty"`
	Text       string   `json:"text,omitempty"`
}

// RpcExtensionUIResponse is a response to an extension UI request.
type RpcExtensionUIResponse struct {
	Type      string `json:"type"` // "extension_ui_response"
	ID        string `json:"id"`
	Value     string `json:"value,omitempty"`
	Confirmed bool   `json:"confirmed,omitempty"`
	Cancelled bool   `json:"cancelled,omitempty"`
}

// ============================================================================
// Slash Command info
// ============================================================================

// RpcSlashCommand describes a command available for invocation via prompt.
type RpcSlashCommand struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"` // "extension", "prompt", "skill"
}

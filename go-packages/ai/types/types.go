// Package types provides core definitions for pi-ai.
package types

import "encoding/json"

// Role defines the participant's role in a conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is an interface representing any type of message in the thread.
type Message interface {
	GetRole() Role
}

type TextContent struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

type ImageContent struct {
	Type     string `json:"type"` // "image"
	MimeType string `json:"mimeType"`
	Data     []byte `json:"data"` // Base64 encoded byte array or raw bytes
}

// UserMessage represents an input from the user.
type UserMessage struct {
	Role    Role          `json:"role"` // "user"
	Content []interface{} `json:"content"` // can be TextContent or ImageContent
}

func (m UserMessage) GetRole() Role { return m.Role }

// ToolCall represents a requested invocation of a tool by the assistant.
type ToolCall struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"` // "function"
	Function  ToolCallDetails `json:"function"`
}

type ToolCallDetails struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// AssistantMessage represents a response from the LLM.
type AssistantMessage struct {
	Role      Role       `json:"role"` // "assistant"
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

func (m AssistantMessage) GetRole() Role { return m.Role }

// ToolResultMessage represents the outcome of a tool execution.
type ToolResultMessage struct {
	Role       Role   `json:"role"` // "tool"
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
}

func (m ToolResultMessage) GetRole() Role { return m.Role }

// Tool defines an external function available to the LLM.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON schema
}

// Context encapsulates the state sent to the LLM.
type Context struct {
	SystemPrompt string    `json:"systemPrompt,omitempty"`
	Messages     []Message `json:"messages"`
	Tools        []Tool    `json:"tools,omitempty"`
}

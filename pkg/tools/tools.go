package tools

import (
	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/bashtool"
	"github.com/badlogic/pi-mono/pkg/edittool"
	"github.com/badlogic/pi-mono/pkg/findtool"
	"github.com/badlogic/pi-mono/pkg/greptool"
	"github.com/badlogic/pi-mono/pkg/lstool"
	"github.com/badlogic/pi-mono/pkg/nativetools"
	"github.com/badlogic/pi-mono/pkg/readtool"
	"github.com/badlogic/pi-mono/pkg/writetool"
)

// ToolName represents a named tool in the registry.
type ToolName string

const (
	ToolRead  ToolName = "read"
	ToolBash  ToolName = "bash"
	ToolEdit  ToolName = "edit"
	ToolWrite ToolName = "write"
	ToolGrep  ToolName = "grep"
	ToolFind  ToolName = "find"
	ToolLs    ToolName = "ls"
)

// AllToolNames lists all available tool names.
var AllToolNames = []ToolName{ToolRead, ToolBash, ToolEdit, ToolWrite, ToolGrep, ToolFind, ToolLs}

// DefaultToolNames lists the default active tool names.
var DefaultToolNames = []ToolName{ToolRead, ToolBash, ToolEdit, ToolWrite}

// ReadOnlyToolNames lists read-only tool names.
var ReadOnlyToolNames = []ToolName{ToolRead, ToolGrep, ToolFind, ToolLs}

// CodingTools is the default set of coding tools.
var CodingTools = []string{"read", "bash", "edit", "write"}

// ReadOnlyTools is the set of read-only tools.
var ReadOnlyTools = []string{"read", "grep", "find", "ls"}

// CreateAllTools returns the full set of built-in tools.
// Default active: read, bash, edit, write.
// Additional: grep, find, ls.
func CreateAllTools(cwd string) []agent.AgentTool {
	return []agent.AgentTool{
		nativetools.NativeReadTool(cwd),
		bashtool.CreateBashTool(cwd),
		edittool.CreateEditTool(cwd),
		writetool.CreateWriteTool(cwd),
		nativetools.NativeGrepTool(cwd),
		nativetools.NativeGlobTool(cwd),
		nativetools.NativeLsTool(cwd),
	}
}

// CreateDefaultTools returns only the default active tool set.
func CreateDefaultTools(cwd string) []agent.AgentTool {
	return []agent.AgentTool{
		nativetools.NativeReadTool(cwd),
		bashtool.CreateBashTool(cwd),
		edittool.CreateEditTool(cwd),
		writetool.CreateWriteTool(cwd),
	}
}

// CreateReadOnlyTools returns the read-only tool set (no file modification).
func CreateReadOnlyTools(cwd string) []agent.AgentTool {
	return []agent.AgentTool{
		nativetools.NativeReadTool(cwd),
		nativetools.NativeGrepTool(cwd),
		nativetools.NativeGlobTool(cwd),
		nativetools.NativeLsTool(cwd),
	}
}

// CreateToolsByName creates a specific set of tools by name.
func CreateToolsByName(cwd string, names []string) []agent.AgentTool {
	var toolList []agent.AgentTool
	for _, name := range names {
		switch name {
		case "read":
			toolList = append(toolList, nativetools.NativeReadTool(cwd))
		case "bash":
			toolList = append(toolList, bashtool.CreateBashTool(cwd))
		case "edit":
			toolList = append(toolList, edittool.CreateEditTool(cwd))
		case "write":
			toolList = append(toolList, writetool.CreateWriteTool(cwd))
		case "grep":
			toolList = append(toolList, nativetools.NativeGrepTool(cwd))
		case "find", "glob":
			toolList = append(toolList, nativetools.NativeGlobTool(cwd))
		case "ls":
			toolList = append(toolList, nativetools.NativeLsTool(cwd))
		}
	}
	return toolList
}

// IsValidToolName checks if a tool name is valid.
func IsValidToolName(name string) bool {
	for _, n := range AllToolNames {
		if string(n) == name {
			return true
		}
	}
	return false
}

// ToolNames returns the names of all available tools.
func ToolNames() []string {
	names := make([]string, len(AllToolNames))
	for i, n := range AllToolNames {
		names[i] = string(n)
	}
	return names
}

// Ensure the new packages are imported and compile
var _ = findtool.Execute
var _ = greptool.Execute
var _ = lstool.Execute
var _ = readtool.Execute

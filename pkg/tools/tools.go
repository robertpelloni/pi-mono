package tools

import (
	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/bashtool"
	"github.com/badlogic/pi-mono/pkg/edittool"
	"github.com/badlogic/pi-mono/pkg/nativetools"
	"github.com/badlogic/pi-mono/pkg/writetool"
)

// CreateAllTools returns the full set of built-in tools matching the TypeScript version.
// The default active tools are: read, bash, edit, write.
// Additional available tools: find, grep, ls.
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

// ToolNames returns the names of all available tools.
func ToolNames() []string {
	return []string{"read", "bash", "edit", "write", "grep", "glob", "ls"}
}

// DefaultToolNames returns the names of the default active tools.
func DefaultToolNames() []string {
	return []string{"read", "bash", "edit", "write"}
}

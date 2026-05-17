package cli

import (
	"fmt"
	"os"
	"strings"
)

// Mode defines the output mode for the agent.
type Mode string

const (
	ModeText Mode = "text"
	ModeJSON Mode = "json"
	ModeRPC  Mode = "rpc"
)

// ThinkingLevel represents the thinking/reasoning level.
type ThinkingLevel string

const (
	ThinkingOff     ThinkingLevel = "off"
	ThinkingMinimal ThinkingLevel = "minimal"
	ThinkingLow     ThinkingLevel = "low"
	ThinkingMedium  ThinkingLevel = "medium"
	ThinkingHigh    ThinkingLevel = "high"
	ThinkingXHigh   ThinkingLevel = "xhigh"
)

var validThinkingLevels = []ThinkingLevel{
	ThinkingOff, ThinkingMinimal, ThinkingLow, ThinkingMedium, ThinkingHigh, ThinkingXHigh,
}

// IsValidThinkingLevel checks if a string is a valid thinking level.
func IsValidThinkingLevel(level string) bool {
	for _, v := range validThinkingLevels {
		if ThinkingLevel(level) == v {
			return true
		}
	}
	return false
}

// Args holds all parsed CLI arguments.
type Args struct {
	Provider           string               `json:"provider,omitempty"`
	Model              string               `json:"model,omitempty"`
	APIKey             string               `json:"apiKey,omitempty"`
	SystemPrompt       string               `json:"systemPrompt,omitempty"`
	AppendSystemPrompt string               `json:"appendSystemPrompt,omitempty"`
	Thinking           ThinkingLevel        `json:"thinking,omitempty"`
	Continue           bool                 `json:"continue,omitempty"`
	Resume             bool                 `json:"resume,omitempty"`
	Help               bool                 `json:"help,omitempty"`
	Version            bool                 `json:"version,omitempty"`
	Mode               Mode                 `json:"mode,omitempty"`
	NoSession          bool                 `json:"noSession,omitempty"`
	Session            string               `json:"session,omitempty"`
	Fork               string               `json:"fork,omitempty"`
	SessionDir         string               `json:"sessionDir,omitempty"`
	Models             []string             `json:"models,omitempty"`
	Tools              []string             `json:"tools,omitempty"`
	NoTools            bool                 `json:"noTools,omitempty"`
	Extensions         []string             `json:"extensions,omitempty"`
	NoExtensions       bool                 `json:"noExtensions,omitempty"`
	Print              bool                 `json:"print,omitempty"`
	Export             string               `json:"export,omitempty"`
	NoSkills           bool                 `json:"noSkills,omitempty"`
	Skills             []string             `json:"skills,omitempty"`
	PromptTemplates    []string             `json:"promptTemplates,omitempty"`
	NoPromptTemplates  bool                 `json:"noPromptTemplates,omitempty"`
	Themes             []string             `json:"themes,omitempty"`
	NoThemes           bool                 `json:"noThemes,omitempty"`
	ListModels         string               `json:"listModels,omitempty"` // "" means not set, "true" means list all
	Offline            bool                 `json:"offline,omitempty"`
	Verbose            bool                 `json:"verbose,omitempty"`
	Messages           []string             `json:"messages,omitempty"`
	FileArgs           []string             `json:"fileArgs,omitempty"`
	UnknownFlags       map[string]interface{} `json:"unknownFlags,omitempty"`
	Diagnostics        []Diagnostic         `json:"diagnostics,omitempty"`
	Dir                string               `json:"dir,omitempty"`
	NoGuard            bool                 `json:"noGuard,omitempty"`
	CompactThreshold   int                  `json:"compactThreshold,omitempty"`
	WebPort            int                  `json:"webPort,omitempty"`
	Frontend           string               `json:"frontend,omitempty"`
	ToolMode           string               `json:"toolMode,omitempty"`
}

// Diagnostic represents a non-fatal parsing issue.
type Diagnostic struct {
	Type    string `json:"type"` // "warning" or "error"
	Message string `json:"message"`
}

// ParseArgs parses CLI arguments into an Args struct.
func ParseArgs(args []string) *Args {
	result := &Args{
		UnknownFlags: make(map[string]interface{}),
		ToolMode:     "parallel",
		CompactThreshold: 100000,
		WebPort:      8080,
		Frontend:     "bubbletea",
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--help" || arg == "-h":
			result.Help = true
		case arg == "--version" || arg == "-v":
			result.Version = true
		case arg == "--mode" && i+1 < len(args):
			i++
			mode := args[i]
			if mode == "text" || mode == "json" || mode == "rpc" {
				result.Mode = Mode(mode)
			}
		case arg == "--continue" || arg == "-c":
			result.Continue = true
		case arg == "--resume" || arg == "-r":
			result.Resume = true
		case arg == "--provider" && i+1 < len(args):
			i++
			result.Provider = args[i]
		case arg == "--model" && i+1 < len(args):
			i++
			result.Model = args[i]
		case arg == "--api-key" && i+1 < len(args):
			i++
			result.APIKey = args[i]
		case arg == "--system-prompt" && i+1 < len(args):
			i++
			result.SystemPrompt = args[i]
		case arg == "--append-system-prompt" && i+1 < len(args):
			i++
			result.AppendSystemPrompt = args[i]
		case arg == "--no-session":
			result.NoSession = true
		case arg == "--session" && i+1 < len(args):
			i++
			result.Session = args[i]
		case arg == "--fork" && i+1 < len(args):
			i++
			result.Fork = args[i]
		case arg == "--session-dir" && i+1 < len(args):
			i++
			result.SessionDir = args[i]
		case arg == "--models" && i+1 < len(args):
			i++
			result.Models = strings.Split(args[i], ",")
		case arg == "--no-tools":
			result.NoTools = true
		case arg == "--tools" && i+1 < len(args):
			i++
			result.Tools = strings.Split(args[i], ",")
		case arg == "--thinking" && i+1 < len(args):
			i++
			level := args[i]
			if IsValidThinkingLevel(level) {
				result.Thinking = ThinkingLevel(level)
			} else {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					Type:    "warning",
					Message: fmt.Sprintf("Invalid thinking level %q. Valid: %v", level, validThinkingLevels),
				})
			}
		case arg == "--print" || arg == "-p":
			result.Print = true
		case arg == "--export" && i+1 < len(args):
			i++
			result.Export = args[i]
		case (arg == "--extension" || arg == "-e") && i+1 < len(args):
			i++
			result.Extensions = append(result.Extensions, args[i])
		case arg == "--no-extensions" || arg == "-ne":
			result.NoExtensions = true
		case arg == "--skill" && i+1 < len(args):
			i++
			result.Skills = append(result.Skills, args[i])
		case arg == "--prompt-template" && i+1 < len(args):
			i++
			result.PromptTemplates = append(result.PromptTemplates, args[i])
		case arg == "--theme" && i+1 < len(args):
			i++
			result.Themes = append(result.Themes, args[i])
		case arg == "--no-skills" || arg == "-ns":
			result.NoSkills = true
		case arg == "--no-prompt-templates" || arg == "-np":
			result.NoPromptTemplates = true
		case arg == "--no-themes":
			result.NoThemes = true
		case arg == "--list-models":
			// Check if next arg is a search pattern (not a flag or file arg)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") && !strings.HasPrefix(args[i+1], "@") {
				i++
				result.ListModels = args[i]
			} else {
				result.ListModels = "true"
			}
		case arg == "--verbose":
			result.Verbose = true
		case arg == "--offline":
			result.Offline = true
		case arg == "--dir" && i+1 < len(args):
			i++
			result.Dir = args[i]
		case arg == "--no-guard":
			result.NoGuard = true
		case arg == "--compact-threshold" && i+1 < len(args):
			i++
			fmt.Sscanf(args[i], "%d", &result.CompactThreshold)
		case arg == "--port" && i+1 < len(args):
			i++
			fmt.Sscanf(args[i], "%d", &result.WebPort)
		case arg == "--frontend" && i+1 < len(args):
			i++
			result.Frontend = args[i]
		case arg == "--tools" && i+1 < len(args):
			i++
			result.ToolMode = args[i]
		case strings.HasPrefix(arg, "@"):
			result.FileArgs = append(result.FileArgs, arg[1:])
		case strings.HasPrefix(arg, "--"):
			eqIndex := strings.Index(arg, "=")
			if eqIndex != -1 {
				flagName := arg[2:eqIndex]
				flagValue := arg[eqIndex+1:]
				result.UnknownFlags[flagName] = flagValue
			} else {
				flagName := arg[2:]
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") && !strings.HasPrefix(args[i+1], "@") {
					i++
					result.UnknownFlags[flagName] = args[i]
				} else {
					result.UnknownFlags[flagName] = true
				}
			}
		case strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--"):
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Type:    "error",
				Message: fmt.Sprintf("Unknown option: %s", arg),
			})
		default:
			result.Messages = append(result.Messages, arg)
		}
	}

	return result
}

// PrintHelp prints the CLI help text.
func PrintHelp() {
	fmt.Fprintf(os.Stderr, `pi-go - AI coding assistant with read, bash, edit, write tools

Usage: pi-go [options] [@files...] [messages...]

Options:
  --provider <name>       Provider name (default: openai)
  --model <pattern>       Model pattern or ID (supports "provider/id")
  --api-key <key>         API key (defaults to env vars)
  --system-prompt <text>  System prompt override
  --append-system-prompt <text>  Append to the system prompt
  --mode <mode>           Output mode: text (default), json, or rpc
  --print, -p             Non-interactive mode: process prompt and exit
  --continue, -c          Continue previous session
  --resume, -r            Select a session to resume
  --session <path>        Use specific session file
  --fork <path>           Fork specific session file
  --session-dir <dir>     Directory for session storage
  --no-session            Don't save session (ephemeral)
  --models <patterns>     Comma-separated model patterns for cycling
  --no-tools              Disable all built-in tools
  --tools <tools>         Comma-separated list of tools to enable
  --thinking <level>      Set thinking level: off, minimal, low, medium, high, xhigh
  --extension, -e <path>  Load an extension file
  --no-extensions, -ne    Disable extension discovery
  --skill <path>          Load a skill file or directory
  --no-skills, -ns        Disable skills discovery
  --no-guard              Disable output guard (secret redaction)
  --export <file>         Export session file to HTML
  --list-models [search]  List available models
  --verbose               Force verbose startup
  --offline               Disable startup network operations
  --dir <path>            Set working directory
  --compact-threshold <n> Token threshold for auto-compaction
  --port <n>              Port for web UI (default: 8080)
  --frontend <type>       Frontend type: bubbletea, cli, web
  --help, -h              Show this help
  --version, -v           Show version number

Environment Variables:
  ANTHROPIC_API_KEY       Anthropic Claude API key
  OPENAI_API_KEY          OpenAI GPT API key
  GEMINI_API_KEY          Google Gemini API key
  PI_OFFLINE              Disable startup network operations (1/true/yes)
  PI_AGENT_DIR            Session storage directory (default: ~/.pi)

Available Tools (default: read, bash, edit, write):
  read  - Read file contents
  bash  - Execute bash commands
  edit  - Edit files with find/replace
  write - Write files (creates/overwrites)
  grep  - Search file contents (read-only, off by default)
  find  - Find files by glob pattern (read-only, off by default)
  ls    - List directory contents (read-only, off by default)

Examples:
  pi-go                                    # Interactive mode
  pi-go "List all .ts files in src/"       # With initial prompt
  pi-go -p "Review this code"              # Non-interactive mode
  pi-go --continue                         # Continue previous session
  pi-go --model gpt-4o                     # Use specific model
  pi-go --provider anthropic --thinking high  # With thinking
  pi-go --tools read,grep,find,ls -p "Review" # Read-only mode
`)
}

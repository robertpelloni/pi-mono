package controlplane

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Tool represents a detected AI coding tool on the system.
type Tool struct {
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	Command      string   `json:"command"`
	Available    bool     `json:"available"`
	Version      string   `json:"version,omitempty"`
	Path         string   `json:"path,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type definition struct {
	Type         string
	Name         string
	Command      string
	VersionArgs  []string
	Capabilities []string
}

// Detector discovers available AI coding tools on the system.
type Detector struct {
	definitions []definition
	timeout     time.Duration
	ttl         time.Duration

	mu       sync.Mutex
	inflight chan struct{}
	cached   []Tool
	cachedAt time.Time
}

// NewDetector creates a tool detector with the given per-command timeout and cache TTL.
func NewDetector(timeout, ttl time.Duration) *Detector {
	return &Detector{
		timeout: timeout,
		ttl:     ttl,
		definitions: []definition{
			{Type: "go", Name: "Go", Command: "go", VersionArgs: []string{"version"}, Capabilities: []string{"build", "test", "server"}},
			{Type: "node", Name: "Node.js", Command: "node", VersionArgs: []string{"--version"}, Capabilities: []string{"runtime", "scripts"}},
			{Type: "python", Name: "Python", Command: "python", VersionArgs: []string{"--version"}, Capabilities: []string{"runtime", "scripts"}},
			{Type: "claude", Name: "Claude CLI", Command: "claude", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "code", "analyze"}},
			{Type: "claude-code", Name: "Claude Code CLI", Command: "claude-code", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "code", "analyze"}},
			{Type: "codex", Name: "Codex CLI", Command: "codex", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "code"}},
			{Type: "aider", Name: "Aider CLI", Command: "aider", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "edit", "git-aware", "multi-file"}},
			{Type: "opencode", Name: "OpenCode CLI", Command: "opencode", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "edit", "multi-file", "autonomous"}},
			{Type: "gemini", Name: "Gemini CLI", Command: "gemini", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "code", "multimodal"}},
			{Type: "ollama", Name: "Ollama CLI", Command: "ollama", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "local", "models"}},
			{Type: "goose", Name: "Goose CLI", Command: "goose", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "code", "agent"}},
			{Type: "grok", Name: "Grok CLI", Command: "grok", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "code", "realtime"}},
			{Type: "copilot", Name: "GitHub Copilot CLI", Command: "github-copilot-cli", VersionArgs: []string{"--version"}, Capabilities: []string{"explain", "suggest", "chat", "terminal", "shell"}},
			{Type: "warp", Name: "Warp CLI", Command: "warp", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "terminal", "collaborative"}},
			{Type: "jules", Name: "Jules CLI", Command: "jules", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "code", "agent"}},
			{Type: "smithery", Name: "Smithery CLI", Command: "smithery", VersionArgs: []string{"--version"}, Capabilities: []string{"mcp", "registry", "tools"}},
			{Type: "litellm", Name: "LiteLLM CLI", Command: "litellm", VersionArgs: []string{"--version"}, Capabilities: []string{"models", "proxy", "routing"}},
			{Type: "mistral-vibe", Name: "Mistral Vibe CLI", Command: "mistral", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "code", "local"}},
			{Type: "pi", Name: "Pi CLI", Command: "pi", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "personal", "assistant"}},
			{Type: "kimi", Name: "Kimi CLI", Command: "kimi", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "code", "long-context"}},
			{Type: "llm", Name: "LLM CLI", Command: "llm", VersionArgs: []string{"--version"}, Capabilities: []string{"chat", "models", "prompt"}},
			{Type: "copilot", Name: "GitHub Copilot CLI", Command: "gh", VersionArgs: []string{"copilot", "--version"}, Capabilities: []string{"copilot", "cli"}},
		},
	}
}

// DetectAll discovers all available tools on the system. Results are cached for the configured TTL.
func (d *Detector) DetectAll(ctx context.Context) ([]Tool, error) {
	d.mu.Lock()

	// Return cached results if still valid
	if len(d.cached) > 0 && time.Since(d.cachedAt) < d.ttl {
		tools := append([]Tool(nil), d.cached...)
		d.mu.Unlock()
		return tools, nil
	}

	// Wait for in-flight scan if another goroutine is already scanning
	if d.inflight != nil {
		wait := d.inflight
		d.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-wait:
			d.mu.Lock()
			tools := append([]Tool(nil), d.cached...)
			d.mu.Unlock()
			return tools, nil
		}
	}

	// Mark in-flight to prevent duplicate scans
	wait := make(chan struct{})
	d.inflight = wait
	d.mu.Unlock()

	tools := make([]Tool, 0, len(d.definitions))
	for _, def := range d.definitions {
		tools = append(tools, d.detectTool(ctx, def))
	}

	d.mu.Lock()
	d.cached = append([]Tool(nil), tools...)
	d.cachedAt = time.Now()
	close(wait)
	d.inflight = nil
	d.mu.Unlock()

	return tools, nil
}

func (d *Detector) detectTool(ctx context.Context, def definition) Tool {
	tool := Tool{
		Type:         def.Type,
		Name:         def.Name,
		Command:      def.Command,
		Available:    false,
		Capabilities: append([]string(nil), def.Capabilities...),
	}

	executable, err := exec.LookPath(def.Command)
	if err != nil {
		return tool
	}

	tool.Available = true
	tool.Path = executable

	commandCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	output, err := exec.CommandContext(commandCtx, executable, def.VersionArgs...).CombinedOutput()
	if err != nil && commandCtx.Err() != nil {
		tool.Version = "timeout"
		return tool
	}

	tool.Version = parseVersion(string(output))
	return tool
}

var versionPattern = regexp.MustCompile(`v?(\d+\.\d+(?:\.\d+)?)`)

func parseVersion(output string) string {
	match := versionPattern.FindStringSubmatch(output)
	if len(match) >= 2 {
		return match[1]
	}

	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "unknown"
	}
	return trimmed
}

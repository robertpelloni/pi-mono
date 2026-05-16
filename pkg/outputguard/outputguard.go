package outputguard

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// OutputGuard checks tool outputs for sensitive content and safety violations.
// It operates as both a BeforeToolCall and AfterToolCall hook.
type OutputGuard struct {
	cwd    string
	rules  []GuardRule
}

// GuardRule is a single check that can allow, block, or modify a tool result.
type GuardRule struct {
	Name        string
	Description string
	Phase       GuardPhase // Before or After execution
	Check       func(ctx GuardContext) GuardResult
}

// GuardPhase indicates when the rule runs.
type GuardPhase string

const (
	PhaseBefore GuardPhase = "before" // Before tool execution (can block)
	PhaseAfter  GuardPhase = "after"  // After tool execution (can redact/modify)
)

// GuardContext provides the information needed to evaluate a guard rule.
type GuardContext struct {
	ToolName   string
	ToolCallID string
	Args       map[string]any
	Result     agent.AgentToolResult
	IsError    bool
	CWD        string
}

// GuardResult represents the outcome of a guard check.
type GuardResult struct {
	Allowed       bool
	Reason        string
	ModifiedResult *agent.AgentToolResult
}

// NewOutputGuard creates an output guard with default safety rules.
func NewOutputGuard(cwd string) *OutputGuard {
	g := &OutputGuard{cwd: cwd}
	g.addDefaultRules()
	return g
}

// BeforeToolCall checks pre-execution rules (blocking).
func (g *OutputGuard) BeforeToolCall(ctx context.Context, callCtx agent.BeforeToolCallContext) (*agent.BeforeToolCallResult, error) {
	guardCtx := GuardContext{
		ToolName:   callCtx.ToolCall.Name,
		ToolCallID: callCtx.ToolCall.ID,
		Args:       callCtx.Args,
		CWD:        g.cwd,
	}

	for _, rule := range g.rules {
		if rule.Phase != PhaseBefore {
			continue
		}
		result := rule.Check(guardCtx)
		if !result.Allowed {
			return &agent.BeforeToolCallResult{
				Block:  true,
				Reason: result.Reason,
			}, nil
		}
	}

	return nil, nil
}

// AfterToolCall checks post-execution rules (redaction, modification).
func (g *OutputGuard) AfterToolCall(ctx context.Context, callCtx agent.AfterToolCallContext) (*agent.AfterToolCallResult, error) {
	guardCtx := GuardContext{
		ToolName:   callCtx.ToolCall.Name,
		ToolCallID: callCtx.ToolCall.ID,
		Args:       callCtx.Args,
		Result:     callCtx.Result,
		IsError:    callCtx.IsError,
		CWD:        g.cwd,
	}

	for _, rule := range g.rules {
		if rule.Phase != PhaseAfter {
			continue
		}
		result := rule.Check(guardCtx)
		if !result.Allowed {
			return &agent.AfterToolCallResult{
				Content: []ai.Content{ai.TextContent{Text: fmt.Sprintf("Output guard blocked: %s", result.Reason)}},
				IsError: boolPtr(true),
			}, nil
		}
		if result.ModifiedResult != nil {
			return &agent.AfterToolCallResult{
				Content: result.ModifiedResult.Content,
				Details: result.ModifiedResult.Details,
			}, nil
		}
	}

	return nil, nil
}

// AddRule adds a custom guard rule.
func (g *OutputGuard) AddRule(rule GuardRule) {
	g.rules = append(g.rules, rule)
}

// --- Default rules ---

func (g *OutputGuard) addDefaultRules() {
	// === PRE-EXECUTION RULES (BeforeToolCall) ===

	// Rule: Block reading of secret files
	g.rules = append(g.rules, GuardRule{
		Name:        "block-secrets",
		Description: "Blocks reading of files that commonly contain secrets",
		Phase:       PhaseBefore,
		Check: func(ctx GuardContext) GuardResult {
			if ctx.ToolName != "read" {
				return GuardResult{Allowed: true}
			}
			path, _ := ctx.Args["path"].(string)
			if path == "" {
				return GuardResult{Allowed: true}
			}

			basename := strings.ToLower(filepath.Base(path))
			secretFiles := map[string]bool{
				".env":                true,
				".env.local":          true,
				".env.production":     true,
				".env.staging":        true,
				".npmrc":              true,
				".pypirc":             true,
				"credentials.json":    true,
				"service-account.json": true,
			}

			if secretFiles[basename] {
				return GuardResult{
					Allowed: false,
					Reason:  fmt.Sprintf("Reading %q is blocked — this file may contain secrets", path),
				}
			}

			return GuardResult{Allowed: true}
		},
	})

	// Rule: Prevent writing to critical system paths
	g.rules = append(g.rules, GuardRule{
		Name:        "block-system-writes",
		Description: "Prevents writing to critical system directories",
		Phase:       PhaseBefore,
		Check: func(ctx GuardContext) GuardResult {
			if ctx.ToolName != "write" && ctx.ToolName != "edit" {
				return GuardResult{Allowed: true}
			}

			path, _ := ctx.Args["path"].(string)
			if path == "" {
				return GuardResult{Allowed: true}
			}

			absPath := path
			if !filepath.IsAbs(path) {
				absPath = filepath.Join(ctx.CWD, path)
			}

			blockedPrefixes := []string{
				"/etc/",
				"/usr/",
				"/bin/",
				"/sbin/",
				"/System/",
				"/Library/System/",
				"C:\\Windows\\",
				"C:\\Program Files\\",
			}

			normalized := filepath.ToSlash(absPath)
			for _, prefix := range blockedPrefixes {
				if strings.HasPrefix(normalized, prefix) {
					return GuardResult{
						Allowed: false,
						Reason:  fmt.Sprintf("Writing to %q is blocked — system directory", path),
					}
				}
			}

			return GuardResult{Allowed: true}
		},
	})

	// Rule: Prevent destructive bash commands
	g.rules = append(g.rules, GuardRule{
		Name:        "block-destructive-commands",
		Description: "Prevents destructive bash commands",
		Phase:       PhaseBefore,
		Check: func(ctx GuardContext) GuardResult {
			if ctx.ToolName != "bash" {
				return GuardResult{Allowed: true}
			}

			command, _ := ctx.Args["command"].(string)
			if command == "" {
				return GuardResult{Allowed: true}
			}

			blockedPatterns := []string{
				"rm -rf /",
				"rm -rf ~",
				"rm -rf /*",
				"mkfs.",
				"dd if=",
				"format ",
				"del /s /q C:\\",
				":(){ :|:& };:",
				"> /dev/sda",
				"chmod -R 777 /",
			}

			lowerCmd := strings.ToLower(command)
			for _, pattern := range blockedPatterns {
				if strings.Contains(lowerCmd, strings.ToLower(pattern)) {
					return GuardResult{
						Allowed: false,
						Reason:  fmt.Sprintf("Command blocked as potentially destructive: %s", pattern),
					}
				}
			}

			return GuardResult{Allowed: true}
		},
	})

	// === POST-EXECUTION RULES (AfterToolCall) ===

	// Rule: Redact API keys and tokens from tool output
	g.rules = append(g.rules, GuardRule{
		Name:        "redact-secrets",
		Description: "Redacts potential API keys and tokens from tool output",
		Phase:       PhaseAfter,
		Check: func(ctx GuardContext) GuardResult {
			modified := false
			var newContent []ai.Content

			for _, c := range ctx.Result.Content {
				txt, ok := c.(ai.TextContent)
				if !ok {
					newContent = append(newContent, c)
					continue
				}

				redacted := redactSecrets(txt.Text)
				if redacted != txt.Text {
					modified = true
					newContent = append(newContent, ai.TextContent{Text: redacted})
				} else {
					newContent = append(newContent, txt)
				}
			}

			if modified {
				result := ctx.Result
				result.Content = newContent
				return GuardResult{
					Allowed:        true,
					ModifiedResult: &result,
				}
			}
			return GuardResult{Allowed: true}
		},
	})

	// Rule: Truncate very large tool outputs
	g.rules = append(g.rules, GuardRule{
		Name:        "truncate-large-output",
		Description: "Truncates tool outputs that exceed a size limit",
		Phase:       PhaseAfter,
		Check: func(ctx GuardContext) GuardResult {
			const maxOutputChars = 50000

			for _, c := range ctx.Result.Content {
				if txt, ok := c.(ai.TextContent); ok {
					if len(txt.Text) > maxOutputChars {
						truncated := txt.Text[:maxOutputChars]
						truncated += fmt.Sprintf("\n\n... [output truncated: %d chars total, showing first %d]", len(txt.Text), maxOutputChars)
						result := ctx.Result
						result.Content = []ai.Content{ai.TextContent{Text: truncated}}
						return GuardResult{
							Allowed:        true,
							ModifiedResult: &result,
						}
					}
				}
			}
			return GuardResult{Allowed: true}
		},
	})
}

// redactSecrets replaces potential API keys and tokens in text with [REDACTED].
func redactSecrets(text string) string {
	patterns := []struct {
		prefix string
		length int
	}{
		{"sk-", 20},       // OpenAI
		{"sk-ant-", 24},   // Anthropic
		{"AIza", 35},      // Google
		{"ghp_", 36},      // GitHub
		{"gho_", 36},      // GitHub OAuth
		{"github_pat_", 82}, // GitHub fine-grained
		{"glpat-", 26},    // GitLab
		{"xai-", 20},      // xAI
		{"AKIA", 20},      // AWS Access Key
		{"ASIA", 20},      // AWS STS
	}

	result := text
	for _, p := range patterns {
		idx := strings.Index(result, p.prefix)
		for idx >= 0 {
			end := idx + len(p.prefix) + p.length
			if end > len(result) {
				end = len(result)
			}
			secret := result[idx:end]
			if isLikelyKey(secret[len(p.prefix):]) {
				result = result[:idx] + p.prefix + "[REDACTED]" + result[end:]
			}
			nextIdx := strings.Index(result[idx+1:], p.prefix)
			if nextIdx >= 0 {
				idx = idx + 1 + nextIdx
			} else {
				idx = -1
			}
		}
	}

	return result
}

// isLikelyKey checks if a string looks like it could be an API key.
func isLikelyKey(s string) bool {
	if len(s) < 8 {
		return false
	}
	alnum := 0
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			alnum++
		}
	}
	return alnum > len(s)*2/3
}

// InitBeforeHook returns a BeforeToolCall hook for pre-execution safety checks.
func InitBeforeHook(cwd string) func(ctx context.Context, callCtx agent.BeforeToolCallContext) (*agent.BeforeToolCallResult, error) {
	guard := NewOutputGuard(cwd)
	return guard.BeforeToolCall
}

// InitAfterHook returns an AfterToolCall hook for post-execution redaction.
func InitAfterHook(cwd string) func(ctx context.Context, callCtx agent.AfterToolCallContext) (*agent.AfterToolCallResult, error) {
	guard := NewOutputGuard(cwd)
	return guard.AfterToolCall
}

// CheckBefore is a test helper that runs before-execution rules.
func (g *OutputGuard) CheckBefore(ctx GuardContext) GuardResult {
	for _, rule := range g.rules {
		if rule.Phase != PhaseBefore {
			continue
		}
		result := rule.Check(ctx)
		if !result.Allowed {
			return result
		}
	}
	return GuardResult{Allowed: true}
}

// CheckAfter is a test helper that runs after-execution rules.
func (g *OutputGuard) CheckAfter(ctx GuardContext) GuardResult {
	for _, rule := range g.rules {
		if rule.Phase != PhaseAfter {
			continue
		}
		result := rule.Check(ctx)
		if !result.Allowed || result.ModifiedResult != nil {
			return result
		}
	}
	return GuardResult{Allowed: true}
}

func boolPtr(b bool) *bool {
	return &b
}

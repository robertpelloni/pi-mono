package toolguardrail

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "strings"
)

// ToolGuardrailConfig holds configurable thresholds for tool‑call loop detection.
type ToolGuardrailConfig struct {
    WarningsEnabled            bool
    HardStopEnabled            bool
    ExactFailureWarnAfter      int
    ExactFailureBlockAfter     int
    SameToolFailureWarnAfter   int
    SameToolFailureHaltAfter  int
    NoProgressWarnAfter        int
    NoProgressBlockAfter       int
    IdempotentTools           map[string]struct{}
    MutatingTools             map[string]struct{}
}

// NewDefaultConfig returns a Config with sensible defaults.
func NewDefaultConfig() *ToolGuardrailConfig {
    // Populate default tool sets (subset of the Python constants).
    idempotent := []string{"read_file", "search_files", "web_search", "web_extract", "session_search"}
    mutating := []string{"terminal", "execute_code", "write_file", "patch", "todo", "memory"}
    idem := make(map[string]struct{})
    for _, n := range idempotent {
        idem[n] = struct{}{}
    }
    mut := make(map[string]struct{})
    for _, n := range mutating {
        mut[n] = struct{}{}
    }
    return &ToolGuardrailConfig{
        WarningsEnabled:           true,
        HardStopEnabled:           false,
        ExactFailureWarnAfter:     2,
        ExactFailureBlockAfter:    5,
        SameToolFailureWarnAfter:  3,
        SameToolFailureHaltAfter:  8,
        NoProgressWarnAfter:       2,
        NoProgressBlockAfter:      5,
        IdempotentTools:           idem,
        MutatingTools:             mut,
    }
}

// ToolCallSignature is a stable identifier for a tool name + canonical args.
type ToolCallSignature struct {
    ToolName string
    ArgsHash string
}

// FromCall creates a signature from tool name and args.
func FromCall(toolName string, args map[string]interface{}) ToolCallSignature {
    canonical := canonicalToolArgs(args)
    h := sha256.Sum256([]byte(canonical))
    return ToolCallSignature{ToolName: toolName, ArgsHash: hex.EncodeToString(h[:])}
}

func (s ToolCallSignature) ToMetadata() map[string]string {
    return map[string]string{"tool_name": s.ToolName, "args_hash": s.ArgsHash}
}

// ToolGuardrailDecision represents a decision from the guardrail controller.
type ToolGuardrailDecision struct {
    Action    string // allow | warn | block | halt
    Code      string
    Message   string
    ToolName  string
    Count     int
    Signature *ToolCallSignature
}

func (d ToolGuardrailDecision) AllowsExecution() bool { return d.Action == "allow" || d.Action == "warn" }
func (d ToolGuardrailDecision) ShouldHalt() bool { return d.Action == "block" || d.Action == "halt" }

func (d ToolGuardrailDecision) ToMetadata() map[string]interface{} {
    m := map[string]interface{}{"action": d.Action, "code": d.Code, "message": d.Message, "tool_name": d.ToolName, "count": d.Count}
    if d.Signature != nil {
        m["signature"] = d.Signature.ToMetadata()
    }
    return m
}

// ToolCallGuardrailController tracks tool calls per turn.
type ToolCallGuardrailController struct {
    config                 *ToolGuardrailConfig
    exactFailureCounts     map[ToolCallSignature]int
    sameToolFailureCounts  map[string]int
    noProgress             map[ToolCallSignature]struct{ resultHash string; count int }
    haltDecision            *ToolGuardrailDecision
}

func NewController(cfg *ToolGuardrailConfig) *ToolCallGuardrailController {
    if cfg == nil {
        cfg = NewDefaultConfig()
    }
    return &ToolCallGuardrailController{config: cfg, exactFailureCounts: make(map[ToolCallSignature]int), sameToolFailureCounts: make(map[string]int), noProgress: make(map[ToolCallSignature]struct{ resultHash string; count int })}
}

func (c *ToolCallGuardrailController) ResetForTurn() {
    c.exactFailureCounts = make(map[ToolCallSignature]int)
    c.sameToolFailureCounts = make(map[string]int)
    c.noProgress = make(map[ToolCallSignature]struct{ resultHash string; count int })
    c.haltDecision = nil
}

func (c *ToolCallGuardrailController) HaltDecision() *ToolGuardrailDecision { return c.haltDecision }

// BeforeCall is invoked before a tool executes.
func (c *ToolCallGuardrailController) BeforeCall(toolName string, args map[string]interface{}) ToolGuardrailDecision {
    sig := FromCall(toolName, args)
    if !c.config.HardStopEnabled {
        return ToolGuardrailDecision{Action: "allow", ToolName: toolName, Signature: &sig}
    }
    // Hard‑stop logic for exact failures is handled in AfterCall.
    // Idempotent no‑progress check.
    if c.isIdempotent(toolName) {
        if rec, ok := c.noProgress[sig]; ok && rec.count >= c.config.NoProgressBlockAfter {
            d := ToolGuardrailDecision{Action: "block", Code: "idempotent_no_progress_block", Message: fmt.Sprintf("Blocked %s: identical result repeated %d times", toolName, rec.count), ToolName: toolName, Count: rec.count, Signature: &sig}
            c.haltDecision = &d
            return d
        }
    }
    return ToolGuardrailDecision{Action: "allow", ToolName: toolName, Signature: &sig}
}

// AfterCall is invoked after a tool finishes.
func (c *ToolCallGuardrailController) AfterCall(toolName string, args map[string]interface{}, result string, failedPtr *bool) ToolGuardrailDecision {
    sig := FromCall(toolName, args)
    // Determine failure status if not provided.
    failed := false
    if failedPtr != nil {
        failed = *failedPtr
    } else {
        failed, _ = classifyToolFailure(toolName, result)
    }
    if failed {
        // Exact failure count.
        count := c.exactFailureCounts[sig] + 1
        c.exactFailureCounts[sig] = count
        // Reset no‑progress for this signature.
        delete(c.noProgress, sig)
        // Same‑tool failure count.
        sameCount := c.sameToolFailureCounts[toolName] + 1
        c.sameToolFailureCounts[toolName] = sameCount
        // Hard‑stop on same‑tool failures.
        if c.config.HardStopEnabled && sameCount >= c.config.SameToolFailureHaltAfter {
            d := ToolGuardrailDecision{Action: "halt", Code: "same_tool_failure_halt", Message: fmt.Sprintf("Stopped %s: failed %d times", toolName, sameCount), ToolName: toolName, Count: sameCount, Signature: &sig}
            c.haltDecision = &d
            return d
        }
        // Warnings for exact failures.
        if c.config.WarningsEnabled && count >= c.config.ExactFailureWarnAfter {
            return ToolGuardrailDecision{Action: "warn", Code: "repeated_exact_failure_warning", Message: fmt.Sprintf("%s has failed %d times with identical arguments", toolName, count), ToolName: toolName, Count: count, Signature: &sig}
        }
        // Warnings for repeated same‑tool failures.
        if c.config.WarningsEnabled && sameCount >= c.config.SameToolFailureWarnAfter {
            return ToolGuardrailDecision{Action: "warn", Code: "same_tool_failure_warning", Message: toolFailureRecoveryHint(toolName, sameCount), ToolName: toolName, Count: sameCount, Signature: &sig}
        }
        // Default failure response.
        return ToolGuardrailDecision{Action: "allow", ToolName: toolName, Count: count, Signature: &sig}
    }
    // Success path – clear failure counters.
    delete(c.exactFailureCounts, sig)
    delete(c.sameToolFailureCounts, toolName)
    // Idempotent no‑progress tracking.
    if c.isIdempotent(toolName) {
        resultHash := resultHash(result)
        rec := c.noProgress[sig]
        if rec.resultHash == resultHash {
            rec.count++
        } else {
            rec.resultHash = resultHash
            rec.count = 1
        }
        c.noProgress[sig] = rec
        if c.config.WarningsEnabled && rec.count >= c.config.NoProgressWarnAfter {
            return ToolGuardrailDecision{Action: "warn", Code: "idempotent_no_progress_warning", Message: fmt.Sprintf("%s returned the same result %d times", toolName, rec.count), ToolName: toolName, Count: rec.count, Signature: &sig}
        }
    } else {
        delete(c.noProgress, sig)
    }
    return ToolGuardrailDecision{Action: "allow", ToolName: toolName, Signature: &sig}
}

func (c *ToolCallGuardrailController) isIdempotent(name string) bool {
    _, mut := c.config.MutatingTools[name]
    if mut {
        return false
    }
    _, idem := c.config.IdempotentTools[name]
    return idem
}

// ---- Helper functions ----

// canonicalToolArgs returns stable JSON for args.
func canonicalToolArgs(args map[string]interface{}) string {
    // Sort keys for stability.
    b, _ := json.Marshal(args)
    var m map[string]interface{}
    _ = json.Unmarshal(b, &m)
    // Re‑marshal with sorted keys via Encoder.
    buf, _ := json.Marshal(m)
    return string(buf)
}

func classifyToolFailure(toolName, result string) (bool, string) {
    lower := strings.ToLower(result)
    if strings.Contains(lower, "error") || strings.Contains(lower, "failed") || strings.HasPrefix(strings.TrimSpace(lower), "error") {
        return true, ""
    }
    return false, ""
}

func resultHash(res string) string {
    h := sha256.Sum256([]byte(res))
    return hex.EncodeToString(h[:])
}

func toolFailureRecoveryHint(toolName string, count int) string {
    common := fmt.Sprintf("%s has failed %d times this turn. This looks like a loop. ", toolName, count)
    if toolName == "terminal" {
        return common + "Run a small diagnostic such as `pwd && ls -la` then try a simpler command."
    }
    return common + "Try different arguments or a different tool."
}

// SyntheticResult returns a JSON string representing a blocked tool call.
func SyntheticResult(decision ToolGuardrailDecision) string {
    data := map[string]interface{}{"error": decision.Message, "guardrail": decision.ToMetadata()}
    b, _ := json.Marshal(data)
    return string(b)
}

// AppendGuidance adds a warning/halt suffix to an existing tool result.
func AppendGuidance(result string, decision ToolGuardrailDecision) string {
    if decision.Action != "warn" && decision.Action != "halt" {
        return result
    }
    label := "Tool loop warning"
    if decision.Action == "halt" {
        label = "Tool loop hard stop"
    }
    suffix := fmt.Sprintf("\n\n[%s: %s; count=%d; %s]", label, decision.Code, decision.Count, decision.Message)
    if result == "" {
        return suffix
    }
    return result + suffix
}

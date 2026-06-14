package toolexecutor

import (
    "fmt"
    "log"
    "time"

    "github.com/badlogic/pi-mono/pkg/toolguardrail"
)

// ToolFunc is the signature for a tool implementation.
// It receives arguments as a map and returns a JSON string result or an error.
type ToolFunc func(args map[string]interface{}) (string, error)

// Registry holds named tools.
type Registry struct {
    tools map[string]ToolFunc
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry { return &Registry{tools: make(map[string]ToolFunc)} }

// Register adds a tool implementation.
func (r *Registry) Register(name string, fn ToolFunc) { r.tools[name] = fn }

// Get retrieves a tool implementation.
func (r *Registry) Get(name string) (ToolFunc, bool) { fn, ok := r.tools[name]; return fn, ok }

// ToolCall represents a single tool invocation request.
type ToolCall struct {
    Name string
    Args map[string]interface{}
}

// Execute runs a slice of tool calls sequentially, applying guardrails.
// It returns a slice of result strings (one per tool) and stops early if a hard stop is triggered.
func Execute(calls []ToolCall, reg *Registry, guard *toolguardrail.ToolCallGuardrailController) ([]string, error) {
    results := []string{}
    for _, call := range calls {
        // Pre‑call guardrail check.
        decision := guard.BeforeCall(call.Name, call.Args)
        if decision.Action != "allow" && decision.Action != "warn" {
            // Block or halt – return synthetic result.
            synthetic := toolguardrail.SyntheticResult(decision)
            results = append(results, synthetic)
            if decision.ShouldHalt() {
                return results, fmt.Errorf("guardrail halted execution: %s", decision.Message)
            }
            continue
        }

        // Execute the tool.
        fn, ok := reg.Get(call.Name)
        var result string
        var err error
        start := time.Now()
        if ok {
            result, err = fn(call.Args)
        } else {
            err = fmt.Errorf("unknown tool %s", call.Name)
        }
        duration := int(time.Since(start).Milliseconds())

        // Determine failure.
        var failed bool
        if err != nil {
            failed = true
            // Include error message in result JSON.
            result = fmt.Sprintf("{\"error\": %q}", err.Error())
        } else {
            failed = false
        }

        // Post‑call guardrail.
        postDecision := guard.AfterCall(call.Name, call.Args, result, &failed)
        // Attach guidance if needed.
        if postDecision.Action == "warn" || postDecision.Action == "halt" {
            result = toolguardrail.AppendGuidance(result, postDecision)
        }
        // Log.
        log.Printf("tool %s executed in %dms, decision %s", call.Name, duration, postDecision.Action)

        results = append(results, result)
        if postDecision.ShouldHalt() {
            return results, fmt.Errorf("guardrail halted after tool %s: %s", call.Name, postDecision.Message)
        }
    }
    return results, nil
}

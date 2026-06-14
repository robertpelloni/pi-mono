package tooldispatch

import (
    "regexp"
)

// Sets of tool names mirroring Hermes‑agent definitions.
var (
    // Tools that must never run in parallel (interactive / user‑facing).
    neverParallelTools = map[string]struct{}{
        "clarify": {},
    }
    // Read‑only tools safe for parallel execution.
    parallelSafeTools = map[string]struct{}{
        "ha_get_state":    {},
        "ha_list_entities": {},
        "ha_list_services": {},
        "read_file":        {},
        "search_files":     {},
        "session_search":  {},
        "skill_view":      {},
        "skills_list":     {},
        "vision_analyze":  {},
        "web_extract":     {},
        "web_search":      {},
    }
    // Tools that operate on paths and can be parallel if targets are independent.
    pathScopedTools = map[string]struct{}{
        "read_file":  {},
        "write_file": {},
        "patch":      {},
    }
)

// IsNeverParallel returns true if the tool must never be run in parallel.
func IsNeverParallel(name string) bool {
    _, ok := neverParallelTools[name]
    return ok
}

// IsParallelSafe returns true if the tool is read‑only and safe for parallel execution.
func IsParallelSafe(name string) bool {
    _, ok := parallelSafeTools[name]
    return ok
}

// IsPathScoped returns true if the tool operates on a file path and can be parallel when paths don't overlap.
func IsPathScoped(name string) bool {
    _, ok := pathScopedTools[name]
    return ok
}

// Heuristic patterns for destructive terminal commands.
var (
    destructivePatterns = regexp.MustCompile(`(?i)(?:^|\s|&&|\|\||;|` + "`" + `)(?:rm\s|rmdir\s|cp\s|install\s|mv\s|sed\s+-i|truncate\s|dd\s|shred\s|git\s+(?:reset|clean|checkout)\s)`)
    // Output redirects that overwrite files (single >, not >>)
    redirectOverwrite = regexp.MustCompile(`[^>]>[^>]|^>[^>]`)
)

// IsDestructiveCommand returns true if the command looks like it modifies or deletes files.
func IsDestructiveCommand(cmd string) bool {
    return destructivePatterns.MatchString(cmd) || redirectOverwrite.MatchString(cmd)
}

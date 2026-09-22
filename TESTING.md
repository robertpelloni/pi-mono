# Testing & Parity Validation Guide

This document outlines the standard procedures for validating the unified execution harness, with a specific focus on ensuring exact parity with integrated external tools (Claude Code, Gemini CLI, Codex, etc.).

## 1. Local Staging Validation

Before submitting changes to core handlers or tool schemas, you must validate them against a live local instance.

1.  **Spin up the staging instance:**
    ```bash
    PI_AGENT_DIR=.pi-staging ./pi --port 8081
    ```
    *Note: This isolates the agent's memory and workspace from your primary environment.*

2.  **Execute the Parity Suite:**
    ```bash
    ./scripts/parity-check.sh --endpoint http://localhost:8081
    ```

## 2. Core Security & Edge Cases

When contributing to `pkg/ai/clean_room_handlers.go` or `pkg/ai/security.go`, you must explicitly verify the following constraints:

### A. Directory Boundary Enforcement (`safePath`)
All direct file manipulation tools (e.g., `read_file`, `write_to_file`, `replace_lines`, `Amp_ReadFile`) **must** pass their target paths through `validatePath()`.
- **Validation:** Attempt to read or write to `../../etc/passwd` via the tool interface. The harness must gracefully return a `security violation` string error, not a Go panic or fatal crash.

### B. URL Validation & Injection Prevention
Browser tools (e.g., `browser_action`, `web_search`) accept raw URLs.
- **Validation:** The underlying handler must use `os/exec` with explicit parameter arguments (e.g., `exec.Command("lynx", "-dump", url)`).
- **NEVER** pass URLs directly into a shell string (e.g., `bash -c "lynx " + url`), as this opens the harness to severe shell injection attacks via `;` or `&&` terminators.

### C. Multi-File Edits
Complex handlers like `multiedit` or `apply_patch` must be tested across multiple directory depths to ensure relative path resolution remains intact during atomic file writes.

## 3. Go Test Suite

Always execute the native test suite to ensure schema routing and fallback mechanisms (like the Community Plugin Hub) remain intact:
```bash
go test ./pkg/ai/... -v
go test ./pkg/plugins/... -v
```

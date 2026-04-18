# Handoff Document

## Current Status
- Created comprehensive LLM instructions and workflow documentation (`VISION.md`, `ROADMAP.md`, `TODO.md`, `SUBMODULES.md`, `IDEAS.md`).
- Added the target submodules (`aider`, `hermes-agent`, `open-interpreter`, `goose`, `vscode-copilot-release`, `ollama`, `ii-agent`).
- Implemented "Total Assimilation" 1:1 Clean Room tools inside TypeScript.
- The `net/http` Server-Sent Events stream implementations for Go (`pkg/ai/openai.go`, `anthropic.go`, `google.go`) are fully complete. They successfully parse text blocks and tool JSON blocks, handle context cancellation natively (via `ctx context.Context`), implement exponential backoff retry loops, and serialize full multi-turn conversational history including tool results.
- The execution loop (`runLoop` in `pkg/agent/agent.go`) is complete! When `finalMsg` contains a `ToolCall`, the agent matches it against `a.tools`, invokes the Go `Execute` function (which leverages the `FileMutationQueue`), streams `EventToolExecutionStart` and `EventToolExecutionEnd` updates, and appends the result as a `ToolResultMessage` to `a.messages` before recursively continuing the turn.
- Created `pkg/tools/tools.go` scaffolding the `read`, `write`, and `bash` tools for the native Go client.

## Urgent Constraints & Next Steps
1. **Flesh out Go Native Tools (Phase 4)**: The `pkg/tools/tools.go` currently implements naive versions of `read`, `bash`, and `write`. You must port the robust, battle-tested logic from the TypeScript implementations (`packages/coding-agent/src/core/tools/read.ts`, `write.ts`, `bash.ts`) to Go.
   - For example, `read` should handle byte/line truncation limits (`DEFAULT_MAX_BYTES`), offsets, and image resizing natively if `autoResizeImages` is set, matching TS behavior.
   - `write` should use atomic writes or standard temp-file moving.
   - `bash` should use PTYs if possible (e.g. `creack/pty`) to capture interactive output, or at least stream output robustly.
2. **Move to TUI (Phase 5)**: After tools are robust, begin porting the TUI library (`@mariozechner/pi-tui`) using `charmbracelet/bubbletea` or standard raw termios logic to match the TS interactive prompt exactly.

## Strict Rules
- *Never execute commands that taskkill all node processes.*
- Do not commit changes inside the `submodules/` folders directly unless specifically adding a patch upstream.

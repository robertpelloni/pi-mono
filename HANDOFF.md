# Handoff Document

## Current Status
- Created comprehensive LLM instructions and workflow documentation (`VISION.md`, `ROADMAP.md`, `TODO.md`, `SUBMODULES.md`, `IDEAS.md`).
- Added the target submodules (`aider`, `hermes-agent`, `open-interpreter`, `goose`, `vscode-copilot-release`, `ollama`, `ii-agent`).
- Implemented "Total Assimilation" 1:1 Clean Room tools inside TypeScript.
- The `net/http` Server-Sent Events stream implementations for Go (`pkg/ai/openai.go`, `anthropic.go`, `google.go`) are fully complete. They successfully parse text blocks and tool JSON blocks, handle context cancellation, implement exponential backoff retry loops, and serialize full multi-turn conversational history including tool results.
- `pkg/agent/agent.go` now correctly accumulates streaming events (`EventTextDelta`, `EventToolCallDelta`) into a `finalMsg`.
- The execution loop (`runLoop` in `pkg/agent/agent.go`) is complete! When `finalMsg` contains a `ToolCall`, the agent matches it against `a.tools`, invokes the Go `Execute` function (which leverages the `FileMutationQueue`), streams `EventToolExecutionStart` and `EventToolExecutionEnd` updates, and appends the result as a `ToolResultMessage` to `a.messages` before recursively continuing the turn.

## Urgent Constraints & Next Steps
1. **Fix Context Cancellation Goroutine Leak (Phase 3.1)**: See `TODO.md`. The `runLoop` cancels context gracefully, but the stream goroutines in the `pkg/ai` providers block forever trying to send events if the loop has exited. Fix this by passing `ctx context.Context` explicitly to `StreamFunction` and tying it directly into the `http.NewRequestWithContext`.
2. **Move to Phase 4 (CLI & Native Implementations)**: Start porting the core tool definitions natively to Go inside `cmd/pi/` or a new `pkg/tools/` directory (e.g. implementing the robust logic for `read`, `bash`, `edit`, `write` identical to `packages/coding-agent/src/core/tools/`).

## Strict Rules
- *Never execute commands that taskkill all node processes.*
- Do not commit changes inside the `submodules/` folders directly unless specifically adding a patch upstream.

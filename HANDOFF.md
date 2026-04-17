# Handoff Document

## Current Status
- Created comprehensive LLM instructions and workflow documentation (`VISION.md`, `ROADMAP.md`, `TODO.md`, `SUBMODULES.md`, `IDEAS.md`).
- Added the target submodules (`aider`, `hermes-agent`, `open-interpreter`, `goose`, `vscode-copilot-release`, `ollama`, `ii-agent`).
- Implemented "Total Assimilation" 1:1 Clean Room tools inside TypeScript.
- The `net/http` Server-Sent Events stream implementations for Go (`pkg/ai/openai.go`, `anthropic.go`, `google.go`) are complete, context-cancelable, and implement exponential backoff retry loops.
- Resolved the critical code review bug: The Go API chunks now successfully parse Tool Call JSON payloads and emit them via the `EventToolCallStart` and `EventToolCallDelta` channels.

## Urgent Constraints & Next Steps
1. **Move to Agent Core Porting (Phase 3)**: Now that the primary API package (`pkg/ai/`) correctly handles data types, registries, and SSE streams, you must begin porting `@mariozechner/pi-agent-core` to Go.
   - This package handles message state, context window limits, token budget calculations, and executing tools based on provider callbacks.
   - Start by porting the `FileMutationQueue` mechanics to Go, ensuring destructive operations behave in a thread-safe manner, analogous to the TS version.

## Strict Rules
- *Never execute commands that taskkill all node processes.*
- Do not commit changes inside the `submodules/` folders directly unless specifically adding a patch upstream.

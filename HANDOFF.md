# Handoff Document

## Current Status
- Created comprehensive LLM instructions and workflow documentation (`VISION.md`, `ROADMAP.md`, `TODO.md`, `SUBMODULES.md`, `IDEAS.md`).
- Added the target submodules (`aider`, `hermes-agent`, `open-interpreter`, `goose`, `vscode-copilot-release`, `ollama`, `ii-agent`).
- Implemented "Total Assimilation" 1:1 Clean Room tools inside TypeScript. The TypeScript architecture for mapping strict external tool schemas (like `claude_code_read_file`, `hermes_terminal`, `aider_replace_lines`, `open_interpreter_execute`) is fully established and wired into the `coding-agent` via `packages/coding-agent/src/core/tools/clean-room-schemas.ts` and `clean-room-tools.ts`. They are exposed dynamically through `allToolDefinitions` and `createAllTools`.

## Urgent Constraints & Next Steps
1. **Next Primary Task (Go Streaming Implementation):** With the TypeScript 'clean-room' schemas successfully wired up and compiling natively, the blocker has been resolved. The next session MUST resume the Go port by implementing the actual HTTP streaming client logic (using `net/http` and SSE parsing) inside `pkg/ai/openai.go`, `pkg/ai/anthropic.go`, and `pkg/ai/google.go`. The interfaces and channels (`AssistantMessageEventStream`) are already in place, but they currently return empty/stub streams.
2. **Next Secondary Task (Go File Mutations):** Port the TypeScript FileMutationQueue and `fs` destructive operations to Go (for schemas like `write_file` or `patch`), ensuring that they behave thread-safely via mutex locks similar to the TS version.

## Strict Rules
- *Never execute commands that taskkill all node processes.*
- Do not commit changes inside the `submodules/` folders directly unless specifically adding a patch upstream.

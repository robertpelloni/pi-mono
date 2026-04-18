# Handoff Document

## Current Status
- Created comprehensive LLM instructions and workflow documentation (`VISION.md`, `ROADMAP.md`, `TODO.md`, `SUBMODULES.md`, `IDEAS.md`).
- Added the target submodules (`aider`, `hermes-agent`, `open-interpreter`, `goose`, `vscode-copilot-release`, `ollama`, `ii-agent`).
- Implemented "Total Assimilation" 1:1 Clean Room tools inside TypeScript.
- The `net/http` Server-Sent Events stream implementations for Go (`pkg/ai/openai.go`, `anthropic.go`, `google.go`) are fully complete. They successfully parse text blocks and tool JSON blocks, handle context cancellation, implement exponential backoff retry loops, and serialize full multi-turn conversational history including tool results.
- Resolved the critical code review bug: `pkg/agent/agent.go` now correctly accumulates streaming events (`EventTextDelta`, `EventToolCallDelta`) into a `finalMsg`, which is appended to the agent's state memory upon turn completion.

## Urgent Constraints & Next Steps
1. **Tool Execution Logic (Phase 3)**: The agent loop (`runLoop` in `pkg/agent/agent.go`) successfully streams and parses tool calls from the AI providers, but currently stops and emits `EventTurnEnd` after generation.
   - The next session MUST implement the execution loop: When `finalMsg` contains a `ToolCall`, the agent must match it against `a.tools`, invoke the Go `Execute` function (which leverages the `FileMutationQueue`), and append the result as a `ToolResultMessage` to `a.messages` before recursively calling the provider again to continue the turn.
   - Look at `packages/agent/src/agent-loop.ts` for the TypeScript reference on how tool execution phases are structured (e.g. parallel vs sequential execution).

## Strict Rules
- *Never execute commands that taskkill all node processes.*
- Do not commit changes inside the `submodules/` folders directly unless specifically adding a patch upstream.

# TODO

This document contains individual features, bug fixes, and other fine details that need to be solved/implemented in the short term.

## Short-term Action Items

- [x] Add all specified submodules (`aider`, `goose`, `open-interpreter`, `hermes-agent`, etc).
- [x] Port `packages/ai/src/types.ts` to Go interfaces and structs.
- [x] Port basic OpenAI stream provider to Go.
- [x] Port basic Anthropic stream provider to Go.
- [x] Port basic Google Gemini stream provider to Go.
- [x] Port global models JSON registry logic to Go.
- [x] Port internal tool structures for generic providers.
- [x] Port context execution loops (`pkg/agent`).
- [x] Create testing harness in Go similar to the TypeScript suite.
- [ ] Set up CI/CD workflows for the new Go project.

## Submodule Tool Assimilation (Phase 6)
- [x] Goose (`developer__shell`, `recipe__final_output`)
- [x] Aider (`replace_lines`, `run_command`)
- [x] Copilot (`vscode_read`)
- [x] Claude Code (`read_file`, `bash`)
- [ ] Open Interpreter (Extract specific OS control modules natively into Go)
- [ ] Hermes Agent (Extract browser controls, home assistant, MOA, memory natively)

# Crucial Code Review Fixes (Next Session)
1. **Fix Missing Tool Registration**: The TypeScript `clean-room-tools.ts` and `clean-room-schemas.ts` were built, but they need to be actively exported from `packages/coding-agent/src/core/tools/index.ts` in order to be functionally available to the user, not just exist as dead code. Ensure they are mapped to `allToolDefinitions` and `allTools`.
2. **Remove Node Scripts**: If any leftover `.cjs` scripts exist from previous automation (e.g. `patch_agent_hooks.cjs`), remove them explicitly. Do not commit scratchpad scripts into the repo.

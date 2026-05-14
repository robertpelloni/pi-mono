# Memory & Observations

This document contains ongoing observations about the codebase and design preferences.

[PROJECT_MEMORY]
## Project Vision & Architecture
This project, `pi-mono`, is an ambitious effort labeled "Total Assimilation". It aims to build the most comprehensive and functional AI coding agent ecosystem. Originally developed as a complex TypeScript monorepo (`packages/ai`, `packages/coding-agent`, `packages/tui`, etc.), the project is currently undergoing a massive systematic rewrite into a robust Go architecture.

### Key Goals
- **Go Port:** Transition the entire TypeScript monorepo into Go while maintaining strict 1:1 feature parity. The Go port is organized into `/cmd/pi` (for the CLI app) and `/pkg/...` (for libraries).
- **Tool Parity ("Clean Room"):** Implement "clean-room" versions of tools used by over 30 leading AI coding agents (Aider, Claude Code, Cursor, Copilot, etc.). The schemas expected by models like `claude-3.5-sonnet` must map precisely to internal execution handlers.
- **Multiple Native Frontends:** Support Terminal UI (using `bubbletea` in Go, or `@mariozechner/pi-tui` in TS) and Web UI (`pkg/server` in Go serves static frontend assets).
- **Submodule Integration:** Third-party agents are cloned into `submodules/` (e.g. `goose`, `aider`, `cline`, `codebuff`) for study and feature extraction.

### Execution Loop & State
The agent interacts with LLMs via Server-Sent Events (SSE). It captures tool execution requests, maps them to native host OS operations (file read/write, bash execution, AST analysis via `grep`, GUI interaction via `xdotool` logic port), and synchronously maintains state in an `AgentContext`. A `FileMutationQueue` manages concurrent modifications to the same file.

### Testing & CI
- **TypeScript:** Uses `vitest` mapped via npm workspaces (`npm test --workspaces`). CI tests run node tasks, but some TUI terminal layout tests are notoriously flaky and have been explicitly mocked out.
- **Go:** Follows standard `go test` paradigms natively mirroring TS functionality. Excludes submodules from unit testing.
- The `.github/workflows` run on every PR and main branch push. There are multiple pipelines dictating cross-compilation binaries building and matrix verifications.

### Documentation Workflow
The system strictly enforces maintaining documentation files (`ROADMAP.md`, `TODO.md`, `CHANGELOG.md`, `VISION.md`, `HANDOFF.md`, `SUBMODULES.md`, `SUBMODULE_INVENTORY.md`) constantly in sync with every iterative step. Each session explicitly leaves breadcrumbs for parallel or future agents inside `HANDOFF.md`.

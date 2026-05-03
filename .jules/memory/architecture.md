*   **Goal & Status:** The primary objective is porting the TypeScript AI agent project to a robust Go backend while preserving 1:1 feature parity. We are currently finalizing Phase 14 with a cross-platform compilation `Makefile`.
*   **Architecture & Patterns:**
    *   **Monorepo Structure:** Contains both the original TypeScript implementation (under `packages/`) and the new Go port (`pkg/`, `cmd/pi/`).
    *   **Agent Core (`pkg/agent`):** Utilizes an event-driven loop with channels. Event listeners are registered via `Subscribe` and executed via `emit`. Supports hook interruptions like `BeforeToolCall` and `AfterToolCall`.
    *   **Community Plugins:** Housed in `pkg/extensions/`. They are configured to follow an opt-in pattern using an `Enabled` flag. Tools are added dynamically via an `AddTools` receiver, and logic hooks are integrated into the `agent.AgentLoopConfig`.
*   **Recent Decisions & Fixes:**
    *   **Deployment Hooks:** Scaffolded a universal `Makefile` to allow users to type `make build-all` and output statically linked AMD64/ARM64 binaries across Linux, Windows, and Darwin.
    *   All plugin integrations were fully merged to `main`, passing all unit and build tests!
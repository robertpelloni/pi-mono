# MEMORY

## Architectural Observations
- The project has successfully moved from a TS monorepo to a unified Go project.
- A **Unified Tool Harness** (`pkg/ai/harness.go`) manages the routing of assimilated tool schemas (Tabby, Warp) to native Go implementations.
- External Integrations served as temporary "clean room" references to achieve tool parity with existing agents (Aider, Cline, Hermes, etc.).
- `pkg/ai/clean_room_handlers.go` and `pkg/ai/clean_room_tools.go` are the core of our parity effort, mapping specific schemas to internal implementations.

## Design Preferences
- **Minimal Core:** The core Go application should remain lean, delegating complex or optional behaviors to the extension system.
- **Parity is Priority:** 1:1 matching of tool schemas (input/output) is critical for models trained on specific agent harnesses.
- **Local-First:** Native execution and local model support (Ollama) are prioritized over cloud-only dependencies.

## Post-Assimilation State (v0.97.0)
- All 11 initial reference submodules (Aider, Cline, etc.) and 6 Phase 19 submodules (Tabby, Warp, etc.) have been assimilated and removed.
- The project is now a self-contained Go ultra-project with 100% functional tool parity for the target agents.
- **Unified Tool Harness (`pkg/ai/harness.go`)**: Manages routing of Tabby, Warp, and Wave API requests to native handlers.
- **RepoMap (`pkg/repomap`)**: Internalized symbol extraction and ranking for optimized context usage.
- **Terminal Parity**: Go TUI supports OSC 8/133 sequences for modern terminal feature matching.
- **Security Logic**: `pkg/ai/security.go` implements prefix-overlap protected path validation using `filepath.Rel`.
- **Binary Naming**: The project core is consolidated into the `pi` binary (from `cmd/pi`), deprecating the `pi-agent` and `pi-server` names for a unified interface.

- Phase 20 (Extended Assimilation) tools (Claude Code, Gemini, etc.) have been integrated directly into the core runtime.

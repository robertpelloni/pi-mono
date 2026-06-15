# Project Vision

This document extensively describes and outlines in full detail the ultimate goal and design of the project.

## Ultimate Goal

The ultimate vision is to create the most comprehensive, robust, and functional open-source AI coding agent ecosystem. The project has successfully transitioned from a TypeScript monorepo to a **Go-first architecture**, with the Go implementation now serving as the primary codebase and the TypeScript version maintained for compatibility and reference only.

We have assimilated the best features, architectures, and functionalities from over 30 leading AI tools, CLIs, and IDE extensions into a single unified platform. This is achieved through our **Unified Tool Harness** (Phase 19), a modular Go-based engine that routes specialized API requests (Tabby, Warp, etc.) to native handlers while ensuring 100% tool parity. The harness provides a clean, extensible interface for tool execution with built-in safety guards, atomic file operations, and deterministic output formatting.

### Success Milestones Achieved

- ✅ **Complete Go Port**: All core functionality from TypeScript has been ported to Go with feature parity
- ✅ **Submodule Assimilation**: Successfully integrated and removed submodules including `aider`, `opencode-cli`, `cursor`, `windsurf`, and others
- ✅ **Unified Tool Harness**: Production-ready tool execution engine with clean room implementations
- ✅ **Native Frontends**: Bubble tea TUI, CLI, and web UI all running on Go backend
- ✅ **Session Management**: Complete tree-based session architecture with branching and compaction
- ✅ **Multi-Provider Support**: Seamless integration with all major LLM providers (OpenAI, Anthropic, Google, Ollama, etc.)

## Core Design Principles

1.  **Go First**: The entire system is built on a high-performance Go foundation. TypeScript is now maintained only for reference and compatibility; all new development occurs in Go.
2.  **Total Assimilation**: We have systematically analyzed and merged capabilities from competing projects (Aider, Claude Code, Cursor, Windsurf, etc.) into our core. Successfully assimilated submodules are removed from the repository once parity is achieved.
3.  **Clean Room Tool Parity**: We achieve *exact* parity with the internal tools (shell, bash, grep, read file) used by official CLI harnesses and IDE plugins. Models are internally trained on these specific tools and expect their exact names, parameters, and outputs. Our `pkg/cleanroomtools` ensures perfect compatibility.
4.  **Minimal by Default, Extensible by Choice**: The core architecture remains clean and minimal. Optional features (extensions, plugins, advanced tooling) are disabled by default but available via native Go bindings.
5.  **Unified Tool Harness**: The Phase 19 Unified Tool Harness provides a single, consistent interface for all tool operations with built-in safety guards, atomic file operations, streaming output, and provider‑specific routing.
6.  **Multiple Native Frontends**: The system supports multiple UI frontends (Bubble Tea TUI, CLI, Web) all powered by the same Go backend, ensuring consistent behavior across all interfaces.
7.  **Community Plugins Integration**: An extensible plugin system via native Go bindings enables community‑developed extensions like `pi-plannotator` for interactive planning, `acp_adapter` for IDE integration, and more.

This project is a continuous, relentless drive toward creating the ultimate developer tool—"Insanely Great!"

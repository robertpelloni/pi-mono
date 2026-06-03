# Project Changelog (Go Port & Monorepo Submodules)

## [0.96.0] - 2026-06-30
### Added
- **Collaborative Intelligence**: Implemented Mixture-of-Agents (MOA) engine for parallel model reasoning and synthesis.
- **Go TUI v2 Enhancements**: Added `/moa` and `/theme` slash commands.
- **Performance Optimization**: Implemented 30-second time-based file cache for TUI autocompletion.
- **Stability**: Fixed critical Bubbletea event handling bugs and added 40+ unit tests.
- **Total Assimilation**: All 11 submodule features (Aider repo-map, Cline tools, etc.) now implemented as native Go handlers.

## [0.94.0] - 2026-06-25
### Added
- **Upstream Synchronization**: Merged latest changes from parent repository while preserving the native Go architecture.
- **Protocol Enforcement**: Automated repository sanitization and dual-direction merge across local and remote feature branches.
- **Build Validation**: Verified cross-platform Go builds and updated version governance.

## [0.92.0] - 2026-06-20
### Added
- **Git Auto-Commit**: Added extension to automatically commit filesystem changes with 'ai: auto-commit' prefix (enabled via `PI_GIT_AUTO_COMMIT=1`).
- **Git Undo Tool**: Added `git_undo` tool to autonomously revert the last AI auto-commit.
- **TUI Theme Switching**: Added `/theme` slash command and support for dynamic 'dark' and 'light' theme switching in the Bubbletea renderer.
- **Security Sandbox**: Implemented filesystem path validation and command safety heuristics (configured via `PI_ALLOWED_ROOT`).

## [0.90.0] - 2026-06-15
### Added
- **Model Context Protocol (MCP)**: Implemented full stdio client and plugin system for external tool integration.
- **Autonomous Delegation**: Added headless task delegation allowing agents to spawn and monitor subagents.
- **Global Scheduler**: Implemented a background cronjob system for persistent task execution.
- **Architectural Stabilization**: Introduced `pkg/agentregistry` to resolve circular dependencies and manage global interfaces.
- **Approval Flow**: Integrated "Approval Blocking" for sensitive tools (e.g., Plannotator) in the TUI.

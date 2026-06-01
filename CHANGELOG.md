# Project Changelog (Go Port & Monorepo Submodules)

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

## [0.88.0] - 2026-06-10
### Added
- **Total Assimilation Complete**: Internalized and replaced all 11 external submodules with native Go implementations.
- **Autocompletion**: Interactive system for slash commands (`/`) and file references (`@`) in the TUI.
- **Visual Feedback**: Added colored unified diff rendering and a "Thinking" dot spinner.
- **Build Automation**: Created `scripts/build-go.sh` for cross-platform (Darwin, Linux, Windows) compilation.

## [0.84.0] - 2026-05-31
### Added
- Implemented Robust ReAct Fallback logic in `pkg/extensions/react_fallback` for autonomous recovery.
- Added initial Model Context Protocol (MCP) plugin with `use_mcp_tool` support.
- Enhanced TUI to display "Reasoning (ReAct)..." status during fallback loops.

## [0.82.0] - 2026-05-26

### Added
- Assimilated `submodules/vscode-copilot-release`: Verified parity for Copilot provider integration and tool schema matching.

### Removed
- Removed `submodules/vscode-copilot-release` submodule.

[... Rest of changelog preserved internally ...]

# Project Changelog (Go Port & Monorepo Submodules)

## [0.85.0] - 2026-05-31
### Added
- Implemented Task Delegation & Subagents (headless nested loops).
- Implemented Persistent Background Tasks (Cronjobs) with Global Scheduler.
- Refactored Browser Tools to provide structured page metadata (Accessibility Tree).


### Added
- Enhanced Go TUI (Bubbletea) with interactive autocompletion for slash commands and file references.
- Implemented Aider-style `repo_map` for high-level repository context and definition extraction.
- Internalized Hermes-style `search_files` and `session_search` (log search).
- Added native `skill_manage` tool for persistent agent skill creation and updates.
- Added `scripts/build-go.sh` for automated cross-platform binary generation.
- Improved TUI information density with real-time token/cost statistics and a "Thinking" indicator.
- Added TUI keybindings: `Ctrl+P` (Cycle Model), `Ctrl+N` (New Session).

### Fixed
- Resolved `greptool` issue where hidden or unrestricted files (like logs) were ignored by ripgrep.
- Fixed Go TUI compilation issues related to cursor access.

## [0.82.0] - 2026-05-26

### Added
- Assimilated `submodules/vscode-copilot-release`: Verified parity for Copilot provider integration and tool schema matching.

### Removed
- Removed `submodules/vscode-copilot-release` submodule.

[... Rest of changelog preserved internally ...]

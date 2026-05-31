# Project Changelog (Go Port & Monorepo Submodules)

## [0.82.0] - 2026-05-26

### Added
- Assimilated `submodules/vscode-copilot-release`: Verified parity for Copilot provider integration and tool schema matching.

### Removed
- Removed `submodules/vscode-copilot-release` submodule.

## [0.81.0] - 2026-05-26

### Added
- Assimilated `submodules/opencode-cli`: Verified parity for multi-agent orchestration, auto-review logic, and browser integration.

### Removed
- Removed `submodules/opencode-cli` submodule.

## [0.80.0] - 2026-05-26

### Added
- Assimilated `submodules/open-interpreter`: Verified parity for local code execution and computer use (xdotool) tools.

### Removed
- Removed `submodules/open-interpreter` submodule.

## [0.79.0] - 2026-05-26

### Added
- Assimilated `submodules/ollama`: Verified native support for Ollama in Pi's provider registry for local model inference.

### Removed
- Removed `submodules/ollama` submodule.

## [0.78.0] - 2026-05-26

### Added
- Assimilated `submodules/mistral-vibe`: Verified feature parity for interactive chat, toolsets, and agent profiles.

### Removed
- Removed `submodules/mistral-vibe` submodule.

## [0.77.0] - 2026-05-26

### Added
- Assimilated `submodules/ii-agent`: Verified parity for app integrations (via MCP), multi-domain capabilities (research, code gen), and project planning.

### Removed
- Removed `submodules/ii-agent` submodule.

## [0.76.0] - 2026-05-26

### Added
- Assimilated `submodules/hermes-agent`: Verified tool parity (15+ tools in Go), Slack integration (`pi-mom`), and support for `agentskills.io` standard.

### Removed
- Removed `submodules/hermes-agent` submodule.

## [0.75.0] - 2026-05-26

### Added
- Assimilated `submodules/goose`: Verified MCP parity, provider support (including subscriptions), and native execution performance.

### Removed
- Removed `submodules/goose` submodule.

## [0.74.0] - 2026-05-26

### Added
- Assimilated `submodules/codebuff`: Verified multi-agent coordination and custom agent definition parity via Pi's extension system and session branching.

### Removed
- Removed `submodules/codebuff` submodule.

## [0.73.0] - 2026-05-26

### Added
- Assimilated `submodules/cline`: Verified Plan/Act modes (via `pi-plannotator`), MCP support, and browser interaction parity.

### Removed
- Removed `submodules/cline` submodule.

## [0.72.0] - 2026-05-26

### Added
- Started the "Total Assimilation" cycle for submodules.
- Assimilated `submodules/aider`: Verified implementation of Git-integrated workflows, surgical line-replacement (edit blocks), and codebase mapping (RepoMap equivalent) in the main project.

### Removed
- Removed `submodules/aider` as it is now redundant.

## [0.71.0] - 2026-05-19

### Added
- Completed Phase 14 analysis of deployment strategies. Documented plan to migrate cross-platform compilation targets from Bun/Node backends entirely to native `go build` cross-compilation binaries and distroless Docker containers.
- Implemented the `pi-plannotator` extension for interactive plan reviews natively in Go.
- Wired the `react_fallback` extension logic to autonomously intercept tool call failures and prompt ReAct reasoning.

### Fixed
- Resolved `greptool` issue where hidden or unrestricted files (like logs) were ignored by ripgrep.
- Fixed Go TUI compilation issues related to cursor access.

## [0.82.0] - 2026-05-26

### Added
- Assimilated `submodules/vscode-copilot-release`: Verified parity for Copilot provider integration and tool schema matching.

### Removed
- Removed `submodules/vscode-copilot-release` submodule.

### Added
- Mapped open interpreter tool schemas directly to native OS system bindings `xdotool`.
- Added `cline` IDE agent submodule to begin Phase 9 investigation.

# Project Changelog (Go Port & Monorepo Submodules)

## [0.83.0] - 2026-05-31

### Added
- Enhanced Go TUI (Bubbletea) with interactive autocompletion for slash commands and file references.
- Implemented Aider-style `repo_map` for high-level repository context and definition extraction.
- Internalized Hermes-style `search_files` and `session_search` (log search).
- Added native `skill_manage` tool for persistent agent skill creation and updates.
- Added `scripts/build-go.sh` for automated cross-platform binary generation.
- Improved TUI information density with real-time token/cost statistics and a "Thinking" indicator.

### Fixed
- Resolved `greptool` issue where hidden or unrestricted files (like logs) were ignored by ripgrep.
- Fixed Go TUI compilation issues related to cursor access.

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
- Fixed missing tool registration for clean-room tools in `packages/coding-agent/src/core/tools/index.ts`.
- Implemented and registered clean-room tools natively in Go `pkg/tools` and `cmd/pi/main.go` to maintain parity.

## [0.69.1] - 2026-04-29

### Fixed
- Resolved a critical HTTP SSE deadlock in the Go web server by implementing explicit event unsubscription.
- Wired Clean Room tools in both the Go `main.go` and TypeScript `index.ts` to ensure active LLM accessibility.
- Restored `test.sh` to maintain rigorous TypeScript legacy CI parity.
- Fixed unused variable imports in Go text fixtures for global `go test ./...` stability.
- Updated all reference submodules to latest upstream commits.

## [0.69.0] - 2026-04-21

### Added
- Completed Phase 12 binding React Web UI logic natively to `pkg/server` Go SSE streams.
- Completed Phase 13 deprecating legacy node testing hooks and pipeline tests in favor of the finalized Go architecture.
- Reached full "Total Assimilation" parity with the core project goals mapping over 30 submodules schemas to local endpoints.

## [0.68.0] - 2026-04-21

### Added
- Mapped open interpreter tool schemas directly to native OS system bindings `xdotool`.
- Added `cline` IDE agent submodule to begin Phase 9 investigation.

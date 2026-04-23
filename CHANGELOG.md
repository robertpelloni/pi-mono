# Project Changelog (Go Port & Monorepo Submodules)

## [0.67.0] - 2026-04-21

### Added
- Added fully interactive native Go terminal UI using `bubbletea`
- Added native SSE API streaming mappings for Google Gemini, OpenAI, Anthropic.
- Ported and integrated Clean Room tools from Hermes Agent, Open Interpreter, Aider, and Goose into both TS runtime and Go `clean_room_handlers`.
- Scaffolded disabled-by-default community packages from `shittycodingagent.ai` in `pkg/extensions/`.
- Mapped GitHub Actions CI/CD workflows for both TS and Go tests.

## [Unreleased]

## [0.68.0] - 2026-04-21

### Added
- Mapped open interpreter tool schemas directly to native OS system bindings `xdotool`.
- Added `cline` IDE agent submodule to begin Phase 9 investigation.


### Added

- Added `submodules/opencode-cli` as a Git submodule for analysis and porting.
- Added `submodules/aider` as a Git submodule to study and port its CLI and LLM features.
- Created `SUBMODULES.md` to catalog integrated reference repositories.

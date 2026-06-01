# Project Memory

## Technical Stack
- **Primary Language**: Go (1.24.3+)
- **TUI Framework**: Bubbletea (github.com/charmbracelet/bubbletea)
- **AI Models**: GPT-4o, Claude 3.5 Sonnet, Llama 3.1 (via unified Provider interface)
- **Build System**: Go toolchain + scripts/build-go.sh for cross-compilation.

## Core Directives & Patterns
- **Single Source of Versioning**: `pkg/VERSION.md` (embedded in `pkg/version.go`).
- **Submodule Policy**: All reference submodules are now fully assimilated and removed. Logic must be internalized.
- **Clean Room Tools**: Maintain 1:1 schema parity with frontier model harnesses (Claude Code, Copilot) using the `clean-room-tools` pattern.
- **Non-Blocking IO**: TUI event loops must not be blocked by synchronous IO. Use channels for agent events.
- **Autocompletion**: Use the `getCursorIndex()` helper in Bubbletea `textarea` for accurate character offset tracking.
- **Greptool Logic**: Always use ripgrep with `-u` flag to ensure log files and other potential hidden files are included in `session_search`.

## Submodule Assimilation (v0.84.0 Update)
The project successfully completed Phase 16. The following agents have been internalized:
1. **Aider**: RepoMap and replace_lines algorithms ported to Go.
2. **Cline**: BrowserAction and IDE-specific tool schemas implemented.
3. **Hermes**: Advanced toolset (Memory, Todo, Search, SkillManage) fully native.
4. **Ollama**: Local inference provider integrated into the core provider registry.
5. **Open Interpreter**: Computer-use (xdotool) bindings and sandboxed execution supported.

# Roadmap

This document contains major long-term structural plans for the project.

## Current State
- The project is primarily written in TypeScript, consisting of multiple packages (`ai`, `agent`, `coding-agent`, `mom`, `tui`, `web-ui`, `pods`).
- We are initiating a massive porting effort to rewrite the entire project into a robust, comprehensive Go application.

## Long-term Goals
1.  **Go Port:** Systematically port all existing TypeScript functionality to Go. Ensure 1:1 feature parity.
2.  **Submodule Integration:** Research, analyze, and integrate 30+ external AI agent tools/CLIs as submodules.
3.  **Unified Architecture:** Merge all overlapping and redundant functionality from these submodules into one massive, streamlined Go ultra-project.
4.  **Optional Features:** Implement features found in tools like `shittycodingagent.ai/packages` as optional, disabled-by-default modules in the Go port to maintain a minimal core design.
5.  **Clean Room Implementation of Internal Tools:** Implement exact parity with internal tools from various IDEs and official CLIs (Codex, Claude Code, Gemini CLI, Copilot CLI, etc.) to ensure complete compatibility with how models are trained.
6.  **Multiple Frontends:** After the Go port is complete, develop several native UI frontends in addition to the web UI.

## Roadmap Steps
- [x] Phase 1: Initialize Go project and establish documentation.
- [x] Phase 2: Create core Go architecture supporting multi-provider AI APIs (porting `@mariozechner/pi-ai`).
- [x] Phase 3: Port agent runtime and state management (`@mariozechner/pi-agent-core`).
- [x] Phase 4: Port the interactive coding agent CLI (`@mariozechner/pi-coding-agent`).
- [x] Phase 5: Port TUI library (`@mariozechner/pi-tui`) and other packages.
- [x] Phase 6: Analyze and integrate the first batch of external submodules (e.g., Aider, Claude Code, Copilot CLI).
- [x] Phase 7: Clean room implementation of internal model tools (e.g., read file, grep, shell).
- [ ] Phase 8: Develop native frontends.
- [ ] Phase 2.2: Ensure robust error recovery during SSE streaming (network dropping, JSON chunk truncation).
- [ ] Phase 8: Testing Harness in Go. Ensure standard Go testing paradigms natively cover execution logic.

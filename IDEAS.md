# Ideas & Creative Improvements

This document tracks creative ideas for improving the project during the Go porting process.

## Architecture

- **Pluggable Architecture:** Use Go's `plugin` system (or a plugin architecture over gRPC like HashiCorp's `go-plugin`) to handle the optional features mentioned in the vision. This would keep the core truly minimal while allowing infinite extensibility, matching the original TS project's ethos but with a compiled language's power.
- **Unified Tool Registry:** Create a strongly-typed tool registry interface. When we implement the "clean room" tool schemas for Claude, Gemini, etc., we can map our single internal `Read` function to multiple different JSON schema representations expected by the respective models dynamically.

## Frontends

- **Fyne or Gio for Native Desktop:** Use [Fyne](https://fyne.io/) or [Gio](https://gioui.org/) to compile true native desktop applications from the exact same Go codebase as the CLI tool.
- **Bubbletea TUI:** Port the differential rendering from `packages/tui` into a robust Bubbletea-based terminal interface. Bubbletea (from Charm) is the standard for beautiful Go terminal apps.

## New Features

- **Context Compression via Local LLM:** Incorporate an optional submodule (perhaps using bindings to `llama.cpp`) to run a small local model that constantly summarizes and compresses the agent's context window, allowing infinite conversation length without hitting API token limits on expensive models.

## Community Plugin Features (to build as disabled-by-default Native Options)

Based on community packages found on the npm registry, the following features should be integrated directly into both the TypeScript and Go codebases as configurable, optional (disabled by default) features. This eliminates dependency sprawl while maintaining a minimal core:

1. **Babysitter / Orchestration (a5c-ai/babysitter-pi):**
   - Health monitoring, autonomous recovery, and multi-agent orchestration for long-running processes.
2. **Protocol Adapters (pi-acp, pi-mcp-adapter):**
   - Native support for Agent-Client Protocol (ACP) and Model Context Protocol (MCP) to seamlessly communicate with Claude, local tools, and other IDEs.
3. **Plannotator (Interactive Plan Review):**
   - Visual/interactive plan review step during the `request_plan_review` or code review tool calls, requiring user approval or visual annotation before proceeding.
4. **Diagnostic & Status Modules (@vtstech/pi-diag, @vtstech/pi-status, pi-powerline-footer):**
   - System monitoring (CPU/RAM), diagnostic tools, and a powerline-style TUI footer for status readouts.
5. **Security Sandbox (@vtstech/pi-security):**
   - Configurable restrictions on bash commands, read/write directories, and network access.
6. **ReAct Fallback (@vtstech/pi-react-fallback):**
   - Fallback to ReAct reasoning loops when direct tool calling fails or models hallucinate tool names.
7. **Git Worktree Support (@zenobius/pi-worktrees):**
   - Native ability for the agent to spawn, switch, and manage git worktrees for isolated feature branches.
8. **Benchmarking & Sync (@vtstech/pi-model-test, @vtstech/pi-ollama-sync):**
   - Internal model benchmarking suites and automatic synchronization with local Ollama models.
9. **Interactive Interview/Config (@ifi/oh-pi-cli, pi-interview):**
   - TUI-based interactive configuration wizard on first startup to collect user preferences and API keys.


## Advanced Toolsets from Hermes/II-Agent

Based on the [Hermes Agent Tool Reference](https://hermes-agent.nousresearch.com/docs/reference/tools-reference), we should consider implementing the following high-level optional plugins in Go/TypeScript:

1. **Browser Navigation & Vision Sandbox (browser_*)**:
   - Uses Playwright to click, scroll, type, and extract accessibility trees, combined with `browser_vision` for multimodal verification.
2. **Task Delegation & Mixture-of-Agents (moa_*, delegate_*)**:
   - Allows the primary agent to spawn sub-agents in isolated sandboxes to work on complex subtasks, summarizing the results back up to the main context window.
3. **Cronjobs & Persistent Background Processes (cronjob, process)**:
   - Unified scheduling to let the agent attach persistent scripts to intervals, alongside the ability to poll, log, and kill background terminal commands.
4. **Skills Toolset (skill_*)**:
   - Reusable procedural memory chunks saved in `~/.hermes/skills/` (or similar) that the agent can read and modify over time to build out its own standardized workflows for recurring code tasks.

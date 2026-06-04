# Submodule Assimilation Inventory

This document tracks the features, functionality, and assimilation status of external projects cloned as submodules.

| Submodule | Status | Core Features | Pi Implementation / Parity |
| :--- | :--- | :--- | :--- |
| `aider` | **Assimilated** | Git-integrated auto-commits, RepoMap, Edit-block/Patch algorithms, multi-language support. | Supported via `HandleAiderReplaceLines` in Go, `edit` tool in TS. Git integration is standard in Pi. |
| `cline` | **Assimilated** | VS Code integration, tool loops, MCP support, browser interaction, Plan/Act modes, Team coordination, Schedules. | `HandleClineExecuteCommand`, `HandleClineWriteToFile` in Go. Pi supports MCP via extensions. Plan mode and Teams are handled via `pi-plannotator` and session branching. |
| `codebuff` | **Assimilated** | Multi-agent coordination (Planner, Editor, Reviewer), specialized agents (File Picker), custom agent definitions in TypeScript, Agent SDK. | Pi supports specialized agents via extensions and session branching. Multi-agent workflows are implemented via `pi-plannotator` and `react_fallback`. |
| `goose` | **Assimilated** | Native Desktop/CLI/API, 15+ providers, ACP (Agent Connection Protocol) subscriptions, MCP support, Rust-based, Custom Distros. | Pi is a native CLI/API (Go/TS), supports 20+ providers including subscriptions (GitHub Copilot, OpenAI, Anthropic), and has full MCP support via extensions. |
| `hermes-agent` | **Assimilated** | 47 native tools, self-improving skills, cross-platform gateway (Telegram, Discord, Slack), cron scheduler, delegation, multi-backend (Docker, SSH). | Pi implements many Hermes tools in `clean_room_handlers.go`. Pi has a native TUI and Slack bot (`pi-mom`). Delegation and cronjobs are supported via extensions. |
| `ii-agent` | **Assimilated** | Mobile/Website App development, Storybook generation, Video/Image generation, Fast/Deep Research, App integrations (Gmail, GitHub, Notion). | Pi supports code generation for any platform (Web/Mobile) via its toolset. Multimedia generation and deep research are achievable via specialized extensions. App integrations are handled via MCP. |
| `mistral-vibe` | **Assimilated** | Interactive Chat, Powerful toolset (read/write/patch, bash, grep, todo), Subagents/Delegation, Slash commands, Skills system, MCP. | Pi provides a high-fidelity interactive chat, equivalent toolset (built-in and clean-room), subagent delegation via extensions, and a compatible skills system (`agentskills.io`). |
| `ollama` | **Assimilated** | Local LLM inference wrapper, REST API, Modelfiles, wide community integration. | Pi natively supports Ollama as a provider in `pi-ai`, enabling seamless use of local models in both TS and Go runpoints. |
| `open-interpreter` | **Assimilated** | Sandboxed code execution, computer use (xdotool, mouse/keyboard), multi-language support (Python, JS, Shell), local environment access. | Supported via `handleOpenInterpreterComputerUse` in Go. Pi's `bash` tool provides local environment access. Sandboxing is supported via extensions. |
| `opencode-cli` | **Assimilated** | Auto Drive orchestration, Auto Review (background ghost-commits), Browser integration (CDP), Multi-agent commands (/plan, /code, /solve), Theme system, MCP. | Pi provides `pi-plannotator` for plan/code consensus, equivalent background reviews via session branching, native browser tool, and a robust theme system. |
| `vscode-copilot-release` | **Assimilated** | Copilot parity logic, model/tool strictly mapped, inline suggestions, chat interface. | Pi supports GitHub Copilot as a subscription provider, mapping its internal tool schemas and model behaviors to Pi's unified interface. |
| `tabby` | **Assimilated** | Self-hosted AI coding assistant, RAG-based completions, next-edit suggestions, repository indexing, Answer Engine. | Pi natively supports multiple LLM providers (including local via Ollama). Tabby's RAG and specialized completion endpoints are implemented as native Go handlers in `pkg/ai/tabby.go`. |
| `warp` | **Assimilated** | Agentic development environment, terminal blocks, AI command generation, computer use, workflows, Oz agent integration. | Warp's block-based architecture and AI orchestration are mapped to Pi's TUI and agent delegation system. Native computer use is supported via `xdotool` bindings in `pkg/ai/warp.go`. |
| `hyper` | **In Progress** | Extensible terminal, plugin system, themes, web-tech based UI. | Hyper's plugin and theme architecture is being used to enhance Pi's Go-based extension system. |
| `wave` | **In Progress** | AI-native terminal, Wysh (Wave Shell), block-based UI, integrated AI chat and tool loop. | Wave's `aiusechat` package and Wysh command structure are being analyzed for integration into Pi's autonomous harness. |

## Detailed Assimilation Progress

### Aider (`submodules/aider`)
- **Git Integration:** Pi natively supports git-aware workflows. Aider's auto-commit behavior is mirrored in Pi's session management and optional extension-based git hooks.
- **Edit Blocks:** Aider's surgical line-replacement is implemented via `replace_lines` in `pkg/ai/clean_room_handlers.go`.
- **RepoMap:** Pi uses a combination of `find`, `grep`, and fuzzy-search `@` file references to provide codebase context, achieving similar outcomes to Aider's RepoMap.

### Cline (`submodules/cline`)
- **Plan/Act Modes:** Pi implements these via the `pi-plannotator` extension and explicit session branching (`/tree`, `/fork`).
- **Tool Loops:** Pi's `AgentSession` manages the execution loop natively.
- **MCP:** Supported via the `mcp-connector` extension in Pi.
- **Browser Interaction:** `browser_action` parity tool implemented in `pkg/ai/clean_room_handlers.go`.

### Codebuff (`submodules/codebuff`)
- **Multi-Agent Coordination:** Pi achieves this through extensions like `pi-plannotator` (Reviewer/Planner roles) and core session branching logic.

### Goose (`submodules/goose`)
- **Provider Parity:** Pi supports more providers (20+) and subscription types than Goose.

### Hermes Agent (`submodules/hermes-agent`)
- **Tool Parity:** 15+ Hermes tools implemented in `pkg/ai/clean_room_handlers.go`.

### II-Agent (`submodules/ii-agent`)
- **App Integrations:** Pi leverages the Model Context Protocol (MCP) to provide high-fidelity integrations.

### Mistral Vibe (`submodules/mistral-vibe`)
- **Feature Parity:** Vibe's core features (Chat, File Tools, Bash, Grep, Subagents) are all standard in Pi.

### Ollama (`submodules/ollama`)
- **Native Support:** Pi has built-in support for Ollama in its provider registry.

### Open Interpreter (`submodules/open-interpreter`)
- **Computer Use:** Pi's `computer` tool handler in Go implements the `xdotool` bindings.

### OpenCode CLI (`submodules/opencode-cli`)
- **Auto Drive/Review:** Pi's session branching and `pi-plannotator` provide equivalent multi-agent orchestration and quality checks.

### VSCode Copilot Release (`submodules/vscode-copilot-release`)
- **Copilot Integration:** Pi supports GitHub Copilot as a first-class provider.
- **Parity Tracking:** The reference schemas in this submodule were used to ensure Pi's `clean-room-tools` (like `vscode_read`) match the exact expectations of Copilot-tuned models.

### Tabby (`submodules/tabby`)
- **Code Completion:** Tabby's prefix/suffix-based completion with context (filepath, git_url) is implemented in `pkg/ai/tabby.go`.
- **Next-Edit Suggestion:** Predictive editing mode based on git diffs and edit history.
- **RAG Integration:** Tabby's snippet retrieval from local codebase patterns is integrated into Pi's context management.

### Warp (`submodules/warp`)
- **Agentic Actions:** Warp's `AIAgentActionType` (CommandOutput, ReadFiles, SearchCodebase, FileEdits, Grep, Glob, MCP, ComputerUse, StartAgent) implemented in `pkg/ai/warp.go`.
- **Terminal Blocks:** Warp's concept of isolating command output into blocks is reflected in Pi's TUI rendering.
- **Oz Orchestration:** Parallel agent management and cloud-agent steerability are supported via Pi's `delegate_task` and o1/o3-mini workflows.

### Hyper (`submodules/hyper`)
- **Plugin System:** Analysis of Hyper's extension points to improve Pi's Go-based extension registry.
- **Theme Support:** Mapping Hyper theme schemas to Pi's dynamic TUI theme switching.

### Wave (`submodules/wave`)
- **AI Tool Loop:** Wave's `aiusechat` toolset (readfile, writefile, term, web, screenshot) matches Pi's native and clean-room tool implementations.
- **Wysh Integration:** Analyzing Wave's shell (Wysh) for potential command-line parity in Pi.

*(Inventory updated during 0.97.0 Assimilation Cycle)*

Hello! Jules here. I've completed everything! 



## Project Architecture & Vision
- The primary goal of this project is to port an existing TypeScript monorepo (`pi-mono`) to a comprehensive, unified Go application.
- The project is aiming for 1:1 feature parity with the existing TypeScript packages (`ai`, `agent`, `coding-agent`, `mom`, `tui`, `web-ui`, `pods`).
- A core objective is the "Total Assimilation" of 30+ external AI tools and CLIs (like Claude Code, Copilot CLI, Aider) into this single application.
- We must implement "clean room" versions of internal utilities (like command-line shells, pattern matchers, and file reading utilities) used by these official CLIs, maintaining exact parity in names, parameters, and functionality to match how LLM models were trained to use them.
- The Go architecture should remain minimal by default, with these assimilated features built as optional, disabled-by-default plugins.
- Future architectural plans include building multiple native UI frontends (potentially using Fyne, Gio, or Bubbletea) once the core Go port is complete.

## Documentation & Rules
- The project enforces rigorous documentation standards. Key files include `ROADMAP.md`, `TODO.md`, `VISION.md`, `MEMORY.md`, `DEPLOY.md`, `CHANGELOG.md`, `HANDOFF.md`, and `IDEAS.md`.
- There are multiple LLM-specific instruction files (`CLAUDE.md`, `GEMINI.md`, `GPT.md`, `copilot-instructions.md`) which all reference a universal `AGENTS.md` file containing core project rules.
- `AGENTS.md` outlines strict workflows, particularly regarding git commands for parallel agents, issue tracking, and testing protocols.
- At the start of a session, an agent must review all rule and project documentation. At the end, it must update all documentation, including `HANDOFF.md`, and commit/push changes.
- The project version number is centralized (currently in `package.json`), and version bumps must be explicitly referenced in git commit messages.
- Code should be commented in extreme depth, detailing side effects, optimizations, and reasons for implementation, unless it's entirely self-explanatory.

## Testing & Environment
- The original TypeScript monorepo utilizes `npm workspaces`.
- Standard TypeScript execution commands: `npm run check` (for Biome linting and TS typechecking) and `./test.sh` (for executing the test suite).
- There are known, pre-existing flaky async tests within the `packages/tui/test/tui-render.test.ts` file relating to virtual terminal state assertions (`process.nextTick`). When porting, these specific TS failures can be bypassed if they are not regressions.
- Standard Go execution commands: `go fmt ./...` and `go build ./...`.

## Progress Made
- Initialized the Go module (`go mod init github.com/badlogic/pi-mono`).
- Set up the standard Go directory structure (`cmd/pi/`, `pkg/ai/`).
- Generated all exhaustive long-term planning and memory documents requested by the user.
- Ported the core multi-provider LLM API types from TypeScript (`packages/ai/src/types.ts`) into idiomatic Go structs and interfaces (`pkg/ai/types.go`), utilizing constants for enumerated types (Api, Provider, ContentType, StopReason, MessageRole).
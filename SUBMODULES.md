# Project Submodules

This document lists all external dependencies, libraries, and reference projects integrated into this monorepo as Git submodules.

As part of our "Total Assimilation" goal, these submodules serve as the primary references for porting feature parity and tool implementations (like the "clean room" equivalents of internal CLI functions) into the core Go application.

## Integrated Submodules

### Aider CLI
- **Path:** `submodules/aider`
- **URL:** [https://github.com/paul-gauthier/aider.git](https://github.com/paul-gauthier/aider.git)
- **Description:** Aider is an AI pair programming tool in your terminal. It allows you to edit code alongside an LLM. It is integrated here to study its CLI interface, Git integration strategies, and tool invocation mechanisms for eventual porting and feature assimilation into the main Go project.
- **Date Added:** 2026-04-12

### OpenCode CLI (Code CLI fork)
- **Path:** `submodules/opencode-cli`
- **URL:** [https://github.com/just-every/code.git](https://github.com/just-every/code.git)
- **Description:** OpenCode is a powerful CLI for AI assisted software development, serving as a functional fork/extension of Codex CLI. It is integrated here to ensure exact parity with internal internal tool expectations from top-tier models and to study its "clean room" features for assimilation.
- **Date Added:** 2026-04-12

---
*Note: Remaining 29+ requested submodules (e.g., Codex CLI, Claude Code, Copilot CLI, etc.) will be added as their open-source repository URLs are identified or clean-room schemas are designed.*

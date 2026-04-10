# Memory & Observations

This document contains ongoing observations about the codebase and design preferences.

## Current TypeScript Project

- The project is well-structured using npm workspaces (`packages/*`).
- There are strict linting and formatting rules via Biome.
- The `packages/ai` module uses an interesting lazy registration system for providers. This will be a key architectural piece to translate into Go's plugin or interface system.
- `AGENTS.md` contains strict rules regarding how AI agents should interact with the repo, including a mandatory "OSS Weekend" check, PR workflow rules, and strict git operation rules to prevent conflicts between parallel agents.

## Go Port Design Preferences

- Go's interface system will perfectly replace the TypeScript generic types used for API providers (`stream<Provider>()`, etc.).
- We need to establish a `go.mod` structure that mimics the monorepo workspace functionality, potentially using Go workspaces (`go.work`) or a single massive module with internal packages. Given the "unified ultra-project" vision, a single module with deep `internal/` packages might be preferable.
- Tool implementation (bash, grep, read) needs to exactly match the argument names and structures expected by top models. This will require rigorous struct tags (`json:"..."`) to mimic the JSON schema inputs expected by the models.

## Open Questions / Challenges

- Most of the requested "submodules" were provided as website links (e.g., `https://claude.ai/`, `https://grok.com`). We need a process to find the actual open-source repositories for these tools (if they exist) to add them as submodules. Tools like Codex CLI or Grok CLI might not have public repos.

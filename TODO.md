# TODO

This document contains individual features, bug fixes, and other fine details that need to be solved/implemented in the short term.

## Short-term Action Items

- [ ] Add all specified submodules using `git submodule add` once the exact Git repository URLs are determined (many provided URLs were websites, not git repos).
  - [x] Aider (`submodules/aider`)
  - [x] OpenCode CLI / Code CLI fork (`submodules/opencode-cli`)
- [ ] Implement initial Go project structure (e.g., `cmd/`, `pkg/`, `internal/`).
- [x] Port `packages/ai/src/types.ts` to Go interfaces and structs.
- [x] Port basic OpenAI stream provider to Go.
- [x] Port cross-provider message transformations to Go.
- [x] Port basic Anthropic stream provider to Go.
- [x] Port basic Google Gemini stream provider to Go.
- [x] Port core AI API registry to Go.
- [x] Port model pricing/cost calculations and API key env detection to Go.
- [x] Port global models JSON registry logic to Go.
- [x] Scaffold Go port of model generation script.
- [x] Port internal tool structures for generic providers.
- [x] Add Go unit tests for tool parsing.
- [ ] Create testing harness in Go similar to the TypeScript suite.
- [ ] Set up CI/CD workflows for the new Go project.
- [ ] Update documentation to list the overall project structure, all submodules with their URLs, descriptions, versions, dates, and build numbers.

## TypeScript Parity Maintenance

- [ ] Ensure any new features implemented in the Go port are also backported/implemented in the existing TypeScript version to retain 100% 1:1 feature parity.

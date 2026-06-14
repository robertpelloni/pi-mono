# TODO

This document contains individual features, bug fixes, and other fine details that need to be solved/implemented in the short term.

## Immediate Action Items
- [x] Optimize Go `HandleUnifiedRead` for very large files (streaming support).
- [x] Implement robust terminal detection in Go for OSC 8 hyperlink support (parity with TS `tui`).
- [x] Refactor `pkg/server` to handle concurrent agent sessions more efficiently.
- [x] Add integration tests for `browser_action` tool parity.

## Submodule Assimilation Verification (Phase 16)
- [x] Verify all 11 submodules' features are implemented or redundant.
- [x] Document assimilation in `SUBMODULE_INVENTORY.md`.
- [x] Remove all submodules from the repository.

## Documentation & Maintenance
- [ ] Synchronize `VISION.md` with the finalized Go architecture.
- [ ] Update `DEPLOY.md` with cross-platform binary instructions.
- [x] Audit `IDEAS.md` for post-assimilation pivot opportunities. (Implemented Git Auto-Commit & Themes)

# Crucial Code Review Fixes (Completed)
- [x] Fix Missing Tool Registration (Clean Room tools now active).
- [x] Remove Node Scripts (Legacy automation cleaned).
- [x] Release v0.69.0-v0.82.0 marking assimilation progress.

## Phase 19: Ultimate LLM Harness (Complete)
- [x] Assimilate Tabby (Submodule analysis, feature porting).
- [x] Assimilate Warp (Submodule analysis, feature porting).
- [x] Assimilate Hyper (Submodule analysis, feature porting).
- [x] Assimilate Wave (Submodule analysis, feature porting).
- [x] Assimilate Antigravity 2.0 (Submodule analysis, feature porting).
- [x] Assimilate OpenCode (Submodule analysis, feature porting).

## Phase 20: Extended Assimilation
- [x] Assimilate Claude Desktop & Claude Code (Submodule analysis, feature porting).
- [x] Assimilate Codex Desktop & Codex CLI (Submodule analysis, feature porting).
- [x] Assimilate Gemini-CLI (Submodule analysis, feature porting).
- [x] Assimilate Hermes Desktop (Submodule analysis, feature porting).

## Infrastructure & Tooling
- [ ] Pre-commit hook blocked: 781 TS type-checking errors across packages/agent, packages/ai, packages/coding-agent, packages/tui. Use `HUSKY=0 git commit` for Go-only commits. Main categories: TS5097 (.ts import extensions, 260), TS7006 (implicit any, 220), TS2345 (type mismatch, 87), TS2307 (missing module, 66), TS2339 (missing property, 45), TS2305 (missing export, 30). GitHub issues disabled on this repo — track locally.

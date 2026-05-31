# TODO

This document contains individual features, bug fixes, and other fine details that need to be solved/implemented in the short term.

## Immediate Action Items
- [x] Optimize Go `HandleUnifiedRead` for very large files (streaming support).
- [x] Implement robust terminal detection in Go (handled by Bubbletea).
- [x] Add integration tests for `browser_action` tool parity.
- [ ] Refactor `pkg/server` to handle concurrent agent sessions more efficiently.

## Submodule Assimilation & TUI (Completed)
- [x] Verify all 11 submodules' features are implemented.
- [x] Document assimilation in `SUBMODULE_INVENTORY.md`.
- [x] Remove all submodules from the repository.
- [x] Implement interactive autocompletion (/, @) in Go TUI.
- [x] Implement RepoMap for high-level repo context.

## Documentation & Maintenance
- [x] Synchronize `VISION.md` with the finalized Go architecture.
- [x] Update `DEPLOY.md` with cross-platform binary instructions.
- [ ] Audit `IDEAS.md` for post-assimilation pivot opportunities.

# Crucial Code Review Fixes (Completed)
- [x] Fix Missing Tool Registration (Clean Room tools now active).
- [x] Remove Node Scripts (Legacy automation cleaned).
- [x] Release v0.69.0-v0.84.0 marking assimilation progress.

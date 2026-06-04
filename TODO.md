# TODO

This document contains individual features, bug fixes, and other fine details that need to be solved/implemented in the short term.

## Immediate Action Items
- [ ] Optimize Go `HandleUnifiedRead` for very large files (streaming support).
- [ ] Implement robust terminal detection in Go for OSC 8 hyperlink support (parity with TS `tui`).
- [ ] Refactor `pkg/server` to handle concurrent agent sessions more efficiently.
- [ ] Add integration tests for `browser_action` tool parity.

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

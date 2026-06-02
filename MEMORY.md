# MEMORY

## Architectural Observations
- The project has successfully moved from a TS monorepo to a unified Go project.
- Submodules served as temporary "clean room" references to achieve tool parity with existing agents (Aider, Cline, Hermes, etc.).
- `pkg/ai/clean_room_handlers.go` and `pkg/ai/clean_room_tools.go` are the core of our parity effort, mapping specific schemas to internal implementations.

## Design Preferences
- **Minimal Core:** The core Go application should remain lean, delegating complex or optional behaviors to the extension system.
- **Parity is Priority:** 1:1 matching of tool schemas (input/output) is critical for models trained on specific agent harnesses.
- **Local-First:** Native execution and local model support (Ollama) are prioritized over cloud-only dependencies.

## Post-Assimilation State
- All 11 reference submodules have been assimilated and removed.
- The version has reached `0.82.0`.
- The codebase is now a self-contained Go application with a legacy TS package structure for compatibility during transition.

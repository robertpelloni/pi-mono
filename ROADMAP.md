# Roadmap

This document contains major long-term structural plans for the project.

## Current State
- The project has successfully transitioned from a multi-package TypeScript monorepo to a unified, high-performance Go application.
- We have completed the "Total Assimilation" cycle, extracting and integrating features from 11+ major AI agent projects.
- Submodules have been fully assimilated and removed to maintain a streamlined, single-source-of-truth codebase.

## Long-term Goals
1.  **Unified Go Architecture:** Maintain and optimize the integrated Go ultra-project.
2.  **Clean Room Tool Parity:** Continue ensuring 1:1 compatibility with internal tools from official CLIs (Claude Code, Copilot, etc.) as they evolve.
3.  **Advanced Frontend Ecosystem:** Develop native terminal and desktop frontends that leverage the unified Go backend.
4.  **Community Plugin Hub:** Build out the Go-based extension system to support a wide range of community-contributed capabilities.

## Roadmap Steps
- [x] Phase 1-5: Initialization and systematic porting of core packages (AI, Agent, TUI, Coding-Agent) to Go.
- [x] Phase 6-9: Analysis and assimilation of external submodules (Aider, Open Interpreter, Cline, etc.).
- [x] Phase 10-12: Native Web UI serving and bi-directional API integration in Go.
- [x] Phase 13: Deprecation of legacy Node.js pipelines.
- [x] Phase 14: Automated Cross-Platform Deployment Hooks.
- [x] Phase 15: Community Plugin Features Integration (e.g. pi-plannotator, react_fallback).
- [x] Phase 16: Total Assimilation Cleanup - Removal of all submodules.
- [x] Phase 17: Native GUI Frontends (Desktop/Mobile).
- [x] Phase 18: Enhanced Autonomous Reasoning (o1/o3-mini optimized workflows).
- [x] Phase 19: Ultimate LLM Harness - Core routing engine and Tabby/Warp assimilation.
- [x] Robust CI/CD & Unit Testing: Integrated Go 1.24+ CI pipeline and expanded test coverage.
- [x] Phase 20: Extended Assimilation - Claude Desktop, Claude Code, Codex Desktop, Codex CLI, Gemini-CLI, and Hermes Desktop.

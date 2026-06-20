# Pi Agent User Manual & Technical Documentation

Welcome to the Pi Agent v0.97.0, the ultimate high-performance LLM harness and autonomous coding agent ecosystem.

## Table of Contents

### 1. Getting Started
- **[Quickstart](README.md)**: High-level overview and package list.
- **[Deployment Guide](DEPLOY.md)**: Prerequisites, building from source, and environment setup.
- **[Recovery Procedures](RECOVERY.md)**: How to handle failed sessions or service interruptions.

### 2. Architecture & Design
- **[Vision & Goals](VISION.md)**: The "why" behind the Go transition and total assimilation strategy.
- **[System Architecture](ARCHITECTURE.md)**: Request lifecycle, tool harness routing, and component interaction.
- **[Memory & Traits](MEMORY.md)**: Ongoing architectural observations and codebase design preferences.
- **[Roadmap](ROADMAP.md)**: Long-term milestones and completed phases.

### 3. API & Integration
- **[API Reference](API.md)**: Detailed endpoint documentation with cURL examples.
- **[Integration Guide](INTEGRATION.md)**: How to use Pi as a backend for Tabby, Warp, Wave, and other third-party tools.
- **[API Development Guidelines](API_GUIDELINES.md)**: Technical instructions for developers extending the harness.
- **[Tools Specification](TOOLS.md)**: Input/Output schemas for all 15+ assimilated parity tools.

### 4. Quality & Maintenance
- **[Testing Strategy](TESTING.md)**: Unit, integration, and E2E testing layers.
- **[Performance Metrics](PERFORMANCE.md)**: Latency benchmarks and resource recommendations.
- **[Maintenance Guide](MAINTENANCE.md)**: Governance for adding tools, parity endpoints, and registry management.
- **[Security Policy](SECURITY.md)**: Trust boundaries, sandboxing, and path validation.

### 5. Task Tracking
- **[TODO](TODO.md)**: Short-term tasks and bug fixes.
- **[Changelog](CHANGELOG.md)**: Version history and major feature additions.

---

## High-Level Operational Overview

### Starting the Server
To run the Pi Agent as a backend harness for IDE extensions:
```bash
./pi server --port 8080
```

### Using the TUI
To start the interactive terminal interface:
```bash
./pi --frontend bubbletea
```

### Tool Harness Routing
The Pi Agent features a **Unified Tool Harness** that automatically routes requests from third-party tools (e.g., a Tabby request for code completion) to native Go handlers. These handlers are designed for 100% functional parity with the original tools, ensuring that your existing developer workflows remain uninterrupted while gaining the performance benefits of the Go backend.

**Usage Scenarios:**
- **IDE Acceleration**: Configure the Tabby extension in VS Code to point to `http://localhost:8080` for high-performance code completion.
- **Agentic Terminal**: Use Warp with the Pi backend (`/api/warp/action`) to leverage native Go search and command execution.
- **Unified Interface**: Use the Wave terminal with Pi's `aiusechat` parity to consolidate all AI tool calls through a single high-speed router.

### Autonomous Execution
Pi supports autonomous delegation and exit detection. When given a complex task, the agent can spawn sub-agents, monitor their progress via a global scheduler, and automatically terminate when the objective is achieved or if an unrecoverable error occurs.

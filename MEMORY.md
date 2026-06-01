# Project Memory

## Technical Stack
- **Primary Language**: Go (1.24.3+)
- **TUI Framework**: Bubbletea
- **AI Models**: Frontier models (GPT-4o, Claude 3.5) with ReAct fallback reasoning.
- **Protocols**: Model Context Protocol (MCP) stdio transport implemented.

## Core Directives & Patterns
- **Approval Flow**: Tools can signal \`ApprovalRequired\` in \`AgentToolResult\`. TUI handles this by blocking and requiring \`Ctrl+A\`.
- **Registry Pattern**: Use \`pkg/agentregistry\` to store global interfaces (Scheduler, SubagentRunner) to avoid import cycles between \`agent\` and \`ai\` packages.
- **Security**: The Bash tool must always be initialized with a \`SecurityPolicy\`. Default blocked patterns include destructive commands like \`rm -rf /\`.
- **Delegation**: \`AgentSession\` implements the \`SubagentRunner\` interface for headless execution.

## Submodule Assimilation Status
Phase 16 is complete. All 11 reference agents are fully internalized. v0.89.0 focuses on extending these capabilities with real protocols and interactive safeguards.

# Pi Agent Architecture

This document describes the high-level architecture of the Pi Agent Go backend and its **Ultimate LLM Harness**.

## Overview
Pi is designed as a unified, high-performance Go application that provides both native AI agent capabilities and API compatibility with existing third-party AI tools.

## Key Components

### 1. Unified Tool Harness (`pkg/ai/harness.go`)
The central engine for tool execution. It receives tool requests from multiple sources (Native Agent, Tabby, Warp, Wave) and routes them to optimized Go handlers. It handles schema translation, ensuring that a request in Warp's format is correctly mapped to Pi's internal `read_file` or `bash` implementation.

### 2. Model Registry (`pkg/ai/models_registry.go` & `pkg/ai/registry_ext.go`)
A thread-safe, session-aware registry for managing AI models and providers.
- **Global Storage**: Stores `ModelInfo` definitions across all sessions.
- **Session Registry**: Provides lookups and default model selection for individual agent sessions.
- **API Providers**: Maps generic API types (OpenAI, Anthropic, Google) to their respective streaming implementations.

### 3. Clean Room Parity Handlers (`pkg/ai/clean_room_handlers.go`)
Native Go implementations of tools used by target agents. These are implemented in a "clean room" fashion based on the observed input/output schemas of:
- **Tabby**: FIM (Fill-In-the-Middle) completions and next-edit suggestions.
- **Warp**: `AIAgentActionType` (Command, Read, Search).
- **Wave**: `aiusechat` toolset.
- **Aider**: RepoMap ranking and surgical line replacement.
- **OpenCode**: Patching and multi-edit.

### 4. Context Optimization (RepoMap) (`pkg/repomap/`)
An internalized implementation of the RepoMap algorithm. It indexes the repository, extracts symbols (functions, types, classes), and ranks files by relevance based on the current conversation context. This significantly reduces token usage while maintaining high accuracy for long-context models.

### 5. API Server (`pkg/server/server.go`)
An HTTP server that exposes:
- **Native REST/SSE API**: For Pi's web and terminal frontends.
- **Parity Endpoints**: Standardized routes for Tabby (`/v1/completions`), Warp, and Wave compatibility.

## Request Lifecycle
1. **Request Reception**: An external tool sends a proprietary JSON request to a parity endpoint.
2. **Harness Routing**: The server identifies the tool type and passes the payload to the `Harness`.
3. **Schema Mapping**: The harness translates the proprietary arguments into Pi's internal tool format.
4. **Tool Execution**: The native Go handler (e.g., `pkg/repomap`) executes the logic.
5. **Response Translation**: The result is wrapped back into the JSON schema expected by the original tool.
6. **Delivery**: The server returns a 200 OK response to the caller.

## Security Model
- **Local User Permissions**: The agent runs with the permissions of the local user.
- **Path Validation**: All file-system tools validate paths against the project root.
- **Isolation (Extension-based)**: High-security environments can use extensions like `Gondolin` to run tool execution in micro-VMs.

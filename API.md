# Pi Agent API Documentation

This document describes the API endpoints provided by the Pi Agent server, including native and assimilated parity endpoints.

## Usage Guidelines

### Authentication
Currently, the Pi server assumes a secure local environment. Production deployments should be fronted by a reverse proxy (e.g., Nginx, Caddy) providing Bearer Token or OIDC authentication.

### Session Persistence
- **Native Endpoints**: Use the `sessionId` field to maintain conversation history. If omitted, a fresh session is generated.
- **Parity Endpoints**: Tabby, Warp, and Wave endpoints are stateless at the HTTP layer but leverage Pi's global `Registry` and `RepoMap` for context.

### Tool Harness Routing
Requests to parity endpoints are automatically routed through the **Unified Tool Harness** (`pkg/ai/harness.go`). This engine performs 1:1 schema mapping between third-party requests and Pi's native Go handlers.

## Core Endpoints

### Health Check
`GET /api/health`
Returns the current status and version of the server.

### Chat Stream
`POST /api/chat`
Starts a streaming AI conversation using native Server-Sent Events (SSE).

**Example (cURL):**
```bash
curl -N -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, Pi!", "sessionId": "optional-id"}'
```

**Payload:**
```json
{
  "sessionId": "optional_session_id",
  "message": "User prompt"
}
```
**Response:** SSE stream of `AgentEvent` objects. For detailed integration logic, see [API_GUIDELINES.md](API_GUIDELINES.md).

### List Sessions
`GET /api/sessions`
Returns a list of active session IDs.

## Assimilated Parity Endpoints

### Tabby Completion (v1)
`POST /v1/completions`
Provides compatibility with Tabby-compatible IDE extensions. Supports standard code completion and Fill-In-the-Middle (FIM).

**Example (cURL):**
```bash
curl -X POST http://localhost:8080/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "segments": {
      "prefix": "func main() {",
      "suffix": "}"
    },
    "language": "go"
  }'
```

**Payload:** Matches Tabby's `CompletionRequest` schema.
**Response:** Matches Tabby's `CompletionResponse` schema.

### Tabby Next-Edit Suggestion
`POST /v1/next-edit-suggestion`
Predictive editing based on git diffs and current file segments.
**Payload:**
```json
{
  "segments": {
    "prefix": "...",
    "suffix": "..."
  },
  "filepath": "main.go"
}
```
**Response:** `NextEditSuggestionResponse` containing the predicted edit.

### Warp Action
`POST /api/warp/action`
Executes an agentic action using the Warp-compatible `AIAgentActionType` schema. Supported types: `RequestCommandOutput`, `ReadFiles`, `Grep`, `FileGlob`, `CallMCPTool`, `UseComputer`.

**Example (cURL):**
```bash
curl -X POST http://localhost:8080/api/warp/action \
  -H "Content-Type: application/json" \
  -d '{
    "type": "RequestCommandOutput",
    "params": {
      "command": "hostname"
    }
  }'
```

**Payload:**
```json
{
  "type": "RequestCommandOutput",
  "params": {
    "command": "ls -la"
  }
}
```
**Response:** Warp-compatible `ActionResponse`.

### Hyper Theme Sync
`POST /api/hyper/theme`
Synchronizes the terminal theme with a Hyper-compatible configuration. This affects the TUI's color palette and styling in real-time.

**Example (cURL):**
```bash
curl -X POST http://localhost:8080/api/hyper/theme \
  -H "Content-Type: application/json" \
  -d '{"config": "{\"config\": {\"colors\": {\"black\": \"#000000\"}}}"}'
```

**Payload:**
```json
{
  "config": "{ \"config\": { \"colors\": { \"black\": \"#000000\", ... } } }"
}
```
**Response:** `{"message": "Hyper theme synchronization initialized"}`

### Wave Action
`POST /api/wave/action`
Executes an agentic action using the Wave-compatible `aiusechat` schema. Supported types: `readfile`, `writefile`, `term`, `web`, `screenshot`.

**Example (cURL):**
```bash
curl -X POST http://localhost:8080/api/wave/action \
  -H "Content-Type: application/json" \
  -d '{
    "type": "readfile",
    "params": {
      "path": "go.mod"
    }
  }'
```

**Payload:**
```json
{
  "type": "readfile",
  "params": {
    "path": "go.mod"
  }
}
```
**Response:** Wave-compatible `ActionResponse`.

## Unified Tool Harness

The server also supports a wide range of assimilated tools via its internal harness. These can be invoked by agents or via specialized adapters. For a detailed specification of all available tools, see [TOOLS.md](TOOLS.md).

### Available Parity Tools
- `antigravity_auto_click`: Scans for and clicks common VS Code AI buttons.
- `computer`: Anthropic-style "Computer Use" (type, key, mouse_move, left_click).
- `memory`: Persistent key-value storage for agent state.
- `todo`: Task management within the project context.
- `search_files`: High-performance file content and name searching.
- `execute_code`: Python-based sandboxed code execution.
- `skill_manage`: Lifecycle management for self-improving agent skills.
- `browser_action`: Browser automation (launch, click, type, screenshot).
- `apply_patch`: OpenCode-style unified diff application (supports add/update/delete).
- `multiedit`: Multiple string replacements in a single file pass.
- `repo_map`: Ranked repository structure generation for LLM context optimization.

## Error Codes

| Status Code | Description |
| :--- | :--- |
| `200 OK` | Request processed successfully. |
| `400 Bad Request` | Invalid JSON payload or missing required parameters. |
| `401 Unauthorized` | (Reserved) Authentication failed. |
| `500 Internal Server Error` | Backend handler failed (e.g., no default model configured). |
| `503 Service Unavailable` | Model provider rate-limited or overloaded. |

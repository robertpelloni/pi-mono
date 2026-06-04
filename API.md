# Pi Agent API Documentation

This document describes the API endpoints provided by the Pi Agent server, including native and assimilated parity endpoints.

## Core Endpoints

### Health Check
`GET /api/health`
Returns the current status and version of the server.

### Chat Stream
`POST /api/chat`
Starts a streaming AI conversation.
**Payload:**
```json
{
  "sessionId": "optional_session_id",
  "message": "User prompt"
}
```
**Response:** SSE (Server-Sent Events) stream of `AgentEvent` objects.

### List Sessions
`GET /api/sessions`
Returns a list of active session IDs.

## Assimilated Parity Endpoints

### Tabby Completion (v1)
`POST /v1/completions`
Provides compatibility with Tabby-compatible IDE extensions.
**Payload:** Matches Tabby's `CompletionRequest` schema.
**Response:** Matches Tabby's `CompletionResponse` schema.

### Warp Action
`POST /api/warp/action`
Executes an agentic action using the Warp-compatible schema.
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

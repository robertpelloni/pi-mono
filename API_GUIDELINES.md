# Pi Agent API Development Guidelines

This document provides technical guidelines for developers extending the Pi Agent API or building new frontend adapters.

## Architecture Overview

The Pi Agent API follows a "Harness-First" design. Instead of building monolithic handlers, functionality is encapsulated into **Tools** and **Clean Room Handlers** in the `pkg/ai` package.

1.  **Parity Endpoints**: Map third-party schemas (Tabby, Warp, Wave) to native Go structures.
2.  **Unified Tool Harness**: Provides a standard interface for executing actions (`ExecuteTool`).
3.  **Registry**: Manages model providers, tool definitions, and session-agnostic state.

## Extending the API

### 1. Adding a New Parity Endpoint

To support a new external tool or IDE extension:

1.  **Define Schemas**: Add request/response structs in `pkg/ai/types.go` or a dedicated file if complex.
2.  **Create Handler Logic**: Add a method to `*ai.Registry` in `pkg/ai/registry_ext.go` (or a specialized clean-room file).
3.  **Register Route**: Add the handler to `pkg/server/server.go`.

**Example Pattern:**
```go
// pkg/ai/registry_ext.go
func (r *Registry) HandleNewToolAction(ctx context.Context, req *NewToolRequest) (*NewToolResponse, error) {
    // 1. Validation
    // 2. Logic (leverage existing tools via ai.Harness)
    // 3. Return response
}
```

### 2. Error Handling Conventions

- **Validation Errors**: Return `400 Bad Request` with a JSON body describing the error.
- **Handler Failures**: Return `500 Internal Server Error`. Do not leak sensitive system paths in error messages; use `ai.SanitizeError` if applicable.
- **Provider Timeouts**: Map LLM provider timeouts to `503 Service Unavailable`.

### 3. State Management

- **Global State**: Store configuration and tool registries in `Server.registry`.
- **Session State**: Store conversation history and agent state in `Server.sessions`.
- **Isolation**: Always use the `context.Context` from the HTTP request to ensure cancellations propagate to long-running AI tasks.

## Security Best Practices

### Input Validation
All file paths received via the API **must** be validated using `pkg/ai/security.go:ValidatePath`. This prevents directory traversal attacks.

```go
if err := security.ValidatePath(req.Filepath); err != nil {
    return nil, fmt.Errorf("security violation: %w", err)
}
```

### Sanitization
When returning command output or logs to the frontend, ensure ANSI escape codes are handled correctly or stripped if the frontend doesn't support them.

## Testing Guidelines

Every new endpoint must have:

1.  **Unit Tests**: Test the handler logic in `pkg/ai` without the HTTP overhead.
2.  **E2E Tests**: Add a case to `pkg/server/e2e_test.go` or `pkg/server/server_test.go` using `httptest`.
3.  **Mocking**: Use `pkg/ai/mock_provider.go` to simulate LLM responses in tests to avoid API costs and non-determinism.

### Running API Tests
```bash
go test -v ./pkg/server/...
```

## Documentation

When adding or modifying endpoints, update:
1.  `API.md`: For consumer-facing documentation.
2.  `CHANGELOG.md`: Record the version bump and new features.
3.  `TOOLS.md`: If new tools were added to the internal harness.

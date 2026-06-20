# API & Harness Maintenance Guide

This document provides instructions for maintaining and extending the Pi Agent's **Unified Tool Harness** and parity API endpoints.

## 1. Extending the Tool Harness
The harness resides in `pkg/ai/harness.go` and acts as the central router for all assimilated tools.

### Adding a New Tool
1. **Define the Handler**: Implement the tool logic in a new file in `pkg/ai/` or within `pkg/ai/clean_room_handlers.go`.
2. **Register in `CleanRoomTools`**: Add your handler to the map in `pkg/ai/clean_room_handlers.go`.
3. **Expose in `Harness`**: Update the `ExecuteTool` switch statement in `pkg/ai/harness.go` if specialized schema mapping is required.
4. **Update API Documentation**: Add the tool's signature to `API.md`.

## 2. Managing Parity Endpoints
Parity endpoints are hosted in `pkg/server/server.go`. They should strictly maintain 1:1 schema compatibility with the target tool (e.g., Tabby, Warp).

### Adding a New Parity Endpoint
1. **Define Request/Response Types**: In `pkg/ai/`, create types that match the target tool's JSON schema.
2. **Implement the Handler**: Add a handler method to the `Registry` in a tool-specific file (e.g., `pkg/ai/claude_code.go`).
3. **Register the Route**: Add the HTTP route to `s.mux` in `pkg/server/server.go`.
4. **Update INTEGRATION.md**: Provide instructions for users on how to point their existing tools to this new endpoint.

## 3. Registry Governance
The `Registry` (`pkg/ai/registry_ext.go`) manages model and provider state.

- **Isolation**: Ensure all registry operations use `r.mu` for thread safety.
- **Persistence**: The server uses a persistent registry instance. CLI instances are transient.
- **Fallback**: Maintain a "least-surprise" model selection policy in `GetDefaultModel()`.

## 4. Verification
After any change to the harness or APIs:
- Run all Go tests: `go test ./pkg/...`
- Run the E2E parity suite: `go test -v ./pkg/server/... -run TestServer_E2E`
- Perform manual black-box verification using `scripts/verify-parity.sh`.

## 5. Recovery Procedures
If the agent enters a FAILED state, follow the protocols defined in [RECOVERY.md](RECOVERY.md).

## 6. Submodule Protocol
When assimilating new tools:
1. Add the repo as a temporary submodule.
2. Document all features in `SUBMODULE_INVENTORY.md`.
3. Port functionality to native Go handlers.
4. Remove the submodule and update documentation.

# Integration Guide: Using Pi as a Backend

Pi Agent provides 1:1 API compatibility with several major AI coding tools and terminals. This guide explains how to configure these tools to use Pi as their primary LLM backend.

## 1. Tabby Integration
Pi supports the Tabby v1 completion and next-edit suggestion APIs.

### Configuration
1. Start the Pi server: `./pi-agent server --port 8080`
2. In your IDE (VS Code, IntelliJ), locate the Tabby extension settings.
3. Set the **Server URL** to: `http://localhost:8080`
4. Pi will now handle code completions and Fill-In-the-Middle (FIM) requests using its unified model registry.

### Features Supported
- Standard code completion (`/v1/completions`)
- Next-edit suggestions (`/v1/next-edit-suggestion`)
- RAG context retrieval (automatically injected via Pi's RepoMap)

---

## 2. Warp Integration
Pi implements Warp's `AIAgentActionType` protocol, allowing it to drive Warp's agentic workflows.

### Configuration
1. Ensure the Pi server is running.
2. Configure Warp's AI backend to point to Pi's action endpoint: `http://localhost:8080/api/warp/action`.
3. Warp actions will be routed through Pi's native Go handlers for terminal execution, file reading, and search.

### Handlers Implemented
- `RequestCommandOutput`: Executes shell commands and returns output.
- `ReadFiles`: Efficient single-file reading.
- `Grep` / `FileGlob`: High-performance search via Pi's internalized `ripgrep` and `find` tools.

---

## 3. Wave Integration
Wave uses the `aiusechat` toolset. Pi provides native parity for these actions.

### Configuration
1. Configure Wave to use Pi as the AI provider.
2. Map Wave's tool calls to Pi's parity route: `http://localhost:8080/api/wave/action`.

### Parity Actions
- `readfile` / `writefile`: Direct filesystem interaction.
- `term`: Command execution parity with Wave's shell.
- `web`: Browser-based information retrieval.

---

## 4. Universal Schema Mapping
Pi's **Unified Tool Harness** (`pkg/ai/harness.go`) performs the following transformations:
1. **Request Reception**: Receives proprietary JSON payloads from the target tool.
2. **Internal Mapping**: Converts parameters (e.g., Tabby's `segments` or Warp's `params`) into Pi's unified tool argument format.
3. **Execution**: Invokes the corresponding Go handler (e.g., `pkg/repomap` for context ranking).
4. **Response Formatting**: Re-wraps the result in the schema expected by the calling tool (e.g., `WarpActionResponse`).

---

## Troubleshooting
- **No Models Error**: Ensure you have registered at least one model in the registry (see `DEPLOY.md`).
- **Permission Denied**: Some tools (like `bash` or `writefile`) require appropriate system permissions for the user running the Pi server.

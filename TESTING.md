# Testing Strategy

Pi Agent uses a multi-layered testing strategy to ensure stability and correctness across its Go-based architecture.

## Unit Testing
Unit tests focus on individual components and packages.
- **`pkg/ai`**: Tests for model registry, tool harness, and parity handlers (Tabby, Warp, Wave).
- **`pkg/repomap`**: Tests for symbol extraction, file ranking, and repo map generation.
- **`pkg/opencode`**: Tests for patch parsing and multi-edit operations.
- **`pkg/server`**: Tests for HTTP handlers and request/response marshaling.

Run all unit tests:
```bash
go test ./pkg/...
```

## Integration Testing
Integration tests verify the interaction between multiple components, particularly the server and the tool harness.
- **`pkg/server/integration_test.go`**: Verifies that the server correctly routes requests to the tool harness.

## End-to-End (E2E) Testing
E2E tests verify the entire system flow, from HTTP request to tool execution.
- **`pkg/server/e2e_test.go`**: Simulates real agent flows for Tabby, Warp, and Wave.

Run E2E tests:
```bash
go test -v ./pkg/server/... -run TestServer_E2E
```

## Performance Benchmarking
Benchmarks are used to track the latency and throughput of critical path components.
- **`pkg/ai/harness_perf_test.go`**: Benchmarks for tool execution overhead and concurrency.
- **`pkg/repomap/repomap_test.go`**: Benchmarks for RepoMap generation on large repositories.

Run benchmarks:
```bash
go test -bench . ./pkg/...
```

## Continuous Integration (CI)
Our GitHub Actions pipeline (`.github/workflows/ci.yml`) automatically runs the following on every push and pull request:
1. **Build Verification**: Ensures the project compiles on Darwin, Linux, and Windows.
2. **Unit & E2E Tests**: Executes the full Go test suite.
3. **Legacy Checks**: Runs build and type-check on existing TypeScript packages.

## Prerequisites for Testing
Some tests require system-level tools:
- `ripgrep` (`rg`)
- `lynx`
- `xdotool`
- `scrot` (for screenshot tests)

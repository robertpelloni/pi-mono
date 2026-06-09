# Integration Testing Guide

This document outlines the strategy and procedures for verifying tool interactions and API parity within the Pi Agent ecosystem.

## 1. Automated Integration Tests
Our primary verification mechanism is the Go test suite located in `pkg/server/`.

### Running the Tests
To run all server-side integration and E2E tests:
```bash
go test -v ./pkg/server/...
```

### Key Test Categories
- **Parity Endpoints**: Verifies that `/v1/completions`, `/api/warp/action`, etc., correctly route and respond.
- **Complex Mutations**: Verifies multi-edit and patching logic via the `Harness`.
- **RepoMap**: Verifies the ranking and generation of codebase context.

## 2. Mocking for Development
Integration tests utilize a mock model provider (`pkg/server/integration_mock_test.go`) to simulate LLM responses without requiring external API keys. This ensures fast, reliable, and cost-effective testing of the routing and mapping logic.

## 3. Tool Verification (Live)
For high-fidelity verification against real tools (e.g., Warp, Tabby), we use the `scripts/verify-parity.sh` script.

### Usage
1. Start the Pi server:
   ```bash
   ./pi-agent server --port 8080
   ```
2. Run the verification script:
   ```bash
   ./scripts/verify-parity.sh 8080
   ```

## 4. Performance Benchmarking
We track the overhead of the **Unified Tool Harness** using benchmarks in `pkg/ai/harness_perf_test.go`.

Run benchmarks:
```bash
go test -bench . ./pkg/ai/...
```

## 5. Adding New Tests
When a new assimilated tool or parity endpoint is added:
1. Add a corresponding test case to `pkg/server/integration_test.go`.
2. (Optional) If the tool involves complex logic, add unit tests in the appropriate package (e.g., `pkg/repomap/`).
3. Update `scripts/verify-parity.sh` with a real-world cURL example.

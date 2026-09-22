# Contributing Guide

Welcome! We are currently transitioning to a Unified Go architecture.

## Workflow Guidelines
1. Do not push intermediate or broken states to `main`.
2. All tools and external LLM schemas must maintain 1:1 parity with their clean-room specifications. Check `docs/parity_schema_discrepancies.md`.
3. Adhere to test parity constraints by adding tests to `pkg/ai` or `pkg/tools` alongside new implementations.

## Community Plugin Hub

We encourage building new agent capabilities via the Community Plugin Hub (`pkg/plugins`) rather than cluttering the core Unified Harness.

To contribute a plugin:
1. Create your plugin struct in `pkg/plugins/your_plugin.go`.
2. Implement the `plugins.Plugin` interface:
   - `Name() string`
   - `Init(ctx context.Context) error`
   - `Validate() error` (Verify prerequisites, like installed binaries)
   - `GetTools() map[string]interface{}` (Return exact JSON schemas)
   - `HandleToolCall(ctx, toolName, args) (string, error)`
3. Register your plugin in `plugins.NewManager()` (`pkg/plugins/plugin.go`).
4. Add lifecycle tests to ensure it passes `Validate()` in standard environments.

# Ideas & Creative Improvements

This document tracks creative ideas for improving the project during the Go porting process.

## Architecture

- **Pluggable Architecture:** Use Go's `plugin` system (or a plugin architecture over gRPC like HashiCorp's `go-plugin`) to handle the optional features mentioned in the vision. This would keep the core truly minimal while allowing infinite extensibility, matching the original TS project's ethos but with a compiled language's power.
- **Unified Tool Registry:** Create a strongly-typed tool registry interface. When we implement the "clean room" tool schemas for Claude, Gemini, etc., we can map our single internal `Read` function to multiple different JSON schema representations expected by the respective models dynamically.

## Frontends

- **Fyne or Gio for Native Desktop:** Use [Fyne](https://fyne.io/) or [Gio](https://gioui.org/) to compile true native desktop applications from the exact same Go codebase as the CLI tool.
- **Bubbletea TUI:** Port the differential rendering from `packages/tui` into a robust Bubbletea-based terminal interface. Bubbletea (from Charm) is the standard for beautiful Go terminal apps.

## New Features

- **Context Compression via Local LLM:** Incorporate an optional submodule (perhaps using bindings to `llama.cpp`) to run a small local model that constantly summarizes and compresses the agent's context window, allowing infinite conversation length without hitting API token limits on expensive models.

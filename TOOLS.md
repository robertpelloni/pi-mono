# Integrated Tools Reference

This document provides a comprehensive list of all tools integrated into the Pi Agent, including their descriptions and input schemas. These tools are natively implemented in Go and provide 1:1 parity with their original implementations.

## Hermes Toolset (Assimilated)

### `execute_code`
Run a Python script that can call Hermes tools programmatically.
- **Properties:**
  - `code` (string, required): The Python script to execute.

### `cronjob`
Unified scheduled-task manager.
- **Properties:**
  - `action` (string, required): `create`, `list`, `update`, `pause`, `resume`, `run`, `remove`.
  - `schedule` (string): The cron schedule string.
  - `command` (string): The command to run on the schedule.

### `delegate_task`
Spawn one or more subagents to work on tasks in isolated contexts.
- **Properties:**
  - `task` (string, required): The task description for the subagent.
  - `context` (string): Additional context for the subagent.

### `write_file`
Write content to a file, completely replacing existing content.
- **Properties:**
  - `file_path` (string, required): Destination path.
  - `content` (string, required): The full content to write.

### `ha_call_service`
Call a Home Assistant service to control a device.
- **Properties:**
  - `domain` (string, required): HA domain (e.g., `light`).
  - `service` (string, required): HA service (e.g., `turn_on`).
  - `entity_id` (string): Target device ID.

### `memory`
Save important information to persistent memory.
- **Properties:**
  - `key` (string, required): Memory key.
  - `value` (string, required): Content to store.

### `mixture_of_agents`
Route a hard problem through multiple frontier LLMs collaboratively.
- **Properties:**
  - `prompt` (string, required): The combined prompt for the MOA engine.

### `web_search`
Search the web for information using Lynx.
- **Properties:**
  - `query` (string, required): Search terms.

---

## Computer Use (Open Interpreter Parity)

### `computer`
Interact with the primary monitor's screen, keyboard, and mouse.
- **Actions:** `key`, `type`, `mouse_move`, `left_click`, `screenshot`, etc.
- **Properties:**
  - `action` (string, required): The interaction type.
  - `text` (string): Text to type or key to press.
  - `coordinate` (number[]): [x, y] coordinates for mouse actions.

---

## IDE & Extension Parity

### `tabby_completion`
Routes request to the native Fill-In-the-Middle (FIM) code completion engine.

### `warp_action`
Executes agentic actions compatible with Warp's AIAgentActionType protocol.

### `hyper_theme_sync`
Synchronizes TUI styles with Hyper-compatible configuration strings.

### `repo_map`
Generates a ranked repository structure map to optimize context for long-context models.

---

## Patching & Editing

### `apply_patch`
Applies an OpenCode-style unified diff to the local filesystem.

### `multiedit`
Performs multiple surgical string replacements in a single file pass.

### `replace_lines` (Aider Parity)
Replaces a specific range of lines with new content.

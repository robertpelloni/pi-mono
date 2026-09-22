# Tool Parity: Schema & Signature Discrepancies

This document logs edge-case discrepancies in tool signatures observed during the Phase 20 assimilation, comparing newly integrated agents (Claude Code, Gemini CLI, Copilot) against native unified routes and earlier integrations (Warp, Tabby). These observations guide the `MapCleanRoomParams` normalization strategy for future patch cycles.

## 1. File Reading Tool Discrepancies
* **Native/Unified:** Expects `path`, `offset` (float/int), `limit` (float/int).
* **Claude Code (`read_file`):** Uses `file_path`, along with `offset` and `limit`.
* **Copilot CLI (`vscode_read`):** Uses `uri`.
* **Amp (`Amp_ReadFile`):** Uses `path`.
* **Resolution/Routing:** The `MapCleanRoomParams` engine must catch `uri`, `file_path`, and `filename` and alias them to `path`. `offset` and `limit` types must be dynamically cast (from float64 JSON outputs to int).

## 2. Command Execution Discrepancies
* **Native/Unified:** Expects `command`.
* **Claude Code (`bash`):** Uses `command`.
* **Aider / Amp (`run_command`, `Amp_RunCommand`):** Uses `cmd`.
* **Gemini CLI (`shell`):** Uses `script`.
* **Resolution/Routing:** The engine must alias `cmd` and `script` to `command`.

## 3. Structural & Nested Parameter Divergences
The most significant divergence occurs between "flat" parameter models and "nested" parameter models.

* **Claude Code & Gemini:** These agents typically produce "flat" JSON properties for their tool calls. For example, Claude's `edit` tool provides `file_path`, `old_string`, and `new_string` at the root of the `InputSchema`.
* **Warp (`warp_action`):** Groups all sub-parameters into a nested `params` object (e.g., `type: "RequestCommandOutput", params: { command: "..." }`).
* **Tabby (`tabby_completion`):** Heavily nested, grouping arguments inside `segments` (e.g., `segments: { prefix: "...", suffix: "..." }`).

**Impact:** Because Warp and Tabby schemas encapsulate their action data within nested JSON objects, they cannot easily be passed through the flat `MapCleanRoomParams` aliasing loop without custom logic. Currently, `Harness.ExecuteTool` handles `warp_action` and `tabby_completion` via hardcoded switch branches and dedicated unmarshaling structs (e.g., `WarpAgentAction`, `TabbyCompletionRequest`), bypassing the generic map parsing.

**Future Recommendation:** For Phase 22, consider implementing a recursive parameter flattening strategy inside `MapCleanRoomParams` or formalizing a nested-struct translation layer for all tool calls to eliminate the need for hardcoded switch statements in `ExecuteTool`.
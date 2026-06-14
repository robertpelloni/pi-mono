# Submodules Catalog (DEPRECATED)

All external submodules have been **fully assimilated** into the Go core as of version 0.83.0.
The submodules have been removed from the repository to reduce dependency sprawl and maintain a unified, high-performance Go architecture.

For a detailed breakdown of which features were internalized, see [SUBMODULE_INVENTORY.md](./SUBMODULE_INVENTORY.md).

| Project | Status | Internalized Implementation |
| :--- | :--- | :--- |
| `aider` | Assimilated | `replace_lines`, `repo_map`, git-integrated auto-commits |
| `cline` | Assimilated | `execute_command`, `write_to_file`, `browser_action` |
| `codebuff` | Assimilated | Multi-agent coordination via session branching |
| `goose` | Assimilated | 20+ native provider support |
| `hermes-agent` | Assimilated | 15+ native OS/Web tools (memory, todo, search, etc.) |
| `ii-agent` | Assimilated | MCP-based app integrations |
| `mistral-vibe` | Assimilated | Chat, file tools, subagent delegation |
| `ollama` | Assimilated | Native Ollama provider support |
| `open-interpreter` | Assimilated | `computer` (xdotool), sandboxed code execution |
| `opencode-cli` | Assimilated | Auto Drive/Review via session branching |
| `vscode-copilot-release`| Assimilated | Strictly mapped `vscode_read` and Copilot provider support |
| `tabby` | Assimilated | `/v1/completions`, RAG context, next-edit suggestions |
| `warp` | Assimilated | `/api/warp/action`, terminal blocks, agentic actions |
| `hyper` | Assimilated | Terminal theme synchronization and config parity |
| `wave` | Assimilated | `/api/wave/action`, `aiusechat` toolset parity |
| `antigravity-autopilot` | Assimilated | Auto-Click, Auto-Bump (Watchdog), Exit Detection |


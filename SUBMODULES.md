# Submodules Catalog

The following external open-source AI agent tooling has been cloned to the local repository as `git submodules` under the `submodules/` directory to serve as reference, analysis, and direct source code port targets for the `pi-mono` "Total Assimilation" effort.

| Project Directory | Git Hash / Version Tag | URL | Description |
| ----------------- | ---------------------- | --- | ----------- |
| `submodules/aider` | f09d70659ae90 (v0.86.3.dev) | https://github.com/paul-gauthier/aider | Comprehensive CLI coding agent. Features heavily reference edit-block algorithms. |
| `submodules/open-interpreter` | 06c796a2ce8b0 (v0.4.2) | https://github.com/OpenInterpreter/open-interpreter | AI executing code inside sandboxes using terminal bindings. |
| `submodules/hermes-agent` | 00ff9a26cd174 | https://github.com/nousresearch/hermes-agent | Powerful multi-tool execution framework containing 47 native OS/web interaction tools. |
| `submodules/ii-agent` | 0e57985d3f6e5 | https://github.com/Intelligent-Internet/ii-agent | Reference for recursive tool generation strategies and internet access. |
| `submodules/goose` | dc052f44470ab | https://github.com/block/goose | Cleanly architected structural generation frameworks. |
| `submodules/vscode-copilot-release` | 626080883d9b5 | https://github.com/microsoft/vscode-copilot-release | Leaked/extracted release versions of Copilot, useful to trace strict model/tool parity logic. |
| `submodules/opencode-cli` | 5b0c8300e56fd | https://github.com/just-every/code | Code CLI (Codex Fork). High-fidelity code interaction primitives. |
| `submodules/cline` | latest | https://github.com/cline/cline | Highly capable VS Code extension agent interface. Investigating its tool execution loops. |
| `submodules/ollama` | 7d271e6dc9fb1 (v0.13.4-rc2) | https://github.com/ollama/ollama | Local LLM inference wrapper, used for `ollama` model integrations and sync. |

## "Clean Room" Schemas Implementation Progress

We map strictly specific tool schemas that external LLM providers expect (from the models listed above) back to our unified TypeScript/Go implementations using the `clean-room-tools` pattern.

Currently tracked implementations:
- `claude_code_read_file` -> `read_file` -> TS `handleRead` / Go `HandleUnifiedRead`
- `vscode_read` -> `read`
- `aider_replace_lines` -> `patch`
- `hermes_terminal` -> `bash`
- `open_interpreter_execute` -> `execute_code`

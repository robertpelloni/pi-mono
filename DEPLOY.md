# Deployment & Environment Setup

## Prerequisites
- Go 1.24.3 or higher
- Ripgrep (`rg`) for advanced search tools
- Lynx (`lynx`) for text-based browser navigation
- Xdotool (`xdotool`) for Linux computer-use tools

## Building from Source
To build the native Go binary for your current platform:
```bash
go build -o pi-agent ./cmd/pi
```

## Cross-Platform Compilation
To generate binaries for all supported platforms (Darwin, Linux, Windows), run the provided build script:
```bash
./scripts/build-go.sh
```
Binaries will be available in the `dist/binaries/` directory.

## Security Sandbox
The agent can be restricted to a specific root directory for all file operations and shell commands by setting the `PI_ALLOWED_ROOT` environment variable.
```bash
export PI_ALLOWED_ROOT=/path/to/safe/workspace
```

## Configuration
The agent stores persistent state (skills, sessions, settings) in `~/.pi/`.
- **Skills**: `~/.pi/skills/`
- **Sessions**: `~/.pi/sessions/`
- **Settings**: `~/.pi/settings.json`

## Environment Variables
Set the following variables in your `.env` or shell:
- `OPENAI_API_KEY`: For OpenAI models.
- `ANTHROPIC_API_KEY`: For Anthropic models.
- `GOOGLE_API_KEY`: For Google Gemini models.
- `PI_AGENT_DIR`: Override default config directory (default: `~/.pi`).

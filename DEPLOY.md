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

## Unified Tool Harness
Pi includes a **Unified Tool Harness** that provides API parity with other major AI coding assistants. This allows you to use Pi as a backend for third-party IDE extensions and CLI tools.

### Enabling Parity Endpoints
The Go server automatically exposes several parity endpoints:
- **Tabby Compatibility**: `/v1/completions`
- **Warp Compatibility**: `/api/warp/action`
- **Wave Compatibility**: `/api/wave/action`

To start the parity server:
```bash
./pi-agent server
```

### Performance & Scaling
For high-concurrency environments, see [PERFORMANCE.md](PERFORMANCE.md) for benchmark results and resource recommendations.

For more details on request/response schemas, see [API.md](API.md).

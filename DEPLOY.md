# Deployment & Environment Setup (v0.97.0)

This guide covers the deployment of the **Ultimate LLM Harness**, a high-performance Go-based AI agent backend.

## Prerequisites
- Go 1.24.3 or higher
- Ripgrep (`rg`) for advanced search tools
- Lynx (`lynx`) for text-based browser navigation
- Xdotool (`xdotool`) for Linux computer-use tools

## Building from Source
To build the native Go binary for your current platform:
```bash
go build -o pi ./cmd/pi
```

### Automated Setup
Use the provided script to verify your system environment:
```bash
./scripts/setup-env.sh
```

## Cross-Platform Compilation
To generate binaries for all supported platforms (Darwin, Linux, Windows), run the provided build script:
```bash
./scripts/build-go.sh
```
Binaries will be available in the `dist/binaries/` as `pi-<os>-<arch>`.

## Staging vs. Production
Pi Agent uses the `PI_AGENT_DIR` environment variable to isolate environments.

### Staging Environment
- **Configuration**: Set `PI_AGENT_DIR=.pi-staging`.
- **Database**: Uses local staging files.
- **Port**: Default 8081.

### Production Environment
- **Configuration**: Default `~/.pi/`.
- **Database**: Uses persistent user data.
- **Port**: Default 8080.

## Configuration
The agent stores persistent state (skills, sessions, settings) in your configured agent directory.
- **Skills**: `skills/`
- **Sessions**: `sessions/`
- **Settings**: `settings.json`

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
./pi server
```

### Performance & Scaling
For high-concurrency environments, see [PERFORMANCE.md](PERFORMANCE.md) for benchmark results and resource recommendations.

For more details on request/response schemas, see [API.md](API.md).

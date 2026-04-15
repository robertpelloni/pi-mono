# Deployment Instructions

This document always contains the latest detailed deployment instructions.

## Current State (TypeScript)

The current TypeScript monorepo relies on npm.

To build and run locally:
```bash
npm install
npm run build
npm run check
./test.sh
```

## Go Port Deployment Strategy (Planned)

The future Go project will compile to a single, statically linked binary, significantly simplifying deployment compared to the Node.js ecosystem.

### Prerequisites
- Go 1.24+

### Building
```bash
go build -o pi-agent ./cmd/pi
```

### Running
```bash
./pi-agent
```

*(This file will be updated with containerization, orchestration, and CI/CD specifics as the Go port develops).*

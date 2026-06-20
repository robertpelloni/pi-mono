#!/usr/bin/env bash
# scripts/setup-env.sh
# Checks for prerequisites required for Pi Agent v0.97.0 deployment.

set -e

echo "Checking Pi Agent Prerequisites..."

# 1. Go
if command -v go >/dev/null 2>&1; then
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo "[PASS] Go $GO_VERSION found."
else
    echo "[FAIL] Go not found. Please install Go 1.24+."
fi

# 2. Ripgrep
if command -v rg >/dev/null 2>&1; then
    echo "[PASS] Ripgrep found."
else
    echo "[WARN] Ripgrep (rg) not found. search_files tool will be limited."
fi

# 3. Lynx
if command -v lynx >/dev/null 2>&1; then
    echo "[PASS] Lynx found."
else
    echo "[WARN] Lynx not found. browser_navigate and web_search tools will fail."
fi

# 4. Xdotool (Linux only)
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    if command -v xdotool >/dev/null 2>&1; then
        echo "[PASS] Xdotool found."
    else
        echo "[WARN] Xdotool not found. use_computer tool will fail."
    fi
fi

echo "Prerequisite check complete."

#!/bin/bash
PORT=${1:-8081}
BASE_URL="http://localhost:$PORT"

echo "Verifying Pi Parity Ecosystem on $BASE_URL..."

# 1. Tabby Completion
echo -n "[Tabby] Verifying Completion... "
curl -s -X POST $BASE_URL/v1/completions   -H "Content-Type: application/json"   -d '{"segments": {"prefix": "func"}, "language": "go"}' | grep -q "tabby-" && echo "PASS" || echo "FAIL"

# 2. Warp Action
echo -n "[Warp] Verifying Action... "
curl -s -X POST $BASE_URL/api/warp/action   -H "Content-Type: application/json"   -d '{"type": "RequestCommandOutput", "params": {"command": "echo warp-parity"}}' | grep -q "success" && echo "PASS" || echo "FAIL"

# 3. Wave Action
echo -n "[Wave] Verifying Action... "
curl -s -X POST $BASE_URL/api/wave/action   -H "Content-Type: application/json"   -d '{"type": "readfile", "params": {"path": "go.mod"}}' | grep -q "success" && echo "PASS" || echo "FAIL"

echo "Verification Complete."

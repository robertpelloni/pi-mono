#!/bin/bash
# scripts/chat-smoke-test.sh
# Verifies the full agent loop (Chat -> Tool -> Answer) via SSE.

PORT=${1:-8080}
URL="http://localhost:$PORT/api/chat"

echo "Running Chat Smoke Test on $URL..."

# Note: In a real test we need a mock model provider to return a tool call.
# This script assumes the server is started with --test-mode or similar
# that injects the mock coord provider used in e2e_test.go.
# For this smoke test, we'll verify the endpoint accepts the request and streams SSE.

TEMP_LOG=$(mktemp)

curl -N -s -X POST "$URL" \
  -H "Content-Type: application/json" \
  -d '{"message": "Verify the agent harness is active."}' > "$TEMP_LOG" &

CURL_PID=$!
sleep 3
kill $CURL_PID

if grep -q "data:" "$TEMP_LOG"; then
  echo "[PASS] SSE stream received."
else
  echo "[FAIL] No SSE data received."
  cat "$TEMP_LOG"
  exit 1
fi

rm "$TEMP_LOG"
echo "Chat Smoke Test Complete."

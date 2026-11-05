#!/bin/bash

# tests/parity/conformance_action_launch.sh
#
# This script executes the tool schema conformance tests for the `browser_action` (launch)
# URL validation edge cases, ensuring 1:1 parity for Claude Code and Gemini schemas.
#
# Prerequisite: A local staging instance must be running:
# PI_AGENT_DIR=.pi-staging ./pi --port 8081

set -e

ENDPOINT="http://localhost:8081/v1/tools/execute"

echo "Running Conformance Tests: browser_action (launch) URL Validation"
echo "------------------------------------------------------------------"

# Helper function to fire curl requests and assert expectations
run_test() {
  local test_name=$1
  local payload=$2
  local expected_status=$3
  local expected_snippet=$4

  echo -n "[TEST] $test_name ... "

  response=$(curl -s -X POST $ENDPOINT \
    -H "Content-Type: application/json" \
    -d "$payload")

  if echo "$response" | grep -q "$expected_snippet"; then
    echo "PASS"
  else
    echo "FAIL"
    echo "  Expected snippet: '$expected_snippet'"
    echo "  Actual response: $response"
    exit 1
  fi
}

# 1. Valid HTTP Schema (Claude Code)
PAYLOAD_CLAUDE_VALID='{
  "tool_name": "browser_action",
  "args": {
    "action": "launch",
    "url": "http://example.com"
  }
}'
run_test "Claude Code - Valid HTTP URL" "$PAYLOAD_CLAUDE_VALID" 200 "Example Domain"

# 2. Malformed / Protocol-less Schema (Gemini)
PAYLOAD_GEMINI_MISSING_PROTOCOL='{
  "tool_name": "browser_action",
  "args": {
    "action": "launch",
    "url": "example.com"
  }
}'
run_test "Gemini - Missing Protocol Rejected" "$PAYLOAD_GEMINI_MISSING_PROTOCOL" 200 "security violation: URL must begin with http:// or https://"

# 3. File Traversal Schema (Claude/Gemini crossover)
PAYLOAD_TRAVERSAL='{
  "tool_name": "browser_action",
  "args": {
    "action": "launch",
    "url": "file:///etc/passwd"
  }
}'
run_test "Cross-Schema - file:// schema rejected" "$PAYLOAD_TRAVERSAL" 200 "security violation"

# 4. Empty URL Schema
PAYLOAD_EMPTY='{
  "tool_name": "browser_action",
  "args": {
    "action": "launch",
    "url": ""
  }
}'
run_test "Cross-Schema - Empty URL rejected" "$PAYLOAD_EMPTY" 200 "URL cannot be empty"

echo "------------------------------------------------------------------"
echo "All browser_action launch conformance tests passed successfully."

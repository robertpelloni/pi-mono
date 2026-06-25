#!/bin/bash
# scripts/load-test.sh
# Simple load test for Pi Parity Endpoints

PORT=${1:-8080}
CONCURRENCY=${2:-5}
REQUESTS=${3:-20}
URL="http://localhost:$PORT"

echo "Starting load test on $URL with concurrency $CONCURRENCY for $REQUESTS requests per endpoint..."

test_endpoint() {
  local name=$1
  local path=$2
  local data=$3
  echo "Testing $name ($path)..."

  start_time=$(date +%s%N)

  for ((i=1; i<=REQUESTS; i++)); do
    (
      curl -s -X POST "$URL$path" \
        -H "Content-Type: application/json" \
        -d "$data" > /dev/null
    ) &

    if (( i % CONCURRENCY == 0 )); then
      wait
    fi
  done
  wait

  end_time=$(date +%s%N)
  duration=$(( (end_time - start_time) / 1000000 ))
  avg=$(( duration / REQUESTS ))

  echo "  $name: Total ${duration}ms, Avg ${avg}ms/req"
}

# 1. Tabby Completion
test_endpoint "Tabby" "/v1/completions" '{"segments": {"prefix": "func test() {"}, "language": "go"}'

# 2. Warp Action
test_endpoint "Warp" "/api/warp/action" '{"type": "RequestCommandOutput", "params": {"command": "echo load-test"}}'

# 3. Wave Action
test_endpoint "Wave" "/api/wave/action" '{"type": "readfile", "params": {"path": "go.mod"}}'

echo "Load Test Complete."

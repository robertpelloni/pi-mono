#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist/binaries
PLATFORMS=("darwin/arm64" "darwin/amd64" "linux/amd64" "linux/arm64" "windows/amd64")
for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS=${PLATFORM%/*}
    GOARCH=${PLATFORM#*/}
    OUTPUT="dist/binaries/pi-${GOOS}-${GOARCH}"
    [ "$GOOS" == "windows" ] && OUTPUT="$OUTPUT.exe"
    echo "Building for $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build -o "$OUTPUT" ./cmd/pi
done
ls -lh dist/binaries/

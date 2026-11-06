#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist/binaries
PLATFORMS=("darwin/arm64" "darwin/amd64" "linux/amd64" "linux/arm64" "windows/amd64")
if [ -f "pkg/VERSION.md" ]; then
    VERSION=$(tr -d '\r\n' < pkg/VERSION.md)
else
    VERSION="0.98.0"
fi
for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS=${PLATFORM%/*}
    GOARCH=${PLATFORM#*/}
    OUTPUT="dist/binaries/pi-${GOOS}-${GOARCH}"
    [ "$GOOS" == "windows" ] && OUTPUT="$OUTPUT.exe"
    echo "Building v$VERSION for $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-X main.Version=$VERSION" -o "$OUTPUT" ./cmd/pi
done
ls -lh dist/binaries/

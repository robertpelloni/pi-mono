#!/usr/bin/env bash
#
# Build pi-go binaries for all platforms.
#
# Usage:
#   ./scripts/build-go-binaries.sh [--platform <platform>]
#
# Options:
#   --platform <name>   Build only for specified platform
#                       (darwin-arm64, darwin-x64, linux-x64, linux-arm64, windows-x64)
#
# Output:
#   binaries/
#     pi-go-darwin-arm64
#     pi-go-darwin-x64
#     pi-go-linux-x64
#     pi-go-linux-arm64
#     pi-go-windows-x64.exe

set -euo pipefail
cd "$(dirname "$0")/.."

VERSION=$(grep 'const Version' pkg/version.go | sed 's/.*= *"\(.*\)".*/\1/')
LDFLAGS="-X main.Version=$VERSION"

PLATFORM=""
while [[ $# -gt 0 ]]; do
  case $1 in
    --platform)
      PLATFORM="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Default: all platforms
if [[ -n "$PLATFORM" ]]; then
  PLATFORMS=("$PLATFORM")
else
  PLATFORMS=(darwin-arm64 darwin-x64 linux-x64 linux-arm64 windows-x64)
fi

echo "==> Building pi-go v${VERSION}..."
echo "==> Running tests..."
go test ./pkg/... -timeout 30s

echo "==> Building binaries..."
rm -rf binaries
mkdir -p binaries

for platform in "${PLATFORMS[@]}"; do
  echo "Building for $platform..."

  case "$platform" in
    darwin-arm64)
      GOOS=darwin GOARCH=arm64 go build -ldflags="$LDFLAGS" -o "binaries/pi-go-$platform" ./cmd/pi/main.go
      ;;
    darwin-x64)
      GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "binaries/pi-go-$platform" ./cmd/pi/main.go
      ;;
    linux-x64)
      GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "binaries/pi-go-$platform" ./cmd/pi/main.go
      ;;
    linux-arm64)
      GOOS=linux GOARCH=arm64 go build -ldflags="$LDFLAGS" -o "binaries/pi-go-$platform" ./cmd/pi/main.go
      ;;
    windows-x64)
      GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "binaries/pi-go-$platform.exe" ./cmd/pi/main.go
      ;;
    *)
      echo "Invalid platform: $platform"
      exit 1
      ;;
  esac
done

echo ""
echo "==> Build complete!"
echo "Binaries available in binaries/:"
ls -lh binaries/ 2>/dev/null || true
echo ""
echo "Version: $VERSION"

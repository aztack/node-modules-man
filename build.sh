#!/usr/bin/env bash
set -euo pipefail

# Build script for node-module-man
# Usage examples:
#   ./build.sh                      # build macOS (arm64, amd64) into ./dist
#   GOOS=linux GOARCH=amd64 ./build.sh  # build one target based on env
#   ./build.sh all                  # build common targets (darwin/linux/win)

APP_NAME="node-module-man"
OUT_DIR="dist"
VERSION=${VERSION:-dev}
LDFLAGS="-s -w -X main.version=${VERSION}"
PKG="./cmd/node-module-man"

mkdir -p "${OUT_DIR}"

build_target() {
  local goos="$1" arch="$2" ext=""
  local bin="${APP_NAME}"
  if [[ "$goos" == "windows" ]]; then ext=".exe"; fi
  bin+="-${goos}-${arch}${ext}"
  echo "==> Building ${bin}"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$arch" \
    go build -trimpath -ldflags "$LDFLAGS" -o "${OUT_DIR}/${bin}" "$PKG"
}

if [[ "${1:-mac}" == "all" ]]; then
  build_target darwin arm64
  build_target darwin amd64
  build_target linux amd64
  build_target windows amd64
  echo "Binaries in ${OUT_DIR}/"
  exit 0
fi

# If GOOS/GOARCH provided, build that single target; otherwise default to macOS universal pair
if [[ -n "${GOOS:-}" && -n "${GOARCH:-}" ]]; then
  build_target "$GOOS" "$GOARCH"
else
  build_target darwin arm64
  build_target darwin amd64
fi

echo "Binaries in ${OUT_DIR}/"

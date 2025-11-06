#!/usr/bin/env bash
set -euo pipefail

# Build script for node-module-man
# Usage examples:
#   ./build.sh                              # build macOS (arm64, amd64) into ./dist
#   GOOS=linux GOARCH=amd64 ./build.sh      # build one target based on env
#   ./build.sh all                          # build common targets (darwin/linux/win)
#   ./build.sh release                      # build + package archives for release (multi-OS)

APP_NAME="node-module-man"
OUT_DIR="dist"
VERSION=${VERSION:-dev}
LDFLAGS="-s -w -X main.version=${VERSION}"
PKG="./cmd/node-module-man"

mkdir -p "${OUT_DIR}"

build_target() {
  local goos="$1" arch="$2" ext="" outbin
  outbin="${OUT_DIR}/${APP_NAME}-${goos}-${arch}"
  if [[ "$goos" == "windows" ]]; then ext=".exe"; fi
  echo "==> Building ${APP_NAME}-${goos}-${arch}${ext}"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$arch" \
    go build -trimpath -ldflags "$LDFLAGS" -o "${outbin}${ext}" "$PKG"
}

# Package a built artifact into tar.gz (unix) or zip (windows)
package_target() {
  local goos="$1" arch="$2" ext="" base name archive tmpbin
  if [[ "$goos" == "windows" ]]; then ext=".exe"; fi
  base="${APP_NAME}-${goos}-${arch}${ext}"
  name="${APP_NAME}_${VERSION}_${goos}_${arch}"
  archive="${OUT_DIR}/${name}"
  tmpbin="${OUT_DIR}/${APP_NAME}${ext}"
  # Repack binary without GOOS/GOARCH suffix inside archive for a clean layout
  cp "${OUT_DIR}/${APP_NAME}-${goos}-${arch}${ext}" "$tmpbin"
  if [[ "$goos" == "windows" ]]; then
    echo "==> Packaging ${name}.zip"
    (cd "${OUT_DIR}" && zip -q -9 "${name}.zip" "${APP_NAME}${ext}")
  else
    echo "==> Packaging ${name}.tar.gz"
    (cd "${OUT_DIR}" && tar -czf "${name}.tar.gz" "${APP_NAME}${ext}")
  fi
  rm -f "$tmpbin"
}

# Compute sha256 checksums for release files
checksum_files() {
  local files=("$@")
  local sumfile="${OUT_DIR}/checksums_${VERSION}.txt"
  : > "$sumfile"
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$OUT_DIR" && sha256sum "${files[@]##${OUT_DIR}/}") >> "$sumfile"
  else
    # macOS fallback
    (cd "$OUT_DIR" && shasum -a 256 "${files[@]##${OUT_DIR}/}") >> "$sumfile"
  fi
  echo "==> Wrote checksums to ${sumfile}"
}

case "${1:-}" in
  all)
    build_target darwin arm64
    build_target darwin amd64
    build_target linux amd64
    build_target linux arm64
    build_target windows amd64
    build_target windows arm64
    echo "Binaries in ${OUT_DIR}/"
    exit 0
    ;;
  release)
    # Build and package for common OS/arch combos
    targets=(
      "darwin arm64"
      "darwin amd64"
      "linux amd64"
      "linux arm64"
      "windows amd64"
      "windows arm64"
    )
    archives=()
    for t in "${targets[@]}"; do
      read -r goos arch <<< "$t"
      build_target "$goos" "$arch"
      package_target "$goos" "$arch"
      if [[ "$goos" == "windows" ]]; then
        archives+=("${OUT_DIR}/${APP_NAME}_${VERSION}_${goos}_${arch}.zip")
      else
        archives+=("${OUT_DIR}/${APP_NAME}_${VERSION}_${goos}_${arch}.tar.gz")
      fi
    done
    checksum_files "${archives[@]}"
    echo "Release artifacts in ${OUT_DIR}/"
    exit 0
    ;;
esac

# If GOOS/GOARCH provided, build that single target; otherwise default to macOS universal pair
if [[ -n "${GOOS:-}" && -n "${GOARCH:-}" ]]; then
  build_target "$GOOS" "$GOARCH"
else
  build_target darwin arm64
  build_target darwin amd64
fi

echo "Binaries in ${OUT_DIR}/"

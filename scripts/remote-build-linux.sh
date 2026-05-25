#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
OUT_DIR="${ROOT_DIR}/artifacts/remote-build/${TIMESTAMP}"
FRONTEND_DIR="${ROOT_DIR}/web"
BACKEND_BIN_NAME="nofx-linux-amd64"
ARCHIVE_NAME="nofx-remote-build-${TIMESTAMP}.tar.gz"

GOOS_TARGET="${GOOS_TARGET:-linux}"
GOARCH_TARGET="${GOARCH_TARGET:-amd64}"
NODE_BIN="${NODE_BIN:-node}"
NPM_BIN="${NPM_BIN:-npm}"
GO_BIN="${GO_BIN:-go}"

log() {
  printf '\n[%s] %s\n' "$(date '+%F %T')" "$*"
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing required command: $1" >&2
    exit 1
  }
}

log "Root: ${ROOT_DIR}"
need_cmd "$NODE_BIN"
need_cmd "$NPM_BIN"
need_cmd "$GO_BIN"
need_cmd tar
need_cmd cp
need_cmd rsync

mkdir -p "${OUT_DIR}/frontend" "${OUT_DIR}/backend" "${OUT_DIR}/meta"

log "Node version: $($NODE_BIN -v)"
log "npm version: $($NPM_BIN -v)"
log "Go version: $($GO_BIN version)"

log "Installing frontend dependencies"
cd "$FRONTEND_DIR"
"$NPM_BIN" ci

log "Type-checking frontend"
./node_modules/.bin/tsc --noEmit

log "Building frontend"
"$NPM_BIN" run build

log "Copying frontend dist"
rsync -a --delete "$FRONTEND_DIR/dist/" "${OUT_DIR}/frontend/dist/"

log "Building backend binary (${GOOS_TARGET}/${GOARCH_TARGET})"
cd "$ROOT_DIR"
CGO_ENABLED=1 GOOS="$GOOS_TARGET" GOARCH="$GOARCH_TARGET" "$GO_BIN" build -trimpath -o "${OUT_DIR}/backend/${BACKEND_BIN_NAME}" ./main.go
chmod +x "${OUT_DIR}/backend/${BACKEND_BIN_NAME}"

log "Writing build metadata"
{
  echo "timestamp=${TIMESTAMP}"
  echo "root=${ROOT_DIR}"
  echo "git_commit=$(git rev-parse HEAD)"
  echo "git_branch=$(git rev-parse --abbrev-ref HEAD)"
  echo "node_version=$($NODE_BIN -v)"
  echo "npm_version=$($NPM_BIN -v)"
  echo "go_version=$($GO_BIN version)"
  echo "goos=${GOOS_TARGET}"
  echo "goarch=${GOARCH_TARGET}"
} > "${OUT_DIR}/meta/build-info.txt"

git status --short > "${OUT_DIR}/meta/git-status.txt" || true
git diff --stat > "${OUT_DIR}/meta/git-diff-stat.txt" || true

log "Creating archive ${ARCHIVE_NAME}"
cd "${ROOT_DIR}/artifacts/remote-build"
tar -czf "${ARCHIVE_NAME}" "${TIMESTAMP}"

cat <<EOF

Build finished.

Artifact directory:
  ${OUT_DIR}

Archive:
  ${ROOT_DIR}/artifacts/remote-build/${ARCHIVE_NAME}

Suggested sync-back targets on the weak machine:
  - web/dist/
  - backend/${BACKEND_BIN_NAME}
  - meta/build-info.txt
EOF

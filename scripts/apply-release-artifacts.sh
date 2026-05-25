#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RELEASE_DIR="${ROOT_DIR}/release-artifacts"
FRONTEND_SRC="${RELEASE_DIR}/frontend/dist"
FRONTEND_DST="${ROOT_DIR}/web/dist"
BACKEND_SRC="${RELEASE_DIR}/backend/nofx-linux-amd64"
BACKEND_DST="${ROOT_DIR}/nofx"
META_FILE="${RELEASE_DIR}/meta/build-info.txt"

log() {
  printf '\n[%s] %s\n' "$(date '+%F %T')" "$*"
}

if [ ! -d "$RELEASE_DIR" ]; then
  echo "release-artifacts directory not found: $RELEASE_DIR" >&2
  exit 1
fi

if [ -f "$BACKEND_SRC" ]; then
  log "Applying backend artifact"
  cp -f "$BACKEND_SRC" "$BACKEND_DST"
  chmod +x "$BACKEND_DST"
  ls -lah "$BACKEND_DST"
else
  log "Backend artifact missing: $BACKEND_SRC"
fi

if [ -d "$FRONTEND_SRC" ] && find "$FRONTEND_SRC" -mindepth 1 -print -quit >/dev/null 2>&1; then
  log "Applying frontend artifact"
  mkdir -p "$FRONTEND_DST"
  rsync -a --delete "$FRONTEND_SRC/" "$FRONTEND_DST/"
  find "$FRONTEND_DST" -maxdepth 2 -type f | sed -n '1,40p'
else
  log "Frontend artifact missing or empty: $FRONTEND_SRC"
fi

if [ -f "$META_FILE" ]; then
  log "Build metadata"
  sed -n '1,80p' "$META_FILE"
fi

log "Done"

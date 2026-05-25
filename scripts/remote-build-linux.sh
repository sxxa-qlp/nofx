#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
OUT_DIR="${ROOT_DIR}/artifacts/remote-build/${TIMESTAMP}"
FRONTEND_DIR="${ROOT_DIR}/web"
TOOLS_DIR="${ROOT_DIR}/.tools"
LOCAL_GO_DIR="${TOOLS_DIR}/go"
BACKEND_BIN_NAME="nofx-linux-amd64"
ARCHIVE_NAME="nofx-remote-build-${TIMESTAMP}.tar.gz"

GOOS_TARGET="${GOOS_TARGET:-linux}"
GOARCH_TARGET="${GOARCH_TARGET:-amd64}"
GO_REQUIRED_VERSION="${GO_REQUIRED_VERSION:-1.25.3}"
GO_INSTALL_VERSION="${GO_INSTALL_VERSION:-1.26.2}"
NODE_BIN="${NODE_BIN:-node}"
NPM_BIN="${NPM_BIN:-npm}"
GO_BIN="${GO_BIN:-go}"

PKG_MANAGER=""
SUDO=""

log() {
  printf '\n[%s] %s\n' "$(date '+%F %T')" "$*"
}

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

version_ge() {
  [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]
}

use_sudo_if_needed() {
  if [ "$(id -u)" -eq 0 ]; then
    SUDO=""
  elif has_cmd sudo; then
    SUDO="sudo"
  else
    fail "Need root or sudo to install missing packages"
  fi
}

detect_package_manager() {
  if has_cmd apt-get; then
    PKG_MANAGER="apt"
  elif has_cmd dnf; then
    PKG_MANAGER="dnf"
  elif has_cmd yum; then
    PKG_MANAGER="yum"
  elif has_cmd apk; then
    PKG_MANAGER="apk"
  else
    PKG_MANAGER=""
  fi
}

pkg_install() {
  detect_package_manager
  use_sudo_if_needed
  case "$PKG_MANAGER" in
    apt)
      $SUDO apt-get update
      DEBIAN_FRONTEND=noninteractive $SUDO apt-get install -y "$@"
      ;;
    dnf)
      $SUDO dnf install -y "$@"
      ;;
    yum)
      $SUDO yum install -y "$@"
      ;;
    apk)
      $SUDO apk add --no-cache "$@"
      ;;
    *)
      fail "No supported package manager found to install: $*"
      ;;
  esac
}

ensure_base_packages() {
  local missing=()
  for cmd in curl tar rsync git; do
    has_cmd "$cmd" || missing+=("$cmd")
  done
  if [ ${#missing[@]} -eq 0 ]; then
    return
  fi

  log "Installing missing base packages: ${missing[*]}"
  case "$PKG_MANAGER" in
    apt) pkg_install curl tar rsync git ;;
    dnf) pkg_install curl tar rsync git ;;
    yum) pkg_install curl tar rsync git ;;
    apk) pkg_install curl tar rsync git ;;
    *) fail "Cannot install base packages automatically" ;;
  esac
}

ensure_node() {
  if has_cmd "$NODE_BIN" && has_cmd "$NPM_BIN"; then
    return
  fi

  log "Node/npm missing; attempting automatic install"
  case "$PKG_MANAGER" in
    apt) pkg_install nodejs npm ;;
    dnf) pkg_install nodejs npm ;;
    yum) pkg_install nodejs npm ;;
    apk) pkg_install nodejs npm ;;
    *) fail "Cannot auto-install node/npm on this distro" ;;
  esac

  has_cmd "$NODE_BIN" || fail "node still missing after installation"
  has_cmd "$NPM_BIN" || fail "npm still missing after installation"
}

get_go_version() {
  local bin="$1"
  "$bin" version | awk '{print $3}' | sed 's/^go//'
}

install_local_go() {
  mkdir -p "$TOOLS_DIR"
  local arch="amd64"
  case "$(uname -m)" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) fail "Unsupported architecture for auto Go install: $(uname -m)" ;;
  esac

  local tarball="go${GO_INSTALL_VERSION}.linux-${arch}.tar.gz"
  local url="https://go.dev/dl/${tarball}"
  local tmp="${TOOLS_DIR}/${tarball}"

  log "Installing local Go ${GO_INSTALL_VERSION} from ${url}"
  rm -rf "$LOCAL_GO_DIR"
  curl -fL "$url" -o "$tmp"
  tar -C "$TOOLS_DIR" -xzf "$tmp"
  rm -f "$tmp"

  [ -x "$LOCAL_GO_DIR/bin/go" ] || fail "Local Go install failed"
}

ensure_go() {
  local chosen_go=""

  if [ -x "$LOCAL_GO_DIR/bin/go" ]; then
    local local_ver
    local_ver="$(get_go_version "$LOCAL_GO_DIR/bin/go")"
    if version_ge "$local_ver" "$GO_REQUIRED_VERSION"; then
      chosen_go="$LOCAL_GO_DIR/bin/go"
    fi
  fi

  if [ -z "$chosen_go" ] && has_cmd "$GO_BIN"; then
    local sys_ver
    sys_ver="$(get_go_version "$GO_BIN")"
    if version_ge "$sys_ver" "$GO_REQUIRED_VERSION"; then
      chosen_go="$(command -v "$GO_BIN")"
    fi
  fi

  if [ -z "$chosen_go" ]; then
    log "No suitable Go found (need >= ${GO_REQUIRED_VERSION}); installing local Go ${GO_INSTALL_VERSION}"
    ensure_base_packages
    install_local_go
    chosen_go="$LOCAL_GO_DIR/bin/go"
  fi

  GO_BIN="$chosen_go"
}

log "Root: ${ROOT_DIR}"
detect_package_manager
ensure_base_packages
ensure_node
ensure_go

log "Using node: $(command -v "$NODE_BIN")"
log "Using npm: $(command -v "$NPM_BIN")"
log "Using go: ${GO_BIN}"
log "Node version: $($NODE_BIN -v)"
log "npm version: $($NPM_BIN -v)"
log "Go version: $($GO_BIN version)"

mkdir -p "${OUT_DIR}/frontend" "${OUT_DIR}/backend" "${OUT_DIR}/meta"

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
  echo "node_path=$(command -v "$NODE_BIN")"
  echo "node_version=$($NODE_BIN -v)"
  echo "npm_path=$(command -v "$NPM_BIN")"
  echo "npm_version=$($NPM_BIN -v)"
  echo "go_path=${GO_BIN}"
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

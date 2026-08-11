#!/usr/bin/env bash
# build.sh — Cross-platform build script for Quill (Go + Wails)
# Supports: macOS (Intel/Apple Silicon), Linux (amd64/arm64)
# Usage: ./build.sh [--platform <os/arch>] [--clean] [--dev]
set -euo pipefail

VERSION="${VERSION:-0.1.0}"
APP_NAME="quill"
BUILD_DIR="build/bin"
FRONTEND_DIR="frontend"

# Ensure Go + Wails are discoverable
export PATH="/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; NC='\033[0m'

log()  { printf "${BLUE}[build]${NC} %s\n" "$*"; }
ok()   { printf "${GREEN}[  ok ]${NC} %s\n" "$*"; }
warn() { printf "${YELLOW}[warn]${NC} %s\n" "$*"; }
die()  { printf "${RED}[FAIL]${NC} %s\n" "$*" >&2; exit 1; }

CLEAN=false
DEV=false
TARGET_PLATFORM=""

usage() {
  cat <<EOF
Usage: $0 [OPTIONS]

Options:
  --platform <os/arch>   Target platform (e.g. darwin/arm64, linux/amd64)
  --clean                Remove build artifacts before building
  --dev                  Build in dev mode (debug info, no optimization)
  --version <ver>        Set version string (default: $VERSION)
  -h, --help             Show this help

Examples:
  ./build.sh                           # Build for current platform
  ./build.sh --platform darwin/arm64   # Cross-compile for macOS ARM
  ./build.sh --clean --dev             # Clean dev build
EOF
  exit 0
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --platform)  TARGET_PLATFORM="$2"; shift 2 ;;
    --clean)     CLEAN=true; shift ;;
    --dev)       DEV=true; shift ;;
    --version)   VERSION="$2"; shift 2 ;;
    -h|--help)   usage ;;
    *)           die "Unknown option: $1" ;;
  esac
done

detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$os" in
    darwin) os="darwin" ;;
    linux)  os="linux" ;;
    *)      die "Unsupported OS: $os" ;;
  esac
  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)             die "Unsupported arch: $arch" ;;
  esac
  echo "${os}/${arch}"
}

CURRENT_PLATFORM="$(detect_platform)"
[[ -z "$TARGET_PLATFORM" ]] && TARGET_PLATFORM="$CURRENT_PLATFORM"
TARGET_OS="${TARGET_PLATFORM%/*}"
TARGET_ARCH="${TARGET_PLATFORM#*/}"

log "Quill Build System v${VERSION}"
log "Target: ${TARGET_PLATFORM}  (host: ${CURRENT_PLATFORM})"

# ── Detect webkit2gtk version (Linux) ────────────────────────────────
detect_webkit_tag() {
  [[ "$TARGET_OS" != "linux" ]] && return
  # Check if webkit2gtk-4.1 is available (Ubuntu 22.04+, Fedora 37+)
  if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
    WEBKIT_TAG="webkit2_41"
    log "Detected webkit2gtk-4.1 → using build tag 'webkit2_41'"
  elif pkg-config --exists webkit2gtk-4.0 2>/dev/null; then
    WEBKIT_TAG=""
    log "Detected webkit2gtk-4.0"
  else
    die "No webkit2gtk found. Install libwebkit2gtk-4.1-dev (Ubuntu 22.04+) or libwebkit2gtk-4.0-dev (Ubuntu 20.04)"
  fi
}

WEBKIT_TAG=""
detect_webkit_tag

# ── Prerequisite checks ─────────────────────────────────────────────
check_prereqs() {
  log "Checking prerequisites..."
  command -v go >/dev/null 2>&1 || die "Go not installed. https://go.dev/dl/"
  log "  Go $(go version | grep -oE '[0-9]+\.[0-9]+' | head -1)"

  command -v node >/dev/null 2>&1 || die "Node.js not installed. https://nodejs.org/"
  log "  Node $(node --version)"

  command -v npm >/dev/null 2>&1 || die "npm not installed"
  log "  npm $(npm --version)"

  # Ensure wails is in PATH
  if ! command -v wails >/dev/null 2>&1; then
    if [[ -x "$HOME/go/bin/wails" ]]; then
      export PATH="$HOME/go/bin:$PATH"
    else
      warn "Wails CLI not found. Installing..."
      go install github.com/wailsapp/wails/v2/cmd/wails@latest
      export PATH="$HOME/go/bin:$PATH"
    fi
  fi
  log "  Wails $(wails version 2>/dev/null | head -1 || echo 'installed')"

  if [[ "$TARGET_OS" == "linux" ]]; then
    local missing=()
    if ! pkg-config --exists gtk+-3.0 2>/dev/null; then
      missing+=("libgtk-3-dev")
    fi
    if ! pkg-config --exists webkit2gtk-4.1 2>/dev/null && ! pkg-config --exists webkit2gtk-4.0 2>/dev/null; then
      missing+=("libwebkit2gtk-4.1-dev or libwebkit2gtk-4.0-dev")
    fi
    if [[ ${#missing[@]} -gt 0 ]]; then
      die "Missing Linux packages: ${missing[*]}. Install with your package manager."
    fi
  fi

  ok "Prerequisites satisfied"
}

do_clean() {
  # Always remove legacy-named binaries from rebrand (office-ai → quill)
  rm -f "${BUILD_DIR}/office-ai" "${BUILD_DIR}/office-ai-"* 2>/dev/null || true

  if $CLEAN; then
    log "Cleaning build artifacts..."
    rm -rf "$BUILD_DIR"
    rm -rf "${FRONTEND_DIR}/dist"
    ok "Cleaned"
  fi
}

# ── Build with Wails ─────────────────────────────────────────────────
build_wails() {
  log "Building with Wails..."

  # Ensure PATH includes ~/go/bin for wails
  export PATH="$HOME/go/bin:$PATH"

  local wails_flags="-clean"
  if $DEV; then
    wails_flags="${wails_flags} -debug"
  fi

  # Add webkit2gtk-4.1 tag if needed
  local tag_flags=""
  if [[ -n "$WEBKIT_TAG" ]]; then
    tag_flags="-tags ${WEBKIT_TAG}"
  fi

  wails build ${wails_flags} ${tag_flags} -platform "${TARGET_PLATFORM}"

  # Verify binary exists
  local binary="${BUILD_DIR}/${APP_NAME}"
  if [[ "$TARGET_OS" == "darwin" ]]; then
    binary="${BUILD_DIR}/${APP_NAME}.app/Contents/MacOS/${APP_NAME}"
    # Also check the .app bundle
    if [[ -d "${BUILD_DIR}/${APP_NAME}.app" ]]; then
      ok "macOS app bundle → ${BUILD_DIR}/${APP_NAME}.app"
    fi
  fi

  if [[ ! -f "$binary" ]]; then
    # Wails might put it directly in build/bin with the outputfilename from wails.json
    binary="$(find "${BUILD_DIR}" -type f -executable -name "${APP_NAME}*" 2>/dev/null | head -1 || true)"
    if [[ -z "$binary" || ! -f "$binary" ]]; then
      die "Build succeeded but binary not found in ${BUILD_DIR}/. Check wails.json 'outputfilename'."
    fi
  fi

  local size
  size="$(du -sh "$binary" | cut -f1)"
  ok "Binary: ${binary} (${size})"
}

# ── Fallback: manual build (without wails CLI) ──────────────────────
build_manual() {
  warn "Wails CLI not available, falling back to manual build..."

  # Frontend
  log "Building frontend..."
  cd "$FRONTEND_DIR"
  [[ -d node_modules ]] || npm ci --silent 2>/dev/null || npm install --silent
  npm run build
  cd ..
  ok "Frontend built"

  # Backend — must use -tags desktop,production to embed frontend
  log "Building Go backend..."
  mkdir -p "$BUILD_DIR"

  local ldflags="-s -w -X main.Version=${VERSION}"
  local tags="desktop,production"
  [[ -n "$WEBKIT_TAG" ]] && tags="${tags},${WEBKIT_TAG}"

  if $DEV; then
    ldflags="-X main.Version=${VERSION}-dev"
    tags="desktop"
    [[ -n "$WEBKIT_TAG" ]] && tags="${tags},${WEBKIT_TAG}"
  fi

  local output="${BUILD_DIR}/${APP_NAME}"
  CGO_ENABLED=1 GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" \
    go build -tags "$tags" -ldflags "$ldflags" -o "$output" .

  if [[ ! -f "$output" ]]; then
    die "Build failed — no binary at ${output}"
  fi

  local size
  size="$(du -sh "$output" | cut -f1)"
  ok "Backend built → ${output} (${size})"
}

# ── macOS app bundle (manual build only) ─────────────────────────────
create_macos_bundle() {
  [[ "$TARGET_OS" != "darwin" ]] && return
  local binary="${BUILD_DIR}/${APP_NAME}"
  [[ -f "$binary" ]] || return

  # Only create bundle if wails didn't already make one
  [[ -d "${BUILD_DIR}/${APP_NAME}.app" ]] && return

  log "Creating macOS app bundle..."
  local app_dir="${BUILD_DIR}/${APP_NAME}.app"
  local contents="${app_dir}/Contents"
  mkdir -p "${contents}/MacOS" "${contents}/Resources"

  cp "$binary" "${contents}/MacOS/${APP_NAME}"

  # Info.plist
  if [[ -f build/darwin/Info.plist ]]; then
    cp build/darwin/Info.plist "${contents}/Info.plist"
  else
    cat > "${contents}/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>Quill</string>
  <key>CFBundleDisplayName</key><string>Quill</string>
  <key>CFBundleIdentifier</key><string>com.officeai.app</string>
  <key>CFBundleVersion</key><string>${VERSION}</string>
  <key>CFBundleShortVersionString</key><string>${VERSION}</string>
  <key>CFBundleExecutable</key><string>${APP_NAME}</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
PLIST
  fi

  [[ -f build/appicon.png ]] && cp build/appicon.png "${contents}/Resources/iconfile.png"
  rm -f "$binary"  # remove loose binary, keep .app

  ok "macOS bundle → ${app_dir}"
}

# ── Linux .desktop entry ─────────────────────────────────────────────
create_linux_desktop_entry() {
  [[ "$TARGET_OS" != "linux" ]] && return

  cat > "${BUILD_DIR}/${APP_NAME}.desktop" <<DESKTOP
[Desktop Entry]
Name=Quill
Comment=AI-powered office suite
Exec=${APP_NAME}
Icon=quill
Type=Application
Categories=Office;WordProcessor;Spreadsheet;Presentation;
Terminal=false
StartupWMClass=Quill
DESKTOP
  ok "Desktop entry → ${BUILD_DIR}/${APP_NAME}.desktop"
}

# ── Package ──────────────────────────────────────────────────────────
package_build() {
  log "Packaging..."
  local archive_name="${APP_NAME}-${VERSION}-${TARGET_OS}-${TARGET_ARCH}"
  cd "$BUILD_DIR"
  case "$TARGET_OS" in
    darwin)
      if [[ -d "${APP_NAME}.app" ]]; then
        tar czf "${archive_name}.tar.gz" "${APP_NAME}.app"
      fi
      ;;
    linux)
      tar czf "${archive_name}.tar.gz" "${APP_NAME}" "${APP_NAME}.desktop" 2>/dev/null || \
        tar czf "${archive_name}.tar.gz" "${APP_NAME}"
      ;;
  esac
  cd - >/dev/null
  ok "Package → ${BUILD_DIR}/${archive_name}.tar.gz"
}

# ── Main ─────────────────────────────────────────────────────────────
main() {
  check_prereqs
  do_clean

  if command -v wails >/dev/null 2>&1 || [[ -x "$HOME/go/bin/wails" ]]; then
    export PATH="$HOME/go/bin:$PATH"
    build_wails
  else
    build_manual
    create_macos_bundle
    create_linux_desktop_entry
  fi

  package_build

  echo ""
  log "═══════════════════════════════════════════"
  ok "Build complete! 🎉"
  log "  Version:  ${VERSION}"
  log "  Platform: ${TARGET_PLATFORM}"
  log "  Output:   ${BUILD_DIR}/"
  log "═══════════════════════════════════════════"
}

main "$@"

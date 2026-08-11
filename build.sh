#!/usr/bin/env bash
# build.sh — Cross-platform build script for Office AI (Go + Wails)
# Supports: macOS (Intel/Apple Silicon), Linux (amd64/arm64)
# Usage: ./build.sh [--platform <os/arch>] [--clean] [--dev]
set -euo pipefail

VERSION="${VERSION:-0.1.0}"
APP_NAME="office-ai"
BUILD_DIR="build/bin"
FRONTEND_DIR="frontend"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log()  { printf "${BLUE}[build]${NC} %s\n" "$*"; }
ok()   { printf "${GREEN}[  ok ]${NC} %s\n" "$*"; }
warn() { printf "${YELLOW}[warn]${NC} %s\n" "$*"; }
die()  { printf "${RED}[FAIL]${NC} %s\n" "$*" >&2; exit 1; }

# Defaults
CLEAN=false
DEV=false
TARGET_PLATFORM=""
NSIS=false

usage() {
  cat <<EOF
Usage: $0 [OPTIONS]

Options:
  --platform <os/arch>   Target platform (e.g. darwin/arm64, linux/amd64)
                         Default: current platform
  --clean                Remove build artifacts before building
  --dev                  Build in dev mode (no optimization, includes debug info)
  --nsis                 Build NSIS installer (Windows only, cross-compile)
  --version <ver>        Set version string (default: $VERSION)
  -h, --help             Show this help

Supported platforms:
  darwin/amd64    macOS Intel
  darwin/arm64    macOS Apple Silicon
  linux/amd64     Linux x86_64
  linux/arm64     Linux ARM64

Examples:
  ./build.sh                           # Build for current platform
  ./build.sh --platform darwin/arm64   # Cross-compile for macOS ARM
  ./build.sh --clean --dev             # Clean build in dev mode
EOF
  exit 0
}

# Parse args
while [[ $# -gt 0 ]]; do
  case "$1" in
    --platform)  TARGET_PLATFORM="$2"; shift 2 ;;
    --clean)     CLEAN=true; shift ;;
    --dev)       DEV=true; shift ;;
    --nsis)      NSIS=true; shift ;;
    --version)   VERSION="$2"; shift 2 ;;
    -h|--help)   usage ;;
    *)           die "Unknown option: $1" ;;
  esac
done

# Detect current platform
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
if [[ -z "$TARGET_PLATFORM" ]]; then
  TARGET_PLATFORM="$CURRENT_PLATFORM"
fi

TARGET_OS="${TARGET_PLATFORM%/*}"
TARGET_ARCH="${TARGET_PLATFORM#*/}"

log "Office AI Build System v${VERSION}"
log "Target: ${TARGET_PLATFORM}  (host: ${CURRENT_PLATFORM})"

# ── Prerequisite checks ──────────────────────────────────────────────
check_prereqs() {
  log "Checking prerequisites..."

  command -v go >/dev/null 2>&1 || die "Go is not installed. Install from https://go.dev/dl/"
  local go_ver
  go_ver="$(go version | grep -oE '[0-9]+\.[0-9]+' | head -1)"
  log "  Go ${go_ver}"

  command -v node >/dev/null 2>&1 || die "Node.js is not installed. Install from https://nodejs.org/"
  local node_ver
  node_ver="$(node --version)"
  log "  Node ${node_ver}"

  command -v npm >/dev/null 2>&1 || die "npm is not installed"
  log "  npm $(npm --version)"

  command -v wails >/dev/null 2>&1 || {
    warn "Wails CLI not found. Installing..."
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
  }
  log "  Wails $(wails version 2>/dev/null | head -1 || echo 'installed')"

  # Platform-specific deps
  if [[ "$TARGET_OS" == "linux" ]]; then
    local missing=()
    for pkg in libgtk-3-dev libwebkit2gtk-4.0-dev; do
      dpkg -s "$pkg" >/dev/null 2>&1 || missing+=("$pkg")
    done
    if [[ ${#missing[@]} -gt 0 ]]; then
      warn "Missing Linux packages: ${missing[*]}"
      warn "Install with: sudo apt-get install ${missing[*]}"
    fi
  fi

  ok "Prerequisites satisfied"
}

# ── Clean ─────────────────────────────────────────────────────────────
do_clean() {
  if $CLEAN; then
    log "Cleaning build artifacts..."
    rm -rf "$BUILD_DIR"
    rm -rf "${FRONTEND_DIR}/dist"
    ok "Cleaned"
  fi
}

# ── Frontend build ────────────────────────────────────────────────────
build_frontend() {
  log "Building frontend..."
  cd "$FRONTEND_DIR"

  if [[ ! -d node_modules ]]; then
    log "  Installing npm dependencies..."
    npm ci --silent 2>/dev/null || npm install --silent
  fi

  if $DEV; then
    npm run build -- --mode development
  else
    npm run build
  fi

  cd ..
  ok "Frontend built → ${FRONTEND_DIR}/dist"
}

# ── Go backend build ─────────────────────────────────────────────────
build_backend() {
  log "Building Go backend (${TARGET_OS}/${TARGET_ARCH})..."

  mkdir -p "$BUILD_DIR"

  local ldflags="-s -w -X main.Version=${VERSION}"
  if $DEV; then
    ldflags="-X main.Version=${VERSION}-dev"
  fi

  local output="${BUILD_DIR}/${APP_NAME}"
  if [[ "$TARGET_OS" == "darwin" ]]; then
    output="${BUILD_DIR}/${APP_NAME}.app/Contents/MacOS/${APP_NAME}"
  fi

  local build_tags=""
  if [[ "$TARGET_OS" == "darwin" ]]; then
    build_tags="desktop,production"
  else
    build_tags="desktop,production"
  fi

  if $DEV; then
    build_tags="desktop"
  fi

  CGO_ENABLED=1 GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" \
    go build \
      -tags "$build_tags" \
      -ldflags "$ldflags" \
      -o "$output" \
      .

  ok "Backend built → ${output}"
}

# ── Wails build (preferred path) ─────────────────────────────────────
build_wails() {
  log "Building with Wails..."

  local wails_flags="-clean"

  if $DEV; then
    wails_flags="${wails_flags} -debug"
  fi

  case "$TARGET_PLATFORM" in
    darwin/*)
      wails build ${wails_flags} -platform "${TARGET_PLATFORM}"
      ;;
    linux/*)
      wails build ${wails_flags} -platform "${TARGET_PLATFORM}"
      ;;
    *)
      die "Unsupported platform for Wails build: ${TARGET_PLATFORM}"
      ;;
  esac

  ok "Wails build complete → ${BUILD_DIR}/"
}

# ── macOS app bundle ─────────────────────────────────────────────────
create_macos_bundle() {
  if [[ "$TARGET_OS" != "darwin" ]]; then return; fi

  local app_bundle="${BUILD_DIR}/${APP_NAME}.app"
  local contents="${app_bundle}/Contents"
  local macos="${contents}/MacOS"
  local resources="${contents}/Resources"

  log "Creating macOS app bundle..."

  mkdir -p "$macos" "$resources"

  # Info.plist
  cat > "${contents}/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key>
  <string>Office AI</string>
  <key>CFBundleDisplayName</key>
  <string>Office AI</string>
  <key>CFBundleIdentifier</key>
  <string>com.officeai.app</string>
  <key>CFBundleVersion</key>
  <string>${VERSION}</string>
  <key>CFBundleShortVersionString</key>
  <string>${VERSION}</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleExecutable</key>
  <string>${APP_NAME}</string>
  <key>CFBundleIconFile</key>
  <string>icon.icns</string>
  <key>NSHighResolutionCapable</key>
  <true/>
  <key>LSMinimumSystemVersion</key>
  <string>11.0</string>
  <key>CFBundleDocumentTypes</key>
  <array>
    <dict>
      <key>CFBundleTypeName</key>
      <string>Word Document</string>
      <key>CFBundleTypeExtensions</key>
      <array><string>docx</string><string>doc</string></array>
      <key>CFBundleTypeRole</key>
      <string>Editor</string>
    </dict>
    <dict>
      <key>CFBundleTypeName</key>
      <string>Excel Spreadsheet</string>
      <key>CFBundleTypeExtensions</key>
      <array><string>xlsx</string><string>xls</string><string>csv</string></array>
      <key>CFBundleTypeRole</key>
      <string>Editor</string>
    </dict>
    <dict>
      <key>CFBundleTypeName</key>
      <string>PowerPoint Presentation</string>
      <key>CFBundleTypeExtensions</key>
      <array><string>pptx</string><string>ppt</string></array>
      <key>CFBundleTypeRole</key>
      <string>Editor</string>
    </dict>
    <dict>
      <key>CFBundleTypeName</key>
      <string>PDF Document</string>
      <key>CFBundleTypeExtensions</key>
      <array><string>pdf</string></array>
      <key>CFBundleTypeRole</key>
      <string>Viewer</string>
    </dict>
    <dict>
      <key>CFBundleTypeName</key>
      <string>Markdown Document</string>
      <key>CFBundleTypeExtensions</key>
      <array><string>md</string><string>markdown</string></array>
      <key>CFBundleTypeRole</key>
      <string>Editor</string>
    </dict>
  </array>
</dict>
</plist>
PLIST

  ok "macOS app bundle created → ${app_bundle}"
}

# ── Linux desktop entry ──────────────────────────────────────────────
create_linux_desktop_entry() {
  if [[ "$TARGET_OS" != "linux" ]]; then return; fi

  local desktop_file="${BUILD_DIR}/${APP_NAME}.desktop"

  cat > "$desktop_file" <<DESKTOP
[Desktop Entry]
Name=Office AI
Comment=AI-powered office suite
Exec=/usr/local/bin/${APP_NAME} %F
Icon=${APP_NAME}
Type=Application
Categories=Office;WordProcessor;Spreadsheet;Presentation;
MimeType=application/vnd.openxmlformats-officedocument.wordprocessingml.document;application/vnd.openxmlformats-officedocument.spreadsheetml.sheet;application/vnd.openxmlformats-officedocument.presentationml.presentation;application/pdf;text/markdown;
Terminal=false
StartupNotify=true
DESKTOP

  ok "Linux .desktop file created → ${desktop_file}"
}

# ── Package ──────────────────────────────────────────────────────────
package_build() {
  log "Packaging..."

  local archive_name="${APP_NAME}-${VERSION}-${TARGET_OS}-${TARGET_ARCH}"

  case "$TARGET_OS" in
    darwin)
      # Create DMG-like tar.gz with the app bundle
      cd "$BUILD_DIR"
      tar czf "${archive_name}.tar.gz" "${APP_NAME}.app" 2>/dev/null || \
        tar czf "${archive_name}.tar.gz" "${APP_NAME}"
      cd ..
      ok "Package → ${BUILD_DIR}/${archive_name}.tar.gz"
      ;;
    linux)
      cd "$BUILD_DIR"
      tar czf "${archive_name}.tar.gz" "${APP_NAME}" "${APP_NAME}.desktop" 2>/dev/null || \
        tar czf "${archive_name}.tar.gz" "${APP_NAME}"
      cd ..
      ok "Package → ${BUILD_DIR}/${archive_name}.tar.gz"
      ;;
  esac
}

# ── Main ─────────────────────────────────────────────────────────────
main() {
  check_prereqs
  do_clean

  # Try Wails build first (it handles frontend + backend + packaging)
  if command -v wails >/dev/null 2>&1 && ! $DEV; then
    build_wails
  else
    # Fallback: manual build
    build_frontend
    build_backend
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

#!/usr/bin/env bash
# install.sh — Single-line curl|bash installer for Office AI
# Usage: curl -fsSL https://raw.githubusercontent.com/ankurCES/office-ai/main/install.sh | bash
# Or:    ./install.sh [--prefix /usr/local] [--version 0.1.0] [--from-source]
set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────
APP_NAME="office-ai"
DISPLAY_NAME="Office AI"
GITHUB_REPO="ankurCES/office-ai"
GITHUB_URL="https://github.com/${GITHUB_REPO}"
DEFAULT_VERSION="latest"
DEFAULT_PREFIX="/usr/local"

# ── Colors ───────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

log()  { printf "${BLUE}▸${NC} %s\n" "$*"; }
ok()   { printf "${GREEN}✓${NC} %s\n" "$*"; }
warn() { printf "${YELLOW}⚠${NC} %s\n" "$*"; }
err()  { printf "${RED}✗${NC} %s\n" "$*" >&2; }
die()  { err "$*"; exit 1; }

banner() {
  printf "\n"
  printf "${BOLD}${BLUE}"
  cat <<'BANNER'
   ____  __  __ _              _    ___
  / __ \/ _|/ _(_)            / \  |_ _|
 | |  | | |_| |_ _  ___ ___ / _ \  | |
 | |  | |  _|  _| |/ __/ _ / ___ \ | |
 | |__| | | | | | | (_|  __/ /   \ \| |
  \____/|_| |_| |_|\___\___/_/   \_\___|

BANNER
  printf "${NC}\n"
  printf "  ${BOLD}${DISPLAY_NAME} Installer${NC}\n"
  printf "  AI-powered office suite (Go + Wails)\n\n"
}

# ── Parse arguments ──────────────────────────────────────────────────
PREFIX="$DEFAULT_PREFIX"
VERSION="$DEFAULT_VERSION"
FROM_SOURCE=false
UNINSTALL=false
SKIP_DEPS=false

usage() {
  cat <<EOF
Usage: $0 [OPTIONS]

Install ${DISPLAY_NAME} on macOS or Linux.

Options:
  --prefix <path>    Installation prefix (default: ${DEFAULT_PREFIX})
  --version <ver>    Version to install (default: latest)
  --from-source      Build from source instead of downloading binary
  --uninstall        Remove ${DISPLAY_NAME}
  --skip-deps        Skip dependency installation
  -h, --help         Show this help

One-liner install:
  curl -fsSL https://raw.githubusercontent.com/${GITHUB_REPO}/main/install.sh | bash

One-liner with options:
  curl -fsSL https://raw.githubusercontent.com/${GITHUB_REPO}/main/install.sh | bash -s -- --prefix ~/.local

EOF
  exit 0
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --prefix)      PREFIX="$2"; shift 2 ;;
    --version)     VERSION="$2"; shift 2 ;;
    --from-source) FROM_SOURCE=true; shift ;;
    --uninstall)   UNINSTALL=true; shift ;;
    --skip-deps)   SKIP_DEPS=true; shift ;;
    -h|--help)     usage ;;
    *)             die "Unknown option: $1. Use --help for usage." ;;
  esac
done

# ── Platform detection ───────────────────────────────────────────────
detect_platform() {
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m)"

  case "$OS" in
    darwin) OS="darwin" ;;
    linux)  OS="linux" ;;
    msys*|mingw*|cygwin*) OS="windows" ;;
    *)      die "Unsupported operating system: $OS" ;;
  esac

  case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)             die "Unsupported architecture: $ARCH" ;;
  esac

  PLATFORM="${OS}/${ARCH}"
}

# ── Dependency checks ───────────────────────────────────────────────
has()     { command -v "$1" >/dev/null 2>&1; }
need()    { has "$1" || die "$1 is required but not found. $2"; }

check_deps_common() {
  need "git" "Install: https://git-scm.com/downloads"
  need "curl" "Install: your package manager (apt/brew/dnf)"
}

install_go() {
  if has go; then
    local ver
    ver="$(go version | grep -oE '[0-9]+\.[0-9]+' | head -1)"
    log "Go ${ver} found"
    return
  fi

  log "Installing Go..."
  local go_ver="1.24.4"
  local go_os="$OS"
  local go_arch="$ARCH"
  local go_url="https://go.dev/dl/go${go_ver}.${go_os}-${go_arch}.tar.gz"

  local tmp
  tmp="$(mktemp -d)"
  curl -fsSL "$go_url" -o "${tmp}/go.tar.gz"

  if [[ -w /usr/local ]]; then
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "${tmp}/go.tar.gz"
  else
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "${tmp}/go.tar.gz"
  fi
  rm -rf "$tmp"

  export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
  ok "Go $(go version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+') installed"
}

install_node() {
  if has node; then
    log "Node $(node --version) found"
    return
  fi

  log "Installing Node.js..."
  if has brew; then
    brew install node
  elif has apt-get; then
    curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
    sudo apt-get install -y nodejs
  elif has dnf; then
    curl -fsSL https://rpm.nodesource.com/setup_22.x | sudo bash -
    sudo dnf install -y nodejs
  elif has pacman; then
    sudo pacman -S --noconfirm nodejs npm
  else
    die "Cannot auto-install Node.js. Install from https://nodejs.org/"
  fi
  ok "Node $(node --version) installed"
}

install_wails() {
  if has wails; then
    log "Wails found"
    return
  fi

  log "Installing Wails CLI..."
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  export PATH="$HOME/go/bin:$PATH"
  ok "Wails installed"
}

install_linux_deps() {
  [[ "$OS" != "linux" ]] && return

  log "Checking Linux build dependencies..."

  local pm=""
  local pkgs=()

  if has apt-get; then
    pm="apt-get"
    # Check each package
    for pkg in libgtk-3-dev libwebkit2gtk-4.0-dev build-essential pkg-config; do
      dpkg -s "$pkg" >/dev/null 2>&1 || pkgs+=("$pkg")
    done
    # Fallback: try 4.1 if 4.0 not available
    if [[ " ${pkgs[*]} " =~ "libwebkit2gtk-4.0-dev" ]]; then
      if apt-cache show libwebkit2gtk-4.1-dev >/dev/null 2>&1; then
        pkgs=("${pkgs[@]/libwebkit2gtk-4.0-dev/libwebkit2gtk-4.1-dev}")
      fi
    fi
  elif has dnf; then
    pm="dnf"
    for pkg in gtk3-devel webkit2gtk4.0-devel gcc gcc-c++ pkgconf-pkg-config; do
      rpm -q "$pkg" >/dev/null 2>&1 || pkgs+=("$pkg")
    done
  elif has pacman; then
    pm="pacman"
    for pkg in gtk3 webkit2gtk base-devel pkgconf; do
      pacman -Q "$pkg" >/dev/null 2>&1 || pkgs+=("$pkg")
    done
  fi

  if [[ ${#pkgs[@]} -eq 0 ]]; then
    ok "All Linux dependencies present"
    return
  fi

  log "Installing: ${pkgs[*]}"
  case "$pm" in
    apt-get) sudo apt-get update -qq && sudo apt-get install -y "${pkgs[@]}" ;;
    dnf)     sudo dnf install -y "${pkgs[@]}" ;;
    pacman)  sudo pacman -S --noconfirm "${pkgs[@]}" ;;
    *)       warn "Unknown package manager. Install manually: ${pkgs[*]}" ;;
  esac
  ok "Linux dependencies installed"
}

install_macos_deps() {
  [[ "$OS" != "darwin" ]] && return

  if ! has xcode-select; then
    log "Installing Xcode command line tools..."
    xcode-select --install 2>/dev/null || true
  fi
  ok "macOS build tools present"
}

install_all_deps() {
  if $SKIP_DEPS; then
    log "Skipping dependency installation (--skip-deps)"
    return
  fi

  check_deps_common
  install_go
  install_node
  install_linux_deps
  install_macos_deps
  install_wails
}

# ── Uninstall ────────────────────────────────────────────────────────
do_uninstall() {
  banner
  log "Uninstalling ${DISPLAY_NAME}..."

  local bin_path="${PREFIX}/bin/${APP_NAME}"

  if [[ -f "$bin_path" ]]; then
    if [[ -w "$bin_path" ]]; then
      rm -f "$bin_path"
    else
      sudo rm -f "$bin_path"
    fi
    ok "Removed ${bin_path}"
  else
    warn "Binary not found at ${bin_path}"
  fi

  # macOS: remove app bundle
  if [[ "$OS" == "darwin" ]]; then
    local app_path="/Applications/${DISPLAY_NAME}.app"
    if [[ -d "$app_path" ]]; then
      rm -rf "$app_path"
      ok "Removed ${app_path}"
    fi
  fi

  # Linux: remove desktop entry
  if [[ "$OS" == "linux" ]]; then
    local desktop_path="$HOME/.local/share/applications/${APP_NAME}.desktop"
    if [[ -f "$desktop_path" ]]; then
      rm -f "$desktop_path"
      ok "Removed ${desktop_path}"
    fi
  fi

  # Config (ask first)
  local config_dir="${XDG_CONFIG_HOME:-$HOME/.config}/${APP_NAME}"
  if [[ -d "$config_dir" ]]; then
    printf "Remove config directory ${config_dir}? [y/N] "
    read -r answer
    if [[ "$answer" =~ ^[Yy] ]]; then
      rm -rf "$config_dir"
      ok "Removed ${config_dir}"
    fi
  fi

  ok "${DISPLAY_NAME} uninstalled"
  exit 0
}

# ── Build from source ────────────────────────────────────────────────
build_from_source() {
  local src_dir
  src_dir="$(mktemp -d)"
  local clone_dir="${src_dir}/${APP_NAME}"

  log "Cloning ${GITHUB_URL}..."
  if [[ "$VERSION" == "latest" ]]; then
    git clone --depth 1 "${GITHUB_URL}.git" "$clone_dir"
  else
    git clone --depth 1 --branch "v${VERSION}" "${GITHUB_URL}.git" "$clone_dir" 2>/dev/null || \
      git clone --depth 1 --branch "${VERSION}" "${GITHUB_URL}.git" "$clone_dir"
  fi

  cd "$clone_dir"

  log "Installing frontend dependencies..."
  cd frontend
  npm ci --silent 2>/dev/null || npm install --silent
  cd ..

  log "Building with Wails..."
  if has wails; then
    wails build -clean 2>&1 || {
      warn "Wails build failed, trying manual build..."
      manual_build "$clone_dir"
    }
  else
    manual_build "$clone_dir"
  fi

  # Find the built binary
  local built_bin=""
  if [[ -f "build/bin/${APP_NAME}" ]]; then
    built_bin="build/bin/${APP_NAME}"
  elif [[ -f "build/bin/${APP_NAME}.app/Contents/MacOS/${APP_NAME}" ]]; then
    built_bin="build/bin/${APP_NAME}.app/Contents/MacOS/${APP_NAME}"
  else
    # Search for it
    built_bin="$(find build/bin -type f -name "${APP_NAME}" 2>/dev/null | head -1 || true)"
  fi

  if [[ -z "$built_bin" || ! -f "$built_bin" ]]; then
    die "Build succeeded but binary not found in build/bin/"
  fi

  echo "$built_bin"
}

manual_build() {
  local dir="$1"
  cd "$dir"

  log "Building frontend..."
  cd frontend && npm run build && cd ..

  log "Building Go backend..."
  mkdir -p build/bin
  CGO_ENABLED=1 go build \
    -tags "desktop,production" \
    -ldflags "-s -w -X main.Version=${VERSION}" \
    -o "build/bin/${APP_NAME}" \
    .
}

# ── Download pre-built binary ────────────────────────────────────────
download_binary() {
  local download_url
  local archive_name="${APP_NAME}-${VERSION}-${OS}-${ARCH}.tar.gz"

  if [[ "$VERSION" == "latest" ]]; then
    download_url="${GITHUB_URL}/releases/latest/download/${archive_name}"
  else
    download_url="${GITHUB_URL}/releases/download/v${VERSION}/${archive_name}"
  fi

  local tmp
  tmp="$(mktemp -d)"

  log "Downloading ${archive_name}..."
  if curl -fsSL "$download_url" -o "${tmp}/${archive_name}" 2>/dev/null; then
    cd "$tmp"
    tar xzf "$archive_name"
    local bin_path
    bin_path="$(find . -type f -name "${APP_NAME}" ! -name "*.tar.gz" | head -1)"
    if [[ -n "$bin_path" ]]; then
      echo "$bin_path"
      return 0
    fi
  fi

  # No pre-built binary available, fall back to source
  warn "No pre-built binary for ${OS}/${ARCH}. Building from source..."
  FROM_SOURCE=true
  return 1
}

# ── Install binary ───────────────────────────────────────────────────
install_binary() {
  local src_bin="$1"
  local dest_dir="${PREFIX}/bin"
  local dest_bin="${dest_dir}/${APP_NAME}"

  log "Installing to ${dest_bin}..."

  mkdir -p "$dest_dir" 2>/dev/null || sudo mkdir -p "$dest_dir"

  if [[ -w "$dest_dir" ]]; then
    cp "$src_bin" "$dest_bin"
    chmod +x "$dest_bin"
  else
    sudo cp "$src_bin" "$dest_bin"
    sudo chmod +x "$dest_bin"
  fi

  ok "Binary installed → ${dest_bin}"
}

# ── macOS integration ────────────────────────────────────────────────
setup_macos() {
  [[ "$OS" != "darwin" ]] && return

  local app_dir="/Applications/${DISPLAY_NAME}.app"

  # If Wails built a .app bundle, copy it to /Applications
  local built_app=""
  built_app="$(find /tmp -maxdepth 3 -name "${APP_NAME}.app" -type d 2>/dev/null | head -1 || true)"

  if [[ -d "$built_app" ]]; then
    log "Installing app bundle to /Applications..."
    rm -rf "$app_dir"
    cp -R "$built_app" "$app_dir"
    ok "App bundle → ${app_dir}"
  else
    # Create a minimal app bundle pointing to the installed binary
    log "Creating macOS app bundle..."
    mkdir -p "${app_dir}/Contents/MacOS"
    mkdir -p "${app_dir}/Contents/Resources"

    # Symlink to the installed binary
    ln -sf "${PREFIX}/bin/${APP_NAME}" "${app_dir}/Contents/MacOS/${APP_NAME}"

    cat > "${app_dir}/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>${DISPLAY_NAME}</string>
  <key>CFBundleDisplayName</key><string>${DISPLAY_NAME}</string>
  <key>CFBundleIdentifier</key><string>com.officeai.app</string>
  <key>CFBundleVersion</key><string>${VERSION}</string>
  <key>CFBundleShortVersionString</key><string>${VERSION}</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleExecutable</key><string>${APP_NAME}</string>
  <key>NSHighResolutionCapable</key><true/>
  <key>CFBundleDocumentTypes</key>
  <array>
    <dict>
      <key>CFBundleTypeExtensions</key>
      <array><string>docx</string><string>xlsx</string><string>pptx</string><string>pdf</string><string>md</string></array>
      <key>CFBundleTypeRole</key><string>Editor</string>
    </dict>
  </array>
</dict>
</plist>
PLIST
    ok "App bundle → ${app_dir}"
  fi
}

# ── Linux integration ────────────────────────────────────────────────
setup_linux() {
  [[ "$OS" != "linux" ]] && return

  local desktop_dir="${HOME}/.local/share/applications"
  mkdir -p "$desktop_dir"

  local desktop_file="${desktop_dir}/${APP_NAME}.desktop"

  cat > "$desktop_file" <<DESKTOP
[Desktop Entry]
Name=${DISPLAY_NAME}
Comment=AI-powered office suite
Exec=${PREFIX}/bin/${APP_NAME} %F
Icon=${APP_NAME}
Type=Application
Categories=Office;WordProcessor;Spreadsheet;Presentation;
MimeType=application/vnd.openxmlformats-officedocument.wordprocessingml.document;application/vnd.openxmlformats-officedocument.spreadsheetml.sheet;application/vnd.openxmlformats-officedocument.presentationml.presentation;application/pdf;text/markdown;text/csv;
Terminal=false
StartupNotify=true
DESKTOP

  # Update desktop database if available
  if has update-desktop-database; then
    update-desktop-database "$desktop_dir" 2>/dev/null || true
  fi

  ok "Desktop entry → ${desktop_file}"

  # MIME type associations
  if has xdg-mime; then
    for mime in \
      application/vnd.openxmlformats-officedocument.wordprocessingml.document \
      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet \
      application/vnd.openxmlformats-officedocument.presentationml.presentation; do
      xdg-mime default "${APP_NAME}.desktop" "$mime" 2>/dev/null || true
    done
    ok "MIME associations set"
  fi
}

# ── Create config directory ──────────────────────────────────────────
setup_config() {
  local config_dir

  if [[ "$OS" == "darwin" ]]; then
    config_dir="$HOME/Library/Application Support/${DISPLAY_NAME}"
  else
    config_dir="${XDG_CONFIG_HOME:-$HOME/.config}/${APP_NAME}"
  fi

  mkdir -p "$config_dir"

  # Default config if none exists
  if [[ ! -f "${config_dir}/settings.json" ]]; then
    cat > "${config_dir}/settings.json" <<JSON
{
  "language": "en",
  "theme": "system",
  "autosave": true,
  "autosave_delay_ms": 2000,
  "ai_provider": "genspark",
  "recent_files_limit": 50,
  "font_size": 14,
  "show_line_numbers": true,
  "word_wrap": true
}
JSON
    ok "Default config → ${config_dir}/settings.json"
  fi
}

# ── Verify installation ─────────────────────────────────────────────
verify_install() {
  local bin_path="${PREFIX}/bin/${APP_NAME}"

  if [[ ! -x "$bin_path" ]]; then
    die "Installation verification failed: ${bin_path} not found or not executable"
  fi

  # Check it's on PATH
  if ! has "$APP_NAME"; then
    warn "${APP_NAME} is installed but not on your PATH"
    warn "Add this to your shell profile:"
    warn "  export PATH=\"${PREFIX}/bin:\$PATH\""
  fi

  ok "Installation verified"
}

# ── Main ─────────────────────────────────────────────────────────────
main() {
  banner
  detect_platform

  if $UNINSTALL; then
    do_uninstall
  fi

  log "Platform: ${BOLD}${PLATFORM}${NC}"
  log "Prefix:   ${BOLD}${PREFIX}${NC}"
  log "Version:  ${BOLD}${VERSION}${NC}"
  log "Method:   ${BOLD}$(if $FROM_SOURCE; then echo "source"; else echo "binary (fallback: source)"; fi)${NC}"
  echo ""

  # Step 1: Dependencies
  install_all_deps

  # Step 2: Get the binary
  local binary_path=""

  if $FROM_SOURCE; then
    binary_path="$(build_from_source)"
  else
    binary_path="$(download_binary)" || binary_path="$(build_from_source)"
  fi

  # Step 3: Install
  install_binary "$binary_path"

  # Step 4: Platform integration
  setup_macos
  setup_linux
  setup_config

  # Step 5: Verify
  verify_install

  # Done!
  echo ""
  printf "${GREEN}${BOLD}"
  cat <<'DONE'
  ┌─────────────────────────────────────────────┐
  │       Office AI installed successfully!      │
  └─────────────────────────────────────────────┘
DONE
  printf "${NC}\n"

  log "Run with:  ${BOLD}${APP_NAME}${NC}"
  if [[ "$OS" == "darwin" ]]; then
    log "Or open:   ${BOLD}/Applications/${DISPLAY_NAME}.app${NC}"
  elif [[ "$OS" == "linux" ]]; then
    log "Or find in your application launcher"
  fi
  echo ""
  log "Uninstall: ${BOLD}curl -fsSL https://raw.githubusercontent.com/${GITHUB_REPO}/main/install.sh | bash -s -- --uninstall${NC}"
  echo ""
}

main "$@"

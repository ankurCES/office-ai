#!/usr/bin/env bash
# install.sh — Single-line curl|bash installer for Quill
# Usage: curl -fsSL https://raw.githubusercontent.com/ankurCES/office-ai/main/install.sh | bash
# Or:    ./install.sh [--prefix /usr/local] [--version 0.1.0] [--from-source]
set -euo pipefail

APP_NAME="quill"
DISPLAY_NAME="Quill"
GITHUB_REPO="ankurCES/office-ai"
GITHUB_URL="https://github.com/${GITHUB_REPO}"
DEFAULT_VERSION="latest"
DEFAULT_PREFIX="/usr/local"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; BOLD='\033[1m'; NC='\033[0m'

log()  { printf "${BLUE}▸${NC} %s\n" "$*" >&2; }
ok()   { printf "${GREEN}✓${NC} %s\n" "$*" >&2; }
warn() { printf "${YELLOW}⚠${NC} %s\n" "$*" >&2; }
err()  { printf "${RED}✗${NC} %s\n" "$*" >&2; }
die()  { err "$*"; exit 1; }

banner() {
  printf "\n${BOLD}${BLUE}"
  cat <<'BANNER'
   ___        _ _ _
  / _ \ _   _(_) | |
 | | | | | | | | | |
 | |_| | |_| | | | |
  \__\_\\__,_|_|_|_|
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

Examples:
  curl -fsSL https://raw.githubusercontent.com/${GITHUB_REPO}/main/install.sh | bash
  curl -fsSL ... | bash -s -- --prefix ~/.local
  curl -fsSL ... | bash -s -- --from-source
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
    *)      die "Unsupported OS: $OS" ;;
  esac
  case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)             die "Unsupported architecture: $ARCH" ;;
  esac
  PLATFORM="${OS}/${ARCH}"
}

# ── Dependency helpers ───────────────────────────────────────────────
has() { command -v "$1" >/dev/null 2>&1; }

install_go() {
  if has go; then
    log "Go $(go version | grep -oE '[0-9]+\.[0-9]+' | head -1) found"
    return
  fi
  log "Installing Go..."
  local go_ver="1.24.4"
  local go_url="https://go.dev/dl/go${go_ver}.${OS}-${ARCH}.tar.gz"
  local tmp; tmp="$(mktemp -d)"
  curl -fsSL "$go_url" -o "${tmp}/go.tar.gz"
  if [[ -w /usr/local ]]; then
    rm -rf /usr/local/go && tar -C /usr/local -xzf "${tmp}/go.tar.gz"
  else
    sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf "${tmp}/go.tar.gz"
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
  # Check PATH and ~/go/bin
  if has wails; then
    log "Wails found"
    return
  fi
  if [[ -x "$HOME/go/bin/wails" ]]; then
    export PATH="$HOME/go/bin:$PATH"
    log "Wails found in ~/go/bin"
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

  if has apt-get; then
    local pkgs=()
    for pkg in libgtk-3-dev build-essential pkg-config; do
      dpkg -s "$pkg" >/dev/null 2>&1 || pkgs+=("$pkg")
    done
    # webkit2gtk: prefer 4.1 (newer distros), fall back to 4.0
    if ! dpkg -s libwebkit2gtk-4.1-dev >/dev/null 2>&1 && ! dpkg -s libwebkit2gtk-4.0-dev >/dev/null 2>&1; then
      if apt-cache show libwebkit2gtk-4.1-dev >/dev/null 2>&1; then
        pkgs+=("libwebkit2gtk-4.1-dev")
      else
        pkgs+=("libwebkit2gtk-4.0-dev")
      fi
    fi
    if [[ ${#pkgs[@]} -gt 0 ]]; then
      log "Installing: ${pkgs[*]}"
      sudo apt-get update -qq && sudo apt-get install -y "${pkgs[@]}"
    fi
  elif has dnf; then
    for pkg in gtk3-devel webkit2gtk4.0-devel gcc gcc-c++ pkgconf-pkg-config; do
      rpm -q "$pkg" >/dev/null 2>&1 || sudo dnf install -y "$pkg"
    done
  elif has pacman; then
    for pkg in gtk3 webkit2gtk base-devel pkgconf; do
      pacman -Q "$pkg" >/dev/null 2>&1 || sudo pacman -S --noconfirm "$pkg"
    done
  fi
  ok "Linux dependencies satisfied"
}

# ── Uninstall ────────────────────────────────────────────────────────
do_uninstall() {
  banner
  log "Uninstalling ${DISPLAY_NAME}..."

  # Binary (current name + legacy office-ai name)
  for name in "${APP_NAME}" "office-ai"; do
    for p in "${PREFIX}/bin/${name}" "/usr/local/bin/${name}" "$HOME/.local/bin/${name}"; do
      if [[ -f "$p" ]]; then
        rm -f "$p" 2>/dev/null || sudo rm -f "$p"
        ok "Removed $p"
      fi
    done
  done

  # macOS app bundle
  if [[ -d "/Applications/${DISPLAY_NAME}.app" ]]; then
    rm -rf "/Applications/${DISPLAY_NAME}.app"
    ok "Removed /Applications/${DISPLAY_NAME}.app"
  fi

  # Linux desktop entry
  for d in "$HOME/.local/share/applications" "/usr/share/applications"; do
    if [[ -f "${d}/${APP_NAME}.desktop" ]]; then
      rm -f "${d}/${APP_NAME}.desktop" 2>/dev/null || sudo rm -f "${d}/${APP_NAME}.desktop"
      ok "Removed desktop entry"
    fi
  done

  # Config (ask first)
  for cfg_dir in "$HOME/.${APP_NAME}" "$HOME/.office-ai"; do
    if [[ -d "$cfg_dir" ]]; then
      warn "Config directory exists: ${cfg_dir}"
      warn "Remove manually with: rm -rf ${cfg_dir}"
    fi
  done

  ok "Uninstall complete"
  exit 0
}

# ── Download pre-built binary ────────────────────────────────────────
download_binary() {
  log "Checking for pre-built binary..."

  local tag_url
  if [[ "$VERSION" == "latest" ]]; then
    tag_url="${GITHUB_URL}/releases/latest/download"
  else
    tag_url="${GITHUB_URL}/releases/download/v${VERSION}"
  fi

  local archive="${APP_NAME}-${OS}-${ARCH}.tar.gz"
  local url="${tag_url}/${archive}"
  local tmp; tmp="$(mktemp -d)"

  log "Downloading ${url}..." >&2
  if curl -fsSL --connect-timeout 10 "$url" -o "${tmp}/${archive}" 2>/dev/null; then
    tar xzf "${tmp}/${archive}" -C "$tmp"
    local binary
    binary="$(find "$tmp" -type f -executable -name "${APP_NAME}*" ! -name "*.tar.gz" | head -1)"
    if [[ -z "$binary" ]]; then
      binary="$(find "$tmp" -type f -name "${APP_NAME}" | head -1)"
    fi
    if [[ -n "$binary" && -f "$binary" ]]; then
      echo "$binary"
      return 0
    fi
  fi

  warn "No pre-built binary available for ${OS}/${ARCH}" >&2
  rm -rf "$tmp"
  return 1
}

# ── Build from source ────────────────────────────────────────────────
build_from_source() {
  log "Building from source..." >&2

  # Install deps
  if ! $SKIP_DEPS; then
    install_go
    install_node
    install_linux_deps
    install_wails
  fi

  # Clone or use existing
  local src_dir
  if [[ -f "wails.json" && -f "go.mod" ]]; then
    src_dir="$(pwd)"
    log "Using current directory as source" >&2
  else
    src_dir="$(mktemp -d)"
    log "Cloning ${GITHUB_URL}..." >&2
    git clone --depth 1 "$GITHUB_URL" "$src_dir" >&2
  fi

  cd "$src_dir"

  # Ensure Go + Wails are in PATH (subshell doesn't inherit parent exports)
  export PATH="/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH"

  # Clean stale binaries from previous builds/rebrands
  rm -f build/bin/office-ai build/bin/office-ai-* 2>/dev/null || true
  rm -f build/bin/quill 2>/dev/null || true
  log "Cleaned stale binaries from build/bin/" >&2

  # Detect webkit2gtk version for Linux
  local webkit_tag=""
  if [[ "$OS" == "linux" ]]; then
    if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
      webkit_tag="webkit2_41"
      log "Detected webkit2gtk-4.1" >&2
    elif pkg-config --exists webkit2gtk-4.0 2>/dev/null; then
      log "Detected webkit2gtk-4.0" >&2
    else
      die "No webkit2gtk found. Install libwebkit2gtk-4.1-dev or libwebkit2gtk-4.0-dev"
    fi
  fi

  # Build
  local tag_flags=""
  [[ -n "$webkit_tag" ]] && tag_flags="-tags ${webkit_tag}"
  log "Running: wails build ${tag_flags}" >&2
  wails build ${tag_flags} >&2

  # Find the binary — it goes to build/bin/<outputfilename> per wails.json
  local binary=""
  local build_bin="build/bin"

  # 1. Check exact expected path from wails.json outputfilename
  if [[ -f "${build_bin}/${APP_NAME}" ]]; then
    binary="${build_bin}/${APP_NAME}"
  fi

  # 2. Check macOS .app bundle
  if [[ -z "$binary" && -d "${build_bin}/${APP_NAME}.app" ]]; then
    binary="${build_bin}/${APP_NAME}.app/Contents/MacOS/${APP_NAME}"
  fi

  # 3. Search build/bin for any executable
  if [[ -z "$binary" || ! -f "$binary" ]]; then
    binary="$(find "${build_bin}" -type f -executable 2>/dev/null | head -1 || true)"
  fi

  # 4. Search broader build directory
  if [[ -z "$binary" || ! -f "$binary" ]]; then
    binary="$(find build -type f -executable -name "${APP_NAME}*" 2>/dev/null | head -1 || true)"
  fi

  if [[ -z "$binary" || ! -f "$binary" ]]; then
    echo "Build directory contents:" >&2
    find build -type f 2>/dev/null | head -20 >&2
    die "Build succeeded but binary not found. Check build/bin/ directory."
  fi

  ok "Built: ${binary} ($(du -sh "$binary" | cut -f1))" >&2
  echo "$binary"
}

# ── Install the binary ───────────────────────────────────────────────
install_binary() {
  local binary="$1"
  [[ -f "$binary" ]] || die "Binary not found: ${binary}"

  local bin_dir="${PREFIX}/bin"
  local dest="${bin_dir}/${APP_NAME}"

  log "Installing to ${dest}..."

  # Remove legacy office-ai binary if it exists (rebrand migration)
  local legacy="${bin_dir}/office-ai"
  if [[ -f "$legacy" ]]; then
    log "Removing legacy office-ai binary at ${legacy}..." >&2
    rm -f "$legacy" 2>/dev/null || sudo rm -f "$legacy"
    ok "Removed legacy binary: ${legacy}"
  fi

  # Remove legacy config dir migration note
  if [[ -d "$HOME/.office-ai" && ! -d "$HOME/.quill" ]]; then
    log "Migrating config from ~/.office-ai to ~/.quill..." >&2
    cp -r "$HOME/.office-ai" "$HOME/.quill" 2>/dev/null || true
    ok "Config migrated to ~/.quill"
  fi

  if [[ -w "$bin_dir" ]] || mkdir -p "$bin_dir" 2>/dev/null; then
    cp "$binary" "$dest"
    chmod +x "$dest"
  else
    sudo mkdir -p "$bin_dir"
    sudo cp "$binary" "$dest"
    sudo chmod +x "$dest"
  fi

  ok "Installed: ${dest}"
}

# ── Platform integration ─────────────────────────────────────────────
setup_macos() {
  [[ "$OS" != "darwin" ]] && return

  # Copy .app bundle to /Applications if it exists
  local app_bundle
  app_bundle="$(find build -name "${APP_NAME}.app" -type d 2>/dev/null | head -1)"
  if [[ -n "$app_bundle" && -d "$app_bundle" ]]; then
    log "Installing app bundle to /Applications..."
    if [[ -w /Applications ]]; then
      rm -rf "/Applications/${DISPLAY_NAME}.app"
      cp -R "$app_bundle" "/Applications/${DISPLAY_NAME}.app"
    else
      sudo rm -rf "/Applications/${DISPLAY_NAME}.app"
      sudo cp -R "$app_bundle" "/Applications/${DISPLAY_NAME}.app"
    fi
    ok "App bundle → /Applications/${DISPLAY_NAME}.app"
  fi
}

setup_linux() {
  [[ "$OS" != "linux" ]] && return

  local desktop_dir="$HOME/.local/share/applications"
  mkdir -p "$desktop_dir"

  cat > "${desktop_dir}/${APP_NAME}.desktop" <<DESKTOP
[Desktop Entry]
Name=${DISPLAY_NAME}
Comment=AI-powered office suite
Exec=${PREFIX}/bin/${APP_NAME}
Icon=${APP_NAME}
Type=Application
Categories=Office;WordProcessor;Spreadsheet;Presentation;
Terminal=false
StartupWMClass=${DISPLAY_NAME}
DESKTOP

  # Copy icon if available
  local icon_dir="$HOME/.local/share/icons/hicolor/256x256/apps"
  if [[ -f build/appicon.png ]]; then
    mkdir -p "$icon_dir"
    cp build/appicon.png "${icon_dir}/${APP_NAME}.png"
  fi

  # Update desktop database
  if has update-desktop-database; then
    update-desktop-database "$desktop_dir" 2>/dev/null || true
  fi

  ok "Desktop entry installed"
}

setup_config() {
  local config_dir="$HOME/.${APP_NAME}"
  mkdir -p "$config_dir"

  if [[ ! -f "${config_dir}/config.json" ]]; then
    cat > "${config_dir}/config.json" <<JSON
{
  "theme": "system",
  "language": "en",
  "fontSize": 14,
  "autoSaveDelay": 2000,
  "provider": "anthropic"
}
JSON
    ok "Default config → ${config_dir}/config.json"
  fi
}

verify_install() {
  local installed="${PREFIX}/bin/${APP_NAME}"
  if [[ -x "$installed" ]]; then
    ok "Verified: ${installed} is executable"
  else
    warn "Binary installed but may not be executable"
    chmod +x "$installed" 2>/dev/null || sudo chmod +x "$installed"
  fi

  # Check PATH
  if ! echo "$PATH" | tr ':' '\n' | grep -q "^${PREFIX}/bin$"; then
    warn "${PREFIX}/bin is not in PATH"
    warn "Add to your shell profile:  export PATH=\"${PREFIX}/bin:\$PATH\""
  fi
}

# ── Main ─────────────────────────────────────────────────────────────
main() {
  banner
  detect_platform

  if $UNINSTALL; then
    do_uninstall
  fi

  log "Platform: ${OS}/${ARCH}"
  log "Prefix:   ${PREFIX}"
  log "Version:  ${VERSION}"
  echo ""

  # Get the binary
  local binary_path=""
  if $FROM_SOURCE; then
    binary_path="$(build_from_source)"
  else
    binary_path="$(download_binary)" || binary_path="$(build_from_source)"
  fi

  install_binary "$binary_path"
  setup_macos
  setup_linux
  setup_config
  verify_install

  echo ""
  printf "${GREEN}${BOLD}"
  cat <<'DONE'
  ┌─────────────────────────────────────────────┐
  │       Quill installed successfully!      │
  └─────────────────────────────────────────────┘
DONE
  printf "${NC}\n"

  log "Run:       ${BOLD}${APP_NAME}${NC}"
  if [[ "$OS" == "darwin" ]]; then
    log "Or open:   ${BOLD}/Applications/${DISPLAY_NAME}.app${NC}"
  elif [[ "$OS" == "linux" ]]; then
    log "Or find:   in your application launcher"
  fi
  echo ""
  log "Uninstall: ${BOLD}curl -fsSL https://raw.githubusercontent.com/${GITHUB_REPO}/main/install.sh | bash -s -- --uninstall${NC}"
  echo ""
}

main "$@"

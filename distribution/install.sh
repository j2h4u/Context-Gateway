#!/bin/sh
# Context-Gateway Install Script
# Usage: curl -fsSL https://compresr.ai/install_gateway.sh | sh
#
# Downloads and installs the context-gateway binary.
# All configs and agent definitions are embedded in the binary.
# Run `context-gateway init` to export configs for customization.

set -e

# Configuration
REPO="Compresr-ai/Context-Gateway"
BINARY_NAME="context-gateway"
INSTALL_DIR="${HOME}/.local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[38;2;23;128;68m'  # #178044
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# ASCII Banner
print_banner() {
    printf "${GREEN}${BOLD}"
    cat << 'EOF'

  ██████╗ ██████╗ ███╗  ██╗████████╗███████╗██╗ ██╗████████╗  ██████╗  █████╗ ████████╗███████╗██╗    ██╗ █████╗ ██╗   ██╗
 ██╔════╝██╔═══██╗████╗ ██║╚══██╔══╝██╔════╝╚██╗██╔╝╚══██╔══╝ ██╔════╝ ██╔══██╗╚══██╔══╝██╔════╝██║    ██║██╔══██╗╚██╗ ██╔╝
 ██║     ██║   ██║██╔██╗██║   ██║   █████╗   ╚███╔╝    ██║    ██║  ███╗███████║   ██║   █████╗  ██║ █╗ ██║███████║ ╚████╔╝
 ██║     ██║   ██║██║╚████║   ██║   ██╔══╝   ██╔██╗    ██║    ██║   ██║██╔══██║   ██║   ██╔══╝  ██║███╗██║██╔══██║  ╚██╔╝
 ╚██████╗╚██████╔╝██║ ╚███║   ██║   ███████╗██╔╝ ██╗   ██║    ╚██████╔╝██║  ██║   ██║   ███████╗╚███╔███╔╝██║  ██║   ██║
  ╚═════╝ ╚═════╝ ╚═╝  ╚══╝   ╚═╝   ╚══════╝╚═╝  ╚═╝   ╚═╝     ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚══════╝ ╚══╝╚══╝ ╚═╝  ╚═╝   ╚═╝

EOF
    printf "${NC}\n\n"
}

info() {
    printf "${GREEN}[INFO]${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1"
    exit 1
}

success() {
    printf "${GREEN}${BOLD}[✓]${NC} %s\n" "$1"
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     OS="linux";;
        Darwin*)    OS="darwin";;
        CYGWIN*|MINGW*|MSYS*) OS="windows";;
        *)          error "Unsupported OS: $(uname -s)";;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   ARCH="amd64";;
        arm64|aarch64)  ARCH="arm64";;
        armv7l)         ARCH="arm";;
        i386|i686)      ARCH="386";;
        *)              error "Unsupported architecture: $(uname -m)";;
    esac
}

# Track installation (silent background request for analytics)
track_install() {
    local os="$1"
    local arch="$2"
    local version="${3:-latest}"

    # Make tracking request in background (don't block on failure)
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "https://api.compresr.ai/install_gateway.sh?os=${os}&arch=${arch}&v=${version}" >/dev/null 2>&1 &
    fi
}

# Get latest version from GitHub
get_latest_version() {
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        error "Failed to get latest version"
    fi
}

# Download and install binary
install_binary() {
    detect_os
    detect_arch

    # Use provided VERSION or get latest
    if [ -z "$VERSION" ]; then
        get_latest_version
    fi

    # Track installation (silent background analytics)
    track_install "${OS}" "${ARCH}" "${VERSION}"

    info "Installing Context-Gateway ${VERSION} for ${OS}/${ARCH}..."

    # Create install directory
    mkdir -p "${INSTALL_DIR}"

    # Construct download URL for raw binary (not tarball)
    FILENAME="gateway-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        FILENAME="${FILENAME}.exe"
    fi
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

    # Download binary directly
    info "Downloading binary from ${URL}..."

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "${URL}" -o "${INSTALL_DIR}/${BINARY_NAME}"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "${URL}" -O "${INSTALL_DIR}/${BINARY_NAME}"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    # Make executable
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

    # Create compresr alias
    if [ "$OS" != "windows" ]; then
        ln -sf "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/compresr"
    fi

    success "Binary installed to ${INSTALL_DIR}/${BINARY_NAME}"
    if [ "$OS" != "windows" ]; then
        success "Alias 'compresr' created"
    fi
}

# Check PATH
check_path() {
    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*) ;;
        *)
            warn "${INSTALL_DIR} is not in your PATH"
            echo ""
            echo "  Add this to your shell profile (~/.bashrc, ~/.zshrc):"
            printf "  ${CYAN}export PATH=\"\$PATH:${INSTALL_DIR}\"${NC}\n"
            echo ""
            ;;
    esac
}

# Print usage instructions
print_usage() {
    printf "\n"
    printf "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
    printf "${GREEN}${BOLD}  ✅ INSTALLATION COMPLETE!${NC}\n"
    printf "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
    printf "\n"
    printf "  ${BOLD}Quick Start:${NC}\n"
    printf "\n"
    printf "  ${YELLOW}1.${NC} Launch the interactive setup wizard:\n"
    printf "     ${CYAN}context-gateway agent${NC}\n"
    printf "\n"
    printf "     The wizard will help you:\n"
    printf "     • Choose an agent (claude_code, cursor, openclaw, custom)\n"
    printf "     • Configure your API key and summarizer model\n"
    printf "     • Set compression threshold (default: 75%%)\n"
    printf "     • Enable optional Slack notifications\n"
    printf "\n"
    printf "  ${YELLOW}2.${NC} Or launch an agent directly (skips wizard):\n"
    printf "     ${CYAN}context-gateway agent claude_code${NC}\n"
    printf "     ${CYAN}context-gateway agent cursor${NC}\n"
    printf "     ${CYAN}context-gateway agent openclaw${NC}\n"
    printf "\n"
    printf "  ${YELLOW}3.${NC} Export configs for manual customization:\n"
    printf "     ${CYAN}context-gateway init${NC}\n"
    printf "\n"
    printf "  ${BOLD}Supported Providers:${NC}\n"
    printf "     Anthropic, OpenAI, AWS Bedrock, Google Gemini, Ollama (local)\n"
    printf "\n"
    printf "  ${BOLD}Documentation:${NC} ${CYAN}https://docs.compresr.ai/gateway${NC}\n"
    printf "  ${BOLD}Discord:${NC}       ${CYAN}https://discord.gg/PeaVWNjT${NC}\n"
    printf "\n"
    printf "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

# Main
main() {
    print_banner
    install_binary
    check_path
    print_usage
}

main

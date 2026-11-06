#!/bin/bash
# install.sh - GroundWork TUI one-line installer
# Usage: curl -fsSL https://raw.githubusercontent.com/Kanishkp2906/groundwork-cli/main/install.sh | bash
# Or: wget -qO- https://raw.githubusercontent.com/Kanishkp2906/groundwork-cli/main/install.sh | bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

REPO="Kanishkp2906/groundwork-cli"
BINARY_NAME="groundwork"
INSTALL_DIR="${HOME}/.local/bin"
GITHUB_API="https://api.github.com/repos/Kanishkp2906/groundwork-cli"

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "${OS}" in
        linux)
            PLATFORM_OS="linux"
            ;;
        darwin)
            PLATFORM_OS="darwin"
            ;;
        *)
            echo -e "${RED}Unsupported OS: ${OS}${NC}"
            exit 1
            ;;
    esac

    case "${ARCH}" in
        x86_64|amd64)
            PLATFORM_ARCH="amd64"
            ;;
        arm64|aarch64)
            PLATFORM_ARCH="arm64"
            ;;
        *)
            echo -e "${RED}Unsupported architecture: ${ARCH}${NC}"
            exit 1
            ;;
    esac

    PLATFORM="${PLATFORM_OS}_${PLATFORM_ARCH}"
}

# Get latest release version
get_latest_version() {
    echo -e "${BLUE}Fetching latest release...${NC}" >&2
    VERSION=$(curl -fsSL "https://api.github.com/repos/Kanishkp2906/groundwork-cli/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "${VERSION}" ]; then
        echo -e "${RED}Failed to get latest version${NC}" >&2
        exit 1
    fi
    echo "${VERSION}"
}

# Download and install
install_binary() {
    local version="${1}"
    local url="https://github.com/Kanishkp2906/groundwork-cli/releases/download/${version}/groundwork_${version#v}_${PLATFORM}.tar.gz"

    if [[ "${OSTYPE}" == "msys" || "${OSTYPE}" == "cygwin" || "${OSTYPE}" == "win32" ]]; then
        url="https://github.com/Kanishkp2906/groundwork-cli/releases/download/${version}/groundwork_${version#v}_${PLATFORM}.zip"
    fi

    echo -e "${BLUE}Downloading groundwork ${version} for ${PLATFORM}...${NC}" >&2

    # Create temp directory
    TMPDIR=$(mktemp -d)
    trap 'rm -rf "${TMPDIR}"' EXIT

    # Download
    if ! curl -fsSL -L "${url}" -o "${TMPDIR}/groundwork_archive"; then
        echo -e "${RED}Failed to download${NC}" >&2
        exit 1
    fi

    # Extract
    if [[ "${url}" == *.zip ]]; then
        unzip -q "${TMPDIR}/groundwork_archive" -d "${TMPDIR}"
    else
        tar -xzf "${TMPDIR}/groundwork_archive" -C "${TMPDIR}"
    fi

    # Find binary
    BINARY_PATH=$(find "${TMPDIR}" -name "groundwork" -type f | head -n1)
    if [ -z "${BINARY_PATH}" ]; then
        echo -e "${RED}Binary not found in archive${NC}" >&2
        exit 1
    fi

    # Make executable
    chmod +x "${BINARY_PATH}"

    # Create install directory
    mkdir -p "${INSTALL_DIR}"

    # Install
    mv "${BINARY_PATH}" "${INSTALL_DIR}/${BINARY_NAME}"
    echo -e "${GREEN}Installed to ${INSTALL_DIR}/${BINARY_NAME}${NC}" >&2
}

# Verify installation
verify_install() {
    if command -v groundwork >/dev/null 2>&1; then
        echo -e "${GREEN}Installation verified!${NC}"
        groundwork --version
    else
        echo -e "${YELLOW}Warning: groundwork not in PATH. Add ${INSTALL_DIR} to your PATH.${NC}"
        echo "Add this to your shell config (~/.bashrc, ~/.zshrc, etc.):"
        echo "  export PATH=\"\${HOME}/.local/bin:\${PATH}\""
    fi
}

# Main
main() {
    echo -e "${BLUE}╔══════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║     GroundWork TUI Installer         ║${NC}"
    echo -e "${BLUE}║   Real CVE Training for Developers   ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════╝${NC}"
    echo ""

    detect_platform
    echo -e "Detected platform: ${GREEN}${PLATFORM}${NC}"

    VERSION=$(get_latest_version)
    echo -e "Latest version: ${GREEN}${VERSION}${NC}"

    install_binary "${VERSION}"
    verify_install

    echo ""
    echo -e "${GREEN}Installation complete!${NC}"
    echo "Run 'groundwork' to start. Set GROUNDWORK_SERVER_URL if needed."
}

main "$@"
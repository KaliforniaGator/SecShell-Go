#!/bin/bash

# Detect operating system
OS="$(uname -s)"
ARCH="$(uname -m)"
case "${OS}" in
    Linux*)     OS_TYPE=linux; OS_NAME="Linux";;
    Darwin*)    OS_TYPE=darwin; OS_NAME="Darwin";;
    *)          echo "Unsupported operating system: ${OS}"; exit 1;;
esac

# Map architecture to release naming
case "${ARCH}" in
    x86_64|amd64)  ARCH_NAME="x86_64";;
    arm64|aarch64) ARCH_NAME="arm64";;
    *)             echo "Unsupported architecture: ${ARCH}"; exit 1;;
esac

# Function to handle errors
handle_error() {
    echo "Error: $1"
    exit 1
}

# Get the latest version from GitHub
echo "Fetching latest version..."
LATEST_VERSION=$(curl -fsSL "https://api.github.com/repos/KaliforniaGator/SecShell-Go/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')
if [ -z "$LATEST_VERSION" ]; then
    # Fallback to VERSION file if API fails
    LATEST_VERSION=$(curl -fsSL "https://raw.githubusercontent.com/KaliforniaGator/SecShell-Go/refs/heads/main/VERSION")
fi
if [ -z "$LATEST_VERSION" ]; then
    handle_error "Failed to determine the latest version"
fi

# Construct download URL
# Archive naming: SecShell-Go_{OS}_{Arch}.tar.gz
DOWNLOAD_URL="https://github.com/KaliforniaGator/SecShell-Go/releases/download/v${LATEST_VERSION}/SecShell-Go_${OS_NAME}_${ARCH_NAME}.tar.gz"

echo "Installing SecShell-Go v${LATEST_VERSION} for ${OS_NAME} ${ARCH_NAME}..."

# Set binary path based on OS
if [ "$OS_TYPE" = "linux" ]; then
    BIN_PATH="/usr/bin/secshell"
    DRAWBOX_BIN_PATH="/usr/bin/drawbox"
elif [ "$OS_TYPE" = "darwin" ]; then
    BIN_PATH="/usr/local/bin/secshell"
    DRAWBOX_BIN_PATH="/usr/local/bin/drawbox"
fi

# Create temporary directory
TMP_DIR=$(mktemp -d)
trap "rm -rf ${TMP_DIR}" EXIT

# Download the release archive
echo "Downloading from: ${DOWNLOAD_URL}"
ARCHIVE_FILE="${TMP_DIR}/secshell-release.tar.gz"
curl -fsSL -o "${ARCHIVE_FILE}" "${DOWNLOAD_URL}" || handle_error "Failed to download release archive"

# Extract the archive
echo "Extracting binary..."
tar -xzf "${ARCHIVE_FILE}" -C "${TMP_DIR}" || handle_error "Failed to extract archive"

# Find the extracted binary
if [ -f "${TMP_DIR}/secshell" ]; then
    EXTRACTED_BIN="${TMP_DIR}/secshell"
elif [ -f "${TMP_DIR}/SecShell-Go" ]; then
    EXTRACTED_BIN="${TMP_DIR}/SecShell-Go"
else
    handle_error "Binary not found in archive"
fi

# Make the binary executable
chmod +x "${EXTRACTED_BIN}" || handle_error "Failed to make binary executable"

# Move to the appropriate location
echo "Installing to ${BIN_PATH}..."
sudo mv "${EXTRACTED_BIN}" "${BIN_PATH}" || handle_error "Failed to install binary. Make sure you have sudo privileges."

# Remove old .secshell directory if it exists
if [ -d ~/.secshell ]; then
    echo "Removing old .secshell directory..."
    rm -rf ~/.secshell
fi

# Create .secshell directory
echo "Creating .secshell directory..."
mkdir -p ~/.secshell &> /dev/null || handle_error "Failed to create ~/.secshell directory."

# Create version file
echo "${LATEST_VERSION}" > ~/.secshell/.ver || handle_error "Failed to create version file"

# Install DrawBox dependency - using direct download
echo "Installing DrawBox dependency..."
DRAWBOX_TMP_FILE="${TMP_DIR}/drawbox-download"

if [ "$OS_TYPE" = "linux" ]; then
    DRAWBOX_URL="https://github.com/KaliforniaGator/DrawBox/releases/download/linux-latest/drawbox-linux"
    curl -fsSL -o "${DRAWBOX_TMP_FILE}" "${DRAWBOX_URL}" || echo "Warning: Failed to download DrawBox binary"
elif [ "$OS_TYPE" = "darwin" ]; then
    DRAWBOX_URL="https://github.com/KaliforniaGator/DrawBox/releases/download/mac-latest/drawbox-mac"
    curl -fsSL -o "${DRAWBOX_TMP_FILE}" "${DRAWBOX_URL}" || echo "Warning: Failed to download DrawBox binary"
fi

if [ -f "${DRAWBOX_TMP_FILE}" ]; then
    # Make DrawBox executable
    chmod +x "${DRAWBOX_TMP_FILE}" || echo "Warning: Failed to make DrawBox binary executable"

    # Move DrawBox to the appropriate location
    sudo mv "${DRAWBOX_TMP_FILE}" "${DRAWBOX_BIN_PATH}" || echo "Warning: Failed to install DrawBox binary"
fi

echo ""
echo "============================================"
echo "SecShell-Go v${LATEST_VERSION} has been successfully installed!"
echo "You can now run 'secshell' to start the shell."
echo "============================================"
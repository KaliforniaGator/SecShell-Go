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

# Initialize progress counter and total steps
CURRENT_STEP=0
TOTAL_STEPS=5

# Function to update progress
update_progress() {
    CURRENT_STEP=$((CURRENT_STEP + 1))
    echo -ne "\033[0K\r"  # Clear the current line and reset cursor to start of line
    case $CURRENT_STEP in
        1) STEP_DESC="Getting latest version" ;;
        2) STEP_DESC="Downloading SecShell-Go release" ;;
        3) STEP_DESC="Extracting and installing binary" ;;
        4) STEP_DESC="Updating DrawBox" ;;
        5) STEP_DESC="Updating version information" ;;
        *) STEP_DESC="Processing" ;;
    esac
    echo -ne "$CURRENT_STEP/$TOTAL_STEPS: $STEP_DESC"
    if command -v drawbox &> /dev/null; then
        drawbox progress $CURRENT_STEP $TOTAL_STEPS 50 "█" "░" green
    else
        echo -ne " [$CURRENT_STEP/$TOTAL_STEPS]"
    fi
    echo -ne "\033[0K\r"
}

# Get the latest version
echo "Checking for updates..."
LATEST_VERSION=$(curl -fsSL "https://raw.githubusercontent.com/KaliforniaGator/SecShell-Go/refs/heads/main/VERSION" | tr -d '[:space:]')
if [ -z "$LATEST_VERSION" ]; then
    handle_error "Failed to determine the latest version"
fi

# Check current version
CURRENT_VERSION=""
if [ -f ~/.secshell/.ver ]; then
    CURRENT_VERSION=$(cat ~/.secshell/.ver | tr -d '[:space:]')
fi

if [ "$CURRENT_VERSION" = "$LATEST_VERSION" ]; then
    echo ""
    echo "You're already up to date! (v${CURRENT_VERSION})"
    exit 0
fi

if [ -n "$CURRENT_VERSION" ]; then
    echo "Current version: v${CURRENT_VERSION}"
fi
echo "Latest version: v${LATEST_VERSION}"
echo ""

# Construct download URL
# Archive naming: SecShell-Go_{OS}_{Arch}.tar.gz
DOWNLOAD_URL="https://github.com/KaliforniaGator/SecShell-Go/releases/download/v${LATEST_VERSION}/SecShell-Go_${OS_NAME}_${ARCH_NAME}.tar.gz"

echo "Updating SecShell-Go for ${OS_NAME} ${ARCH_NAME}..."

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
update_progress
echo ""
echo "Downloading from: ${DOWNLOAD_URL}"
ARCHIVE_FILE="${TMP_DIR}/secshell-release.tar.gz"
curl -fsSL -o "${ARCHIVE_FILE}" "${DOWNLOAD_URL}" || handle_error "Failed to download release archive"

# Extract and install the binary
update_progress
echo ""
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

# Download and install DrawBox
update_progress
echo ""
echo "Updating DrawBox..."
DRAWBOX_TMP_FILE="${TMP_DIR}/drawbox-download"

if [ "$OS_TYPE" = "linux" ]; then
    DRAWBOX_URL="https://github.com/KaliforniaGator/DrawBox/releases/download/linux-latest/drawbox-linux"
    curl -fsSL -o "${DRAWBOX_TMP_FILE}" "${DRAWBOX_URL}" || echo "Warning: Failed to download DrawBox binary"
elif [ "$OS_TYPE" = "darwin" ]; then
    DRAWBOX_URL="https://github.com/KaliforniaGator/DrawBox/releases/download/mac-latest/drawbox-mac"
    curl -fsSL -o "${DRAWBOX_TMP_FILE}" "${DRAWBOX_URL}" || echo "Warning: Failed to download DrawBox binary"
fi

if [ -f "${DRAWBOX_TMP_FILE}" ]; then
    chmod +x "${DRAWBOX_TMP_FILE}" || echo "Warning: Failed to make DrawBox binary executable"
    sudo mv "${DRAWBOX_TMP_FILE}" "${DRAWBOX_BIN_PATH}" || echo "Warning: Failed to install DrawBox binary"
    echo "DrawBox updated successfully."
fi

# Update version file
update_progress
echo ""

# Create .secshell directory if it doesn't exist
mkdir -p ~/.secshell &> /dev/null || handle_error "Failed to create ~/.secshell directory."

# Create version file
echo "${LATEST_VERSION}" > ~/.secshell/.ver || handle_error "Failed to create version file"

echo ""
echo "============================================"
echo "Update complete! SecShell-Go updated to v${LATEST_VERSION}"
echo "Restart SecShell to use the new version."
echo "============================================"
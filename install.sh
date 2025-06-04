#!/bin/bash

# Detect operating system
OS="$(uname -s)"
case "${OS}" in
    Linux*)     OS_TYPE=linux;;
    Darwin*)    OS_TYPE=mac;;
    *)          echo "Unsupported operating system: ${OS}"; exit 1;;
esac

# Function to handle errors
handle_error() {
    echo "Error: $1"
    exit 1
}

echo "Installing SecShell for ${OS_TYPE}..."

# Set download URL and binary path based on OS
if [ "$OS_TYPE" = "linux" ]; then
    DOWNLOAD_URL="https://github.com/KaliforniaGator/SecShell-Go/releases/download/linux-latest/secshell-linux-latest"
    BIN_PATH="/usr/bin/secshell"
    DRAWBOX_BIN_PATH="/usr/bin/drawbox"
elif [ "$OS_TYPE" = "mac" ]; then
    DOWNLOAD_URL="https://github.com/KaliforniaGator/SecShell-Go/releases/download/mac-latest/secshell-mac-latest"
    BIN_PATH="/usr/local/bin/secshell"
    DRAWBOX_BIN_PATH="/usr/local/bin/drawbox"
fi

# Create temporary directory
TMP_DIR=$(mktemp -d)
TMP_FILE="${TMP_DIR}/secshell-download"

# Download the binary
echo "Downloading SecShell binary..."
curl -L -o "${TMP_FILE}" "${DOWNLOAD_URL}" || handle_error "Failed to download binary"

# Make the binary executable
chmod +x "${TMP_FILE}" || handle_error "Failed to make binary executable"

# Move to the appropriate location
echo "Installing to ${BIN_PATH}..."
sudo mv "${TMP_FILE}" "${BIN_PATH}" || handle_error "Failed to install binary. Make sure you have sudo privileges."

# Clean up
rm -rf "${TMP_DIR}"

# Create .secshell directory if it doesn't exist
mkdir -p ~/.secshell &> /dev/null || handle_error "Failed to create ~/.secshell directory."

# Get the version from the binary
CURRENT_VERSION=$(${BIN_PATH} --version 2>&1 | grep -o '[0-9]\+\.[0-9]\+\.[0-9]\+')
if [ -z "$CURRENT_VERSION" ]; then
    CURRENT_VERSION="latest"
fi

# Create version file
echo "${CURRENT_VERSION}" > ~/.secshell/.ver || handle_error "Failed to create version file"

# Install DrawBox dependency
echo "Installing DrawBox dependency..."
curl -s https://raw.githubusercontent.com/KaliforniaGator/DrawBox/main/update.sh | bash || handle_error "Failed to install DrawBox dependency"

# Check if DrawBox binary was downloaded to temp directory
DRAWBOX_TMP_PATH="/tmp/drawbox"
if [ -f "$DRAWBOX_TMP_PATH" ]; then
    echo "Moving DrawBox binary to ${DRAWBOX_BIN_PATH}..."
    # Make the binary executable if it's not already
    chmod +x "$DRAWBOX_TMP_PATH" || handle_error "Failed to make DrawBox binary executable"
    # Move to the appropriate location
    sudo mv "$DRAWBOX_TMP_PATH" "$DRAWBOX_BIN_PATH" || handle_error "Failed to install DrawBox binary"
else
    echo "Warning: DrawBox binary not found at expected location. Installation may be incomplete."
fi

echo "SecShell has been successfully installed!"
echo "You can now run 'secshell' to start the shell." 
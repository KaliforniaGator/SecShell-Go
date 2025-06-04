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

# Remove old .secshell directory if it exists
if [ -d ~/.secshell ]; then
    echo "Removing old .secshell directory..."
    rm -rf ~/.secshell
fi

# Create .secshell directory
echo "Creating .secshell directory..."
mkdir -p ~/.secshell &> /dev/null || handle_error "Failed to create ~/.secshell directory."

# Get the version from the binary
CURRENT_VERSION=$(${BIN_PATH} --version 2>&1 | grep -o '[0-9]\+\.[0-9]\+\.[0-9]\+')
if [ -z "$CURRENT_VERSION" ]; then
    CURRENT_VERSION="latest"
fi

# Create version file
echo "${CURRENT_VERSION}" > ~/.secshell/.ver || handle_error "Failed to create version file"

# Install DrawBox dependency - using direct download instead of compiling
echo "Installing DrawBox dependency..."
DRAWBOX_TMP_FILE="${TMP_DIR}/drawbox-download"
mkdir -p "${TMP_DIR}"

if [ "$OS_TYPE" = "linux" ]; then
    DRAWBOX_URL="https://github.com/KaliforniaGator/DrawBox/releases/download/linux-latest/drawbox-linux"
    curl -L -o "${DRAWBOX_TMP_FILE}" "${DRAWBOX_URL}" || handle_error "Failed to download DrawBox binary"
elif [ "$OS_TYPE" = "mac" ]; then
    DRAWBOX_URL="https://github.com/KaliforniaGator/DrawBox/releases/download/mac-latest/drawbox-mac"
    curl -L -o "${DRAWBOX_TMP_FILE}" "${DRAWBOX_URL}" || handle_error "Failed to download DrawBox binary"
fi

# Make DrawBox executable
chmod +x "${DRAWBOX_TMP_FILE}" || handle_error "Failed to make DrawBox binary executable"

# Move DrawBox to the appropriate location
sudo mv "${DRAWBOX_TMP_FILE}" "${DRAWBOX_BIN_PATH}" || handle_error "Failed to install DrawBox binary"

echo "SecShell has been successfully installed!"
echo "You can now run 'secshell' to start the shell." 
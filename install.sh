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
    DOWNLOAD_URL="https://github.com/KaliforniaGator/SecShell-Go/releases/download/linux-latest/secshell-lin-133"
    BIN_PATH="/usr/bin/secshell"
elif [ "$OS_TYPE" = "mac" ]; then
    DOWNLOAD_URL="https://github.com/KaliforniaGator/SecShell-Go/releases/download/mac-latest/secshell-mac-133"
    BIN_PATH="/usr/local/bin/secshell"
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

# Create version file
echo "1.3.3" > ~/.secshell/.ver || handle_error "Failed to create version file"

echo "SecShell has been successfully installed!"
echo "You can now run 'secshell' to start the shell." 
#!/bin/bash

# Detect operating system
OS="$(uname -s)"
case "${OS}" in
    Linux*)     OS_TYPE=linux;;
    Darwin*)    OS_TYPE=mac;;
    *)          OS_TYPE=unknown;;
esac

# Function to handle errors
handle_error() {
    echo "Error: $1"
    exit 1
}

# Initialize progress counter and total steps
CURRENT_STEP=0
TOTAL_STEPS=6  # Reduced total steps since we're not compiling from source

# Function to update progress
update_progress() {
    CURRENT_STEP=$((CURRENT_STEP + 1))
    echo -ne "\033[0K\r"  # Clear the current line and reset cursor to start of line
    case $CURRENT_STEP in
        1) STEP_DESC="Checking system requirements" ;;
        2) STEP_DESC="Setting up DrawBox" ;;
        3) STEP_DESC="Downloading latest SecShell binary" ;;
        4) STEP_DESC="Installing SecShell binary" ;;
        5) STEP_DESC="Updating version information" ;;
        6) STEP_DESC="Cleaning up" ;;
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

# Check system requirements
if [ "$OS_TYPE" = "unknown" ]; then
    handle_error "Unsupported operating system: ${OS}"
fi
update_progress

# Create temporary directory
TMP_DIR=$(mktemp -d)
if [ ! -d "$TMP_DIR" ]; then
    handle_error "Failed to create temporary directory"
fi

# Define URLs and paths
if [ "$OS_TYPE" = "linux" ]; then
    DOWNLOAD_URL="https://github.com/KaliforniaGator/SecShell-Go/releases/download/linux-latest/secshell-linux-latest"
    BIN_PATH="/usr/bin/secshell"
    DRAWBOX_BIN_PATH="/usr/bin/drawbox"
    DRAWBOX_URL="https://github.com/KaliforniaGator/DrawBox/releases/download/linux-latest/drawbox-linux"
elif [ "$OS_TYPE" = "mac" ]; then
    DOWNLOAD_URL="https://github.com/KaliforniaGator/SecShell-Go/releases/download/mac-latest/secshell-mac-latest"
    BIN_PATH="/usr/local/bin/secshell"
    DRAWBOX_BIN_PATH="/usr/local/bin/drawbox"
    DRAWBOX_URL="https://github.com/KaliforniaGator/DrawBox/releases/download/mac-latest/drawbox-mac"
fi

# Set up DrawBox
if ! command -v drawbox &> /dev/null; then
    echo "Installing DrawBox..."
    
    # Create temporary directory for DrawBox if it doesn't exist
    DRAWBOX_TMP_FILE="${TMP_DIR}/drawbox-download"
    
    # Download DrawBox binary
    curl -L -o "${DRAWBOX_TMP_FILE}" "${DRAWBOX_URL}" || handle_error "Failed to download DrawBox binary"
    
    # Make DrawBox executable
    chmod +x "${DRAWBOX_TMP_FILE}" || handle_error "Failed to make DrawBox binary executable"
    
    # Move DrawBox to the appropriate location
    sudo mv "${DRAWBOX_TMP_FILE}" "${DRAWBOX_BIN_PATH}" || handle_error "Failed to install DrawBox binary"
else
    echo "DrawBox is already installed. Checking for updates..."
    
    # Create temporary directory for DrawBox if it doesn't exist
    DRAWBOX_TMP_FILE="${TMP_DIR}/drawbox-download"
    
    # Download DrawBox binary
    curl -L -o "${DRAWBOX_TMP_FILE}" "${DRAWBOX_URL}" || handle_error "Failed to download DrawBox binary"
    
    # Make DrawBox executable
    chmod +x "${DRAWBOX_TMP_FILE}" || handle_error "Failed to make DrawBox binary executable"
    
    # Move DrawBox to the appropriate location
    sudo mv "${DRAWBOX_TMP_FILE}" "${DRAWBOX_BIN_PATH}" || handle_error "Failed to install DrawBox binary"
fi
update_progress

# Create temporary directory
TMP_DIR=$(mktemp -d)
TMP_FILE="${TMP_DIR}/secshell-download"

# Download the latest SecShell binary
echo "Downloading latest SecShell binary..."
curl -L -o "${TMP_FILE}" "${DOWNLOAD_URL}" || handle_error "Failed to download binary"
update_progress

# Make the binary executable
chmod +x "${TMP_FILE}" || handle_error "Failed to make binary executable"

# Move to the appropriate location
echo "Installing to ${BIN_PATH}..."
sudo mv "${TMP_FILE}" "${BIN_PATH}" || handle_error "Failed to install binary. Make sure you have sudo privileges."
update_progress

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
update_progress

# Clean up
rm -rf "${TMP_DIR}"
update_progress

echo -e "\nUpdate complete. You can now run 'secshell' to start the shell."

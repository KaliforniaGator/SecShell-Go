#!/bin/bash

# Function to check if a package is installed
is_installed() {
    dpkg -l | grep -qw "$1"
}

# Function to handle errors
handle_error() {
    echo "Error: $1"
    exit 1
}

# Initialize progress counter and total steps
CURRENT_STEP=0
TOTAL_STEPS=10  # Total number of major steps in the script

# Function to update progress
update_progress() {
    CURRENT_STEP=$((CURRENT_STEP + 1))
    echo -ne "\033[0K\r"  # Clear the current line and reset cursor to start of line
    case $CURRENT_STEP in
        1) STEP_DESC="Updating package lists" ;;
        2) STEP_DESC="Installing required packages" ;;
        3) STEP_DESC="Setting up DrawBox" ;;
        4) STEP_DESC="Cloning SecShell repository" ;;
        5) STEP_DESC="Initializing Go module" ;;
        6) STEP_DESC="Downloading dependencies" ;;
        7) STEP_DESC="Compiling SecShell" ;;
        8) STEP_DESC="Installing binary" ;;
        9) STEP_DESC="Updating version information" ;;
        10) STEP_DESC="Cleaning up" ;;
        *) STEP_DESC="Processing" ;;
    esac
    echo -ne "$CURRENT_STEP/$TOTAL_STEPS: $STEP_DESC"
    drawbox progress $CURRENT_STEP $TOTAL_STEPS 50 "█" "░" green
    echo -ne "\033[0K\r"
}

# Update package lists
sudo apt-get update || handle_error "Failed to update package lists."
update_progress

# Install necessary packages if not already installed
for package in golang-go libpam0g-dev; do
    if is_installed "$package"; then
        #echo "$package is already installed. Skipping..."
        continue
    else
        #echo "$package is not installed. Installing..."
        sudo apt-get install -y "$package" || handle_error "Failed to install $package."
    fi
done
update_progress

# Define the GitHub repositories
SECSHELL_REPO="https://github.com/KaliforniaGator/SecShell-Go.git"
DRAWBOX_URL="https://raw.githubusercontent.com/KaliforniaGator/DrawBox/refs/heads/main/update.sh"
SECSHELL_DIR="SecShell-Go"

# Check if DrawBox is installed
if ! command -v drawbox &> /dev/null; then
    curl -fsSL "$DRAWBOX_URL" | bash -s -- -q || handle_error "Failed to download or execute DrawBox update script."
    sudo mv drawbox /usr/local/bin/ || handle_error "Failed to move DrawBox binary."
fi
update_progress

# Clone SecShell-Go repository
if [ -d "$SECSHELL_DIR" ]; then
    #echo "Directory $SECSHELL_DIR already exists. Pulling latest changes..."
    cd "$SECSHELL_DIR"
    git pull -q origin main || handle_error "Failed to pull latest changes from $SECSHELL_REPO."
else
    git clone -q "$SECSHELL_REPO" "$SECSHELL_DIR" || handle_error "Failed to clone SecShell-Go repository."
    cd "$SECSHELL_DIR"
fi
update_progress

# Initialize the Go module if needed
if [ ! -f "go.mod" ] || ! grep -q "^module " "go.mod"; then
    go mod init github.com/KaliforniaGator/SecShell-Go > /dev/null || handle_error "Failed to initialize Go module."
fi
update_progress

# Download dependencies
go mod tidy || handle_error "Failed to download Go dependencies."
update_progress

# Compile the program
go build -o secshell secshell.go > /dev/null || handle_error "Compilation failed."
update_progress

# Move the binary to /usr/bin
sudo mv secshell /usr/bin/ || handle_error "Failed to move secshell binary to /usr/bin."
update_progress

# Update version file
# Create .secshell directory if it doesn't exist
mkdir -p ~/.secshell &> /dev/null || handle_error "Failed to create ~/.secshell directory."
# Get the version from GitHub and save it to .ver file
curl -s https://raw.githubusercontent.com/KaliforniaGator/SecShell-Go/refs/heads/main/VERSION > ~/.secshell/.ver 2> /dev/null || handle_error "Failed to update version information."
update_progress

# Clean up
cd ..
rm -rf "$SECSHELL_DIR"
update_progress

echo -e "\nUpdate complete. You can now run 'secshell' to start the shell."

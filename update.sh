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

# Update package lists
echo "Updating package lists..."
sudo apt-get update || handle_error "Failed to update package lists."

# Install necessary packages if not already installed
for package in golang-go libpam0g-dev; do
    if is_installed "$package"; then
        echo "$package is already installed. Skipping..."
    else
        echo "$package is not installed. Installing..."
        sudo apt-get install -y "$package" || handle_error "Failed to install $package."
    fi
done

# Define the GitHub repositories
SECSHELL_REPO="https://github.com/KaliforniaGator/SecShell-Go.git"
DRAWBOX_URL="https://raw.githubusercontent.com/KaliforniaGator/DrawBox/refs/heads/main/update.sh"
SECSHELL_DIR="SecShell-Go"

# Check if DrawBox is installed
if command -v drawbox &> /dev/null; then
    echo "DrawBox is already installed. Skipping installation."
else
    echo "Downloading and executing DrawBox update script..."
    curl -fsSL "$DRAWBOX_URL" | bash || handle_error "Failed to download or execute DrawBox update script."
    echo "DrawBox update script executed successfully."

    # Move DrawBox binary to /usr/local/bin
    echo "Moving DrawBox binary to /usr/local/bin..."
    sudo mv drawbox /usr/local/bin/ || handle_error "Failed to move DrawBox binary."
    echo "DrawBox binary installed successfully."
fi

# Clone SecShell-Go repository
if [ -d "$SECSHELL_DIR" ]; then
    echo "Directory $SECSHELL_DIR already exists. Pulling latest changes..."
    cd "$SECSHELL_DIR"
    git pull origin main || handle_error "Failed to pull latest changes from $SECSHELL_REPO."
else
    git clone "$SECSHELL_REPO" "$SECSHELL_DIR" || handle_error "Failed to clone SecShell-Go repository."
    cd "$SECSHELL_DIR"
fi

# Initialize the Go module if needed
if [ ! -f "go.mod" ] || ! grep -q "^module " "go.mod"; then
    echo "Initializing Go module..."
    go mod init github.com/KaliforniaGator/SecShell-Go || handle_error "Failed to initialize Go module."
fi

# Download dependencies
echo "Downloading Go dependencies..."
go mod tidy || handle_error "Failed to download Go dependencies."

# Compile the program
echo "Compiling secshell.go..."
go build -o secshell secshell.go || handle_error "Compilation failed."

# Move the binary to /usr/bin
echo "Moving 'secshell' binary to /usr/bin..."
sudo mv secshell /usr/bin/ || handle_error "Failed to move secshell binary to /usr/bin."

# Clean up
echo "Cleaning up..."
cd ..
rm -rf "$SECSHELL_DIR"

echo "Update complete. You can now run 'secshell' to start the shell."

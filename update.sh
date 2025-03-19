#!/bin/bash

# Function to check if a package is installed
is_installed() {
    dpkg -l | grep -qw "$1"
}

# Update package lists
sudo apt-get update

# Install necessary packages if not already installed
for package in golang-go libpam0g-dev systemctl; do
    if is_installed "$package"; then
        echo "$package is already installed. Skipping..."
    else
        echo "$package is not installed. Installing..."
        sudo apt-get install -y "$package"
        if [ $? -ne 0 ]; then
            echo "Failed to install $package."
            exit 1
        fi
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
    curl -fsSL "$DRAWBOX_URL" | bash
    if [ $? -ne 0 ]; then
        echo "Failed to download or execute DrawBox update script."
        exit 1
    fi

    echo "DrawBox update script executed successfully."

    # Move DrawBox binary to /usr/local/bin
    echo "Moving DrawBox binary to /usr/local/bin..."
    sudo mv drawbox /usr/local/bin/
    if [ $? -ne 0 ]; then
        echo "Failed to move DrawBox binary."
        exit 1
    fi
    echo "DrawBox binary installed successfully."
fi

# Clone SecShell-Go repository
if [ -d "$SECSHELL_DIR" ]; then
    echo "Directory $SECSHELL_DIR already exists. Pulling latest changes..."
    cd "$SECSHELL_DIR"
    git pull origin main
else
    git clone "$SECSHELL_REPO" "$SECSHELL_DIR"
    if [ $? -ne 0 ]; then
        echo "Failed to clone SecShell-Go repository."
        exit 1
    fi
    cd "$SECSHELL_DIR"
fi

# Initialize the Go module if needed
if [ ! -f "go.mod" ] || ! grep -q "^module " "go.mod"; then
    echo "Initializing Go module..."
    go mod init github.com/KaliforniaGator/SecShell-Go
    if [ $? -ne 0 ]; then
        echo "Failed to initialize Go module."
        exit 1
    fi
fi

# Download dependencies
echo "Downloading Go dependencies..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "Failed to download Go dependencies."
    exit 1
fi

# Compile the program
echo "Compiling secshell.go..."
go build -o secshell secshell.go
if [ $? -ne 0 ]; then
    echo "Compilation failed."
    exit 1
fi

# Move the binary to the current directory
echo "Moving 'secshell' binary to the current directory..."
mv secshell ..

# Clean up
echo "Cleaning up..."
cd ..
rm -rf "$SECSHELL_DIR"

echo "Update complete. You can now run './secshell' to start the shell."

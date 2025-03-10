#!/bin/bash

# Check if Go is installed, and install it if not
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Installing Go..."
    sudo apt-get update
    sudo apt-get install -y golang-go
    if [ $? -ne 0 ]; then
        echo "Failed to install Go. Please install it manually and try again."
        exit 1
    fi
    echo "Go installed successfully."
else
    echo "Go is already installed."
fi

# Define the GitHub repository
REPO_URL="https://github.com/KaliforniaGator/SecShell-Go.git"
REPO_DIR="SecShell-Go"

# Clone the repository
echo "Cloning repository from $REPO_URL..."
if [ -d "$REPO_DIR" ]; then
    echo "Directory $REPO_DIR already exists. Pulling latest changes..."
    cd "$REPO_DIR"
    git pull origin main
else
    git clone "$REPO_URL" "$REPO_DIR"
    if [ $? -ne 0 ]; then
        echo "Failed to clone the repository. Please check your internet connection and try again."
        exit 1
    fi
    cd "$REPO_DIR"
fi

# Initialize the Go module if go.mod is missing or incomplete
if [ ! -f "go.mod" ] || ! grep -q "^module " "go.mod"; then
    echo "go.mod is missing or incomplete. Initializing module..."
    go mod init github.com/KaliforniaGator/SecShell-Go
    if [ $? -ne 0 ]; then
        echo "Failed to initialize Go module. Please check the error message above."
        exit 1
    fi
fi

# Download dependencies
echo "Downloading Go dependencies..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "Failed to download Go dependencies. Please check the error message above."
    exit 1
fi

# Compile the program
echo "Compiling secshell.go..."
go build -o secshell secshell.go
if [ $? -ne 0 ]; then
    echo "Compilation failed. Please check the error message above."
    exit 1
fi

# Move the binary to the current working directory (PWD)
echo "Moving 'secshell' binary to the current directory..."
mv secshell ..

# Clean up: Remove the cloned repository
echo "Cleaning up..."
cd ..
rm -rf "$REPO_DIR"

echo "Compilation successful. The 'secshell' binary has been placed in the current directory."
echo "Update complete. You can now run './secshell' to start the shell."

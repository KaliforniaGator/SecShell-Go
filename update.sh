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

# Define the GitHub repository and files to download
REPO_URL="https://github.com/KaliforniaGator/SecShell-Go"
FILES=("secshell.go" "go.mod" "go.sum" "colors/colors.go")

# Download each file from the repository to the current directory
for FILE in "${FILES[@]}"; do
    FILE_URL="$REPO_URL/raw/main/$FILE"
    echo "Downloading $FILE..."
    mkdir -p "$(dirname "$FILE")"  # Create directories if needed (e.g., for colors/colors.go)
    curl -s -o "$FILE" "$FILE_URL"
    if [ $? -ne 0 ]; then
        echo "Failed to download $FILE. Please check your internet connection and try again."
        exit 1
    fi
done

# Compile the program
echo "Compiling secshell.go..."
go build -o secshell secshell.go
if [ $? -ne 0 ]; then
    echo "Compilation failed. Please check the downloaded files and troubleshoot the issue."
    exit 1
fi

echo "Compilation successful. The 'secshell' binary has been placed in the current directory."
echo "Update complete. You can now run './secshell' to start the shell."

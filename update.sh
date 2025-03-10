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

# Create a temporary directory for downloading files
TMP_DIR=$(mktemp -d)
echo "Downloading files to temporary directory: $TMP_DIR"

# Download each file from the repository
for FILE in "${FILES[@]}"; do
    FILE_URL="$REPO_URL/raw/main/$FILE"
    echo "Downloading $FILE..."
    mkdir -p "$(dirname "$TMP_DIR/$FILE")"
    curl -s -o "$TMP_DIR/$FILE" "$FILE_URL"
    if [ $? -ne 0 ]; then
        echo "Failed to download $FILE. Please check your internet connection and try again."
        exit 1
    fi
done

# Move to the temporary directory
cd "$TMP_DIR"

# Compile the program
echo "Compiling secshell.go..."
go build -o secshell secshell.go
if [ $? -ne 0 ]; then
    echo "Compilation failed. The downloaded files are located in: $TMP_DIR"
    echo "You can inspect the files and troubleshoot the issue."
    exit 1
fi

# Move the compiled binary to the current directory
mv secshell "$(dirname "$0")"
echo "Compilation successful. The 'secshell' binary has been placed in the current directory."

# Keep the downloaded files for inspection
echo "Downloaded files are kept in: $TMP_DIR"
echo "You can inspect or reuse them if needed."

echo "Update complete. You can now run './secshell' to start the shell."

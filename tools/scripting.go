package tools

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"secshell/ui"
	"strings"
)

// ExecuteScript detects if a command is a script file (starting with "./")
// and executes it with the appropriate interpreter based on the shebang or extension
func ExecuteScript(command string) (string, error) {
	// Check if command starts with "./"
	if !strings.HasPrefix(command, "./") {
		return "", fmt.Errorf("not a script execution command")
	}

	// Extract arguments
	parts := strings.Split(command, " ")
	scriptPath := parts[0]
	args := parts[1:]

	// Check if file exists
	info, err := os.Stat(scriptPath)
	if err != nil {
		return "", fmt.Errorf("script file not found: %v", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory, not a script file", scriptPath)
	}

	// Check for executable permission
	if info.Mode().Perm()&0111 == 0 {
		return "", fmt.Errorf("script is not executable, use 'chmod +x %s' first", scriptPath)
	}

	// Open file to check for shebang
	file, err := os.Open(scriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to open script: %v", err)
	}
	defer file.Close()

	// Read first line to check for shebang
	scanner := bufio.NewScanner(file)
	var interpreter string
	if scanner.Scan() {
		firstLine := scanner.Text()
		if strings.HasPrefix(firstLine, "#!") {
			// Extract interpreter from shebang
			interpreterPath := strings.TrimPrefix(firstLine, "#!")
			interpreterPath = strings.TrimSpace(interpreterPath)
			interpreter = strings.Split(interpreterPath, " ")[0]
		}
	}

	// If no shebang, determine interpreter from extension
	if interpreter == "" {
		ext := filepath.Ext(scriptPath)
		switch ext {
		case ".sh":
			interpreter = "sh"
		case ".bash":
			interpreter = "bash"
		case ".py":
			interpreter = "python"
		case ".rb":
			interpreter = "ruby"
		case ".pl":
			interpreter = "perl"
		case ".js":
			interpreter = "node"
		default:
			// Default to bash for executable files without extension
			interpreter = "bash"
		}
	}

	ui.NewLine()
	// Prepare command execution
	var cmd *exec.Cmd
	if len(args) > 0 {
		cmd = exec.Command(interpreter, append([]string{scriptPath}, args...)...)
	} else {
		cmd = exec.Command(interpreter, scriptPath)
	}

	// Execute command and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("script execution failed: %v", err)
	}

	return string(output), nil
}

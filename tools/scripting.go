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
func ExecuteScript(command string) error {
	// Check if command starts with "./"
	if !strings.HasPrefix(command, "./") {
		return fmt.Errorf("not a script execution command")
	}

	// Extract arguments
	parts := strings.Split(command, " ")
	scriptPath := parts[0]
	args := parts[1:]

	// Check if file exists
	info, err := os.Stat(scriptPath)
	if err != nil {
		return fmt.Errorf("script file not found: %v", err)
	}

	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a script file", scriptPath)
	}

	// Open file to check for shebang
	file, err := os.Open(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to open script: %v", err)
	}
	defer file.Close()

	// Read first line to check for shebang
	scanner := bufio.NewScanner(file)
	var interpreter string
	var interpreterArgs []string

	if scanner.Scan() {
		firstLine := scanner.Text()
		if strings.HasPrefix(firstLine, "#!") {
			// Extract interpreter from shebang
			shebangLine := strings.TrimPrefix(firstLine, "#!")
			shebangLine = strings.TrimSpace(shebangLine)
			shebangParts := strings.Fields(shebangLine)

			if len(shebangParts) > 0 {
				interpreterPath := shebangParts[0]
				// Extract interpreter name from full path
				interpreter = filepath.Base(interpreterPath)

				// Handle common interpreter variations
				switch interpreter {
				case "python3", "python2":
					interpreter = "python3" // Prefer python3
				case "python":
					// Check if python3 is available, fallback to python
					if _, err := exec.LookPath("python3"); err == nil {
						interpreter = "python3"
					}
				case "sh", "bash", "zsh", "dash":
					// Keep as is
				case "node", "nodejs":
					interpreter = "node"
				case "ruby":
					interpreter = "ruby"
				case "perl":
					interpreter = "perl"
				}

				// Store additional arguments from shebang if any
				if len(shebangParts) > 1 {
					interpreterArgs = shebangParts[1:]
				}
			}
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
		case ".zsh":
			interpreter = "zsh"
		case ".py":
			// Prefer python3 if available
			if _, err := exec.LookPath("python3"); err == nil {
				interpreter = "python3"
			} else {
				interpreter = "python"
			}
		case ".rb":
			interpreter = "ruby"
		case ".pl":
			interpreter = "perl"
		case ".js":
			interpreter = "node"
		case ".php":
			interpreter = "php"
		case ".lua":
			interpreter = "lua"
		case ".r", ".R":
			interpreter = "R"
		default:
			// For executable files without extension, try to execute directly first
			// If that fails, fall back to shell
			interpreter = ""
		}
	}

	ui.NewLine()
	// Prepare command execution
	var cmd *exec.Cmd

	if interpreter == "" {
		// For files without a detected interpreter, check executable permissions
		if info.Mode().Perm()&0111 == 0 {
			return fmt.Errorf("script is not executable and no interpreter detected, use 'chmod +x %s' first", scriptPath)
		}

		// Try to execute the file directly (for compiled executables or scripts with proper shebang)
		if len(args) > 0 {
			cmd = exec.Command(scriptPath, args...)
		} else {
			cmd = exec.Command(scriptPath)
		}
	} else {
		// Use the detected interpreter - no executable permission needed
		cmdArgs := []string{scriptPath}

		// Add interpreter arguments from shebang if any
		if len(interpreterArgs) > 0 {
			cmdArgs = append(interpreterArgs, cmdArgs...)
		}

		// Add script arguments
		if len(args) > 0 {
			cmdArgs = append(cmdArgs, args...)
		}

		cmd = exec.Command(interpreter, cmdArgs...)
	}

	// Set up stdin, stdout, stderr to allow interactive processes
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Execute command interactively
	err = cmd.Run()
	if err != nil {
		// If direct execution failed and we didn't use an interpreter, try with shell as fallback
		if interpreter == "" {
			fallbackCmd := exec.Command("sh", append([]string{scriptPath}, args...)...)
			fallbackCmd.Stdin = os.Stdin
			fallbackCmd.Stdout = os.Stdout
			fallbackCmd.Stderr = os.Stderr

			fallbackErr := fallbackCmd.Run()
			if fallbackErr != nil {
				return fmt.Errorf("script execution failed: %v (fallback with sh also failed: %v)", err, fallbackErr)
			}
			return nil
		}
		return fmt.Errorf("script execution failed: %v", err)
	}

	return nil
}

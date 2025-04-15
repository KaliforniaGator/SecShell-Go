package core

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"secshell/drawbox"
	"secshell/update"
	"strings"
)

var AllowedCommands = []string{}
var BlacklistedCommands = []string{}
var BuiltInCommands = []string{}
var ProgramCommands = []string{}
var AllowedDirs = []string{}
var History = []string{}
var DefaultBlacklist = []string{"rm", "mv", "cp", "dd", "mkfs", "reboot", "shutdown", "halt", "poweroff", "init", "systemctl", "service", "killall", "pkill"}
var DefaultWhitelist = []string{"sudo", "apt", "ls", "cd", "pwd", "cp", "mv", "rm", "mkdir", "rmdir", "touch", "cat", "echo", "grep", "find", "chmod", "chown", "ps", "kill", "top", "df", "du", "ifconfig", "netstat", "ping", "ip", "clear", "vim", "nano", "emacs", "nvim"}

// getExecutablePath returns the path to the executable directory
func GetExecutablePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "." // Fallback to current directory if home directory cannot be determined
	}
	return filepath.Join(homeDir, ".secshell") // Use ~/.secshell for config files
}

// ensureFilesExist checks and creates blacklist and whitelist files if they don't exist
func EnsureFilesExist(blacklist, whitelist, version, history string) {
	// Ensure the config directory exists
	configDir := filepath.Dir(blacklist)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to create config directory: %s", err))
		return
	}

	// Ensure directory exists
	exePath := GetExecutablePath()
	if err := os.MkdirAll(exePath, 0755); err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to create directory for config files: %s", err))
		return
	}

	// Create blacklist if it doesn't exist
	if _, err := os.Stat(blacklist); os.IsNotExist(err) {
		file, err := os.Create(blacklist)
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to create blacklist file: %s", err))
		} else {
			file.Close()
			drawbox.PrintAlert(fmt.Sprintf("Created new blacklist file at %s", blacklist))
		}
	}

	// Create/update whitelist if needed
	if _, err := os.Stat(whitelist); os.IsNotExist(err) {
		file, err := os.Create(whitelist)
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to create whitelist file: %s", err))
		} else {
			for _, cmd := range DefaultWhitelist {
				file.WriteString(cmd + "\n")
			}
			file.Close()
			drawbox.PrintAlert(fmt.Sprintf("Created new whitelist file at %s with default commands", whitelist))
		}
	} else {
		// Update existing whitelist with any missing default commands
		existingCommands := make(map[string]bool)
		file, err := os.OpenFile(whitelist, os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to open whitelist file: %s", err))
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			existingCommands[scanner.Text()] = true
		}

		for _, cmd := range DefaultWhitelist {
			if !existingCommands[cmd] {
				file.WriteString(cmd + "\n")
			}
		}
	}

	// Create version file if it doesn't exist
	if _, err := os.Stat(version); os.IsNotExist(err) {
		file, err := os.Create(version)
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to create version file: %s", err))
		} else {
			file.Close()
			update.UpdateVersionFile(update.DefaultVersion, version)
			drawbox.PrintAlert(fmt.Sprintf("Created new version file at %s", version))
		}
	}
	update.CheckForUpdates(version)

	// Create History file if it doesn't exist
	if _, err := os.Stat(history); os.IsNotExist(err) {
		file, err := os.Create(history)
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to create history file: %s", err))
		} else {
			file.Close()
			drawbox.PrintAlert(fmt.Sprintf("Created new history file at %s", history))
		}
	}
}

// Load History loads command history from a file
func LoadHistory(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to open history file: %s", filename))
		return
	}
	defer file.Close()
	History = []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())
		if command != "" {
			History = append(History, command)
		}
	}
	if err := scanner.Err(); err != nil {
		drawbox.PrintError(fmt.Sprintf("Error reading history file: %s", err))
	}
}

// Save History saves command history to a file
func SaveHistory(filename string, command string) {
	// Append to History slice
	History = append(History, command)

	// Append to history file
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to open history file for writing: %s", filename))
		return
	}
	defer file.Close()

	if _, err := file.WriteString(command + "\n"); err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to write to history file: %s", err))
		return
	}
}

func ClearHistory(filename string) {
	// Clear the History slice
	History = []string{}

	// Truncate the history file
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to clear history file: %s", err))
		return
	}
	defer file.Close()
	drawbox.PrintAlert("History cleared.")
}

// loadBlacklist loads blacklisted commands from a file
func LoadBlacklist(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to open blacklist file: %s", filename))
		return
	}
	defer file.Close()
	BlacklistedCommands = []string{} // Reset blacklisted commands
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())
		if command != "" {
			BlacklistedCommands = append(BlacklistedCommands, command)
		}
	}
	if err := scanner.Err(); err != nil {
		drawbox.PrintError(fmt.Sprintf("Error reading blacklist file: %s", err))
	}
}

// editBlacklist opens the blacklist file in an editor
func EditBlacklist(filename string) {
	cmd := exec.Command("nano", filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

// listBlacklistCommands lists all blacklisted commands
func ListBlacklistCommands(filename string) {
	drawbox.RunDrawbox("Blacklisted Commands", "bold_white")
	file, err := os.Open(filename)
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Error: Could not open file '%s'.", filename))
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		fmt.Printf(" %d. %s\n", lineNumber, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		drawbox.PrintError(fmt.Sprintf("Error reading file: %s", err))
	}
}

// loadWhitelist loads whitelisted commands from a file
func LoadWhitelist(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		drawbox.PrintAlert(fmt.Sprintf("Notice: No whitelist file found at %s. Using default allowed commands.", filename))
		return
	}
	defer file.Close()

	AllowedCommands = []string{} // Reset allowed commands
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())
		if command != "" {
			AllowedCommands = append(AllowedCommands, command)
		}
	}
	if err := scanner.Err(); err != nil {
		drawbox.PrintError(fmt.Sprintf("Error reading whitelist file: %s", err))
		return
	}

	if len(AllowedCommands) == 0 {
		drawbox.PrintAlert("Warning: Whitelist file is empty. Allowing hard-coded commands and any command within allowed directories.")
		AllowedCommands = DefaultWhitelist
	}
}

// editWhitelist opens the whitelist file in an editor
func EditWhitelist(filename string) {
	cmd := exec.Command("nano", filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

// listWhitelistCommands lists all whitelisted commands
func ListWhitelistCommands() {
	drawbox.RunDrawbox("Whitelisted Commands", "bold_white")
	for i, cmd := range AllowedCommands {
		fmt.Printf(" %d. %s\n", i+1, cmd)
	}
	if len(AllowedCommands) == 0 {
		drawbox.PrintAlert("No commands are whitelisted.")
	}
}

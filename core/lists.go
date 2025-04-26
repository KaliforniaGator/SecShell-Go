package core

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"secshell/admin"
	"secshell/ui/gui"
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

// createSecureFile creates a file with admin-only permissions and writes initial content if provided
func createSecureFile(filepath string, initialContent []string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %s", err)
	}
	defer file.Close()

	// Write initial content if provided
	if len(initialContent) > 0 {
		for _, line := range initialContent {
			if _, err := file.WriteString(line + "\n"); err != nil {
				return fmt.Errorf("failed to write to file: %s", err)
			}
		}
	}

	// Set proper permissions
	if err := admin.SetFilePermissions(filepath); err != nil {
		return fmt.Errorf("failed to set file permissions: %s", err)
	}

	return nil
}

// EnsureFilesExist checks and creates blacklist and whitelist files if they don't exist
func EnsureFilesExist(blacklist, whitelist, version, history, logfile string) {
	// Ensure the .secshell directory exists with proper permissions
	configDir := GetExecutablePath()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		gui.ErrorBox(fmt.Sprintf("Failed to create config directory: %s", err))
		return
	}
	if err := admin.SetFolderPermissions(configDir); err != nil {
		gui.ErrorBox(fmt.Sprintf("Failed to set directory permissions: %s", err))
		return
	}

	// Create files if they don't exist
	files := map[string][]string{
		blacklist: {},               // Empty blacklist
		whitelist: DefaultWhitelist, // Default whitelist commands
		version:   {},               // Empty version file
		history:   {},               // Empty history file
		logfile:   {},               // Empty log file
	}

	for filepath, content := range files {
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			if err := createSecureFile(filepath, content); err != nil {
				gui.ErrorBox(fmt.Sprintf("Failed to create %s: %s", filepath, err))
				continue
			}
			gui.AlertBox(fmt.Sprintf("Created new file at %s", filepath))

			// Special handling for version file
			if filepath == version {
				update.UpdateVersionFile(update.GetLatestVersion(), version)
				update.CheckForUpdates(version)
			}
		}
	}

	// Check for updates after ensuring version file exists
	update.CheckForUpdates(version)
}

// Load History loads command history from a file
func LoadHistory(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		gui.ErrorBox(fmt.Sprintf("Failed to open history file: %s", filename))
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
		gui.ErrorBox(fmt.Sprintf("Error reading history file: %s", err))
	}
}

// Save History saves command history to a file
func SaveHistory(filename string, command string) {
	// Append to History slice
	History = append(History, command)

	// Append to history file
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		gui.ErrorBox(fmt.Sprintf("Failed to open history file for writing: %s", filename))
		return
	}
	defer file.Close()

	if _, err := file.WriteString(command + "\n"); err != nil {
		gui.ErrorBox(fmt.Sprintf("Failed to write to history file: %s", err))
		return
	}
}

func ClearHistory(filename string) {
	// Clear the History slice
	History = []string{}

	// Truncate the history file
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		gui.ErrorBox(fmt.Sprintf("Failed to clear history file: %s", err))
		return
	}
	defer file.Close()
	gui.AlertBox("History cleared.")
}

// loadBlacklist loads blacklisted commands from a file
func LoadBlacklist(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		gui.ErrorBox(fmt.Sprintf("Failed to open blacklist file: %s", filename))
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
		gui.ErrorBox(fmt.Sprintf("Error reading blacklist file: %s", err))
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
	gui.TitleBox("Blacklisted Commands")
	file, err := os.Open(filename)
	if err != nil {
		gui.ErrorBox(fmt.Sprintf("Error: Could not open file '%s'.", filename))
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
		gui.ErrorBox(fmt.Sprintf("Error reading file: %s", err))
	}
}

// loadWhitelist loads whitelisted commands from a file
func LoadWhitelist(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		gui.AlertBox(fmt.Sprintf("Notice: No whitelist file found at %s. Using default allowed commands.", filename))
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
		gui.ErrorBox(fmt.Sprintf("Error reading whitelist file: %s", err))
		return
	}

	if len(AllowedCommands) == 0 {
		gui.AlertBox("Warning: Whitelist file is empty. Allowing hard-coded commands and any command within allowed directories.")
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
	gui.TitleBox("Whitelisted Commands")
	for i, cmd := range AllowedCommands {
		fmt.Printf(" %d. %s\n", i+1, cmd)
	}
	if len(AllowedCommands) == 0 {
		gui.AlertBox("No commands are whitelisted.")
	}
}

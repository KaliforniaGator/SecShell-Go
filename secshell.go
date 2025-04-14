package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"secshell/colors"
	"secshell/commands"
	"secshell/core"
	"secshell/download"
	"secshell/drawbox"
	"secshell/help"
	"secshell/jobs"
	"secshell/sanitize"
	"secshell/services"
	"secshell/ui"
	"secshell/update"

	"github.com/msteinert/pam"
	"golang.org/x/term"
)

// Constants for key inputs
const (
	KeyUp        = "\x1b[A"
	KeyDown      = "\x1b[B"
	KeyTab       = "\t"
	KeyDelete    = "\x7f" // Add backspace key constant
	KeyBackspace = "\b"   // Add backspace key constant
	KeyLeft      = "\x1b[D"
	KeyRight     = "\x1b[C"
)

// SecShell struct to hold shell state and configurations
type SecShell struct {
	jobs                map[int]*jobs.Job
	running             bool
	allowedDirs         []string
	allowedCommands     []string
	blacklist           string
	blacklistedCommands []string
	history             []string
	whitelist           string
	versionFile         string
	historyIndex        int
}

// Define a list of built-in commands
var builtInCommands = []string{
	"allowed", "help", "exit", "services", "jobs", "cd", "history", "export", "env", "unset",
	"reload-blacklist", "blacklist", "edit-blacklist", "whitelist", "edit-whitelist",
	"reload-whitelist", "download", "--version", "--update", "time", "date"}

// NewSecShell initializes a new SecShell instance
func NewSecShell(blacklistPath, whitelistPath string) *SecShell {
	shell := &SecShell{
		jobs:        make(map[int]*jobs.Job),
		running:     true,
		allowedDirs: []string{"/usr/bin/", "/bin/", "/opt/", "/usr/local/bin/"},
		blacklist:   blacklistPath,
		whitelist:   whitelistPath,
		versionFile: filepath.Join(filepath.Dir(blacklistPath), ".ver"),
		history:     []string{},
	}
	shell.ensureFilesExist()
	shell.loadBlacklist(blacklistPath)
	shell.loadWhitelist(whitelistPath)
	return shell
}

// Get current Time
func (s *SecShell) getTime() {
	now := time.Now()
	drawbox.RunDrawbox(fmt.Sprintf("Current time: %s", now.Format("3:04 PM")), "bold_white")
}

// Get current Date
func (s *SecShell) getDate() {
	now := time.Now()
	drawbox.RunDrawbox(fmt.Sprintf("Current date: %s", now.Format("02-Jan-2006")), "bold_white")

}

// ensureFilesExist checks and creates blacklist and whitelist files if they don't exist
func (s *SecShell) ensureFilesExist() {
	// Ensure the config directory exists
	configDir := filepath.Dir(s.blacklist)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to create config directory: %s", err))
		return
	}

	defaultWhitelistCommands := []string{"ls", "cd", "pwd", "cp", "mv", "rm", "mkdir", "rmdir", "touch", "cat", "echo", "grep", "find", "chmod", "chown", "ps", "kill", "top", "df", "du", "ifconfig", "netstat", "ping", "ip", "clear", "vim", "nano", "emacs", "nvim"}

	// Ensure directory exists
	exePath := getExecutablePath()
	if err := os.MkdirAll(exePath, 0755); err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to create directory for config files: %s", err))
		return
	}

	// Create blacklist if it doesn't exist
	if _, err := os.Stat(s.blacklist); os.IsNotExist(err) {
		file, err := os.Create(s.blacklist)
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to create blacklist file: %s", err))
		} else {
			file.Close()
			drawbox.PrintAlert(fmt.Sprintf("Created new blacklist file at %s", s.blacklist))
		}
	}

	// Create/update whitelist if needed
	if _, err := os.Stat(s.whitelist); os.IsNotExist(err) {
		file, err := os.Create(s.whitelist)
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to create whitelist file: %s", err))
		} else {
			for _, cmd := range defaultWhitelistCommands {
				file.WriteString(cmd + "\n")
			}
			file.Close()
			drawbox.PrintAlert(fmt.Sprintf("Created new whitelist file at %s with default commands", s.whitelist))
		}
	} else {
		// Update existing whitelist with any missing default commands
		existingCommands := make(map[string]bool)
		file, err := os.OpenFile(s.whitelist, os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to open whitelist file: %s", err))
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			existingCommands[scanner.Text()] = true
		}

		for _, cmd := range defaultWhitelistCommands {
			if !existingCommands[cmd] {
				file.WriteString(cmd + "\n")
			}
		}
	}

	// Create version file if it doesn't exist
	if _, err := os.Stat(s.versionFile); os.IsNotExist(err) {
		file, err := os.Create(s.versionFile)
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to create version file: %s", err))
		} else {
			file.Close()
			update.UpdateVersionFile(update.DefaultVersion, s.versionFile)
			drawbox.PrintAlert(fmt.Sprintf("Created new version file at %s", s.versionFile))
		}
	}
	update.CheckForUpdates(s.versionFile)
}

// loadBlacklist loads blacklisted commands from a file
func (s *SecShell) loadBlacklist(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to open blacklist file: %s", filename))
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())
		if command != "" {
			s.blacklistedCommands = append(s.blacklistedCommands, command)
		}
	}
}

// loadWhitelist loads whitelisted commands from a file
func (s *SecShell) loadWhitelist(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		drawbox.PrintAlert(fmt.Sprintf("Notice: No whitelist file found at %s. Using default allowed commands.", filename))
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())
		if command != "" {
			s.allowedCommands = append(s.allowedCommands, command)
		}
	}

	if len(s.allowedCommands) == 0 {
		drawbox.PrintAlert("Warning: Whitelist file is empty. Allowing hard-coded commands and any command within allowed directories.")
		s.allowedCommands = []string{"ls", "cd", "pwd", "cp", "mv", "rm", "mkdir", "rmdir", "touch", "cat", "echo", "grep", "find", "chmod", "chown", "ps", "kill", "top", "df", "du", "ifconfig", "netstat", "ping", "ip", "clear", "vim", "nano", "emacs", "nvim"}
	}

	commands.AllowedCommands = s.allowedCommands
	commands.BuiltInCommands = builtInCommands
	commands.AllowedDirs = s.allowedDirs
	commands.Init()
}

// Check if the current user is in an admin group
func isAdmin() bool {
	currentUser, err := user.Current()
	if err != nil {
		return false // Fail-safe: assume not an admin
	}

	// Root (UID 0) is always an admin
	if currentUser.Uid == "0" {
		return true
	}

	// Get the user's group IDs
	groups, err := currentUser.GroupIds()
	if err != nil {
		return false
	}

	// Define admin groups (adjust as needed)
	adminGroups := []string{"sudo", "admin", "wheel", "root"}

	// Check if the user belongs to an admin group
	for _, groupID := range groups {
		group, err := user.LookupGroupId(groupID)
		if err == nil {
			for _, adminGroup := range adminGroups {
				if group.Name == adminGroup {
					return true
				}
			}
		}
	}

	return false
}

// Check if update is needed
func IsUpdateNeeded(currentVersion, latestVersion string) bool {
	// Already in correct format - no need to strip 'v' prefix
	currentParts := strings.Split(currentVersion, ".")
	latestParts := strings.Split(latestVersion, ".")

	// Validate version format
	if len(currentParts) != 3 || len(latestParts) != 3 {
		return false
	}

	// Convert version parts to integers
	current := make([]int, 3)
	latest := make([]int, 3)

	for i := 0; i < 3; i++ {
		var err error
		current[i], err = strconv.Atoi(currentParts[i])
		if err != nil {
			return false
		}
		latest[i], err = strconv.Atoi(latestParts[i])
		if err != nil {
			return false
		}
	}

	// Compare versions
	return latest[0] > current[0] || // Major version
		(latest[0] == current[0] && latest[1] > current[1]) || // Minor version
		(latest[0] == current[0] && latest[1] == current[1] && latest[2] > current[2]) // Patch version
}

// run starts the shell and listens for user input
func (s *SecShell) run() {

	currentVersion := update.GetCurrentVersion(s.versionFile)
	latestVersion := update.GetLatestVersion()
	needsUpdate := IsUpdateNeeded(currentVersion, latestVersion)

	// Display welcome screen
	ui.DisplayWelcomeScreen(currentVersion, needsUpdate)

	// Create a signal channel
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)

	// Handle signals in a goroutine
	go func() {
		for range signalChan {
			// Just ignore SIGINT - this prevents the shell from exiting
			fmt.Print("\r\n\n")
			fmt.Printf("%s Command exited with Ctrl+C %s\n", colors.BoldYellow, colors.Reset)
			fmt.Print("\r\n")
		}
	}()

	for s.running {
		input := s.getInput()
		s.processCommand(input)
	}

	// Clear the screen and move cursor to start before exiting
	fmt.Print("\033[H\033[2J")
}

// getInput reads user input from the terminal
func (s *SecShell) getInput() string {
	ui.DisplayPrompt()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to set terminal to raw mode: %s", err))
		return ""
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	line := ""
	pos := 0
	buf := make([]byte, 8192) // Increased buffer size to handle pasting

	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to read input: %s", err))
			return ""
		}

		// Handle input bytes one by one
		for i := 0; i < n; i++ {
			// Handle bracketed paste mode
			if i+5 < n &&
				buf[i] == 27 && buf[i+1] == '[' && buf[i+2] == '2' &&
				buf[i+3] == '0' && buf[i+4] == '0' && buf[i+5] == '~' {
				i += 6
				continue
			}
			// Handle regular input
			switch buf[i] {
			case 3: // Ctrl+C
				fmt.Print("\r\n\n")
				fmt.Printf("%s To exit, type 'exit' %s\n", colors.BoldRed, colors.Reset)
				fmt.Print("\r\n")
				ui.DisplayPrompt()
				ui.ClearLineAndPrintBottom()
			case 27: // ESC sequence
				if i+2 < n { // Check if we have enough bytes for an escape sequence
					if buf[i+1] == '[' {
						switch buf[i+2] {
						case 'A': // Up arrow
							if s.historyIndex > 0 {
								s.historyIndex--
								newLine := strings.TrimSpace(s.history[s.historyIndex])
								fmt.Printf("\x1b[%dD\x1b[K%s", pos, newLine)
								line = newLine
								pos = len(line)
							}
							i += 2
						case 'B': // Down arrow
							if s.historyIndex < len(s.history)-1 {
								s.historyIndex++
								newLine := strings.TrimSpace(s.history[s.historyIndex])
								fmt.Printf("\x1b[%dD\x1b[K%s", pos, newLine)
								line = newLine
								pos = len(line)
							}
							i += 2
						case 'C': // Right arrow
							if pos < len(line) {
								pos++
								fmt.Print("\x1b[C")
							}
							i += 2
						case 'D': // Left arrow
							if pos > 0 {
								pos--
								fmt.Print("\x1b[D")
							}
							i += 2
						}
					}
				}
			case 127, 8: // Backspace and Delete
				if pos > 0 {
					line = line[:pos-1] + line[pos:]
					pos--
					fmt.Print("\x1b[D\x1b[K")
					if pos < len(line) {
						fmt.Print(line[pos:])
						fmt.Printf("\x1b[%dD", len(line)-pos)
					}
				}
			case 9: // Tab
				line, pos = commands.CompleteCommand(line, pos)
			case 13, 10: // Enter (CR or LF)
				fmt.Println()
				input := s.sanitizeInput(strings.TrimSpace(line), true)
				if input != "" {
					s.history = append(s.history, input)
					s.historyIndex = len(s.history)
				}
				return input
			default:
				if buf[i] >= 32 { // Printable characters
					// Insert character at current position
					line = line[:pos] + string(buf[i]) + line[pos:]
					fmt.Print(line[pos:])
					pos++
					if pos < len(line) {
						fmt.Printf("\x1b[%dD", len(line)-pos)
					}
				}
			}
		}
	}
}

// sanitizeInput uses the sanitize package to clean input
func (s *SecShell) sanitizeInput(input string, allowSpecialChars ...bool) string {
	allow := true
	if len(allowSpecialChars) > 0 {
		allow = allowSpecialChars[0]
	}
	return sanitize.Input(input, allow)
}

// displayHistory shows the command history
func (s *SecShell) displayHistory() {
	drawbox.RunDrawbox("Command History", "bold_white")
	for i, cmd := range s.history {
		fmt.Printf("%d: %s\n", i+1, cmd)
	}
}

func (s *SecShell) searchHistory(query string) {
	drawbox.RunDrawbox("History Search: "+query, "bold_white")
	found := false

	for i, cmd := range s.history {
		if strings.Contains(strings.ToLower(cmd), strings.ToLower(query)) {
			highlightedCmd := highlightText(cmd, query)
			fmt.Printf("%d: %s\n", i+1, highlightedCmd)
			found = true
		}
	}

	if !found {
		drawbox.PrintAlert("No matching commands found.")
	}
}

func (s *SecShell) runHistoryCommand(number int) bool {
	if number <= 0 || number > len(s.history) {
		drawbox.PrintError(fmt.Sprintf("Invalid history number: %d", number))
		return false
	}

	cmd := s.history[number-1]
	drawbox.PrintAlert(fmt.Sprintf("Running: %s", cmd))
	s.processCommand(cmd)
	return true
}

func (s *SecShell) interactiveHistorySearch() {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to set terminal to raw mode: %s", err))
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	query := ""
	selectedIndex := 0
	filteredHistory := []string{}

	// Hide cursor while navigating
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h") // Ensure cursor is shown when function exits

	// Helper function to refresh display
	refreshDisplay := func() {
		fmt.Print("\033[H\033[2J\033[3J") // Clear screen and scrollback buffer
		// Display header with drawbox
		fmt.Print("\n")
		ui.ClearLine()
		fmt.Print(colors.BoldGreen + "┌─[Interactive History Search]" + colors.Reset + "\n")
		ui.ClearLine()
		fmt.Printf(colors.BoldGreen+"└─"+colors.Reset+"$ %s", query)

		// Print instructions
		fmt.Print("\n")
		ui.ClearLine()
		fmt.Println("Type to search, Up/Down arrows to navigate, Enter to select, Esc to cancel")

		// Filter history based on query
		filteredHistory = []string{}
		for _, cmd := range s.history {
			if query == "" || strings.Contains(strings.ToLower(cmd), strings.ToLower(query)) {
				filteredHistory = append(filteredHistory, cmd)
			}
		}

		// Display results with selection highlight
		for i, cmd := range filteredHistory {
			if i == selectedIndex {
				ui.ClearLine()
				fmt.Printf("%s→ %d: %s%s\n", colors.BoldGreen, i+1, cmd, colors.Reset)
			} else {
				ui.ClearLine()
				fmt.Printf("  %d: %s\n", i+1, cmd)
			}
		}

		if len(filteredHistory) == 0 {
			ui.ClearLine()
			fmt.Println("  No matching commands found.")
		}
	}

	// Initial display
	refreshDisplay()

	// Input loop
	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to read input: %s", err))
			return
		}

		if n == 1 {
			switch buf[0] {
			case 27: // ESC
				// Clear screen before returning to normal mode
				fmt.Print("\033[H\033[2J\033[3J") // Clear screen and scrollback buffer
				return

			case 13: // Enter
				if len(filteredHistory) > 0 && selectedIndex >= 0 && selectedIndex < len(filteredHistory) {
					selectedCmd := filteredHistory[selectedIndex]
					// Restore terminal and run command
					fmt.Print("\033[?25h") // Make sure cursor is visible
					term.Restore(int(os.Stdin.Fd()), oldState)
					fmt.Print("\033[H\033[2J\033[3J") // Clear screen and scrollback buffer
					drawbox.PrintAlert("Running: " + selectedCmd)
					s.processCommand(selectedCmd)
					return
				}

			case 127, 8: // Backspace/Delete
				if len(query) > 0 {
					query = query[:len(query)-1]
					selectedIndex = 0
					refreshDisplay()
				}

			default:
				// Add printable characters to query
				if buf[0] >= 32 && buf[0] <= 126 {
					query += string(buf[0])
					selectedIndex = 0
					refreshDisplay()
				}
			}
		} else if n == 3 && buf[0] == 27 && buf[1] == 91 {
			// Handle arrow keys
			switch buf[2] {
			case 65: // Up arrow
				if selectedIndex > 0 {
					selectedIndex--
					refreshDisplay()
				}

			case 66: // Down arrow
				if len(filteredHistory) > 0 && selectedIndex < len(filteredHistory)-1 {
					selectedIndex++
					refreshDisplay()
				}
			}
		}
	}
}

func highlightText(text, query string) string {
	if query == "" {
		return text
	}

	// Case-insensitive search
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)

	var result strings.Builder
	lastIndex := 0

	for {
		index := strings.Index(lowerText[lastIndex:], lowerQuery)
		if index == -1 {
			break
		}

		// Adjust index to account for the slice
		index += lastIndex

		// Append text before the match
		result.WriteString(text[lastIndex:index])

		// Append the highlighted match
		result.WriteString(colors.BoldYellow)
		result.WriteString(text[index : index+len(query)])
		result.WriteString(colors.Reset)

		// Update lastIndex
		lastIndex = index + len(query)
	}

	// Append the remaining text
	result.WriteString(text[lastIndex:])

	return result.String()
}

// exportVariable sets an environment variable
func (s *SecShell) exportVariable(args []string) {
	if len(args) < 2 {
		drawbox.PrintError("Usage: export VAR=value")
		return
	}

	varValue := s.sanitizeInput(args[1], false)
	equalsPos := strings.Index(varValue, "=")
	if equalsPos == -1 {
		drawbox.PrintError("Invalid export syntax. Use VAR=value")
		return
	}

	varName := varValue[:equalsPos]
	value := varValue[equalsPos+1:]

	if err := os.Setenv(varName, value); err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to set environment variable: %s", err))
	} else {
		drawbox.PrintAlert(fmt.Sprintf("Successfully exported %s=%s", varName, value))
	}
}

// listEnvVariables lists all environment variables
func (s *SecShell) listEnvVariables() {
	drawbox.RunDrawbox("Environment Variables", "bold_white")
	for _, env := range os.Environ() {
		fmt.Println(env)
	}
}

// unsetEnvVariable unsets an environment variable
func (s *SecShell) unsetEnvVariable(args []string) {
	if len(args) < 2 {
		drawbox.PrintError("Usage: unset VAR")
		return
	}

	varName := s.sanitizeInput(args[1], false)
	if err := os.Unsetenv(varName); err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to unset environment variable: %s", err))
	} else {
		drawbox.PrintAlert(fmt.Sprintf("Successfully unset environment variable: %s", varName))
	}
}

// reloadBlacklist reloads the blacklist from the file
func (s *SecShell) reloadBlacklist() {
	s.blacklistedCommands = nil
	s.loadBlacklist(s.blacklist)
	drawbox.PrintAlert("Successfully reloaded blacklist commands")
	if len(s.blacklistedCommands) > 0 {
		drawbox.PrintAlert(fmt.Sprintf("Loaded %d blacklisted commands", len(s.blacklistedCommands)))
	}
}

// editBlacklist opens the blacklist file in an editor
func (s *SecShell) editBlacklist() {
	cmd := exec.Command("nano", s.blacklist)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

// listBlacklistCommands lists all blacklisted commands
func (s *SecShell) listBlacklistCommands() {
	drawbox.RunDrawbox("Blacklisted Commands", "bold_white")
	file, err := os.Open(s.blacklist)
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Error: Could not open file '%s'.", s.blacklist))
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		fmt.Printf(" %d. %s\n", lineNumber, scanner.Text())
	}
}

// reloadWhitelist reloads the whitelist from the file
func (s *SecShell) reloadWhitelist() {
	s.allowedCommands = []string{}
	s.loadWhitelist(s.whitelist)
	drawbox.PrintAlert("Successfully reloaded whitelist commands")
	if len(s.allowedCommands) > 0 {
		drawbox.PrintAlert(fmt.Sprintf("Loaded %d whitelisted commands", len(s.allowedCommands)))
	}
}

// editWhitelist opens the whitelist file in an editor
func (s *SecShell) editWhitelist() {
	cmd := exec.Command("nano", s.whitelist)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

// listWhitelistCommands lists all whitelisted commands
func (s *SecShell) listWhitelistCommands() {
	drawbox.RunDrawbox("Whitelisted Commands", "bold_white")
	for i, cmd := range s.allowedCommands {
		fmt.Printf(" %d. %s\n", i+1, cmd)
	}
}

// processCommand processes and executes a user command
func (s *SecShell) processCommand(input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		drawbox.PrintAlert("Please enter a valid command")
		return
	}

	// Handle history execution with ! prefix
	if strings.HasPrefix(input, "!") {
		if input == "!!" {
			if len(s.history) > 1 { // Ensure there's a valid previous command
				lastCommand := s.history[len(s.history)-2] // Get the second-to-last command
				if lastCommand == "!!" {
					drawbox.PrintError("Cannot execute '!!' recursively")
					return
				}
				drawbox.PrintAlert(fmt.Sprintf("Running: %s", lastCommand))
				s.processCommand(lastCommand) // Execute it safely
			} else {
				drawbox.PrintError("No previous command to execute.")
			}
		} else if num, err := strconv.Atoi(input[1:]); err == nil {
			s.runHistoryCommand(num)
		}
		return
	}

	splitCommands := strings.Split(input, "|")
	for i, command := range splitCommands {
		splitCommands[i] = strings.TrimSpace(command)
	}

	if len(splitCommands) > 1 {
		s.executePipedCommands(splitCommands)
	} else {
		args := strings.Fields(splitCommands[0])
		if len(args) == 0 {
			return
		}

		background := false
		if args[len(args)-1] == "&" {
			background = true
			args = args[:len(args)-1]
		}

		if s.isCommandBlacklisted(args[0]) {
			drawbox.PrintError(fmt.Sprintf("Command is blacklisted: %s", args[0]))
			return
		}

		// Clear the current line before executing the command
		fmt.Print("\r\033[K")

		switch args[0] {
		case "--version":
			update.DisplayVersion(s.versionFile)
		case "--update":
			update.UpdateSecShell(isAdmin(), s.versionFile)
		case "services":
			s.manageServices(args)
		case "jobs":
			s.manageJobs(args)
		case "help":
			if len(args) > 1 {
				help.DisplayHelp(args[1])
			} else {
				help.DisplayHelp()
			}
		case "cd":
			core.ChangeDirectory(args)
		case "time":
			s.getTime()
		case "date":
			s.getDate()
		case "allowed":
			if len(args) > 1 {
				switch args[1] {
				case "dirs":
					drawbox.RunDrawbox("Allowed Directories", "bold_white")
					commands.PrintAllowedDirs()
				case "commands":
					drawbox.RunDrawbox("Allowed Commands", "bold_white")
					commands.PrintAllowedCommands()
				case "bins":
					drawbox.RunDrawbox("Allowed Binaries", "bold_white")
					commands.PrintProgramCommands()
				case "builtins":
					drawbox.RunDrawbox("Built-in Commands", "bold_white")
					commands.PrintBuiltInCommands()
				case "all":
					drawbox.RunDrawbox("All Allowed", "bold_white")
					commands.PrintAllCommands()
				}
			} else {
				drawbox.PrintError("Usage: allowed <dirs|commands|bins|builtins|all>")
			}
		case "history":
			if len(args) == 1 {
				s.displayHistory()
			} else {
				switch args[1] {
				case "-s":
					if len(args) < 3 {
						drawbox.PrintError("Usage: history -s <query>")
						return
					}
					s.searchHistory(strings.Join(args[2:], " ")) // Search history for the given query
				case "-i":
					s.interactiveHistorySearch() // Run interactive history search
				default:
					drawbox.PrintError("Invalid history option. Use -s for search or -i for interactive mode.")
				}
			}
		case "export":
			s.exportVariable(args)
		case "env":
			s.listEnvVariables()
		case "unset":
			s.unsetEnvVariable(args)
		case "blacklist":
			s.listBlacklistCommands()
		case "whitelist":
			s.listWhitelistCommands()
		case "edit-blacklist", "edit-whitelist", "reload-whitelist", "reload-blacklist", "exit":
			// Require admin privileges for these commands
			if !isAdmin() {
				drawbox.PrintError("Permission denied: Admin privileges required.")
				return
			}

			switch args[0] {
			case "edit-blacklist":
				s.editBlacklist()
			case "edit-whitelist":
				s.editWhitelist()
			case "reload-whitelist":
				s.reloadWhitelist()
			case "reload-blacklist":
				s.reloadBlacklist()
			case "exit":
				s.running = false
			}
		case "toggle-security":
			s.toggleSecurity()
		case "download":
			download.DownloadFiles(args)
		default:
			// Handle quoted arguments
			args = s.parseQuotedArgs(args)
			s.executeSystemCommand(args, background)
		}
	}
}

// parseQuotedArgs handles quoted arguments in commands
func (s *SecShell) parseQuotedArgs(args []string) []string {
	var parsedArgs []string
	var currentArg string
	inQuotes := false

	for _, arg := range args {
		if strings.HasPrefix(arg, "\"") && strings.HasSuffix(arg, "\"") {
			parsedArgs = append(parsedArgs, strings.Trim(arg, "\""))
		} else if strings.HasPrefix(arg, "\"") {
			inQuotes = true
			currentArg = strings.TrimPrefix(arg, "\"")
		} else if strings.HasSuffix(arg, "\"") {
			inQuotes = false
			currentArg += " " + strings.TrimSuffix(arg, "\"")
			parsedArgs = append(parsedArgs, currentArg)
			currentArg = ""
		} else if inQuotes {
			currentArg += " " + arg
		} else {
			parsedArgs = append(parsedArgs, arg)
		}
	}

	if inQuotes {
		parsedArgs = append(parsedArgs, currentArg)
	}

	return parsedArgs
}

// manageServices manages system services
func (s *SecShell) manageServices(args []string) {
	if len(args) < 2 {
		drawbox.PrintError("Usage: services <start|stop|restart|status|list> <service_name>")
		return
	}

	action := args[1]
	serviceName := ""
	if len(args) > 2 {
		serviceName = args[2]
	}

	if action != "start" && action != "stop" && action != "restart" && action != "status" && action != "list" && action != "help" {
		drawbox.PrintError("Invalid action. Use start, stop, restart, status, or list.")
		return
	}

	if action == "list" {
		services.GetServices()
	} else if action == "status" {
		services.RunServicesCommand("status", serviceName)
	} else if action == "help" || action == "--help" || action == "-h" {
		services.ShowHelp()
	} else {
		services.RunServicesCommand(action, serviceName)
	}
}

// executePipedCommands executes a series of piped commands
func (s *SecShell) executePipedCommands(commands []string) {
	var cmds []*exec.Cmd
	files := make([]*os.File, 0)

	// Clear the line before executing command
	fmt.Print("\r\033[K")

	for _, command := range commands {
		args := strings.Fields(strings.TrimSpace(command))
		if len(args) == 0 {
			continue
		}

		// Add color flags for supported commands
		switch args[0] {
		case "grep":
			hasColorFlag := false
			for _, arg := range args {
				if strings.HasPrefix(arg, "--color") {
					hasColorFlag = true
					break
				}
			}
			if !hasColorFlag {
				args = append([]string{args[0], "--color=always"}, args[1:]...)
			}
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmds = append(cmds, cmd)
	}

	if len(cmds) == 0 {
		return
	}

	// Set up the pipeline
	for i := 0; i < len(cmds)-1; i++ {
		stdout, err := cmds[i].StdoutPipe()
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to set up pipeline: %s", err))
			return
		}
		cmds[i+1].Stdin = stdout
	}

	// Set up input/output for first and last commands
	cmds[0].Stdin = os.Stdin
	lastCmd := cmds[len(cmds)-1]
	lastCmd.Stdout = os.Stdout
	lastCmd.Stderr = os.Stderr

	// Set up process group for all commands in the pipeline
	for _, cmd := range cmds {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	// Start all commands
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to start command: %s", err))
			return
		}
	}

	// Forward SIGINT to the process group
	go func() {
		for range sigChan {
			for _, cmd := range cmds {
				if cmd.Process != nil {
					syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
				}
			}
		}
	}()

	// Wait for all commands to finish
	for _, cmd := range cmds {
		err := cmd.Wait()
		if err != nil && !isSignalKilled(err) {
			if _, ok := err.(*exec.ExitError); !ok {
				drawbox.PrintError(fmt.Sprintf("Command execution failed: %s", err))
			}
		}
	}

	signal.Stop(sigChan)
	close(sigChan)

	// Close any opened files
	for _, file := range files {
		file.Close()
	}
}

// Determine if command is an editor
func isTerminalEditor(cmd string) bool {
	editors := []string{"nano", "vim", "vi", "emacs", "nvim", "pico"}
	for _, editor := range editors {
		if cmd == editor {
			return true
		}
	}
	return false
}

// executeSystemCommand executes a system command
func (s *SecShell) executeSystemCommand(args []string, background bool) {
	// Sanitize command and arguments
	sanitizedArgs := make([]string, len(args))
	for i, arg := range args {
		if i == 0 {
			sanitizedArgs[i] = sanitize.Command(arg)
		} else {
			sanitizedArgs[i] = sanitize.Path(arg)
		}
	}
	args = sanitizedArgs

	if !s.isCommandAllowed(args[0]) {
		drawbox.PrintError(fmt.Sprintf("Command not permitted: %s", args[0]))
		return
	}

	// Special handling for terminal editors
	if isTerminalEditor(args[0]) {
		// Restore terminal to normal mode
		oldState, err := term.GetState(int(os.Stdin.Fd()))
		if err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to get terminal state: %s", err))
			return
		}
		term.Restore(int(os.Stdin.Fd()), oldState)

		// Execute editor
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Run()

		// After editor exits, clear screen and redraw prompt
		fmt.Print("\033[H\033[2J") // Clear screen
		if err != nil && !isSignalKilled(err) {
			drawbox.PrintError(fmt.Sprintf("Editor execution failed: %s", err))
		}
		return
	}

	cmdArgs := []string{}
	var stdout io.Writer = os.Stdout
	var stdin io.Reader = os.Stdin

	// Special handling for ls and grep commands to add color
	switch args[0] {
	case "ls":
		hasColorFlag := false
		for _, arg := range args[1:] {
			if strings.HasPrefix(arg, "--color") {
				hasColorFlag = true
				break
			}
		}
		if !hasColorFlag {
			cmdArgs = append(cmdArgs, "--color=auto")
		}
	case "grep":
		hasColorFlag := false
		for _, arg := range args[1:] {
			if strings.HasPrefix(arg, "--color") {
				hasColorFlag = true
				break
			}
		}
		if !hasColorFlag {
			cmdArgs = append(cmdArgs, "--color=auto")
		}
	}

	// Handle input/output redirection
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case ">":
			if i+1 < len(args) {
				file, err := os.Create(args[i+1])
				if err != nil {
					drawbox.PrintError(fmt.Sprintf("Failed to create file: %s", err))
					return
				}
				defer file.Close()
				stdout = file
				i++ // Skip the next argument
			}
		case "<":
			if i+1 < len(args) {
				file, err := os.Open(args[i+1])
				if err != nil {
					drawbox.PrintError(fmt.Sprintf("Failed to open file: %s", err))
					return
				}
				defer file.Close()
				stdin = file
				i++ // Skip the next argument
			}
		default:
			cmdArgs = append(cmdArgs, args[i])
		}
	}

	// Expand environment variables for echo command
	if args[0] == "echo" {
		for i, arg := range cmdArgs {
			cmdArgs[i] = os.ExpandEnv(arg)
		}
	}

	// Create command with proper arguments
	cmd := exec.Command(args[0], cmdArgs...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr

	// Create a new process group
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if background {
		if err := cmd.Start(); err != nil {
			drawbox.PrintError(fmt.Sprintf("Failed to start background job: %s", err))
			return
		}
		jobs.AddJob(s.jobs, cmd.Process.Pid, args[0], cmd.Process)

		// Goroutine to wait for the command to finish and update its status
		go func(pid int) {
			err := cmd.Wait()
			exitCode := 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				}
				drawbox.PrintError(fmt.Sprintf("Command execution failed: %s", err))
			}

			// Update job status
			if job, ok := s.jobs[pid]; ok {
				job.Lock()
				job.EndTime = time.Now()
				job.ExitCode = exitCode
				if err != nil {
					job.Status = fmt.Sprintf("failed with code %d", exitCode)
				} else {
					job.Status = "completed"
				}
				job.Unlock()
			}
		}(cmd.Process.Pid)
	} else {
		// Set up signal handling for foreground processes
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT)

		if err := cmd.Start(); err != nil {
			drawbox.PrintError(fmt.Sprintf("Command execution failed: %s", err))
			return
		}

		// Forward SIGINT to the process group
		go func() {
			for range sigChan {
				syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
			}
		}()

		err := cmd.Wait()
		signal.Stop(sigChan)
		close(sigChan)

		if err != nil && !isSignalKilled(err) {
			drawbox.PrintError(fmt.Sprintf("Command execution failed: %s", err))
		}
	}
}

// isCommandAllowed checks if a command is allowed
func (s *SecShell) isCommandAllowed(cmd string) bool {
	// Bypass security checks for built-in commands
	if isAdmin() && !securityEnabled {
		return true // Admins bypass whitelist
	}

	// Sanitize the command first
	cmd = sanitize.Command(cmd)

	// Define a list of restricted network commands
	networkCommands := []string{"wget", "curl", "nc", "nmap", "scp", "rsync"}

	for _, netCmd := range networkCommands {
		if cmd == netCmd {
			drawbox.PrintError("Network access restricted for non-admin users.")
			return false
		}
	}

	// First check if command is in whitelist
	for _, allowedCmd := range s.allowedCommands {
		if cmd == allowedCmd {
			return true
		}
	}

	// Verify the command exists in allowed directories
	for _, dir := range s.allowedDirs {
		path := filepath.Join(dir, cmd)
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}

// isCommandBlacklisted checks if a command is blacklisted
func (s *SecShell) isCommandBlacklisted(cmd string) bool {
	// Bypass security checks for built-in commands
	if isAdmin() && !securityEnabled {
		return false // Admins bypass blacklist
	}

	for _, blacklistedCmd := range s.blacklistedCommands {
		if cmd == blacklistedCmd {
			return true
		}
	}
	return false
}

var securityEnabled = true

// toggleSecurity prompts for a password before allowing admins to toggle security.
func (s *SecShell) toggleSecurity() {
	if !isAdmin() {
		drawbox.PrintError("Permission denied: Only admins can toggle security settings.")
		return
	}

	// Request password authentication
	fmt.Print("Enter your password: ")
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // Move to the next line after password input
	if err != nil {
		drawbox.PrintError("Failed to read password.")
		return
	}

	password := strings.TrimSpace(string(bytePassword))

	// Authenticate the user
	if !authenticateUser(password) {
		drawbox.PrintError("Authentication failed. Incorrect password.")
		return
	}

	// Toggle security state
	securityEnabled = !securityEnabled
	if securityEnabled {
		drawbox.PrintAlert("Security enforcement ENABLED.")
	} else {
		drawbox.PrintAlert("Security enforcement DISABLED. All commands are now allowed.")
	}
}

func authenticateUser(password string) bool {
	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("Error getting current user:", err)
		return false
	}

	// Start a PAM authentication transaction
	transaction, err := pam.StartFunc("login", currentUser.Username, func(s pam.Style, msg string) (string, error) {
		return password, nil
	})
	if err != nil {
		fmt.Println("PAM transaction start failed:", err)
		return false
	}

	// Attempt authentication
	err = transaction.Authenticate(0)
	if err != nil {
		fmt.Println("Authentication failed:", err)
		return false
	}

	return true // Authentication successful
}

// Add this function near the top of the file after the imports
func getExecutablePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "." // Fallback to current directory if home directory cannot be determined
	}
	return filepath.Join(homeDir, ".secshell") // Use ~/.secshell for config files
}

// manageJobs manages background jobs
func (s *SecShell) manageJobs(args []string) {
	if len(args) < 2 {
		jobs.RunJobsCommand("list", 0, s.jobs)
	} else if args[1] == "--help" {
		jobs.ShowHelp()
	} else {
		action := args[1]
		pid := 0
		if len(args) > 2 {
			pidStr := args[2]
			var err error
			pid, err = strconv.Atoi(pidStr)
			if err != nil {
				drawbox.PrintError("Invalid PID. Please enter a valid integer.")
				return
			}
		}
		jobs.RunJobsCommand(action, pid, s.jobs)
	}
}

// Helper function to check if error was due to signal
func isSignalKilled(err error) bool {
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.Signaled() && status.Signal() == syscall.SIGINT
		}
	}
	return false
}

// main function to start the shell
func main() {
	// Check for version flags
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		configDir := getExecutablePath()
		shell := NewSecShell(
			filepath.Join(configDir, ".blacklist"),
			filepath.Join(configDir, ".whitelist"),
		)
		update.DisplayVersion(shell.versionFile)
		return
	}
	// Check for update flag
	if len(os.Args) > 1 && os.Args[1] == "--update" {
		configDir := getExecutablePath()
		shell := NewSecShell(
			filepath.Join(configDir, ".blacklist"),
			filepath.Join(configDir, ".whitelist"),
		)
		update.UpdateSecShell(isAdmin(), shell.versionFile)
		return
	}

	// Create config directory if it doesn't exist
	configDir := getExecutablePath()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Failed to create config directory: %s\n", err)
		return
	}

	blacklistPath := filepath.Join(configDir, ".blacklist")
	whitelistPath := filepath.Join(configDir, ".whitelist")
	shell := NewSecShell(blacklistPath, whitelistPath)
	shell.run()
}

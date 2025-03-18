package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
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
	jobs                map[int]string
	running             bool
	allowedDirs         []string
	allowedCommands     []string
	blacklist           string
	blacklistedCommands []string
	history             []string
	whitelist           string
	historyIndex        int
}

// Define a list of built-in commands
var builtInCommands = []string{
	"help", "exit", "services", "jobs", "cd", "history", "export", "env", "unset",
	"reload-blacklist", "blacklist", "edit-blacklist", "whitelist", "edit-whitelist",
	"reload-whitelist", "download"}

// NewSecShell initializes a new SecShell instance
func NewSecShell(blacklistPath, whitelistPath string) *SecShell {
	shell := &SecShell{
		jobs:        make(map[int]string),
		running:     true,
		allowedDirs: []string{"/usr/bin/", "/bin/", "/opt/"},
		//allowedCommands: []string{"ls", "cd", "pwd", "download"},
		blacklist: blacklistPath,
		whitelist: whitelistPath,
		history:   []string{},
	}
	shell.ensureFilesExist()
	shell.loadBlacklist(blacklistPath)
	shell.loadWhitelist(whitelistPath)
	return shell
}

// ensureFilesExist checks and creates blacklist and whitelist files if they don't exist
func (s *SecShell) ensureFilesExist() {
	// Ensure the config directory exists
	configDir := filepath.Dir(s.blacklist)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		s.printError(fmt.Sprintf("Failed to create config directory: %s", err))
		return
	}

	defaultWhitelistCommands := []string{"ls", "cd", "pwd", "cp", "mv", "rm", "mkdir", "rmdir", "touch", "cat", "echo", "grep", "find", "chmod", "chown", "ps", "kill", "top", "df", "du", "ifconfig", "netstat", "ping", "ip", "clear", "vim", "nano", "emacs", "nvim"}

	// Ensure directory exists
	exePath := getExecutablePath()
	if err := os.MkdirAll(exePath, 0755); err != nil {
		s.printError(fmt.Sprintf("Failed to create directory for config files: %s", err))
		return
	}

	// Create blacklist if it doesn't exist
	if _, err := os.Stat(s.blacklist); os.IsNotExist(err) {
		file, err := os.Create(s.blacklist)
		if err != nil {
			s.printError(fmt.Sprintf("Failed to create blacklist file: %s", err))
		} else {
			file.Close()
			s.printAlert(fmt.Sprintf("Created new blacklist file at %s", s.blacklist))
		}
	}

	// Create/update whitelist if needed
	if _, err := os.Stat(s.whitelist); os.IsNotExist(err) {
		file, err := os.Create(s.whitelist)
		if err != nil {
			s.printError(fmt.Sprintf("Failed to create whitelist file: %s", err))
		} else {
			for _, cmd := range defaultWhitelistCommands {
				file.WriteString(cmd + "\n")
			}
			file.Close()
			s.printAlert(fmt.Sprintf("Created new whitelist file at %s with default commands", s.whitelist))
		}
	} else {
		// Update existing whitelist with any missing default commands
		existingCommands := make(map[string]bool)
		file, err := os.OpenFile(s.whitelist, os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			s.printError(fmt.Sprintf("Failed to open whitelist file: %s", err))
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
}

// loadBlacklist loads blacklisted commands from a file
func (s *SecShell) loadBlacklist(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		s.printError(fmt.Sprintf("Failed to open blacklist file: %s", filename))
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
		s.printAlert(fmt.Sprintf("Notice: No whitelist file found at %s. Using default allowed commands.", filename))
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
		s.printAlert("Warning: Whitelist file is empty. Allowing hard-coded commands and any command within allowed directories.")
		s.allowedCommands = []string{"ls", "cd", "pwd", "cp", "mv", "rm", "mkdir", "rmdir", "touch", "cat", "echo", "grep", "find", "chmod", "chown", "ps", "kill", "top", "df", "du", "ifconfig", "netstat", "ping", "ip", "clear", "vim", "nano", "emacs", "nvim"}
	}
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

// run starts the shell and listens for user input
func (s *SecShell) run() {
	// Display welcome screen
	s.displayWelcomeScreen()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTSTP)

	go func() {
		for sig := range signalChan {
			fmt.Printf("\nReceived signal %d. Use 'exit' to quit. Press ENTER to continue...\n", sig)
		}
	}()

	for s.running {
		input := s.getInput()
		s.processCommand(input)
	}

	// Clear the screen and move cursor to start before exiting
	fmt.Print("\033[H\033[2J")
}

// Change the displayPrompt function:
func (s *SecShell) displayPrompt() {
	user := os.Getenv("USER")
	if user == "" {
		user = "unknown"
	}
	host, _ := os.Hostname()
	cwd, err := os.Getwd()
	if err != nil {
		s.printError("Failed to get current working directory")
		return
	}

	/* fmt.Print(colors.BoldGreen + "┌─[SecShell]" + colors.Reset + " " +
	colors.BoldBlue + "(" + user + ")" + colors.Reset + " " +
	colors.BoldWhite + "[" + cwd + "]" + colors.Reset + "\n" +
	colors.BoldGreen + "└─" + colors.Reset + "$ ") */

	// Background color for the bar
	textReset := colors.Reset
	bgReset := colors.BgReset
	frameColor := colors.BoldGreen
	bgColor := colors.BgGray2
	endCapColor := colors.Gray2   // End caps should match the background
	logoColor := colors.BoldGreen // Text should contrast with the background
	userColor := colors.BoldCyan  // User/host should have a different color
	dirColor := colors.BoldYellow // Directory should have a different color

	// Print top bar with seamless end caps and proper alignment
	fmt.Printf("\n%s╭─%s%s%s [SecShell] %s %s@%s %s%s %s %s%s%s\n",
		frameColor, endCapColor, bgColor, logoColor, userColor, user, host, frameColor, dirColor, cwd, bgReset, endCapColor, textReset)

	// Print bottom input line
	fmt.Printf("%s╰─%s$ %s", frameColor, colors.BoldWhite, textReset)
}

// getInput reads user input from the terminal
func (s *SecShell) getInput() string {
	s.displayPrompt()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		s.printError(fmt.Sprintf("Failed to set terminal to raw mode: %s", err))
		return ""
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	line := ""
	pos := 0
	buf := make([]byte, 1024) // Increased buffer size to handle pasting
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			s.printError(fmt.Sprintf("Failed to read input: %s", err))
			return ""
		}

		// Handle input bytes one by one
		for i := 0; i < n; i++ {
			switch buf[i] {
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
				line, pos = s.completeCommand(line, pos)
			case 13, 10: // Enter (CR or LF)
				fmt.Println()
				input := s.sanitizeInput(strings.TrimSpace(line))
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

// Modify the completeCommand function:
func (s *SecShell) completeCommand(line string, pos int) (string, int) {
	if line == "" {
		return line, pos
	}

	words := strings.Fields(line)
	if len(words) == 0 {
		return line, pos
	}

	lastWord := words[len(words)-1]
	prefix := lastWord

	// Check if we are completing a command or a path
	if len(words) == 1 {
		// Command completion
		matches := s.getCommandMatches(prefix)
		if len(matches) == 0 {
			return line, pos
		}

		// Replace the last word with the first match
		words[len(words)-1] = matches[0]
		newLine := strings.Join(words, " ")
		s.clearLine()
		s.clearLineAndPrintBottom()
		fmt.Print(newLine)

		// If there are multiple matches, show them below
		if len(matches) > 1 {
			for _, match := range matches {
				fmt.Print(match + "  ")
			}
			s.clearLine()
			s.clearLineAndPrintBottom() // Clear line and print the bottom prompt
			fmt.Print(newLine)          // Reprint the new input with the first match
		}

		return newLine, len(newLine)

	} else {
		// Path completion
		matches, _ := filepath.Glob(prefix + "*")
		if len(matches) == 0 {
			return line, pos
		}

		// Replace the last word with the first match
		words[len(words)-1] = matches[0]
		newLine := strings.Join(words, " ")
		s.clearLine()
		s.clearLineAndPrintBottom()
		fmt.Print(newLine)

		// If there are multiple matches, show them below
		if len(matches) > 1 {
			for _, match := range matches {
				fmt.Print(match + "  ")
			}
			s.clearLine()
			s.clearLineAndPrintBottom() // Clear line and print the bottom prompt
			fmt.Print(newLine)          // Reprint the new input with the first match
		}

		return newLine, len(newLine)
	}
}

// Add this new method to get command matches:
func (s *SecShell) getCommandMatches(prefix string) []string {
	var matches []string
	for _, cmd := range s.allowedCommands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}

	// Include built-in commands
	for _, cmd := range builtInCommands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

// sanitizeInput removes forbidden characters from input
func (s *SecShell) sanitizeInput(input string, allowSpecialChars ...bool) string {
	allow := true
	if len(allowSpecialChars) > 0 {
		allow = allowSpecialChars[0]
	}

	forbidden := ";`\\"
	if !allow {
		forbidden += "&|><"
	}

	for _, char := range forbidden {
		input = strings.ReplaceAll(input, string(char), "")
	}
	return input
}

// changeDirectory changes the current working directory
func (s *SecShell) changeDirectory(args []string) {
	var dir string
	if len(args) < 2 {
		home := os.Getenv("HOME")
		if home == "" {
			s.printError("cd failed: HOME environment variable not set")
			return
		}
		dir = home
	} else {
		dir = args[1]
	}

	if err := os.Chdir(dir); err != nil {
		s.printError(fmt.Sprintf("cd failed: %s", err))
	}
}

// displayHistory shows the command history
func (s *SecShell) displayHistory() {
	s.runDrawbox("Command History", "bold_white")
	for i, cmd := range s.history {
		fmt.Printf("%d: %s\n", i+1, cmd)
	}
}

func (s *SecShell) searchHistory(query string) {
	s.runDrawbox("History Search: "+query, "bold_white")
	found := false

	for i, cmd := range s.history {
		if strings.Contains(strings.ToLower(cmd), strings.ToLower(query)) {
			highlightedCmd := highlightText(cmd, query)
			fmt.Printf("%d: %s\n", i+1, highlightedCmd)
			found = true
		}
	}

	if !found {
		s.printAlert("No matching commands found.")
	}
}

func (s *SecShell) runHistoryCommand(number int) bool {
	if number <= 0 || number > len(s.history) {
		s.printError(fmt.Sprintf("Invalid history number: %d", number))
		return false
	}

	cmd := s.history[number-1]
	s.printAlert(fmt.Sprintf("Running: %s", cmd))
	s.processCommand(cmd)
	return true
}

func (s *SecShell) interactiveHistorySearch() {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		s.printError(fmt.Sprintf("Failed to set terminal to raw mode: %s", err))
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
		// Clear screen and scrollback buffer
		fmt.Print("\033[3J\033[H\033[2J") // Clear screen and scrollback buffer

		// Display header with drawbox
		fmt.Print("\n")
		s.clearLine()
		fmt.Print(colors.BoldGreen + "┌─[Interactive History Search]" + colors.Reset + "\n")
		s.clearLine()
		fmt.Printf(colors.BoldGreen+"└─"+colors.Reset+"$ %s", query)

		// Print instructions
		fmt.Print("\n")
		s.clearLine()
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
				s.clearLine()
				fmt.Printf("%s→ %d: %s%s\n", colors.BoldGreen, i+1, cmd, colors.Reset)
			} else {
				s.clearLine()
				fmt.Printf("  %d: %s\n", i+1, cmd)
			}
		}

		if len(filteredHistory) == 0 {
			s.clearLine()
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
			s.printError(fmt.Sprintf("Failed to read input: %s", err))
			return
		}

		if n == 1 {
			switch buf[0] {
			case 27: // ESC
				// Clear screen before returning to normal mode
				fmt.Print("\033[3J\033[H\033[2J") // Clear screen and scrollback buffer
				return

			case 13: // Enter
				if len(filteredHistory) > 0 && selectedIndex >= 0 && selectedIndex < len(filteredHistory) {
					selectedCmd := filteredHistory[selectedIndex]
					// Restore terminal and run command
					fmt.Print("\033[?25h") // Make sure cursor is visible
					term.Restore(int(os.Stdin.Fd()), oldState)
					// Clear screen and scrollback buffer
					fmt.Print("\033[3J\033[H\033[2J")
					s.printAlert("Running: " + selectedCmd)
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
		s.printError("Usage: export VAR=value")
		return
	}

	varValue := s.sanitizeInput(args[1], false)
	equalsPos := strings.Index(varValue, "=")
	if equalsPos == -1 {
		s.printError("Invalid export syntax. Use VAR=value")
		return
	}

	varName := varValue[:equalsPos]
	value := varValue[equalsPos+1:]

	if err := os.Setenv(varName, value); err != nil {
		s.printError(fmt.Sprintf("Failed to set environment variable: %s", err))
	} else {
		s.printAlert(fmt.Sprintf("Successfully exported %s=%s", varName, value))
	}
}

// listEnvVariables lists all environment variables
func (s *SecShell) listEnvVariables() {
	s.runDrawbox("Environment Variables", "bold_white")
	for _, env := range os.Environ() {
		fmt.Println(env)
	}
}

// unsetEnvVariable unsets an environment variable
func (s *SecShell) unsetEnvVariable(args []string) {
	if len(args) < 2 {
		s.printError("Usage: unset VAR")
		return
	}

	varName := s.sanitizeInput(args[1], false)
	if err := os.Unsetenv(varName); err != nil {
		s.printError(fmt.Sprintf("Failed to unset environment variable: %s", err))
	} else {
		s.printAlert(fmt.Sprintf("Successfully unset environment variable: %s", varName))
	}
}

// reloadBlacklist reloads the blacklist from the file
func (s *SecShell) reloadBlacklist() {
	s.blacklistedCommands = nil
	s.loadBlacklist(s.blacklist)
	s.printAlert("Successfully reloaded blacklist commands")
	if len(s.blacklistedCommands) > 0 {
		s.printAlert(fmt.Sprintf("Loaded %d blacklisted commands", len(s.blacklistedCommands)))
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
	s.runDrawbox("Blacklisted Commands", "bold_white")
	file, err := os.Open(s.blacklist)
	if err != nil {
		s.printError(fmt.Sprintf("Error: Could not open file '%s'.", s.blacklist))
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
	s.printAlert("Successfully reloaded whitelist commands")
	if len(s.allowedCommands) > 0 {
		s.printAlert(fmt.Sprintf("Loaded %d whitelisted commands", len(s.allowedCommands)))
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
	s.runDrawbox("Whitelisted Commands", "bold_white")
	for i, cmd := range s.allowedCommands {
		fmt.Printf(" %d. %s\n", i+1, cmd)
	}
}

// processCommand processes and executes a user command
func (s *SecShell) processCommand(input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		s.printAlert("Please enter a valid command")
		return
	}

	// Handle history execution with ! prefix
	if strings.HasPrefix(input, "!") {
		if input == "!!" {
			if len(s.history) > 1 { // Ensure there's a valid previous command
				lastCommand := s.history[len(s.history)-2] // Get the second-to-last command
				s.printAlert(fmt.Sprintf("Running: %s", lastCommand))
				s.processCommand(lastCommand) // Execute it safely
			} else {
				s.printError("No previous command to execute.")
			}
		} else if num, err := strconv.Atoi(input[1:]); err == nil {
			s.runHistoryCommand(num)
		}
		return
	}

	commands := strings.Split(input, "|")
	for i, command := range commands {
		commands[i] = strings.TrimSpace(command)
	}

	if len(commands) > 1 {
		s.executePipedCommands(commands)
	} else {
		args := strings.Fields(commands[0])
		if len(args) == 0 {
			return
		}

		background := false
		if args[len(args)-1] == "&" {
			background = true
			args = args[:len(args)-1]
		}

		if s.isCommandBlacklisted(args[0]) {
			s.printError(fmt.Sprintf("Command is blacklisted: %s", args[0]))
			return
		}

		// Clear the current line before executing the command
		fmt.Print("\r\033[K")

		switch args[0] {
		case "services":
			s.manageServices(args)
		case "jobs":
			s.listJobs()
		case "help":
			s.displayHelp()
		case "cd":
			s.changeDirectory(args)
		case "history":
			if len(args) == 1 {
				s.displayHistory()
			} else {
				switch args[1] {
				case "-s":
					if len(args) < 3 {
						s.printError("Usage: history -s <query>")
						return
					}
					s.searchHistory(strings.Join(args[2:], " ")) // Search history for the given query
				case "-i":
					s.interactiveHistorySearch() // Run interactive history search
				default:
					s.printError("Invalid history option. Use -s for search or -i for interactive mode.")
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
				s.printError("Permission denied: Admin privileges required.")
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
			if len(args) != 2 {
				s.printError("Usage: download <url>")
				return
			}
			s.downloadFile(args[1])
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
		s.printError("Usage: services <start|stop|restart|status|list> <service_name>")
		return
	}

	action := args[1]
	serviceName := ""
	if len(args) > 2 {
		serviceName = args[2]
	}

	if action != "start" && action != "stop" && action != "restart" && action != "status" && action != "list" {
		s.printError("Invalid action. Use start, stop, restart, status, or list.")
		return
	}

	var command string
	if action == "list" {
		command = "systemctl list-units --type=service"
	} else if action == "status" {
		command = "systemctl status " + serviceName
	} else {
		command = "sudo systemctl " + action + " " + serviceName
	}

	s.runDrawbox("Service Manager", "bold_white")
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		s.printError("Failed to execute service command.")
	} else {
		s.printAlert("Service command executed successfully.")
	}
}

// listJobs lists all active background jobs
func (s *SecShell) listJobs() {
	s.runDrawbox("Jobs", "bold_white")
	fmt.Println("Active Jobs:")
	for pid, job := range s.jobs {
		fmt.Printf("PID: %d - %s\n", pid, job)
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
			s.printError(fmt.Sprintf("Failed to set up pipeline: %s", err))
			return
		}
		cmds[i+1].Stdin = stdout
	}

	// Set up input/output for first and last commands
	cmds[0].Stdin = os.Stdin
	lastCmd := cmds[len(cmds)-1]
	lastCmd.Stdout = os.Stdout
	lastCmd.Stderr = os.Stderr

	// Start all commands
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			s.printError(fmt.Sprintf("Failed to start command: %s", err))
			return
		}
	}

	// Wait for all commands to finish
	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				s.printError(fmt.Sprintf("Command execution failed: %s", err))
			}
		}
	}

	// Close any opened files
	for _, file := range files {
		file.Close()
	}
}

// executeSystemCommand executes a system command
func (s *SecShell) executeSystemCommand(args []string, background bool) {
	if !s.isCommandAllowed(args[0]) {
		s.printError(fmt.Sprintf("Command not permitted: %s", args[0]))
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
					s.printError(fmt.Sprintf("Failed to create file: %s", err))
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
					s.printError(fmt.Sprintf("Failed to open file: %s", err))
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

	// Create command with proper arguments
	cmd := exec.Command(args[0], cmdArgs...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr

	if background {
		if err := cmd.Start(); err != nil {
			s.printError(fmt.Sprintf("Failed to start background job: %s", err))
			return
		}
		s.jobs[cmd.Process.Pid] = args[0]
		s.printAlert(fmt.Sprintf("[%d] %s running in background", cmd.Process.Pid, args[0]))
	} else {
		if err := cmd.Run(); err != nil {
			s.printError(fmt.Sprintf("Command execution failed: %s", err))
		}
	}
}

// displayHelp shows the help message
func (s *SecShell) displayHelp() {
	s.runDrawbox("SecShell Help", "bold_white")
	fmt.Fprintf(os.Stdout, `
Built-in Commands:
  %shelp%s       - Show this help message
  %sexit%s       - Exit the shell
  %sservices%s   - Manage system services
               		Usage: services <start|stop|restart|status|list> <service_name>

  %sjobs%s       - List active background jobs
  %scd%s         - Change directory
               		Usage: cd [directory]

  %shistory%s    - Show command history
  			Usage: history [-s <query>] [-i]
			   -s: Search history for a query
			   -i: Interactive history search
			   ![number]: Execute command by number
			   !!: Execute last command

  %sexport%s     - Set an environment variable
               		Usage: export VAR=value

  %senv%s        - List all environment variables
  %sunset%s      - Unset an environment variable
               		Usage: unset VAR

  %sblacklist%s  - List blacklisted commands
  %swhitelist%s  - List whitelisted commands
  %sedit-blacklist%s - Edit the blacklist file
  %sedit-whitelist%s - Edit the whitelist file
  %sreload-blacklist%s - Reload the blacklisted commands
  %sreload-whitelist%s - Reload the whitelisted commands

  %sdownload%s    - Download a file from URL
               		Usage: download <url>

%sAllowed System Commands:%s
  ls, ps, netstat, tcpdump, cd, clear, ifconfig

%sSecurity Features:%s
  - Command whitelisting
  - Input sanitization
  - Process isolation
  - Job tracking
  - Service Management
  - Background job execution
  - Piped command execution
  - Input/output redirection

%sExamples:%s
  > ls -l
  > jobs
  > services list
  > export MY_VAR=value
  > env
  > unset MY_VAR
  > history
  > blacklist
  > edit-blacklist
  > reload-blacklist
  > whitelist
  > edit-whitelist
  > reload-whitelist
  > exit

%sNote:%s
All commands are subject to security checks and sanitization.
Only executables from trusted directories are permitted.
`,
		colors.BoldWhite, colors.Reset, // help
		colors.BoldWhite, colors.Reset, // exit
		colors.BoldWhite, colors.Reset, // services
		colors.BoldWhite, colors.Reset, // jobs
		colors.BoldWhite, colors.Reset, // cd
		colors.BoldWhite, colors.Reset, // history
		colors.BoldWhite, colors.Reset, // export
		colors.BoldWhite, colors.Reset, // env
		colors.BoldWhite, colors.Reset, // unset
		colors.BoldWhite, colors.Reset, // blacklist
		colors.BoldWhite, colors.Reset, // whitelist
		colors.BoldWhite, colors.Reset, // edit-blacklist
		colors.BoldWhite, colors.Reset, // edit-whitelist
		colors.BoldWhite, colors.Reset, // reload-blacklist
		colors.BoldWhite, colors.Reset, // reload-whitelist
		colors.BoldWhite, colors.Reset, // download
		colors.Cyan, colors.Reset, // Allowed System Commands
		colors.Cyan, colors.Reset, // Security Features
		colors.Cyan, colors.Reset, // Examples
		colors.Cyan, colors.Reset, // Note
	)
}

// isCommandAllowed checks if a command is allowed
func (s *SecShell) isCommandAllowed(cmd string) bool {
	// Bypass security checks for built-in commands
	if isAdmin() && !securityEnabled {
		return true // Admins bypass whitelist
	}

	// Define a list of restricted network commands
	networkCommands := []string{"wget", "curl", "nc", "nmap", "scp", "rsync"}

	for _, netCmd := range networkCommands {
		if cmd == netCmd {
			s.printError("Network access restricted for non-admin users.")
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
		s.printError("Permission denied: Only admins can toggle security settings.")
		return
	}

	// Request password authentication
	fmt.Print("Enter your password: ")
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // Move to the next line after password input
	if err != nil {
		s.printError("Failed to read password.")
		return
	}

	password := strings.TrimSpace(string(bytePassword))

	// Authenticate the user
	if !authenticateUser(password) {
		s.printError("Authentication failed. Incorrect password.")
		return
	}

	// Toggle security state
	securityEnabled = !securityEnabled
	if securityEnabled {
		s.printAlert("Security enforcement ENABLED.")
	} else {
		s.printAlert("Security enforcement DISABLED. All commands are now allowed.")
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

// runDrawbox runs the drawbox command to display a message box
func (s *SecShell) runDrawbox(title, color string) {
	fmt.Print("\n") // Add newline before box

	// Use exec.LookPath to find the drawbox executable in the PATH
	drawboxPath, err := exec.LookPath("drawbox")
	if err != nil {
		// If drawbox is not found, fallback to the custom box drawing
		fmt.Fprintf(os.Stdout, "%s╔══%s %s %s══╗%s\n",
			colors.BoldWhite, colors.Reset, title, colors.BoldWhite, colors.Reset)
		return
	}

	// Execute the drawbox command
	cmd := exec.Command(drawboxPath, title, color)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Fallback if drawbox fails
		fmt.Fprintf(os.Stdout, "%s╔══%s %s %s══╗%s\n",
			colors.BoldWhite, colors.Reset, title, colors.BoldWhite, colors.Reset)
	}
}

// printAlert prints an alert message
func (s *SecShell) printAlert(message string) {
	s.runDrawbox("ALERT: "+message, "bold_yellow")
}

// printError prints an error message
func (s *SecShell) printError(message string) {
	s.runDrawbox("ERROR: "+message, "bold_red")
}

// Add this function near the top of the file after the imports
func getExecutablePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "." // Fallback to current directory if home directory cannot be determined
	}
	return filepath.Join(homeDir, ".secshell") // Use ~/.secshell for config files
}

// Add this after the printError method and before main:
func (s *SecShell) displayWelcomeScreen() {
	// Clear the screen first
	fmt.Print("\033[H\033[2J")

	// ASCII art logo
	logo := `
    ███████╗███████╗ ██████╗███████╗██╗  ██╗███████╗██╗     ██╗     
    ██╔════╝██╔════╝██╔════╝██╔════╝██║  ██║██╔════╝██║     ██║     
    ███████╗█████╗  ██║     ███████╗███████║█████╗  ██║     ██║     
    ╚════██║██╔══╝  ██║     ╚════██║██╔══██║██╔══╝  ██║     ██║     
    ███████║███████╗╚██████╗███████║██║  ██║███████╗███████╗███████╗
    ╚══════╝╚══════╝ ╚═════╝╚══════╝╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝
    `

	fmt.Printf("%s%s%s\n", colors.BoldYellow, logo, colors.Reset)

	s.runDrawbox("Welcome to SecShell - A Secure Command Shell", "bold_green")
	fmt.Printf("\n%sFeatures:%s\n", colors.BoldWhite, colors.Reset)
	features := []string{
		"✓ Command whitelisting and blacklisting",
		"✓ Secure input handling",
		"✓ Process isolation",
		"✓ Service management",
		"✓ Background job support",
		"✓ Command history",
		"✓ Tab completion",
	}

	for _, feature := range features {
		fmt.Printf("  %s%s%s\n", colors.BoldGreen, feature, colors.Reset)
	}

	fmt.Printf("\n%sType 'help' for available commands%s\n\n", colors.BoldCyan, colors.Reset)
}

// Add this new method:
func (s *SecShell) downloadFile(url string) {
	// Sanitize URL
	url = s.sanitizeInput(url, false)

	// Extract filename from URL
	fileName := filepath.Base(url)
	if fileName == "" || fileName == "." {
		fileName = "downloaded_file"
	}

	// Sanitize filename
	fileName = s.sanitizeInput(fileName, false)

	// Create the file
	out, err := os.Create(fileName)
	if err != nil {
		s.printError(fmt.Sprintf("Error creating file: %v", err))
		return
	}
	defer out.Close()

	// Get the data
	s.printAlert(fmt.Sprintf("Downloading %s...", url))
	resp, err := http.Get(url)
	if err != nil {
		s.printError(fmt.Sprintf("Error downloading file: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.printError(fmt.Sprintf("Bad status: %s", resp.Status))
		return
	}

	// Create progress bar
	size := resp.ContentLength
	progress := 0
	startTime := time.Now()

	// Create counter proxy reader with proper initialization
	counter := &WriteCounter{
		Total:      size,
		Downloaded: 0,
		progress:   &progress,
		shell:      s,
	}

	// Copy data with progress updates
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		s.printError(fmt.Sprintf("Error saving file: %v", err))
		return
	}

	// Calculate download speed
	duration := time.Since(startTime).Seconds()
	speed := float64(size) / duration / 1024 / 1024 // MB/s

	fmt.Print("\r\033[K") // Clear progress line
	s.printAlert(fmt.Sprintf("Downloaded %s (%.2f MB/s)", fileName, speed))
}

// Replace the WriteCounter struct and its Write method:
type WriteCounter struct {
	Total      int64
	Downloaded int64
	progress   *int
	shell      *SecShell
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Downloaded += int64(n)

	// Calculate percentage
	if wc.Total > 0 {
		percentage := float64(wc.Downloaded) / float64(wc.Total) * 100
		newProgress := int(percentage)

		// Only update if progress has changed
		if newProgress != *wc.progress {
			*wc.progress = newProgress

			// Clear line and show progress
			fmt.Printf("\r\033[K[")
			for i := 0; i < 50; i++ {
				if i < *wc.progress/2 {
					fmt.Print("=")
				} else {
					fmt.Print(" ")
				}
			}
			fmt.Printf("] %d%%", *wc.progress)
		}
	}
	return n, nil
}

// Print only top:
func (s *SecShell) clearLine() {
	// Clear the entire current line and return carriage
	fmt.Print("\033[2K\r")

}

// Add this new method:
func (s *SecShell) clearLineAndPrintBottom() {
	// Clear the entire current line and return carriage
	fmt.Print("\033[2K\r")
	// Print only the bottom prompt exactly as defined
	fmt.Print(colors.BoldGreen + "└─" + colors.Reset + "$ ")
}

// main function to start the shell
func main() {
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

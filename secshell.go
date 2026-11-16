package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"secshell/admin"
	"secshell/auth"
	"secshell/cmdmap"
	"secshell/colors"
	"secshell/core"
	"secshell/globals"
	"secshell/history"
	"secshell/jobs"
	"secshell/logging"
	"secshell/services"
	"secshell/tools"
	"secshell/tools/editor"
	"secshell/ui"
	"secshell/ui/gui"
	"secshell/update"

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
	historyFile         string
	historyIndex        int
	continuationMode    bool // true while reading a PS2 continuation line
}

// NewSecShell initializes a new SecShell instance
func NewSecShell(blacklistPath, whitelistPath string) *SecShell {
	shell := &SecShell{
		jobs:        make(map[int]*jobs.Job),
		running:     true,
		allowedDirs: globals.TrustedDirs,
		blacklist:   globals.BlacklistPath,
		whitelist:   globals.WhitelistPath,
		versionFile: globals.VersionPath,
		historyFile: globals.HistoryPath,
		history:     []string{},
	}
	core.EnsureFilesExist(globals.BlacklistPath, globals.WhitelistPath, globals.VersionPath, globals.HistoryPath, logging.LogFile)
	core.LoadBlacklist(globals.BlacklistPath)
	core.LoadWhitelist(globals.WhitelistPath)
	core.LoadHistory(globals.HistoryPath)

	shell.blacklistedCommands = core.BlacklistedCommands
	shell.allowedCommands = core.AllowedCommands
	shell.history = history.GetHistoryFromFile(globals.HistoryPath)
	shell.historyIndex = len(shell.history)

	// Log that we're loading blacklisted commands
	if len(core.BlacklistedCommands) > 0 {
		logging.LogAlert(fmt.Sprintf("Loaded %d blacklisted commands", len(core.BlacklistedCommands)))
	}

	// Update command map to remove blacklisted commands
	for _, cmd := range core.BlacklistedCommands {
		if _, exists := cmdmap.GlobalCommandMap[cmd]; exists {
			logging.LogAlert(fmt.Sprintf("Removing blacklisted command from command map: %s", cmd))
			delete(cmdmap.GlobalCommandMap, cmd)
		}
	}

	// Log the state of restricted commands
	restrictedCount := len(globals.RestrictedCommands)
	if restrictedCount > 0 {
		logging.LogAlert(fmt.Sprintf("Loaded %d admin-restricted commands", restrictedCount))
	}

	return shell
}

// Get current Time
func (s *SecShell) getTime() {
	now := time.Now()
	gui.TitleBox(fmt.Sprintf("Current time: %s", now.Format("3:04 PM")))
}

// Get current Date
func (s *SecShell) getDate() {
	now := time.Now()
	gui.TitleBox(fmt.Sprintf("Current date: %s", now.Format("02-Jan-2006")))

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
			logging.LogError(err)
			return false
		}
		latest[i], err = strconv.Atoi(latestParts[i])
		if err != nil {
			logging.LogError(err)
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
		// If the line is an unfinished compound command, quote, or ends in a
		// connector, keep reading continuation lines (PS2) until it is complete.
		for s.running && strings.TrimSpace(input) != "" && cmdmap.IsIncompleteCommand(input) {
			s.continuationMode = true
			more := s.getInput()
			s.continuationMode = false
			input = input + "\n" + more
		}
		s.processCommand(input)
	}

	// Clear the screen and move cursor to start before exiting
	fmt.Print("\033[H\033[2J")
}

// getInput reads user input from the terminal
func (s *SecShell) getInput() string {
	if s.continuationMode {
		fmt.Print("> ")
	} else {
		ui.DisplayPrompt()
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Failed to set terminal to raw mode: %s", err))
		return ""
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	line := ""
	pos := 0
	buf := make([]byte, 8192) // Increased buffer size to handle pasting

	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to read input: %s", err))
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
							if len(s.history) > 0 && s.historyIndex > 0 {
								s.historyIndex--
								newLine := strings.TrimSpace(s.history[s.historyIndex])
								fmt.Printf("\x1b[%dD\x1b[K%s", pos, newLine)
								line = newLine
								pos = len(line)
							}
							i += 2
						case 'B': // Down arrow
							if len(s.history) > 0 && s.historyIndex < len(s.history)-1 {
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
				newLine, newPos := cmdmap.CompleteCommand(line, pos)
				if newLine != line {
					// The completion function already handles clearing and reprinting
					line = newLine
					pos = newPos
				}
			case 13, 10: // Enter (CR or LF)
				fmt.Println()
				// The command line is passed through verbatim; the shell parser and
				// expansion stage interpret quoting and metacharacters (parity-first).
				// Sanitizing here would silently corrupt valid syntax such as
				// backticks, $(...), and quotes.
				input := strings.TrimSpace(line)
				if input != "" {
					logging.LogCommand(input, 0)
					core.SaveHistory(s.historyFile, input)
					s.history = core.History
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

// reloadBlacklist reloads the blacklist from the file
func (s *SecShell) reloadBlacklist() {
	s.blacklistedCommands = nil
	core.LoadBlacklist(s.blacklist)
	s.blacklistedCommands = core.BlacklistedCommands
	logging.LogAlert("Successfully reloaded blacklist commands")
	gui.AlertBox("Successfully reloaded blacklist commands")
	if len(s.blacklistedCommands) > 0 {
		gui.AlertBox(fmt.Sprintf("Loaded %d blacklisted commands", len(s.blacklistedCommands)))
	}
}

// reloadWhitelist reloads the whitelist from the file
func (s *SecShell) reloadWhitelist() {
	s.allowedCommands = []string{}
	core.LoadWhitelist(s.whitelist)
	s.allowedCommands = core.AllowedCommands
	logging.LogAlert("Successfully reloaded whitelist commands")
	gui.AlertBox("Successfully reloaded whitelist commands")
	if len(s.allowedCommands) > 0 {
		gui.AlertBox(fmt.Sprintf("Loaded %d whitelisted commands", len(s.allowedCommands)))
	}
}

// processCommand processes and executes a user command
func (s *SecShell) processCommand(input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		gui.AlertBox("Please enter a valid command")
		return
	}

	// Handle history execution with ! prefix
	if strings.HasPrefix(input, "!") {
		if input == "!!" {
			if len(s.history) > 1 { // Ensure there's a valid previous command
				lastCommand := s.history[len(s.history)-2] // Get the second-to-last command
				if lastCommand == "!!" {
					logging.LogAlert("Cannot execute '!!' recursively.")
					gui.ErrorBox("Cannot execute '!!' recursively")
					return
				}
				gui.AlertBox(fmt.Sprintf("Running: %s", lastCommand))
				s.processCommand(lastCommand) // Execute it safely
			} else {
				logging.LogAlert("No previous command to execute.")
				gui.ErrorBox("No previous command to execute.")
			}
		} else if num, err := strconv.Atoi(input[1:]); err == nil {
			logging.LogError(err)
			history.RunHistoryCommand(s.history, num, s.processCommand)
		}
		return
	}

	// Check for script execution (files with ./ prefix)
	if strings.HasPrefix(input, "./") {
		// Extract the command name from the input to check if it's blacklisted
		parts := strings.Fields(input)
		if len(parts) > 0 {
			scriptCmd := parts[0]
			// Only check blacklist if security is enabled or user is not admin
			if s.isCommandBlacklisted(scriptCmd) {
				logging.LogAlert(fmt.Sprintf("Script execution denied: '%s' is blacklisted", scriptCmd))
				gui.ErrorBox(fmt.Sprintf("Script execution denied: '%s' is blacklisted", scriptCmd))
				return
			}
		}

		err := tools.ExecuteScript(input)
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Script execution failed: %s", err))
			return
		}
		return
	}

	// Handle command with our command mapping system
	cmdmap.ExecuteCommand(input, s.jobs)
}

// isCommandBlacklisted checks if a command is blacklisted
func (s *SecShell) isCommandBlacklisted(cmd string) bool {
	// If security is disabled and user is admin, bypass the blacklist
	if admin.IsAdmin() && !securityEnabled {
		return false // Admins bypass blacklist when security is disabled
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
	if !admin.IsAdmin() {
		logging.LogAlert("Permission denied: Only admins can toggle security settings.")
		gui.ErrorBox("Permission denied: Only admins can toggle security settings.")
		return
	}

	// Request password authentication
	fmt.Print("Enter your password: ")
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // Move to the next line after password input
	if err != nil {
		logging.LogError(err)
		gui.ErrorBox("Failed to read password.")
		return
	}

	password := strings.TrimSpace(string(bytePassword))

	// Authenticate the user
	if !auth.AuthenticateUser(password) {
		logging.LogAlert("Authentication failed. Incorrect password.")
		gui.ErrorBox("Authentication failed. Incorrect password.")
		return
	}

	// Toggle security state
	securityEnabled = !securityEnabled
	// Update the security flag in the cmdmap package
	cmdmap.SetSecurityEnabled(securityEnabled)

	if securityEnabled {
		logging.LogAlert("Security enforcement ENABLED.")
		gui.AlertBox("Security enforcement ENABLED.")
	} else {
		logging.LogAlert("Security enforcement DISABLED. All commands are now allowed for admin users.")
		gui.AlertBox("Security enforcement DISABLED. Admin user can now execute ANY command including blacklisted ones.")
		// Print the state of the security flag in cmdmap for debugging
		fmt.Printf("Security state in cmdmap: %v\n", cmdmap.GetSecurityEnabledFlag())
	}
}

// manageServices manages system services
func (s *SecShell) manageServices(args []string) {
	if len(args) < 2 {
		services.RunServicesCommand("list", "")
	} else if args[1] == "--help" {
		services.ShowHelp()
	} else {
		action := args[1]
		serviceName := ""
		if len(args) > 2 {
			serviceName = args[2]
		}
		services.RunServicesCommand(action, serviceName)
	}
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
				logging.LogError(err)
				gui.ErrorBox("Invalid PID. Please enter a valid integer.")
				return
			}
		}
		jobs.RunJobsCommand(action, pid, s.jobs)
	}
}

// Helper function to check if error was due to signal
func isSignalKilled(err error) bool {
	logging.LogError(err)
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.Signaled() && status.Signal() == syscall.SIGINT
		}
	}
	return false
}

// main function to start the shell
func main() {

	// Set language environment variable for consistent character set handling
	os.Setenv("LANG", "en_US.UTF-8")

	// Check for version flags
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		shell := NewSecShell(globals.BlacklistPath, globals.WhitelistPath)
		update.DisplayVersion(shell.versionFile)
		return
	}
	// Check for update flag
	if len(os.Args) > 1 && os.Args[1] == "--update" {
		shell := NewSecShell(globals.BlacklistPath, globals.WhitelistPath)
		update.UpdateSecShell(admin.IsAdmin(), shell.versionFile)
		return
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(globals.ConfigDir, 0755); err != nil {
		logging.LogError(err)
		fmt.Printf("Failed to create config directory: %s\n", err)
		return
	}

	// Initialize command mapping system
	cmdmap.InitCommandMap()

	// Initialize security state in cmdmap package
	cmdmap.SetSecurityEnabled(securityEnabled)

	// Register command handlers
	cmdmap.RegisterBuiltInCommandHandlers()

	// Create shell instance
	shell := NewSecShell(globals.BlacklistPath, globals.WhitelistPath)

	// Register shell-specific handlers
	registerShellHandlers(shell)

	// Run the shell
	shell.run()
}

// registerShellHandlers registers handlers for commands that need direct access to the shell state
func registerShellHandlers(shell *SecShell) {
	// Register exit handler
	exitHandler := func(args []string) (int, error) {
		// Set the running flag to false to exit the shell
		shell.running = false
		return 0, nil
	}

	// Register reload-blacklist handler
	reloadBlacklistHandler := func(args []string) (int, error) {
		if !admin.IsAdmin() {
			logging.LogAlert("Permission denied: Admin privileges required.")
			gui.ErrorBox("Permission denied: Admin privileges required.")
			return 1, fmt.Errorf("permission denied")
		}
		shell.reloadBlacklist()
		return 0, nil
	}

	// Register reload-whitelist handler
	reloadWhitelistHandler := func(args []string) (int, error) {
		if !admin.IsAdmin() {
			logging.LogAlert("Permission denied: Admin privileges required.")
			gui.ErrorBox("Permission denied: Admin privileges required.")
			return 1, fmt.Errorf("permission denied")
		}
		shell.reloadWhitelist()
		return 0, nil
	}

	// Register toggle-security handler
	toggleSecurityHandler := func(args []string) (int, error) {
		shell.toggleSecurity()
		return 0, nil
	}

	// Register jobs handler
	jobsHandler := func(args []string) (int, error) {
		if len(args) > 1 {
			if args[1] == "-i" || args[1] == "--interactive" {
				jobs.InteractiveJobManager(shell.jobs)
			}
		} else {
			shell.manageJobs(args)
		}
		return 0, nil
	}

	// Register history handler
	historyHandler := func(args []string) (int, error) {
		if len(args) == 1 {
			history.DisplayHistory(shell.history) // Display command history
		} else {
			switch args[1] {
			case "-s":
				if len(args) < 3 {
					logging.LogAlert("Usage: history -s <query>")
					gui.ErrorBox("Usage: history -s <query>")
					return 1, fmt.Errorf("invalid usage")
				}
				history.SearchHistory(shell.history, strings.Join(args[2:], " ")) // Search history for the given query
			case "-i":
				history.InteractiveHistorySearch(shell.history, shell.processCommand) // Run interactive history search
			case "clear":
				shell.history = []string{}
				core.ClearHistory(shell.historyFile)
			default:
				logging.LogAlert("Invalid history option. Use -s for search or -i for interactive mode.")
				gui.ErrorBox("Invalid history option. Use -s for search or -i for interactive mode.")
				return 1, fmt.Errorf("invalid option")
			}
		}
		return 0, nil
	}

	// Register edit-prompt handler
	editPromptHandler := func(args []string) (int, error) {
		editor.EditCommand([]string{globals.PromptConfigFile})
		return 0, nil
	}

	// Register reload-prompt handler
	reloadPromptHandler := func(args []string) (int, error) {
		version := update.GetCurrentVersion(shell.versionFile)
		latestVersion := update.GetLatestVersion()
		needsUpdate := IsUpdateNeeded(version, latestVersion)
		// Reload the prompt
		ui.ReloadPrompt(version, needsUpdate)
		return 0, nil
	}

	// Update command handlers in the command map
	cmdmap.RegisterCommand(cmdmap.Command{
		Name:     "exit",
		Handler:  exitHandler,
		Category: cmdmap.CategorySystem,
	})

	cmdmap.RegisterCommand(cmdmap.Command{
		Name:     "reload-blacklist",
		Handler:  reloadBlacklistHandler,
		Category: cmdmap.CategorySecurity,
		Admin:    true,
	})

	cmdmap.RegisterCommand(cmdmap.Command{
		Name:     "reload-whitelist",
		Handler:  reloadWhitelistHandler,
		Category: cmdmap.CategorySecurity,
		Admin:    true,
	})

	cmdmap.RegisterCommand(cmdmap.Command{
		Name:     "toggle-security",
		Handler:  toggleSecurityHandler,
		Category: cmdmap.CategorySecurity,
		Admin:    true,
	})

	cmdmap.RegisterCommand(cmdmap.Command{
		Name:     "jobs",
		Handler:  jobsHandler,
		Category: cmdmap.CategoryProcess,
	})

	cmdmap.RegisterCommand(cmdmap.Command{
		Name:     "history",
		Handler:  historyHandler,
		Category: cmdmap.CategorySystem,
	})

	cmdmap.RegisterCommand(cmdmap.Command{
		Name:     "edit-prompt",
		Handler:  editPromptHandler,
		Category: cmdmap.CategorySystem,
	})

	cmdmap.RegisterCommand(cmdmap.Command{
		Name:     "reload-prompt",
		Handler:  reloadPromptHandler,
		Category: cmdmap.CategorySystem,
	})
}

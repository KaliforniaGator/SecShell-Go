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

	"secshell/admin"
	"secshell/colors"
	"secshell/commands"
	"secshell/core"
	"secshell/download"
	"secshell/env"
	"secshell/globals"
	"secshell/help"
	"secshell/history"
	"secshell/jobs"
	"secshell/logging"
	"secshell/pentest"
	"secshell/sanitize"
	"secshell/services"
	"secshell/tools"
	"secshell/tools/editor"
	filemanager "secshell/tools/file-manager"
	"secshell/ui"
	"secshell/ui/gui"
	"secshell/ui/gui/tests"
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
	historyFile         string
	historyIndex        int
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
	commands.AllowedCommands = core.AllowedCommands
	commands.BuiltInCommands = globals.BuiltInCommands
	commands.AllowedDirs = globals.TrustedDirs
	commands.Init()
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

// sanitizeInput uses the sanitize package to clean input
func (s *SecShell) sanitizeInput(input string, allowSpecialChars ...bool) string {
	allow := true
	if len(allowSpecialChars) > 0 {
		allow = allowSpecialChars[0]
	}
	return sanitize.Input(input, allow)
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
		output, err := tools.ExecuteScript(input)
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Script execution failed: %s", err))
			return
		}
		// Script executed successfully
		fmt.Print(output)
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

		// Prevent deletion of critical files and the config directory
		if args[0] == "rm" {
			// Get absolute paths of critical files/dirs for comparison
			criticalPaths := make(map[string]string)
			criticalItems := []string{
				logging.LogFile,
				globals.BlacklistPath,
				globals.WhitelistPath,
				globals.VersionPath,
				globals.HistoryPath,
				globals.ConfigDir,
			}
			for _, item := range criticalItems {
				absPath, err := filepath.Abs(item)
				if err == nil { // Only add if we can resolve the absolute path
					criticalPaths[absPath] = item // Store original name for logging if needed
				} else {
					logging.LogError(fmt.Errorf("error resolving critical path %s: %w", item, err))
				}
			}

			for _, arg := range args[1:] {
				// Ignore flags like -r, -f, -rf etc.
				if strings.HasPrefix(arg, "-") {
					continue
				}

				// Resolve potential relative paths for the argument
				absArg, err := filepath.Abs(arg)
				if err != nil {
					logging.LogError(fmt.Errorf("error resolving path %s: %w", arg, err))
					continue // Skip if path resolution fails for the argument
				}

				// Check if the argument matches any critical path
				if _, isCritical := criticalPaths[absArg]; isCritical {
					alertMsg := fmt.Sprintf("Attempt to delete the critical file/directory '%s' is forbidden.", arg)
					logging.LogAlert(alertMsg)
					ui.NewLine() // Ensure error box appears on a new line
					gui.ErrorBox(alertMsg)
					return // Prevent execution
				}
			}
		}

		// Check if command requires admin privileges
		if globals.RestrictedCommands[args[0]] {
			if !admin.IsAdmin() {
				logging.LogAlert(fmt.Sprintf("Permission denied: '%s' requires admin privileges", args[0]))
				gui.ErrorBox(fmt.Sprintf("Permission denied: '%s' requires admin privileges", args[0]))
				return
			}
		}

		background := false
		if args[len(args)-1] == "&" {
			background = true
			args = args[:len(args)-1]
		}

		if s.isCommandBlacklisted(args[0]) {
			logging.LogAlert(fmt.Sprintf("Blacklisted command: %s", args[0]))
			gui.ErrorBox(fmt.Sprintf("Command is blacklisted: %s", args[0]))
			return
		}

		// Clear the current line before executing the command
		fmt.Print("\r\033[K")

		switch args[0] {
		case "--version":
			update.DisplayVersion(s.versionFile)
		case "--update":
			update.UpdateSecShell(admin.IsAdmin(), s.versionFile)
		case "services":
			s.manageServices(args)
		case "jobs":
			if len(args) > 1 {
				if args[1] == "-i" || args[1] == "--interactive" {
					jobs.InteractiveJobManager(s.jobs)
				}
			} else {
				s.manageJobs(args)
			}
		case "help":
			if len(args) > 1 {
				if args[1] == "-i" || args[1] == "--interactive" {
					help.InteractiveHelpApp()
				} else {
					help.DisplayHelp(args[1])
				}
			} else {
				help.DisplayHelp()
			}
		case "cd":
			core.ChangeDirectory(args)
		case "time":
			s.getTime()
		case "date":
			s.getDate()
		case "features": // Added features command case
			help.DisplayFeatures()
		case "allowed":
			if len(args) > 1 {
				switch args[1] {
				case "dirs":
					gui.TitleBox("Allowed Directories")
					commands.PrintAllowedDirs()
				case "commands":
					gui.TitleBox("Allowed Commands")
					commands.PrintAllowedCommands()
				case "bins":
					gui.TitleBox("Allowed Binaries")
					commands.PrintProgramCommands()
				case "builtins":
					gui.TitleBox("Built-in Commands")
					commands.PrintBuiltInCommands()
				case "all":
					gui.TitleBox("All Allowed")
					commands.PrintAllCommands()
				}
			} else {
				logging.LogAlert("Usage: allowed <dirs|commands|bins|builtins|all>")
				gui.ErrorBox("Usage: allowed <dirs|commands|bins|builtins|all>")
			}
		case "history":
			if len(args) == 1 {
				history.DisplayHistory(s.history) // Display command history
			} else {
				switch args[1] {
				case "-s":
					if len(args) < 3 {
						logging.LogAlert("Usage: history -s <query>")
						gui.ErrorBox("Usage: history -s <query>")
						return
					}
					history.SearchHistory(s.history, strings.Join(args[2:], " ")) // Search history for the given query
				case "-i":
					history.InteractiveHistorySearch(s.history, s.processCommand) // Run interactive history search
				case "clear":
					s.history = []string{}
					core.ClearHistory(s.historyFile)
				default:
					logging.LogAlert("Invalid history option. Use -s for search or -i for interactive mode.")
					gui.ErrorBox("Invalid history option. Use -s for search or -i for interactive mode.")
				}
			}
		case "export":
			env.ExportVariable(args)
		case "env":
			env.ListEnvVariables()
		case "unset":
			env.UnsetEnvVariable(args)
		case "logs":
			if len(args) < 2 {
				logging.LogAlert("Usage: logs list") // Updated usage message
				gui.ErrorBox("Usage: logs list")     // Updated usage message
				return
			} else {
				switch args[1] {
				case "list":
					err := logging.PrintLog()
					if err != nil {
						logging.LogError(err)
						gui.ErrorBox("Failed to read log file")
					}
				default: // Added default case for invalid options
					logging.LogAlert("Invalid logs option. Use 'list'.")
					gui.ErrorBox("Invalid logs option. Use 'list'.")
				}
			}
		case "blacklist":
			core.ListBlacklistCommands(s.blacklist)
		case "whitelist":
			core.ListWhitelistCommands()
		case "edit-blacklist", "edit-whitelist", "reload-whitelist", "reload-blacklist", "exit":
			// Require admin privileges for these commands
			if !admin.IsAdmin() {
				logging.LogAlert("Permission denied: Admin privileges required.")
				gui.ErrorBox("Permission denied: Admin privileges required.")
				return
			}

			switch args[0] {
			case "edit-blacklist":
				core.EditBlacklist(s.blacklist)
			case "edit-whitelist":
				core.EditWhitelist(s.whitelist)
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
		case "portscan":
			if len(args) < 2 {
				gui.ErrorBox("Usage: portscan [-p ports] [-udp] [-t timing] [-v] [-j|-html] [-o file] [-syn] [-os] [-e] <target>")
				return
			}

			options := &pentest.ScanOptions{
				Protocol:       "tcp",
				Timing:         3,
				ShowVersion:    false,
				Format:         "text",
				OutputFile:     "",
				SynScan:        false,
				DetectOS:       false,
				EnhancedDetect: false,
			}

			target := ""
			portRange := ""

			// Parse arguments
			for i := 1; i < len(args); i++ {
				switch args[i] {
				case "-p":
					if i+1 < len(args) {
						portRange = args[i+1]
						i++
					}
				case "-udp":
					options.Protocol = "udp"
				case "-t":
					if i+1 < len(args) {
						if t, err := strconv.Atoi(args[i+1]); err == nil && t >= 1 && t <= 5 {
							options.Timing = t
							i++
						}
					}
				case "-v":
					options.ShowVersion = true
				case "-j":
					options.Format = "json"
				case "-html":
					options.Format = "html"
				case "-syn":
					options.SynScan = true
				case "-os":
					options.DetectOS = true
				case "-e":
					options.EnhancedDetect = true
				case "-o":
					if i+1 < len(args) {
						options.OutputFile = args[i+1]
						i++
					}
				default:
					if !strings.HasPrefix(args[i], "-") {
						target = args[i]
					}
				}
			}

			if target == "" {
				gui.ErrorBox("No target specified")
				return
			}

			pentest.RunPortScan(target, portRange, options)

		case "hostscan":
			if len(args) < 2 {
				gui.ErrorBox("Usage: hostscan <network-range>")
				return
			}
			pentest.RunHostDiscovery(args[1])

		case "webscan":
			if len(args) < 2 {
				help.DisplayHelp("webscan")
				return
			}

			options := &pentest.WebScanOptions{
				Timeout:       10,
				Threads:       10,
				CustomHeaders: make(map[string]string),
				SkipSSL:       false,
				MaxDepth:      5,
				TestMethods:   []string{"GET", "POST", "HEAD"},
				SafetyChecks:  true,
			}

			target := ""
			for i := 1; i < len(args); i++ {
				switch args[i] {
				case "-t", "--timeout":
					if i+1 < len(args) {
						if t, err := strconv.Atoi(args[i+1]); err == nil {
							options.Timeout = t
						}
						i++
					}
				case "-H", "--header":
					if i+1 < len(args) {
						parts := strings.SplitN(args[i+1], ":", 2)
						if len(parts) == 2 {
							options.CustomHeaders[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
						}
						i++
					}
				case "-k", "--insecure":
					options.SkipSSL = true
				case "-A", "--user-agent":
					if i+1 < len(args) {
						options.UserAgent = args[i+1]
						i++
					}
				case "--threads":
					if i+1 < len(args) {
						if t, err := strconv.Atoi(args[i+1]); err == nil {
							options.Threads = t
						}
						i++
					}
				case "-w", "--wordlist":
					if i+1 < len(args) {
						options.WordlistPath = args[i+1]
						i++
					}
				case "-m", "--methods":
					if i+1 < len(args) {
						options.TestMethods = strings.Split(args[i+1], ",")
						i++
					}
				case "-v", "--verbose":
					options.VerboseMode = true
				case "--follow-redirects":
					options.FollowRedirect = true
				case "--cookie":
					if i+1 < len(args) {
						options.Cookies = args[i+1]
						i++
					}
				case "--auth":
					if i+1 < len(args) {
						options.Authentication = args[i+1]
						i++
					}
				case "-f", "--format":
					if i+1 < len(args) {
						options.OutputFormat = args[i+1]
						i++
					}
				case "-o", "--output":
					if i+1 < len(args) {
						options.OutputFile = args[i+1]
						i++
					}
				default:
					if !strings.HasPrefix(args[i], "-") {
						target = args[i]
					}
				}
			}

			if target == "" {
				gui.ErrorBox("No target specified")
				return
			}

			pentest.WebScan(target, options)

		case "payload":
			if len(args) < 3 {
				gui.ErrorBox("Usage: payload <ip-address> <port>")
				return
			}
			pentest.GenerateReverseShellPayload(args[1], args[2])

		case "session":
			if len(args) < 2 {
				pentest.ListSessions()
				return
			}

			switch args[1] {
			case "-l":
				pentest.ListSessions()
			case "-i":
				if len(args) < 3 {
					gui.ErrorBox("Usage: session -i <id>")
					return
				}
				id, err := strconv.Atoi(args[2])
				if err != nil {
					logging.LogError(err)
					gui.ErrorBox("Invalid session ID")
					return
				}
				pentest.InteractWithSession(id)
			case "-c":
				if len(args) < 3 {
					gui.ErrorBox("Usage: session -c <port>")
					return
				}
				port := args[2]
				id := pentest.ListenForConnections(port)
				if id != -1 {
					gui.AlertBox(fmt.Sprintf("Created session %d", id))
				}
			case "-k":
				if len(args) < 3 {
					gui.ErrorBox("Usage: session -k <id>")
					return
				}
				id, err := strconv.Atoi(args[2])
				if err != nil {
					logging.LogError(err)
					gui.ErrorBox("Invalid session ID")
					return
				}
				pentest.CloseSession(id)
			default:
				gui.ErrorBox("Unknown session command. Use -l, -i, -c, or -k")
			}

		case "base64":
			err := tools.ExecuteEncodingCommand(args, tools.Base64Encoding)
			if err != nil {
				gui.ErrorBox(fmt.Sprintf("Base64 operation failed: %v", err))
			}
			return

		case "hex":
			err := tools.ExecuteEncodingCommand(args, tools.HexEncoding)
			if err != nil {
				gui.ErrorBox(fmt.Sprintf("Hex operation failed: %v", err))
			}
			return

		case "urlencode", "url":
			err := tools.ExecuteEncodingCommand(args, tools.URLEncoding)
			if err != nil {
				gui.ErrorBox(fmt.Sprintf("URL encoding operation failed: %v", err))
			}
			return

		case "binary":
			err := tools.ExecuteEncodingCommand(args, tools.BinaryEncoding)
			if err != nil {
				gui.ErrorBox(fmt.Sprintf("Binary operation failed: %v", err))
			}
			return

		case "hash":
			result, err := tools.HashCommand(args[1:])
			if err != nil {
				gui.ErrorBox(fmt.Sprintf("Hash operation failed: %v", err))
				return
			}
			fmt.Println(result)
			return

		case "extract-strings":
			// Extract strings from binary files
			if len(args) < 2 {
				gui.ErrorBox("Usage: extract-strings <file> [-n min-len]")
				return
			}
			// Remove "extract-strings" from the arguments and pass the rest to StringExtractCmd
			err := tools.RunStringExtract(args[1:])
			if err != nil {
				gui.ErrorBox(fmt.Sprintf("String extraction failed: %v", err))
			}
			return

		case "more":
			// Handle the more command
			if len(args) < 2 {
				// If no argument is provided, show usage
				gui.ErrorBox("Usage: more <file> or command | more")
				return
			}

			// Remove "more" from the args and pass the rest to RunMore
			err := core.RunMore(args[1:])
			if err != nil {
				logging.LogError(err)
				gui.ErrorBox(fmt.Sprintf("Error: %v", err))
			}
			return

		case "edit":
			// Open the file in the default editor
			editor.EditCommand(args[1:])
			return

		case "prompt":
			if len(args) > 1 {
				if args[1] == "-r" || args[1] == "--reset" {
					// Reset the prompt to default
					ui.ResetPrompt()
					return
				}
			}
			ui.DisplayPromptOptions()

		case "edit-prompt":
			// Open the prompt file in the default editor
			editor.EditCommand([]string{globals.PromptConfigFile})
		case "reload-prompt":
			version := update.GetCurrentVersion(s.versionFile)
			latestVersion := update.GetLatestVersion()
			needsUpdate := IsUpdateNeeded(version, latestVersion)
			// Reload the prompt
			ui.ReloadPrompt(version, needsUpdate)
		case "colors":
			colors.DisplayColors()

		case "testwindow": // Add this case
			gui.TestWindowApp() // Call the test function from the gui package

		case "test":
			tests.TestSegmentsApp()
		case "changelog":
			// Display changelog
			update.DisplayChangelog()
		case "files":
			filemanager.FileManagerApp()
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

// executePipedCommands executes a series of piped commands
func (s *SecShell) executePipedCommands(commands []string) {
	// Special handling for 'more' as the last command
	if strings.TrimSpace(commands[len(commands)-1]) == "more" {
		// Create a pipe for the output of all previous commands
		pr, pw := io.Pipe()

		// Execute all commands except 'more'
		var cmds []*exec.Cmd
		for i := 0; i < len(commands)-1; i++ {
			args := strings.Fields(strings.TrimSpace(commands[i]))
			if len(args) == 0 {
				continue
			}
			cmd := exec.Command(args[0], args[1:]...)
			// Set process group for each command
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			cmds = append(cmds, cmd)
		}

		// Set up signal handling to kill all processes in pipeline
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT)
		go func() {
			<-sigChan
			for _, cmd := range cmds {
				if cmd.Process != nil {
					// Kill entire process group
					pgid, err := syscall.Getpgid(cmd.Process.Pid)
					if err == nil {
						syscall.Kill(-pgid, syscall.SIGKILL)
					}
				}
			}
			pw.Close()
		}()
		defer signal.Stop(sigChan)

		// Set up the pipeline
		for i := 0; i < len(cmds)-1; i++ {
			stdout, err := cmds[i].StdoutPipe()
			if err != nil {
				logging.LogError(err)
				gui.ErrorBox(fmt.Sprintf("Failed to set up pipeline: %s", err))
				return
			}
			cmds[i+1].Stdin = stdout
		}

		// Set first command's stdin to os.Stdin and last command's stdout to our pipe
		if len(cmds) > 0 {
			cmds[0].Stdin = os.Stdin
			cmds[len(cmds)-1].Stdout = pw
			cmds[len(cmds)-1].Stderr = os.Stderr
		}

		// Start all commands
		for _, cmd := range cmds {
			if err := cmd.Start(); err != nil {
				logging.LogError(err)
				gui.ErrorBox(fmt.Sprintf("Failed to start command: %s", err))
				return
			}
		}

		// Read the output in a goroutine
		var lines []string
		go func() {
			scanner := bufio.NewScanner(pr)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			pw.Close()
		}()

		// Wait for all commands to finish
		for _, cmd := range cmds {
			if err := cmd.Wait(); err != nil && !isSignalKilled(err) {
				logging.LogError(err)
			}
		}

		// Close the pipe writer to signal EOF
		pw.Close()

		// Now pass the collected lines to More
		err := core.More(lines)
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("More failed: %s", err))
		}
		return
	}

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
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to set up pipeline: %s", err))
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
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to start command: %s", err))
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
				logging.LogError(err)
				gui.ErrorBox(fmt.Sprintf("Command execution failed: %s", err))
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

// Check if command needs terminal reset
func needsTerminalReset(cmd string) bool {
	// Commands that need direct terminal access
	return cmd == "sudo" || cmd == "su" || isTerminalEditor(cmd)
}

// executeSystemCommand executes a system command
func (s *SecShell) executeSystemCommand(args []string, background bool) {
	if len(args) == 0 {
		return
	}

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
		logging.LogAlert(fmt.Sprintf("Command not permitted: %s", args[0]))
		gui.ErrorBox(fmt.Sprintf("Command not permitted: %s", args[0]))
		return
	}

	// Special handling for commands that need terminal reset
	if needsTerminalReset(args[0]) {
		// Save terminal state
		oldState, err := term.GetState(int(os.Stdin.Fd()))
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to get terminal state: %s", err))
			return
		}

		// Reset terminal to normal mode for password input
		term.Restore(int(os.Stdin.Fd()), oldState)

		// For sudo or su commands, we need more careful terminal handling
		if args[0] == "sudo" || args[0] == "su" || isTerminalEditor(args[0]) {
			// Check if user is admin before allowing sudo/su
			if !admin.IsAdmin() {
				logging.LogAlert("Permission denied: Only admins can use sudo/su commands")
				gui.ErrorBox("Permission denied: Only admins can use sudo/su commands")
				return
			}

			// Execute sudo/su in a way that properly handles the password prompt
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			// Different process group handling for su vs sudo
			if args[0] == "su" {
				// For su, we need to create a new session
				cmd.SysProcAttr = &syscall.SysProcAttr{
					Setsid:  true,  // Create new session
					Setpgid: false, // Don't create process group
				}
			} else {
				// For sudo, keep existing behavior
				cmd.SysProcAttr = &syscall.SysProcAttr{
					Setpgid: false,
				}
			}

			// Run the command directly
			err = cmd.Run()

			// Get a new terminal state after command completes
			_, _ = term.GetState(int(os.Stdin.Fd()))

			if err != nil && !isSignalKilled(err) {
				logging.LogError(err)
				gui.ErrorBox(fmt.Sprintf("Command execution failed: %s", err))
			}

			if isTerminalEditor(args[0]) {
				// If the command is an editor, we need to restore the terminal state
				term.Restore(int(os.Stdin.Fd()), oldState)
			}
		} else {
			// Handle other terminal-dependent commands normally
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			err = cmd.Run()
		}

		// After command exits, clear screen
		//fmt.Print("\033[H\033[2J")

		if err != nil && !isSignalKilled(err) {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Command execution failed: %s", err))
		}
		return
	}

	// Process standard commands
	cmdArgs := []string{}
	var stdout io.Writer = os.Stdout
	var stdin io.Reader = os.Stdin

	// Special handling for ls and grep commands to add color
	switch args[0] {
	case "ls", "grep":
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
					logging.LogError(err)
					gui.ErrorBox(fmt.Sprintf("Failed to create file: %s", err))
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
					logging.LogError(err)
					gui.ErrorBox(fmt.Sprintf("Failed to open file: %s", err))
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
		// Handle background jobs (unchanged)
		if err := cmd.Start(); err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to start background job: %s", err))
			return
		}
		jobs.AddJob(s.jobs, cmd.Process.Pid, args[0], cmd.Process)

		// Goroutine to wait for the command to finish and update its status
		go func(pid int) {
			waitErr := cmd.Wait() // Capture the error from cmd.Wait()
			exitCode := 0

			job, jobExists := s.jobs[pid]
			suppressErrorLoggingAndDisplay := false

			if jobExists {
				job.Lock()
				// If waitErr is a signal interrupt and the job was already marked as "stopped"
				// (by StopJobClean or similar mechanism), then suppress the generic error display and logging.
				if waitErr != nil && isSignalKilled(waitErr) && job.Status == "stopped" {
					suppressErrorLoggingAndDisplay = true
				}
				job.Unlock()
			}

			if waitErr != nil {
				if exitError, ok := waitErr.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				}

				if !suppressErrorLoggingAndDisplay {
					// Log the error only if it's not a user-initiated stop that's already handled/logged.
					logging.LogError(waitErr)
				}
			}

			// Update job status
			if jobExists {
				job.Lock()
				job.EndTime = time.Now()
				job.ExitCode = exitCode // Store exit code

				if waitErr != nil {
					// If the job status is already "stopped" (set by StopJobClean, for instance),
					// and the error is a signal interrupt, keep status as "stopped".
					// Otherwise, mark as failed.
					if !(job.Status == "stopped" && isSignalKilled(waitErr)) {
						job.Status = fmt.Sprintf("failed with code %d", exitCode)
					}
					// If job.Status was "stopped" and it was a SIGINT, it remains "stopped".
				} else {
					job.Status = "completed"
				}
				job.Unlock()
			}
		}(cmd.Process.Pid)
	} else {
		// Handle foreground execution
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT)
		defer signal.Stop(sigChan)
		defer close(sigChan)

		if err := cmd.Start(); err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Command execution failed: %s", err))
			return
		}

		// Forward SIGINT to the process group
		go func() {
			for range sigChan {
				syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
			}
		}()

		err := cmd.Wait()
		if err != nil && !isSignalKilled(err) {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Command execution failed: %s", err))
		}
	}
}

// isCommandAllowed checks if a command is allowed
func (s *SecShell) isCommandAllowed(cmd string) bool {
	// Bypass security checks for built-in commands
	if admin.IsAdmin() && !securityEnabled {
		return true // Admins bypass whitelist
	}

	// Sanitize the command first
	cmd = sanitize.Command(cmd)

	// Define a list of restricted network commands
	networkCommands := []string{"wget", "curl", "nc", "nmap", "scp", "rsync"}

	for _, netCmd := range networkCommands {
		if cmd == netCmd {
			logging.LogAlert("Network access restricted for non-admin users.")
			gui.ErrorBox("Network access restricted for non-admin users.")
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
			logging.LogError(err)
			return true
		}
	}

	return false
}

// isCommandBlacklisted checks if a command is blacklisted
func (s *SecShell) isCommandBlacklisted(cmd string) bool {
	// Bypass security checks for built-in commands
	if admin.IsAdmin() && !securityEnabled {
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
	if !authenticateUser(password) {
		logging.LogAlert("Authentication failed. Incorrect password.")
		gui.ErrorBox("Authentication failed. Incorrect password.")
		return
	}

	// Toggle security state
	securityEnabled = !securityEnabled
	if securityEnabled {
		logging.LogAlert("Security enforcement ENABLED.")
		gui.AlertBox("Security enforcement ENABLED.")
	} else {
		logging.LogAlert("Security enforcement DISABLED. All commands are now allowed.")
		gui.AlertBox("Security enforcement DISABLED. All commands are now allowed.")
	}
}

func authenticateUser(password string) bool {
	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		logging.LogError(err)
		fmt.Println("Error getting current user:", err)
		return false
	}

	// Start a PAM authentication transaction
	transaction, err := pam.StartFunc("login", currentUser.Username, func(s pam.Style, msg string) (string, error) {
		return password, nil
	})
	if err != nil {
		logging.LogError(err)
		fmt.Println("PAM transaction start failed:", err)
		return false
	}

	// Attempt authentication
	err = transaction.Authenticate(0)
	if err != nil {
		logging.LogError(err)
		fmt.Println("Authentication failed:", err)
		return false
	}

	return true // Authentication successful
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

	shell := NewSecShell(globals.BlacklistPath, globals.WhitelistPath)
	shell.run()
}

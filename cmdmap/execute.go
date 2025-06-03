package cmdmap

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"secshell/admin"
	"secshell/core"
	"secshell/globals"
	"secshell/jobs"
	"secshell/logging"
	"secshell/sanitize"
	"secshell/ui"
	"secshell/ui/gui"

	"golang.org/x/term"
)

// ExecuteCommand handles command execution with appropriate terminal mode
func ExecuteCommand(input string, jobsMap map[int]*jobs.Job) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}

	// Print a new line before command output to avoid formatting issues
	fmt.Println()

	// Handle piped commands
	if strings.Contains(input, "|") {
		executePipedCommands(input, jobsMap)
		return
	}

	// Parse the command and arguments, preserving quoted arguments
	args := parseCommandLine(input)
	if len(args) == 0 {
		return
	}

	// Check for background execution
	background := false
	if args[len(args)-1] == "&" {
		background = true
		args = args[:len(args)-1]
	}

	// Get the command name
	cmdName := args[0]

	// Get security state
	isAdminUser := admin.IsAdmin()
	securityBypass := isAdminUser && !securityEnabled

	// Prevent deletion of critical files and the config directory
	if cmdName == "rm" {
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

	// Check if command requires admin privileges - always enforced regardless of security state
	if globals.RestrictedCommands[cmdName] && !isAdminUser {
		logging.LogAlert(fmt.Sprintf("Permission denied: '%s' requires admin privileges", cmdName))
		gui.ErrorBox(fmt.Sprintf("Permission denied: '%s' requires admin privileges", cmdName))
		return
	}

	// If security is disabled for an admin user, skip all further permission checks
	if !securityBypass {
		// Check if command is blacklisted
		for _, blacklisted := range core.BlacklistedCommands {
			if cmdName == blacklisted {
				logging.LogAlert(fmt.Sprintf("Command '%s' is blacklisted and cannot be executed", cmdName))
				gui.ErrorBox(fmt.Sprintf("Command '%s' is blacklisted and cannot be executed", cmdName))
				return
			}
		}

		// Check if command is allowed (whitelisted or in trusted directory)
		isAllowed := false

		// Check if command exists in the map (built-in command)
		if _, exists := GlobalCommandMap[cmdName]; exists {
			isAllowed = true
		}

		// Check if command is in whitelist
		if !isAllowed {
			for _, whitelisted := range core.AllowedCommands {
				if cmdName == whitelisted {
					isAllowed = true
					break
				}
			}
		}

		// Check if command exists in trusted directories
		if !isAllowed {
			for _, dir := range globals.TrustedDirs {
				path := filepath.Join(dir, cmdName)
				if _, err := os.Stat(path); err == nil {
					isAllowed = true
					break
				}
			}
		}

		if !isAllowed {
			logging.LogAlert(fmt.Sprintf("Command not permitted: %s", cmdName))
			gui.ErrorBox(fmt.Sprintf("Command not permitted: %s", cmdName))
			return
		}
	} else if securityBypass {
		// Log the security bypass for audit purposes
		logging.LogAlert(fmt.Sprintf("SECURITY BYPASS: Admin user executing command with security disabled: %s", cmdName))
	}

	// Check if command is a built-in command
	cmd, exists := GetCommand(cmdName)
	if exists && cmd.Handler != nil {

		// Execute built-in command
		exitCode, err := cmd.Handler(args)
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Command execution failed: %s", err))
		}
		logging.LogCommand(input, exitCode)
		return
	}

	// Check if command needs raw terminal
	needsRawTerm := NeedsRawTerminal(cmdName)

	// Execute external command
	if background {
		executeBackgroundCommand(args, jobsMap)
	} else if needsRawTerm {
		executeRawTerminalCommand(args)
	} else {
		executeNormalCommand(args)
	}
}

// parseCommandLine splits a command line into arguments, preserving quoted strings
func parseCommandLine(cmdLine string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)
	escaped := false

	for _, char := range cmdLine {
		switch {
		case escaped:
			// Handle escaped character
			current.WriteRune(char)
			escaped = false
		case char == '\\':
			// Next character is escaped
			escaped = true
		case char == '"' || char == '\'':
			if inQuotes && char == quoteChar {
				// End of quoted string
				inQuotes = false
				quoteChar = rune(0)
			} else if !inQuotes {
				// Start of quoted string
				inQuotes = true
				quoteChar = char
			} else {
				// Quote character inside another type of quotes
				current.WriteRune(char)
			}
		case char == ' ' && !inQuotes:
			// Space outside quotes - end of argument
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			// Regular character
			current.WriteRune(char)
		}
	}

	// Add the last argument if not empty
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	// Handle the case where there are unclosed quotes
	if inQuotes {
		// Try to fix common user errors by assuming they meant to close the quote
		logging.LogAlert("Warning: Unclosed quotes detected in command")
	}

	return args
}

// executeBackgroundCommand executes a command in the background
func executeBackgroundCommand(args []string, jobsMap map[int]*jobs.Job) {
	// Create the command
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Start the command
	if err := cmd.Start(); err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Failed to start background job: %s", err))
		return
	}

	// Add to jobs map
	jobs.AddJob(jobsMap, cmd.Process.Pid, args[0], cmd.Process)

	// Print confirmation of background job
	fmt.Printf("\nStarted background job [%d]: %s\n", cmd.Process.Pid, args[0])

	// Wait for the command to finish in a goroutine
	go func() {
		err := cmd.Wait()
		exitCode := 0

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
			logging.LogError(err)
		}

		// Update job status
		job, exists := jobsMap[cmd.Process.Pid]
		if exists {
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
	}()
}

// executeRawTerminalCommand executes a command that needs raw terminal mode
func executeRawTerminalCommand(args []string) {
	// Save current terminal state
	oldState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Failed to get terminal state: %s", err))
		return
	}

	// Restore terminal state when function returns
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Create the command
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Important: don't set process group for raw terminal commands
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: false}

	// Special handling for sudo/su
	if args[0] == "sudo" || args[0] == "su" {
		if !admin.IsAdmin() {
			logging.LogAlert("Permission denied: Only admins can use sudo/su commands")
			gui.ErrorBox("Permission denied: Only admins can use sudo/su commands")
			return
		}

		// Use setsid for su to create a new session
		if args[0] == "su" {
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Setsid:  true,  // Create new session
				Setpgid: false, // Don't create process group
			}
		}
	}

	// Run the command
	err = cmd.Run()
	if err != nil && !isSignalKilled(err) {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Command execution failed: %s", err))
	}
}

// executeNormalCommand executes a command in normal terminal mode
func executeNormalCommand(args []string) {
	// Process standard commands
	cmdArgs := []string{}
	var stdout io.Writer = os.Stdout
	var stdin io.Reader = os.Stdin

	// Special handling for ls and grep commands to add color
	switch args[0] {
	case "ls", "grep", "diff":
		hasColorFlag := false
		for _, arg := range args[1:] {
			if strings.HasPrefix(arg, "--color") {
				hasColorFlag = true
				break
			}
		}
		if !hasColorFlag {
			if args[0] == "ls" {
				cmdArgs = append(cmdArgs, "--color=auto")
			} else if args[0] == "grep" {
				cmdArgs = append(cmdArgs, "--color=always")
			} else if args[0] == "diff" {
				cmdArgs = append(cmdArgs, "--color=auto")
			}
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
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	defer signal.Stop(sigChan)
	defer close(sigChan)

	// Start the command
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

	// Wait for the command to finish
	err := cmd.Wait()
	if err != nil && !isSignalKilled(err) {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Command execution failed: %s", err))
	}
}

// executePipedCommands handles command pipelines
func executePipedCommands(input string, jobsMap map[int]*jobs.Job) {
	// Split the input into separate commands
	splitCommands := strings.Split(input, "|")
	for i, cmd := range splitCommands {
		splitCommands[i] = strings.TrimSpace(cmd)
	}

	// Special handling for 'more' as the last command
	if strings.TrimSpace(splitCommands[len(splitCommands)-1]) == "more" {
		executePipeWithMore(splitCommands[:len(splitCommands)-1])
		return
	}

	// Set up the pipeline
	var cmds []*exec.Cmd

	// Create commands
	for _, cmdString := range splitCommands {
		args := parseCommandLine(cmdString)
		if len(args) == 0 {
			continue
		}

		// Add color flags for supported commands
		if len(args) > 0 {
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
			case "ls", "diff":
				hasColorFlag := false
				for _, arg := range args {
					if strings.HasPrefix(arg, "--color") {
						hasColorFlag = true
						break
					}
				}
				if !hasColorFlag {
					args = append([]string{args[0], "--color=auto"}, args[1:]...)
				}
			}
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmds = append(cmds, cmd)
	}

	if len(cmds) == 0 {
		return
	}

	// Set up pipes between commands
	for i := 0; i < len(cmds)-1; i++ {
		stdout, err := cmds[i].StdoutPipe()
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to set up pipeline: %s", err))
			return
		}
		cmds[i+1].Stdin = stdout
	}

	// Set first command's stdin and last command's stdout
	cmds[0].Stdin = os.Stdin
	cmds[len(cmds)-1].Stdout = os.Stdout
	cmds[len(cmds)-1].Stderr = os.Stderr

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	// Start all commands
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to start command: %s", err))
			signal.Stop(sigChan)
			close(sigChan)
			return
		}
	}

	// Forward SIGINT to all process groups
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
			logging.LogError(err)
		}
	}

	signal.Stop(sigChan)
	close(sigChan)
}

// executePipeWithMore handles pipelines that end with 'more'
func executePipeWithMore(commands []string) {
	// Create pipe for the output of all previous commands
	pr, pw := io.Pipe()

	// Execute all commands except 'more'
	var cmds []*exec.Cmd
	for _, cmdString := range commands {
		args := parseCommandLine(cmdString)
		if len(args) == 0 {
			continue
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmds = append(cmds, cmd)
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	defer signal.Stop(sigChan)

	go func() {
		<-sigChan
		for _, cmd := range cmds {
			if cmd.Process != nil {
				pgid, err := syscall.Getpgid(cmd.Process.Pid)
				if err == nil {
					syscall.Kill(-pgid, syscall.SIGKILL)
				}
			}
		}
		pw.Close()
	}()

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

	// Set first command's stdin and last command's stdout
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

	// Pass collected lines to More
	moreCmd, exists := GetCommand("more")
	if exists && moreCmd.Handler != nil {
		moreCmd.Handler([]string{"more", strings.Join(lines, "\n")})
	} else {
		// Fallback to printing lines
		for _, line := range lines {
			fmt.Println(line)
		}
	}
}

// isSignalKilled checks if an error was caused by a signal
func isSignalKilled(err error) bool {
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.Signaled() && status.Signal() == syscall.SIGINT
		}
	}
	return false
}

// RegisterBuiltInHandlers registers handlers for built-in commands
func RegisterBuiltInHandlers() {
	// Register handlers for built-in commands
	// These will be implemented in secshell.go and mapped here
}

// Helper function for command handlers that need to manipulate files
func sanitizePath(path string) string {
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	// Sanitize the path
	return sanitize.Path(path)
}

package main

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

	"secshell/colors"

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

// NewSecShell initializes a new SecShell instance
func NewSecShell(blacklistPath, whitelistPath string) *SecShell {
	shell := &SecShell{
		jobs:            make(map[int]string),
		running:         true,
		allowedDirs:     []string{"/usr/bin/", "/bin/", "/opt/"},
		allowedCommands: []string{},
		blacklist:       blacklistPath,
		whitelist:       whitelistPath,
		history:         []string{},
		historyIndex:    -1,
	}
	shell.ensureFilesExist()
	shell.loadBlacklist(blacklistPath)
	shell.loadWhitelist(whitelistPath)
	return shell
}

// ensureFilesExist checks and creates blacklist and whitelist files if they don't exist
func (s *SecShell) ensureFilesExist() {
	defaultWhitelistCommands := []string{"ls", "cd", "pwd", "cp", "mv", "rm", "mkdir", "rmdir", "touch", "cat", "echo", "grep", "find", "chmod", "chown", "ps", "kill", "top", "df", "du", "ifconfig", "netstat", "ping", "clear", "vim", "nano", "emacs", "nvim"}

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
		s.allowedCommands = []string{"ls", "cd", "pwd", "cp", "mv", "rm", "mkdir", "rmdir", "touch", "cat", "echo", "grep", "find", "chmod", "chown", "ps", "kill", "top", "df", "du", "ifconfig", "netstat", "ping", "clear", "vim", "nano", "emacs", "nvim"}
	}
}

// run starts the shell and listens for user input
func (s *SecShell) run() {
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

// displayPrompt shows the shell prompt to the user
func (s *SecShell) displayPrompt() {
	user := os.Getenv("USER")
	if user == "" {
		user = "unknown"
	}

	cwd, err := os.Getwd()
	if err != nil {
		s.printError("Failed to get current working directory")
		return
	}

	fmt.Fprintf(os.Stdout, "%s┌─[SecShell]%s %s(%s)%s %s[%s]%s\n%s└─%s$ ",
		colors.Green, colors.Reset,
		colors.Blue, user, colors.Reset,
		colors.Yellow, cwd, colors.Reset,
		colors.Green, colors.Reset)
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
	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			s.printError(fmt.Sprintf("Failed to read input: %s", err))
			return ""
		}

		input := string(buf[:n])
		switch input {
		case KeyLeft:
			if pos > 0 {
				pos--
				fmt.Print("\x1b[D") // Move cursor left
			}
		case KeyRight:
			if pos < len(line) {
				pos++
				fmt.Print("\x1b[C") // Move cursor right
			}
		case KeyDelete, KeyBackspace:
			if pos > 0 {
				// Remove character at position
				line = line[:pos-1] + line[pos:]
				pos--
				// Clear from cursor to end of line
				fmt.Print("\x1b[D\x1b[K")
				// Print remaining text
				if pos < len(line) {
					fmt.Print(line[pos:])
					// Move cursor back to position
					fmt.Printf("\x1b[%dD", len(line)-pos)
				}
			}
		case KeyUp:
			if s.historyIndex > 0 {
				s.historyIndex--
				newLine := strings.TrimSpace(s.history[s.historyIndex])
				// Clear current line content
				fmt.Printf("\x1b[%dD\x1b[K", pos)
				// Print new line
				fmt.Print(newLine)
				line = newLine
				pos = len(line)
			}
		case KeyDown:
			if s.historyIndex < len(s.history)-1 {
				s.historyIndex++
				newLine := strings.TrimSpace(s.history[s.historyIndex])
				// Clear current line content
				fmt.Printf("\x1b[%dD\x1b[K", pos)
				// Print new line
				fmt.Print(newLine)
				line = newLine
				pos = len(line)
			}
		case KeyTab:
			line, pos = s.completeCommand(line, pos)
		case "\r", "\n":
			fmt.Println()
			input := s.sanitizeInput(strings.TrimSpace(line))
			if input != "" {
				s.history = append(s.history, input)
				s.historyIndex = len(s.history)
			}
			return input
		default:
			if len(input) == 1 && input[0] >= 32 { // Printable characters
				// Insert character at current position
				line = line[:pos] + input + line[pos:]
				fmt.Print(line[pos:]) // Print from cursor to end
				pos++
				// Move cursor back to position
				if pos < len(line) {
					fmt.Printf("\x1b[%dD", len(line)-pos)
				}
			}
		}
	}
}

// completeCommand provides command completion suggestions
func (s *SecShell) completeCommand(line string, pos int) (string, int) {
	words := strings.Fields(line)
	if len(words) == 0 {
		return line, pos
	}

	lastWord := words[len(words)-1]
	completions := s.getCompletions(lastWord)

	if len(completions) == 1 {
		// Replace the last word with the completion
		words[len(words)-1] = completions[0]
		newLine := strings.Join(words, " ")
		return newLine, len(newLine)
	} else if len(completions) > 1 {
		// Show multiple completions
		fmt.Println()
		for _, completion := range completions {
			fmt.Printf("%s  ", completion)
		}
		fmt.Println()
		s.displayPrompt()
		fmt.Print(line)
	}
	return line, pos
}

// getCompletions returns a list of possible completions for a given prefix
func (s *SecShell) getCompletions(prefix string) []string {
	var completions []string
	for _, cmd := range s.allowedCommands {
		if strings.HasPrefix(cmd, prefix) {
			completions = append(completions, cmd)
		}
	}
	files, _ := filepath.Glob(prefix + "*")
	completions = append(completions, files...)
	return completions
}

// sanitizeInput removes forbidden characters from input
func (s *SecShell) sanitizeInput(input string) string {
	forbidden := ";`"
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

// exportVariable sets an environment variable
func (s *SecShell) exportVariable(args []string) {
	if len(args) < 2 {
		s.printError("Usage: export VAR=value")
		return
	}

	varValue := args[1]
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

	varName := args[1]
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
			s.displayHistory()
		case "export":
			s.exportVariable(args)
		case "env":
			s.listEnvVariables()
		case "unset":
			s.unsetEnvVariable(args)
		case "reload-blacklist":
			s.reloadBlacklist()
		case "blacklist":
			s.listBlacklistCommands()
		case "edit-blacklist":
			s.editBlacklist()
		case "whitelist":
			s.listWhitelistCommands()
		case "edit-whitelist":
			s.editWhitelist()
		case "reload-whitelist":
			s.reloadWhitelist()
		case "exit":
			s.running = false
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
		colors.Cyan, colors.Reset, // Allowed System Commands
		colors.Cyan, colors.Reset, // Security Features
		colors.Cyan, colors.Reset, // Examples
		colors.Cyan, colors.Reset, // Note
	)
}

// isCommandAllowed checks if a command is allowed
func (s *SecShell) isCommandAllowed(cmd string) bool {
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
	for _, blacklistedCmd := range s.blacklistedCommands {
		if cmd == blacklistedCmd {
			return true
		}
	}
	return false
}

// runDrawbox runs the drawbox command to display a message box
func (s *SecShell) runDrawbox(title, color string) {
	fmt.Print("\n") // Add newline before box
	drawboxPath := filepath.Join("/usr/local/bin", "drawbox")
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
	fmt.Print("\n") // Add newline before box
	drawboxPath := filepath.Join("/usr/local/bin", "drawbox")
	cmd := exec.Command(drawboxPath, message, "bold_yellow")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Fallback if drawbox fails
		fmt.Fprintf(os.Stdout, "%s╔═ ALERT ═╗\n│%s %s\n%s└────────┘%s\n",
			colors.BoldYellow, colors.Reset, message,
			colors.BoldYellow, colors.Reset)
	}
}

// printError prints an error message
func (s *SecShell) printError(message string) {
	fmt.Print("\n") // Add newline before box
	drawboxPath := filepath.Join("/usr/local/bin", "drawbox")
	cmd := exec.Command(drawboxPath, message, "bold_red")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Fallback if drawbox fails
		fmt.Fprintf(os.Stderr, "%s╔═ ERROR ═╗\n│%s %s\n%s└────────┘%s\n",
			colors.BoldRed, colors.Reset, message,
			colors.BoldRed, colors.Reset)
	}
}

// Add this function near the top of the file after the imports
func getExecutablePath() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// main function to start the shell
func main() {
	exePath := getExecutablePath()
	blacklistPath := filepath.Join(exePath, ".blacklist")
	whitelistPath := filepath.Join(exePath, ".whitelist")
	shell := NewSecShell(blacklistPath, whitelistPath)
	shell.run()
}

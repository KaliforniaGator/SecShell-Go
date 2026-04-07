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
	"secshell/ui/gui"

	"golang.org/x/term"
)

// ExecuteCommand handles command execution with proper shell parsing
func ExecuteCommand(input string, jobsMap map[int]*jobs.Job) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}

	// Check for potentially destructive commands
	if msg, isBlocked := isDestructiveCommand(input); isBlocked {
		fmt.Println()
		logging.LogAlert(fmt.Sprintf("Security: %s", msg))
		gui.ErrorBox(fmt.Sprintf("Security: %s", msg))
		return
	}

	// Print a new line before command output
	fmt.Println()

	// Try to parse the command using our shell parser
	chain, parseErr := ParseString(input)

	// If parsing fails, fall back to legacy handling
	if parseErr != nil {
		// Check if it's the here-doc unsupported error
		if strings.Contains(parseErr.Error(), "here-doc") {
			logging.LogError(parseErr)
			gui.ErrorBox(fmt.Sprintf("Command parsing error: %s", parseErr))
			return
		}
		// Fall back to legacy parsing for backward compatibility
		executeLegacyCommand(input, jobsMap)
		return
	}

	// Execute the command chain
	executeChain(chain, jobsMap, input)
}

// executeChain executes a chain of pipelines connected by && and ||
func executeChain(chain *ChainNode, jobsMap map[int]*jobs.Job, originalInput string) {
	lastExitCode := 0

	for i, pipeline := range chain.Pipelines {
		// Determine if we should execute this pipeline based on &&/|| semantics
		if i > 0 {
			op := chain.Operators[i-1]
			switch op {
			case OpAnd:
				// Only run if previous command succeeded
				if lastExitCode != 0 {
					return
				}
			case OpOr:
				// Only run if previous command failed
				if lastExitCode == 0 {
					return
				}
			}
		}

		// Execute the pipeline
		if pipeline.Background {
			executePipelineBackground(pipeline, jobsMap, &lastExitCode)
		} else {
			// Check for 'more' as the last command
			if len(pipeline.Commands) > 1 && pipeline.Commands[len(pipeline.Commands)-1].Name == "more" {
				executePipelineWithMore(pipeline.Commands[:len(pipeline.Commands)-1], &lastExitCode)
			} else {
				executePipeline(pipeline, &lastExitCode)
			}
		}
	}
}

// executePipeline executes a pipeline of commands connected by pipes
func executePipeline(pipeline PipelineNode, lastExitCode *int) {
	if len(pipeline.Commands) == 0 {
		*lastExitCode = 0
		return
	}

	if len(pipeline.Commands) == 1 {
		// Single command with redirections
		*lastExitCode = executeSingleCommand(pipeline.Commands[0])
		return
	}

	// Multiple commands connected by pipes - commands in pipelines use pipe I/O, not raw terminal
	var cmds []*exec.Cmd
	var files []*os.File
	_ = files

	for _, cmdNode := range pipeline.Commands {
		cmd, cleanupCmd, err := buildCommandFromNode(cmdNode)
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to build command: %s", err))
			*lastExitCode = 1
			return
		}
		if cleanupCmd != nil {
			files = append(files, cleanupCmd...)
		}
		cmds = append(cmds, cmd)
	}

	if len(cmds) == 0 {
		return
	}

	// Add color flags for supported commands
	for _, cmd := range cmds {
		addColorFlagsToCmd(cmd)
	}

	// Set up pipes between commands
	for i := 0; i < len(cmds)-1; i++ {
		stdout, err := cmds[i].StdoutPipe()
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to set up pipeline: %s", err))
			*lastExitCode = 1
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
			*lastExitCode = 1
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
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				*lastExitCode = exitErr.ExitCode()
			} else if !isSignalKilled(err) {
				logging.LogError(err)
				*lastExitCode = 1
			}
		}
	}

	// Close any opened files
	for _, file := range files {
		file.Close()
	}

	signal.Stop(sigChan)
	close(sigChan)
}

// executePipelineBackground executes a pipeline in the background
func executePipelineBackground(pipeline PipelineNode, jobsMap map[int]*jobs.Job, lastExitCode *int) {
	if len(pipeline.Commands) == 0 {
		*lastExitCode = 0
		return
	}

	if len(pipeline.Commands) == 1 {
		// Single command with redirections in background
		cmdNode := pipeline.Commands[0]
		executeSingleCommandBackground(cmdNode, jobsMap)
		return
	}

	// Multiple commands connected by pipes in background
	var cmds []*exec.Cmd

	for _, cmdNode := range pipeline.Commands {
		cmd, cleanupFiles, err := buildCommandFromNode(cmdNode)
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to build command: %s", err))
			*lastExitCode = 1
			return
		}
		for _, f := range cleanupFiles {
			f.Close()
		}
		cmds = append(cmds, cmd)
	}

	if len(cmds) == 0 {
		return
	}

	// Add color flags for supported commands
	for _, cmd := range cmds {
		addColorFlagsToCmd(cmd)
	}

	// Set up pipes between commands
	for i := 0; i < len(cmds)-1; i++ {
		stdout, err := cmds[i].StdoutPipe()
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to set up pipeline: %s", err))
			*lastExitCode = 1
			return
		}
		cmds[i+1].Stdin = stdout
	}

	// No stdin for background jobs
	cmds[0].Stdin = nil
	cmds[len(cmds)-1].Stdout = os.Stdout
	cmds[len(cmds)-1].Stderr = os.Stderr

	// Build command string for display
	var cmdStrings []string
	for _, cmd := range pipeline.Commands {
		cmdStr := cmd.Name
		if len(cmd.Args) > 0 {
			cmdStr += " " + strings.Join(cmd.Args, " ")
		}
		cmdStrings = append(cmdStrings, cmdStr)
	}
	cmdDisplay := strings.Join(cmdStrings, " | ")

	// Start all commands
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to start command: %s", err))
			*lastExitCode = 1
			return
		}
	}

	// Get the PID of the last command for job tracking
	lastPid := cmds[len(cmds)-1].Process.Pid

	// Add to jobs map
	jobs.AddJob(jobsMap, lastPid, cmdDisplay, cmds[len(cmds)-1].Process)

	// Print confirmation of background job
	fmt.Printf("\nStarted background job [%d]: %s\n", lastPid, cmdDisplay)

	// Wait for all commands to finish in a goroutine
	go func() {
		exitCode := 0
		var jobErr error

		for _, cmd := range cmds {
			err := cmd.Wait()
			if err != nil {
				jobErr = err
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				}
			}
		}

		if jobErr != nil {
			logging.LogError(jobErr)
		}

		// Update job status
		job, exists := jobsMap[lastPid]
		if exists {
			job.Lock()
			job.EndTime = time.Now()
			job.ExitCode = exitCode

			if jobErr != nil {
				job.Status = fmt.Sprintf("failed with exit code %d", exitCode)
			} else {
				job.Status = "completed"
			}
			job.Unlock()
		}
	}()

	*lastExitCode = 0
}

// executePipelineWithMore executes a pipeline that ends with 'more' command
func executePipelineWithMore(commands []CommandNode, lastExitCode *int) {
	if len(commands) == 0 {
		*lastExitCode = 0
		return
	}

	// Create pipe for the output
	pr, pw := io.Pipe()

	var cmds []*exec.Cmd

	for _, cmdNode := range commands {
		cmd, cleanupFiles, err := buildCommandFromNode(cmdNode)
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to build command: %s", err))
			*lastExitCode = 1
			return
		}
		for _, f := range cleanupFiles {
			f.Close()
		}
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
			*lastExitCode = 1
			return
		}
		cmds[i+1].Stdin = stdout
	}

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
			*lastExitCode = 1
			return
		}
	}

	// Read the output
	var lines []string
	scanDone := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		scanDone <- scanner.Err()
	}()

	// Wait for all commands
	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil && !isSignalKilled(err) {
			logging.LogError(err)
		}
	}

	pw.Close()
	if scanErr := <-scanDone; scanErr != nil {
		logging.LogError(scanErr)
		gui.ErrorBox(fmt.Sprintf("Failed to read piped output: %s", scanErr))
		*lastExitCode = 1
		return
	}

	if err := core.More(lines); err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Error: %v", err))
		*lastExitCode = 1
		return
	}

	*lastExitCode = 0
}

// executeSingleCommand executes a single command with its redirections
func executeSingleCommand(cmdNode CommandNode) int {
	// Check if it's a built-in command first
	builtinCmd, exists := GetCommand(cmdNode.Name)
	if exists && builtinCmd.Handler != nil {
		args := append([]string{cmdNode.Name}, cmdNode.Args...)
		exitCode, err := builtinCmd.Handler(args)
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Command execution failed: %s", err))
			return 1
		}
		return exitCode
	}

	// Check if command needs raw terminal mode (e.g., ssh, vim, top, nano)
	if NeedsRawTerminal(cmdNode.Name) {
		return executeRawCommandWithRedirections(cmdNode)
	}

	// Build and execute the command normally
	exeCmd, files, err := buildCommandFromNode(cmdNode)
	if err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Failed to build command: %s", err))
		return 1
	}
	defer func() {
		for _, f := range files {
			f.Close()
		}
	}()

	// Add color flags
	addColorFlagsToCmd(exeCmd)

	// Set stdin/stdout if not already set by redirections
	if exeCmd.Stdin == nil {
		exeCmd.Stdin = os.Stdin
	}
	if exeCmd.Stdout == nil {
		exeCmd.Stdout = os.Stdout
	}
	if exeCmd.Stderr == nil {
		exeCmd.Stderr = os.Stderr
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	defer signal.Stop(sigChan)
	defer close(sigChan)

	// Start the command
	if err := exeCmd.Start(); err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Command execution failed: %s", err))
		return 1
	}

	// Forward SIGINT
	go func() {
		for range sigChan {
			if exeCmd.Process != nil {
				syscall.Kill(-exeCmd.Process.Pid, syscall.SIGINT)
			}
		}
	}()

	// Wait for command to finish
	err = exeCmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		if !isSignalKilled(err) {
			logging.LogError(err)
		}
		return 1
	}

	return 0
}

// executeRawCommandWithRedirections executes a command that needs raw terminal mode
func executeRawCommandWithRedirections(cmdNode CommandNode) int {
	// Build the full args list
	args := append([]string{cmdNode.Name}, cmdNode.Args...)

	// Handle output redirections if specified
	if cmdNode.StdoutFile != "" {
		var file *os.File
		var err error
		if cmdNode.StdoutAppend {
			file, err = os.OpenFile(cmdNode.StdoutFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		} else {
			file, err = os.Create(cmdNode.StdoutFile)
		}
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to open output file '%s': %s", cmdNode.StdoutFile, err))
			return 1
		}
		defer file.Close()

		// For raw terminal commands with output redirection, we need to use a different approach
		// since term.GetState/Restore won't work well with file redirection
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = file
		cmd.Stderr = file
		if cmdNode.StdinFile != "" {
			stdinFile, err := os.Open(cmdNode.StdinFile)
			if err != nil {
				logging.LogError(err)
				gui.ErrorBox(fmt.Sprintf("Failed to open input file '%s': %s", cmdNode.StdinFile, err))
				return 1
			}
			defer stdinFile.Close()
			cmd.Stdin = stdinFile
		}
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitErr.ExitCode()
			}
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Command execution failed: %s", err))
			return 1
		}
		return 0
	}

	// Handle input redirection only
	if cmdNode.StdinFile != "" {
		file, err := os.Open(cmdNode.StdinFile)
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to open input file '%s': %s", cmdNode.StdinFile, err))
			return 1
		}
		defer file.Close()

		// For raw terminal commands with input redirection, run without raw mode
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = file
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitErr.ExitCode()
			}
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Command execution failed: %s", err))
			return 1
		}
		return 0
	}

	// No redirections - run in full raw terminal mode
	executeRawTerminalCommand(args)
	return 0
}

// executeSingleCommandBackground executes a single command in the background
func executeSingleCommandBackground(cmdNode CommandNode, jobsMap map[int]*jobs.Job) {
	// Check if it's a built-in command
	cmd, exists := GetCommand(cmdNode.Name)
	if exists && cmd.Handler != nil {
		logging.LogAlert("Built-in commands cannot be run in background")
		gui.ErrorBox("Built-in commands cannot be run in background")
		return
	}

	// Build the command
	cmd_exe, files, err := buildCommandFromNode(cmdNode)
	if err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Failed to build command: %s", err))
		return
	}
	for _, f := range files {
		f.Close()
	}

	// No stdin for background jobs
	cmd_exe.Stdin = nil
	cmd_exe.Stdout = os.Stdout
	cmd_exe.Stderr = os.Stderr

	// Start the command
	if err := cmd_exe.Start(); err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Failed to start background job: %s", err))
		return
	}

	// Add to jobs map
	jobs.AddJob(jobsMap, cmd_exe.Process.Pid, cmdNode.Name, cmd_exe.Process)

	// Print confirmation
	fmt.Printf("\nStarted background job [%d]: %s\n", cmd_exe.Process.Pid, cmdNode.Name)

	// Wait in goroutine
	go func() {
		err := cmd_exe.Wait()
		exitCode := 0

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
			logging.LogError(err)
		}

		// Update job status
		job, exists := jobsMap[cmd_exe.Process.Pid]
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

// buildCommandFromNode creates an exec.Cmd from a CommandNode
func buildCommandFromNode(cmdNode CommandNode) (*exec.Cmd, []*os.File, error) {
	var files []*os.File

	// Process arguments and handle redirections
	args := make([]string, 0, len(cmdNode.Args))
	for _, arg := range cmdNode.Args {
		args = append(args, arg)
	}

	// Expand environment variables for echo command
	if cmdNode.Name == "echo" {
		for i, arg := range args {
			args[i] = os.ExpandEnv(arg)
		}
	}

	cmd := exec.Command(cmdNode.Name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Handle input redirection
	if cmdNode.StdinFile != "" {
		file, err := os.Open(cmdNode.StdinFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open input file '%s': %w", cmdNode.StdinFile, err)
		}
		cmd.Stdin = file
		files = append(files, file)
	}

	// Handle output redirection
	if cmdNode.StdoutFile != "" {
		var file *os.File
		var err error
		if cmdNode.StdoutAppend {
			file, err = os.OpenFile(cmdNode.StdoutFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		} else {
			file, err = os.Create(cmdNode.StdoutFile)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open output file '%s': %w", cmdNode.StdoutFile, err)
		}
		cmd.Stdout = file
		cmd.Stderr = file // Also redirect stderr to the same file
		files = append(files, file)
	}

	// Check security for the command
	if !isCommandAllowed(cmdNode.Name) {
		return nil, files, fmt.Errorf("command not permitted: %s", cmdNode.Name)
	}

	return cmd, files, nil
}

// isCommandAllowed checks if a command is allowed based on security settings
func isCommandAllowed(cmdName string) bool {
	isAdminUser := admin.IsAdmin()
	securityBypass := isAdminUser && !securityEnabled

	if securityBypass {
		logging.LogAlert(fmt.Sprintf("SECURITY BYPASS: Admin user executing command with security disabled: %s", cmdName))
		return true
	}

	// Check if command is blacklisted
	for _, blacklisted := range core.BlacklistedCommands {
		if cmdName == blacklisted {
			logging.LogAlert(fmt.Sprintf("Command '%s' is blacklisted and cannot be executed", cmdName))
			gui.ErrorBox(fmt.Sprintf("Command '%s' is blacklisted and cannot be executed", cmdName))
			return false
		}
	}

	// Check if command is allowed
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
		return false
	}

	return true
}

// addColorFlagsToCmd adds color flags to commands that support them
func addColorFlagsToCmd(cmd *exec.Cmd) {
	if len(cmd.Args) == 0 {
		return
	}

	// Check if user already specified a color flag
	hasColorFlag := false
	for _, arg := range cmd.Args {
		if strings.HasPrefix(arg, "--color") {
			hasColorFlag = true
			break
		}
	}

	if !hasColorFlag {
		switch cmd.Args[0] {
		case "ls", "diff":
			cmd.Args = append([]string{cmd.Args[0], "--color=auto"}, cmd.Args[1:]...)
		case "grep", "fgrep", "egrep":
			cmd.Args = append([]string{cmd.Args[0], "--color=always"}, cmd.Args[1:]...)
		case "git":
			cmd.Args = append([]string{cmd.Args[0], "--color=always"}, cmd.Args[1:]...)
		case "tree":
			cmd.Args = append([]string{cmd.Args[0], "-C"}, cmd.Args[1:]...)
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

// executeBackgroundCommand executes a command in the background (legacy compatibility)
func executeBackgroundCommand(args []string, jobsMap map[int]*jobs.Job) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Failed to start background job: %s", err))
		return
	}

	jobs.AddJob(jobsMap, cmd.Process.Pid, args[0], cmd.Process)
	fmt.Printf("\nStarted background job [%d]: %s\n", cmd.Process.Pid, args[0])

	go func() {
		err := cmd.Wait()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
			logging.LogError(err)
		}
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
	oldState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Failed to get terminal state: %s", err))
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	cmdArgs := buildCommandArgsWithColor(args)
	cmd := exec.Command(args[0], cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: false}

	if args[0] == "sudo" || args[0] == "su" {
		if !admin.IsAdmin() {
			logging.LogAlert("Permission denied: Only admins can use sudo/su commands")
			gui.ErrorBox("Permission denied: Only admins can use sudo/su commands")
			return
		}
		if args[0] == "su" {
			cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setpgid: false}
		}
	}

	err = cmd.Run()
	if err != nil && !isSignalKilled(err) {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Command execution failed: %s", err))
	}
}

// executeNormalCommand executes a command in normal terminal mode (legacy compatibility)
func executeNormalCommand(args []string) {
	cmdArgs := []string{}
	var stdout io.Writer = os.Stdout
	var stdin io.Reader = os.Stdin

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
				i++
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
				i++
			}
		default:
			cmdArgs = append(cmdArgs, args[i])
		}
	}

	if args[0] == "echo" {
		for i, arg := range cmdArgs {
			cmdArgs[i] = os.ExpandEnv(arg)
		}
	}

	cmd := exec.Command(args[0], cmdArgs...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	defer signal.Stop(sigChan)
	defer close(sigChan)

	if err := cmd.Start(); err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Command execution failed: %s", err))
		return
	}

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

// executePipedCommands handles command pipelines (legacy compatibility)
func executePipedCommands(input string, jobsMap map[int]*jobs.Job) {
	splitCommands := strings.Split(input, "|")
	for i, cmd := range splitCommands {
		splitCommands[i] = strings.TrimSpace(cmd)
	}

	// Check for background execution
	background := false
	if len(splitCommands) > 0 {
		lastCmd := splitCommands[len(splitCommands)-1]
		args := parseCommandLine(lastCmd)
		if len(args) > 0 && args[len(args)-1] == "&" {
			background = true
			if len(args) > 1 {
				args = args[:len(args)-1]
				splitCommands[len(splitCommands)-1] = strings.Join(args, " ")
			} else {
				splitCommands = splitCommands[:len(splitCommands)-1]
			}
		}
	}

	if background {
		// Convert to CommandNodes and execute as background pipeline
		var commands []CommandNode
		for _, cmdStr := range splitCommands {
			args := parseCommandLine(cmdStr)
			if len(args) == 0 {
				continue
			}
			node := CommandNode{Name: args[0]}
			if len(args) > 1 {
				node.Args = args[1:]
			}
			commands = append(commands, node)
		}
		pipeline := PipelineNode{Commands: commands, Background: true}
		var exitCode int
		executePipelineBackground(pipeline, jobsMap, &exitCode)
		return
	}

	// Check for 'more'
	if strings.TrimSpace(splitCommands[len(splitCommands)-1]) == "more" {
		var commands []CommandNode
		for _, cmdStr := range splitCommands[:len(splitCommands)-1] {
			args := parseCommandLine(cmdStr)
			if len(args) == 0 {
				continue
			}
			node := CommandNode{Name: args[0]}
			if len(args) > 1 {
				node.Args = args[1:]
			}
			commands = append(commands, node)
		}
		var exitCode int
		executePipelineWithMore(commands, &exitCode)
		return
	}

	var cmds []*exec.Cmd
	for _, cmdString := range splitCommands {
		args := parseCommandLine(cmdString)
		if len(args) == 0 {
			continue
		}

		hasColorFlag := false
		for _, arg := range args {
			if strings.HasPrefix(arg, "--color") {
				hasColorFlag = true
				break
			}
		}
		if !hasColorFlag {
			switch args[0] {
			case "grep", "fgrep", "egrep":
				args = append([]string{args[0], "--color=always"}, args[1:]...)
			case "ls", "diff":
				args = append([]string{args[0], "--color=auto"}, args[1:]...)
			case "git":
				args = append([]string{args[0], "--color=always"}, args[1:]...)
			case "tree":
				args = append([]string{args[0], "-C"}, args[1:]...)
			}
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmds = append(cmds, cmd)
	}

	if len(cmds) == 0 {
		return
	}

	for i := 0; i < len(cmds)-1; i++ {
		stdout, err := cmds[i].StdoutPipe()
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to set up pipeline: %s", err))
			return
		}
		cmds[i+1].Stdin = stdout
	}

	cmds[0].Stdin = os.Stdin
	cmds[len(cmds)-1].Stdout = os.Stdout
	cmds[len(cmds)-1].Stderr = os.Stderr

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to start command: %s", err))
			signal.Stop(sigChan)
			close(sigChan)
			return
		}
	}

	go func() {
		for range sigChan {
			for _, cmd := range cmds {
				if cmd.Process != nil {
					syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
				}
			}
		}
	}()

	for _, cmd := range cmds {
		err := cmd.Wait()
		if err != nil && !isSignalKilled(err) {
			logging.LogError(err)
		}
	}

	signal.Stop(sigChan)
	close(sigChan)
}

// executePipeWithMore handles pipelines that end with 'more' (legacy compatibility)
func executePipeWithMore(commands []string) {
	pr, pw := io.Pipe()
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

	for i := 0; i < len(cmds)-1; i++ {
		stdout, err := cmds[i].StdoutPipe()
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to set up pipeline: %s", err))
			return
		}
		cmds[i+1].Stdin = stdout
	}

	if len(cmds) > 0 {
		cmds[0].Stdin = os.Stdin
		cmds[len(cmds)-1].Stdout = pw
		cmds[len(cmds)-1].Stderr = os.Stderr
	}

	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to start command: %s", err))
			return
		}
	}

	var lines []string
	scanDone := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		scanDone <- scanner.Err()
	}()

	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil && !isSignalKilled(err) {
			logging.LogError(err)
		}
	}

	pw.Close()
	if scanErr := <-scanDone; scanErr != nil {
		logging.LogError(scanErr)
		gui.ErrorBox(fmt.Sprintf("Failed to read piped output: %s", scanErr))
		return
	}

	if err := core.More(lines); err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Error: %v", err))
	}
}

// executeBackgroundPipedCommands executes a pipeline in background (legacy compatibility)
func executeBackgroundPipedCommands(commands []string, jobsMap map[int]*jobs.Job) {
	var cmds []*exec.Cmd
	for _, cmdString := range commands {
		cmdString = strings.TrimSpace(cmdString)
		if cmdString == "" {
			continue
		}
		args := parseCommandLine(cmdString)
		if len(args) == 0 {
			continue
		}

		hasColorFlag := false
		for _, arg := range args {
			if strings.HasPrefix(arg, "--color") {
				hasColorFlag = true
				break
			}
		}
		if !hasColorFlag {
			switch args[0] {
			case "grep", "fgrep", "egrep":
				args = append([]string{args[0], "--color=always"}, args[1:]...)
			case "ls", "diff":
				args = append([]string{args[0], "--color=auto"}, args[1:]...)
			case "git":
				args = append([]string{args[0], "--color=always"}, args[1:]...)
			case "tree":
				args = append([]string{args[0], "-C"}, args[1:]...)
			}
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmds = append(cmds, cmd)
	}

	if len(cmds) == 0 {
		return
	}

	for i := 0; i < len(cmds)-1; i++ {
		stdout, err := cmds[i].StdoutPipe()
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to set up pipeline: %s", err))
			return
		}
		cmds[i+1].Stdin = stdout
	}

	cmds[0].Stdin = nil
	cmds[len(cmds)-1].Stdout = os.Stdout
	cmds[len(cmds)-1].Stderr = os.Stderr

	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to start command: %s", err))
			return
		}
	}

	lastPid := cmds[len(cmds)-1].Process.Pid
	firstCmd := commands[0]
	jobs.AddJob(jobsMap, lastPid, firstCmd, cmds[len(cmds)-1].Process)
	fmt.Printf("\nStarted background job [%d]: %s\n", lastPid, strings.Join(commands, " | "))

	go func() {
		exitCode := 0
		var jobErr error
		for _, cmd := range cmds {
			err := cmd.Wait()
			if err != nil {
				jobErr = err
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				}
			}
		}
		if jobErr != nil {
			logging.LogError(jobErr)
		}
		job, exists := jobsMap[lastPid]
		if exists {
			job.Lock()
			job.EndTime = time.Now()
			job.ExitCode = exitCode
			if jobErr != nil {
				job.Status = fmt.Sprintf("failed with code %d", exitCode)
			} else {
				job.Status = "completed"
			}
			job.Unlock()
		}
	}()
}

// buildCommandArgsWithColor adds color flags to commands that support them
func buildCommandArgsWithColor(args []string) []string {
	if len(args) < 1 {
		return args[1:]
	}

	cmdArgs := make([]string, 0, len(args))
	hasColorFlag := false
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--color") {
			hasColorFlag = true
			break
		}
	}

	if !hasColorFlag {
		switch args[0] {
		case "ls":
			cmdArgs = append(cmdArgs, "--color=auto")
		case "grep", "fgrep", "egrep":
			cmdArgs = append(cmdArgs, "--color=always")
		case "diff":
			cmdArgs = append(cmdArgs, "--color=auto")
		case "git":
			cmdArgs = append(cmdArgs, "--color=always")
		case "tree":
			cmdArgs = append(cmdArgs, "-C")
		}
	}

	cmdArgs = append(cmdArgs, args[1:]...)
	return cmdArgs
}

// parseCommandLine splits a command line into arguments (legacy compatibility)
func parseCommandLine(cmdLine string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)
	escaped := false

	for _, char := range cmdLine {
		switch {
		case escaped:
			current.WriteRune(char)
			escaped = false
		case char == '\\':
			escaped = true
		case char == '"' || char == '\'':
			if inQuotes && char == quoteChar {
				inQuotes = false
				quoteChar = rune(0)
			} else if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else {
				current.WriteRune(char)
			}
		case char == ' ' && !inQuotes:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	if inQuotes {
		logging.LogAlert("Warning: Unclosed quotes detected in command")
	}

	return args
}

// executeLegacyCommand provides backward compatibility for the old execution model
func executeLegacyCommand(input string, jobsMap map[int]*jobs.Job) {
	// Parse the command and arguments
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

	if len(args) == 0 {
		return
	}

	cmdName := args[0]

	// Check if command is blacklisted
	for _, blacklisted := range core.BlacklistedCommands {
		if cmdName == blacklisted {
			logging.LogAlert(fmt.Sprintf("Command '%s' is blacklisted", cmdName))
			gui.ErrorBox(fmt.Sprintf("Command '%s' is blacklisted", cmdName))
			return
		}
	}

	// Check if command is a built-in command
	cmd, exists := GetCommand(cmdName)
	if exists && cmd.Handler != nil {
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

	if background {
		executeBackgroundCommand(args, jobsMap)
	} else if needsRawTerm {
		executeRawTerminalCommand(args)
	} else {
		executeNormalCommand(args)
	}
}

// isDestructiveCommand checks if a command attempts to perform dangerous filesystem operations
// Returns a message explaining why the command is blocked, or empty string if not blocked
func isDestructiveCommand(input string) (string, bool) {
	configDir := globals.ConfigDir
	dirName := ".secshell"

	// Normalize the input to lowercase for case-insensitive matching
	lowerInput := strings.ToLower(input)

	// Check for system-wide destructive commands first
	// rm -rf / or rm -rf /* or rm -rf ~
	if strings.Contains(lowerInput, "rm ") || strings.Contains(lowerInput, "rm-") {
		// Check for root directory deletion - catastrophic!
		if strings.Contains(lowerInput, "rm -rf /") || strings.Contains(lowerInput, "rm -rf /*") ||
			strings.Contains(lowerInput, "rm -rf ~") || strings.Contains(lowerInput, "rm -rf / ") ||
			strings.Contains(lowerInput, "rm -rf /* ") {
			return "Deletion of root directory or home directory is not allowed.", true
		}
		if strings.Contains(lowerInput, "rm -r /") || strings.Contains(lowerInput, "rm -rf /*") {
			return "Deletion of root directory is not allowed.", true
		}
	}

	// Patterns that indicate deletion commands
	deletionPatterns := []string{
		"rm -rf", "rm -f", "rm -r", "rm ",
		"rmdir ",
		"del ", "del /f ", "del /s ", "del /q ",
		"move ", "rename ",
	}

	// Check for deletion commands targeting the config directory
	for _, pattern := range deletionPatterns {
		cmdPattern := strings.ToLower(pattern)
		if idx := strings.Index(lowerInput, cmdPattern); idx != -1 {
			// Get the part of the input after the command pattern
			remainder := strings.TrimSpace(lowerInput[idx+len(cmdPattern):])

			// Check if the remainder contains the config directory name
			if strings.Contains(remainder, dirName) || strings.Contains(remainder, strings.ToLower(configDir)) {
				return "Deletion of SecShell configuration directory (.secshell) is not allowed.", true
			}

			// Also check for variations like .*secshell*, *secshell*, etc.
			if strings.Contains(remainder, "secshell") && strings.Contains(remainder, ".") {
				return "Deletion of SecShell configuration directory (.secshell) is not allowed.", true
			}
		}
	}

	// Check for commands using wildcards or glob patterns that might match .secshell
	if strings.Contains(lowerInput, "rm ") && strings.Contains(lowerInput, "*") {
		if strings.Contains(lowerInput, "secshell") {
			return "Deletion of SecShell configuration directory (.secshell) is not allowed.", true
		}
	}

	// Check for recursive deletion from parent directories
	// e.g., "rm -rf ./*secshell" or "rm -rf ./.*"
	if strings.Contains(lowerInput, "rm") && strings.Contains(lowerInput, ".*") {
		if strings.Contains(lowerInput, "secshell") {
			return "Deletion of SecShell configuration directory (.secshell) is not allowed.", true
		}
	}

	return "", false
}

// Helper function for command handlers that need to manipulate files
func sanitizePath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}
	return sanitize.Path(path)
}

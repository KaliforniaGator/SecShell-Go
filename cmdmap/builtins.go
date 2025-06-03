package cmdmap

import (
	"secshell/admin"
	"secshell/colors"
	"secshell/core"
	"secshell/download"
	"secshell/env"
	"secshell/globals"
	"secshell/help"
	"secshell/history"
	"secshell/logging"
	"secshell/pentest"
	"secshell/services"
	"secshell/tools"
	"secshell/tools/editor"
	filemanager "secshell/tools/file-manager"
	"secshell/ui"
	"secshell/ui/gui"
	"secshell/update"

	"fmt"
	"strconv"
	"strings"
	"time"
)

// RegisterBuiltInCommandHandlers registers handlers for all built-in commands
func RegisterBuiltInCommandHandlers() {
	// System commands
	registerCommandHandler("help", handleHelp)
	registerCommandHandler("exit", handleExit)
	registerCommandHandler("time", handleTime)
	registerCommandHandler("date", handleDate)
	registerCommandHandler("cd", handleCd)
	registerCommandHandler("--version", handleVersion)
	registerCommandHandler("--update", handleUpdate)
	registerCommandHandler("features", handleFeatures)
	registerCommandHandler("changelog", handleChangelog)
	registerCommandHandler("more", handleMore)
	registerCommandHandler("colors", handleColors)

	// Environment commands
	registerCommandHandler("export", handleExport)
	registerCommandHandler("env", handleEnv)
	registerCommandHandler("unset", handleUnset)

	// Security commands
	registerCommandHandler("allowed", handleAllowed)
	registerCommandHandler("blacklist", handleBlacklist)
	registerCommandHandler("whitelist", handleWhitelist)
	registerCommandHandler("edit-blacklist", handleEditBlacklist)
	registerCommandHandler("edit-whitelist", handleEditWhitelist)
	registerCommandHandler("reload-blacklist", handleReloadBlacklist)
	registerCommandHandler("reload-whitelist", handleReloadWhitelist)
	registerCommandHandler("toggle-security", handleToggleSecurity)

	// Process commands
	registerCommandHandler("jobs", handleJobs)
	registerCommandHandler("services", handleServices)

	// Network/Pentest commands
	registerCommandHandler("download", handleDownload)
	registerCommandHandler("portscan", handlePortscan)
	registerCommandHandler("hostscan", handleHostscan)
	registerCommandHandler("webscan", handleWebscan)
	registerCommandHandler("payload", handlePayload)
	registerCommandHandler("session", handleSession)

	// Utility commands
	registerCommandHandler("logs", handleLogs)
	registerCommandHandler("history", handleHistory)
	registerCommandHandler("base64", handleBase64)
	registerCommandHandler("hex", handleHex)
	registerCommandHandler("urlencode", handleUrlEncode)
	registerCommandHandler("url", handleUrlEncode)
	registerCommandHandler("binary", handleBinary)
	registerCommandHandler("hash", handleHash)
	registerCommandHandler("extract-strings", handleExtractStrings)
	registerCommandHandler("edit", handleEdit)
	registerCommandHandler("files", handleFiles)

	// UI commands
	registerCommandHandler("prompt", handlePrompt)
	registerCommandHandler("edit-prompt", handleEditPrompt)
	registerCommandHandler("reload-prompt", handleReloadPrompt)
}

// registerCommandHandler registers a handler for a command
func registerCommandHandler(name string, handler CommandHandler) {
	cmd, exists := GlobalCommandMap[name]
	if exists {
		cmd.Handler = handler
		GlobalCommandMap[name] = cmd
	}
}

// Command handler implementations

// handleHelp handles the help command
func handleHelp(args []string) (int, error) {
	if len(args) > 1 {
		if args[1] == "-i" || args[1] == "--interactive" {
			help.InteractiveHelpApp()
		} else {
			help.DisplayHelp(args[1])
		}
	} else {
		help.DisplayHelp()
	}
	return 0, nil
}

// handleExit handles the exit command
func handleExit(args []string) (int, error) {
	// This will be handled in secshell.go by setting the running flag to false
	return 0, nil
}

// handleTime handles the time command
func handleTime(args []string) (int, error) {
	now := time.Now()
	gui.TitleBox(fmt.Sprintf("Current time: %s", now.Format("3:04 PM")))
	return 0, nil
}

// handleDate handles the date command
func handleDate(args []string) (int, error) {
	now := time.Now()
	gui.TitleBox(fmt.Sprintf("Current date: %s", now.Format("02-Jan-2006")))
	return 0, nil
}

// handleCd handles the cd command
func handleCd(args []string) (int, error) {
	core.ChangeDirectory(args)
	return 0, nil
}

// handleVersion handles the --version command
func handleVersion(args []string) (int, error) {
	// Assuming globals.VersionPath is accessible
	update.DisplayVersion(globals.VersionPath)
	return 0, nil
}

// handleUpdate handles the --update command
func handleUpdate(args []string) (int, error) {
	update.UpdateSecShell(admin.IsAdmin(), globals.VersionPath)
	return 0, nil
}

// handleFeatures handles the features command
func handleFeatures(args []string) (int, error) {
	help.DisplayFeatures()
	return 0, nil
}

// handleChangelog handles the changelog command
func handleChangelog(args []string) (int, error) {
	update.DisplayChangelog()
	return 0, nil
}

// handleColors handles the colors command
func handleColors(args []string) (int, error) {
	colors.DisplayColors()
	return 0, nil
}

// handleExport handles the export command
func handleExport(args []string) (int, error) {
	env.ExportVariable(args)
	return 0, nil
}

// handleEnv handles the env command
func handleEnv(args []string) (int, error) {
	env.ListEnvVariables()
	return 0, nil
}

// handleUnset handles the unset command
func handleUnset(args []string) (int, error) {
	env.UnsetEnvVariable(args)
	return 0, nil
}

// handleAllowed handles the allowed command
func handleAllowed(args []string) (int, error) {
	if len(args) > 1 {
		switch args[1] {
		case "dirs":
			gui.TitleBox("Allowed Directories")
			for _, dir := range GetAllowedDirs() {
				fmt.Println(" - " + dir)
			}
		case "commands":
			gui.TitleBox("Allowed Commands")
			commands := GetAllowedCommands()
			for _, cmd := range commands {
				fmt.Println(" - " + cmd)
			}
		case "bins":
			gui.TitleBox("Allowed Binaries")
			bins := GetExternalCommands()
			for _, bin := range bins {
				fmt.Println(" - " + bin)
			}
		case "builtins":
			gui.TitleBox("Built-in Commands")
			builtins := GetBuiltInCommands()
			for _, cmd := range builtins {
				fmt.Println(" - " + cmd)
			}
		case "all":
			gui.TitleBox("All Allowed")
			fmt.Println("Allowed Directories:")
			ui.NewLine()
			for _, dir := range GetAllowedDirs() {
				fmt.Println(" - " + dir)
			}
			fmt.Println("\nAvailable Commands:")
			ui.NewLine()
			for _, cmd := range GetAllowedCommands() {
				fmt.Println(" - " + cmd)
			}
			fmt.Println("\nBuilt-In Commands:")
			ui.NewLine()
			for _, cmd := range GetBuiltInCommands() {
				fmt.Println(" - " + cmd)
			}
		}
	} else {
		logging.LogAlert("Usage: allowed <dirs|commands|bins|builtins|all>")
		gui.ErrorBox("Usage: allowed <dirs|commands|bins|builtins|all>")
	}
	return 0, nil
}

// GetAllowedDirs returns a list of allowed directories
func GetAllowedDirs() []string {
	// Use trusted dirs from globals
	return globals.TrustedDirs
}

// GetAllowedCommands returns a list of allowed commands
func GetAllowedCommands() []string {
	var commands []string
	for name, cmd := range GlobalCommandMap {
		if cmd.Category != CategoryExternal {
			commands = append(commands, name)
		}
	}
	return commands
}

// GetExternalCommands returns a list of external commands
func GetExternalCommands() []string {
	var commands []string
	for name, cmd := range GlobalCommandMap {
		if cmd.Category == CategoryExternal {
			commands = append(commands, name)
		}
	}
	return commands
}

// GetBuiltInCommands returns a list of built-in commands
func GetBuiltInCommands() []string {
	var commands []string
	for name, cmd := range GlobalCommandMap {
		if cmd.Category == CategorySystem ||
			cmd.Category == CategorySecurity ||
			cmd.Category == CategoryUtility {
			commands = append(commands, name)
		}
	}
	return commands
}

// handleBlacklist handles the blacklist command
func handleBlacklist(args []string) (int, error) {
	core.ListBlacklistCommands(globals.BlacklistPath)
	return 0, nil
}

// handleWhitelist handles the whitelist command
func handleWhitelist(args []string) (int, error) {
	core.ListWhitelistCommands()
	return 0, nil
}

// handleEditBlacklist handles the edit-blacklist command
func handleEditBlacklist(args []string) (int, error) {
	if !admin.IsAdmin() {
		logging.LogAlert("Permission denied: Admin privileges required.")
		gui.ErrorBox("Permission denied: Admin privileges required.")
		return 1, fmt.Errorf("permission denied")
	}

	core.EditBlacklist(globals.BlacklistPath)
	return 0, nil
}

// handleEditWhitelist handles the edit-whitelist command
func handleEditWhitelist(args []string) (int, error) {
	if !admin.IsAdmin() {
		logging.LogAlert("Permission denied: Admin privileges required.")
		gui.ErrorBox("Permission denied: Admin privileges required.")
		return 1, fmt.Errorf("permission denied")
	}

	core.EditWhitelist(globals.WhitelistPath)
	return 0, nil
}

// handleReloadBlacklist handles the reload-blacklist command
func handleReloadBlacklist(args []string) (int, error) {
	if !admin.IsAdmin() {
		logging.LogAlert("Permission denied: Admin privileges required.")
		gui.ErrorBox("Permission denied: Admin privileges required.")
		return 1, fmt.Errorf("permission denied")
	}

	// This will be implemented in secshell.go
	return 0, nil
}

// handleReloadWhitelist handles the reload-whitelist command
func handleReloadWhitelist(args []string) (int, error) {
	if !admin.IsAdmin() {
		logging.LogAlert("Permission denied: Admin privileges required.")
		gui.ErrorBox("Permission denied: Admin privileges required.")
		return 1, fmt.Errorf("permission denied")
	}

	// This will be implemented in secshell.go
	return 0, nil
}

// handleToggleSecurity handles the toggle-security command
func handleToggleSecurity(args []string) (int, error) {
	// Currently being handled in secshell.go and execute.go
	return 0, nil
}

// handleJobs handles the jobs command
func handleJobs(args []string) (int, error) {
	// Currently being handled in secshell.go
	return 0, nil
}

// handleServices handles the services command
func handleServices(args []string) (int, error) {
	// Check if user has admin privileges
	isAdmin := admin.IsAdmin()
	if !isAdmin && globals.RestrictedCommands["services"] {
		logging.LogAlert("Permission denied: 'services' requires admin privileges")
		gui.ErrorBox("Permission denied: 'services' requires admin privileges")
		return 1, fmt.Errorf("permission denied")
	}

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
	return 0, nil
}

// handleDownload handles the download command
func handleDownload(args []string) (int, error) {
	download.DownloadFiles(args)
	return 0, nil
}

// handlePortscan handles the portscan command
func handlePortscan(args []string) (int, error) {
	if len(args) < 2 {
		gui.ErrorBox("Usage: portscan [-p ports] [-udp] [-t timing] [-v] [-j|-html] [-o file] [-syn] [-os] [-e] <target>")
		return 1, fmt.Errorf("invalid usage")
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
		return 1, fmt.Errorf("no target specified")
	}

	pentest.RunPortScan(target, portRange, options)
	return 0, nil
}

// handleHostscan handles the hostscan command
func handleHostscan(args []string) (int, error) {
	if len(args) < 2 {
		gui.ErrorBox("Usage: hostscan <network-range>")
		return 1, fmt.Errorf("invalid usage")
	}

	pentest.RunHostDiscovery(args[1])
	return 0, nil
}

// handleWebscan handles the webscan command
func handleWebscan(args []string) (int, error) {
	if len(args) < 2 {
		help.DisplayHelp("webscan")
		return 1, fmt.Errorf("invalid usage")
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
		return 1, fmt.Errorf("no target specified")
	}

	pentest.WebScan(target, options)
	return 0, nil
}

// handlePayload handles the payload command
func handlePayload(args []string) (int, error) {
	if len(args) < 3 {
		gui.ErrorBox("Usage: payload <ip-address> <port>")
		return 1, fmt.Errorf("invalid usage")
	}

	pentest.GenerateReverseShellPayload(args[1], args[2])
	return 0, nil
}

// handleSession handles the session command
func handleSession(args []string) (int, error) {
	if len(args) < 2 {
		pentest.ListSessions()
		return 0, nil
	}

	switch args[1] {
	case "-l":
		pentest.ListSessions()
	case "-i":
		if len(args) < 3 {
			gui.ErrorBox("Usage: session -i <id>")
			return 1, fmt.Errorf("invalid usage")
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox("Invalid session ID")
			return 1, err
		}
		pentest.InteractWithSession(id)
	case "-c":
		if len(args) < 3 {
			gui.ErrorBox("Usage: session -c <port>")
			return 1, fmt.Errorf("invalid usage")
		}
		port := args[2]
		id := pentest.ListenForConnections(port)
		if id != -1 {
			gui.AlertBox(fmt.Sprintf("Created session %d", id))
		}
	case "-k":
		if len(args) < 3 {
			gui.ErrorBox("Usage: session -k <id>")
			return 1, fmt.Errorf("invalid usage")
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox("Invalid session ID")
			return 1, err
		}
		pentest.CloseSession(id)
	default:
		gui.ErrorBox("Unknown session command. Use -l, -i, -c, or -k")
		return 1, fmt.Errorf("invalid option: %s", args[1])
	}

	return 0, nil
}

// handleLogs handles the logs command
func handleLogs(args []string) (int, error) {
	if len(args) < 2 {
		logging.LogAlert("Usage: logs list")
		gui.ErrorBox("Usage: logs list")
		return 1, fmt.Errorf("invalid usage")
	}

	switch args[1] {
	case "list":
		err := logging.PrintLog()
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox("Failed to read log file")
			return 1, err
		}
	default:
		logging.LogAlert("Invalid logs option. Use 'list'.")
		gui.ErrorBox("Invalid logs option. Use 'list'.")
		return 1, fmt.Errorf("invalid option: %s", args[1])
	}

	return 0, nil
}

// handleHistory handles the history command
func handleHistory(args []string) (int, error) {
	if len(args) == 1 {
		history.DisplayHistory(core.History) // Display command history
	} else {
		switch args[1] {
		case "-s":
			if len(args) < 3 {
				logging.LogAlert("Usage: history -s <query>")
				gui.ErrorBox("Usage: history -s <query>")
				return 1, fmt.Errorf("invalid usage")
			}
			history.SearchHistory(core.History, strings.Join(args[2:], " ")) // Search history for the given query
		case "-i":
			history.InteractiveHistorySearch(core.History, nil) // Run interactive history search
		case "clear":
			core.ClearHistory(globals.HistoryPath)
			gui.AlertBox("History cleared")
		default:
			logging.LogAlert("Invalid history option. Use -s for search or -i for interactive mode.")
			gui.ErrorBox("Invalid history option. Use -s for search or -i for interactive mode.")
			return 1, fmt.Errorf("invalid option")
		}
	}
	return 0, nil
}

// handleMore handles the more command
func handleMore(args []string) (int, error) {
	if len(args) < 2 {
		gui.ErrorBox("Usage: more <file> or command | more")
		return 1, fmt.Errorf("invalid usage")
	}

	err := core.RunMore(args[1:])
	if err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Error: %v", err))
		return 1, err
	}

	return 0, nil
}

// handleBase64 handles the base64 command
func handleBase64(args []string) (int, error) {
	err := tools.ExecuteEncodingCommand(args, tools.Base64Encoding)
	if err != nil {
		gui.ErrorBox(fmt.Sprintf("Base64 operation failed: %v", err))
		return 1, err
	}
	return 0, nil
}

// handleHex handles the hex command
func handleHex(args []string) (int, error) {
	err := tools.ExecuteEncodingCommand(args, tools.HexEncoding)
	if err != nil {
		gui.ErrorBox(fmt.Sprintf("Hex operation failed: %v", err))
		return 1, err
	}
	return 0, nil
}

// handleUrlEncode handles the urlencode/url command
func handleUrlEncode(args []string) (int, error) {
	err := tools.ExecuteEncodingCommand(args, tools.URLEncoding)
	if err != nil {
		gui.ErrorBox(fmt.Sprintf("URL encoding operation failed: %v", err))
		return 1, err
	}
	return 0, nil
}

// handleBinary handles the binary command
func handleBinary(args []string) (int, error) {
	err := tools.ExecuteEncodingCommand(args, tools.BinaryEncoding)
	if err != nil {
		gui.ErrorBox(fmt.Sprintf("Binary operation failed: %v", err))
		return 1, err
	}
	return 0, nil
}

// handleHash handles the hash command
func handleHash(args []string) (int, error) {
	if len(args) < 2 {
		gui.ErrorBox("Usage: hash <algorithm> <data>")
		return 1, fmt.Errorf("invalid usage")
	}

	result, err := tools.HashCommand(args[1:])
	if err != nil {
		gui.ErrorBox(fmt.Sprintf("Hash operation failed: %v", err))
		return 1, err
	}

	fmt.Println(result)
	return 0, nil
}

// handleExtractStrings handles the extract-strings command
func handleExtractStrings(args []string) (int, error) {
	if len(args) < 2 {
		gui.ErrorBox("Usage: extract-strings <file> [-n min-len]")
		return 1, fmt.Errorf("invalid usage")
	}

	err := tools.RunStringExtract(args[1:])
	if err != nil {
		gui.ErrorBox(fmt.Sprintf("String extraction failed: %v", err))
		return 1, err
	}

	return 0, nil
}

// handleEdit handles the edit command
func handleEdit(args []string) (int, error) {
	editor.EditCommand(args[1:])
	return 0, nil
}

// handleFiles handles the files command
func handleFiles(args []string) (int, error) {
	filemanager.FileManagerApp()
	return 0, nil
}

// handlePrompt handles the prompt command
func handlePrompt(args []string) (int, error) {
	if len(args) > 1 {
		if args[1] == "-r" || args[1] == "--reset" {
			ui.ResetPrompt()
			return 0, nil
		}
	}

	ui.DisplayPromptOptions()
	return 0, nil
}

// handleEditPrompt handles the edit-prompt command
func handleEditPrompt(args []string) (int, error) {
	editor.EditCommand([]string{globals.PromptConfigFile})
	return 0, nil
}

// handleReloadPrompt handles the reload-prompt command
func handleReloadPrompt(args []string) (int, error) {
	version := update.GetCurrentVersion(globals.VersionPath)
	latestVersion := update.GetLatestVersion()

	// Check if update is needed
	needsUpdate := false

	// Parse version strings
	currentParts := strings.Split(version, ".")
	latestParts := strings.Split(latestVersion, ".")

	// Validate version format
	if len(currentParts) == 3 && len(latestParts) == 3 {
		// Convert version parts to integers
		current := make([]int, 3)
		latest := make([]int, 3)

		valid := true
		for i := 0; i < 3; i++ {
			var err error
			current[i], err = strconv.Atoi(currentParts[i])
			if err != nil {
				logging.LogError(err)
				valid = false
				break
			}
			latest[i], err = strconv.Atoi(latestParts[i])
			if err != nil {
				logging.LogError(err)
				valid = false
				break
			}
		}

		if valid {
			// Compare versions
			needsUpdate = latest[0] > current[0] || // Major version
				(latest[0] == current[0] && latest[1] > current[1]) || // Minor version
				(latest[0] == current[0] && latest[1] == current[1] && latest[2] > current[2]) // Patch version
		}
	}

	// Reload the prompt
	ui.ReloadPrompt(version, needsUpdate)
	return 0, nil
}

package cmdmap

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"secshell/admin"
	"secshell/core"
	"secshell/globals"
	"secshell/help"
	"secshell/logging"
	"secshell/sanitize"
	"secshell/ui"
	"secshell/ui/gui"
	"strings"
)

// GlobalCommandMap holds all registered commands
var GlobalCommandMap CommandMap

// SpecialCommands that need raw terminal mode
var SpecialCommands = map[string]bool{
	"ssh":     true,
	"vim":     true,
	"vi":      true,
	"nano":    true,
	"pico":    true,
	"emacs":   true,
	"less":    true,
	"more":    true,
	"top":     true,
	"htop":    true,
	"nvim":    true,
	"python":  true,
	"python3": true,
	"ruby":    true,
	"node":    true,
	"mysql":   true,
	"psql":    true,
	"su":      true,
	"sudo":    true,
	"screen":  true,
	"tmux":    true,
	"man":     true,
	"info":    true,
}

// Variable to track security state - exported at package level for all cmdmap files
var securityEnabled = true

// InitCommandMap initializes the command mapping system
func InitCommandMap() {
	GlobalCommandMap = make(CommandMap)
	registerBuiltInCommands()
	registerAllowedCommands()
	syncWithWhitelist()
}

// registerBuiltInCommands registers all built-in commands
func registerBuiltInCommands() {
	// Combine built-in commands from globals and help
	builtInCommands := make(map[string]bool)

	// Add commands from globals.BuiltInCommands
	for _, cmd := range globals.BuiltInCommands {
		builtInCommands[cmd] = true
	}

	// Add commands from help.HelpCommands
	for _, cmd := range help.HelpCommands {
		builtInCommands[cmd] = true
	}

	// Register all the combined built-in commands
	for cmdName := range builtInCommands {
		// Get help topic if it exists
		helpTopic, exists := help.HelpTopics[cmdName]

		if !exists {
			// Create a default help topic
			helpTopic = help.HelpTopic{
				Command:     cmdName,
				Description: "Built-in command",
				Usage:       cmdName,
				Examples:    []string{cmdName},
				Category:    "System",
			}
		}

		// Create command with default handler (will be overridden in execute.go)
		cmd := Command{
			Name:        cmdName,
			Description: helpTopic.Description,
			Usage:       helpTopic.Usage,
			Examples:    helpTopic.Examples,
			Category:    CommandCategory(helpTopic.Category),
			TermMode:    ModeNormal,
			Admin:       false,
			AllowArgs:   true,
		}

		// Add to command map
		GlobalCommandMap[cmdName] = cmd
	}
}

// registerAllowedCommands scans allowed directories for executable commands
func registerAllowedCommands() {
	// Scan directories for executables
	for _, dir := range globals.TrustedDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*"))
		if err != nil {
			logging.LogError(err)
			continue
		}

		for _, file := range files {
			// Check if file is executable
			info, err := os.Stat(file)
			if err != nil || info.IsDir() {
				continue
			}

			// Check executable bit
			if info.Mode()&0111 == 0 {
				continue
			}

			// Get base name and sanitize
			baseName := filepath.Base(file)
			sanitizedName, err := sanitize.SanitizeFileName(baseName)
			if err != nil {
				continue
			}

			// Skip if already registered
			if _, exists := GlobalCommandMap[sanitizedName]; exists {
				continue
			}

			// Check if it's a special command that needs raw terminal
			termMode := ModeNormal
			if SpecialCommands[sanitizedName] {
				termMode = ModeRaw
			}

			// Create command
			cmd := Command{
				Name:        sanitizedName,
				Description: fmt.Sprintf("External command found in %s", dir),
				Usage:       sanitizedName,
				Examples:    []string{sanitizedName},
				Category:    CategoryExternal,
				TermMode:    termMode,
				Admin:       false,
				AllowArgs:   true,
			}

			// Add to command map
			GlobalCommandMap[sanitizedName] = cmd
		}
	}
}

// syncWithWhitelist ensures all mapped built-in commands are in the whitelist
func syncWithWhitelist() {
	whitelistedCommands := core.AllowedCommands
	needsUpdate := false

	// Only add built-in commands to the whitelist, not all external commands
	for cmdName, cmd := range GlobalCommandMap {
		// Only add built-in commands, not external commands from trusted dirs
		if cmd.Category != CategoryExternal {
			found := false
			for _, whitelisted := range whitelistedCommands {
				if cmdName == whitelisted {
					found = true
					break
				}
			}

			if !found && cmdName != "" {
				// Add to whitelist
				whitelistedCommands = append(whitelistedCommands, cmdName)
				needsUpdate = true
			}
		}
	}

	// Update whitelist if needed
	if needsUpdate && admin.IsAdmin() {
		core.AllowedCommands = whitelistedCommands

		// Save to whitelist file
		err := saveWhitelist(globals.WhitelistPath, whitelistedCommands)
		if err != nil {
			logging.LogError(err)
			gui.ErrorBox(fmt.Sprintf("Failed to update whitelist: %s", err))
		}
	}
}

// saveWhitelist saves the whitelist to a file
func saveWhitelist(path string, commands []string) error {
	// Create whitelist content
	content := strings.Join(commands, "\n")

	// Write to file
	return os.WriteFile(path, []byte(content), 0644)
}

// RegisterCommand adds a new command to the global command map
func RegisterCommand(cmd Command) {
	GlobalCommandMap[cmd.Name] = cmd
}

// GetCommand returns a command by name
func GetCommand(name string) (Command, bool) {
	cmd, exists := GlobalCommandMap[name]
	return cmd, exists
}

// IsCommandAllowed checks if a command is allowed to be executed
func IsCommandAllowed(cmdName string) bool {
	// Get the securityEnabled variable
	securityEnabled := GetSecurityEnabledFlag()

	// Admin bypass - if security is disabled and user is an admin, allow all commands
	if admin.IsAdmin() && !securityEnabled {
		return true
	}

	// Check blacklist first - blacklisted commands are never allowed regardless of other checks
	for _, blacklisted := range core.BlacklistedCommands {
		if cmdName == blacklisted {
			return false
		}
	}

	// Check if command exists in the map (built-in command)
	if _, exists := GlobalCommandMap[cmdName]; exists {
		// Built-in commands are allowed if they're in the command map and not blacklisted
		return true
	}

	// Check if the command is in the whitelist
	for _, whitelisted := range core.AllowedCommands {
		if cmdName == whitelisted {
			return true
		}
	}

	// Check if command exists in trusted directories
	for _, dir := range globals.TrustedDirs {
		path := filepath.Join(dir, cmdName)
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	// If not found in any trusted source, deny by default
	return false
}

// SetSecurityEnabled sets the security enabled flag
func SetSecurityEnabled(enabled bool) {
	securityEnabled = enabled
}

// GetSecurityEnabledFlag returns the current state of the security flag
func GetSecurityEnabledFlag() bool {
	return securityEnabled
}

// GetAllCommandNames returns a list of all registered command names
func GetAllCommandNames() []string {
	cmdNames := make([]string, 0, len(GlobalCommandMap))
	for name := range GlobalCommandMap {
		cmdNames = append(cmdNames, name)
	}
	return cmdNames
}

// GetCommandsByCategory returns commands grouped by category
func GetCommandsByCategory() map[CommandCategory][]Command {
	categories := make(map[CommandCategory][]Command)

	for _, cmd := range GlobalCommandMap {
		categories[cmd.Category] = append(categories[cmd.Category], cmd)
	}

	return categories
}

// NeedsRawTerminal checks if a command needs raw terminal mode
func NeedsRawTerminal(cmdName string) bool {
	cmd, exists := GlobalCommandMap[cmdName]
	if !exists {
		return SpecialCommands[cmdName] // Check special commands as fallback
	}

	return cmd.TermMode == ModeRaw
}

// CompleteCommand provides command completion functionality
func CompleteCommand(line string, pos int) (string, int) {
	if line == "" {
		return line, pos
	}

	words := strings.Fields(line)
	if len(words) == 0 {
		return line, pos
	}

	lastWord := words[len(words)-1]
	prefix := lastWord

	// Special handling for help command completion
	if len(words) == 2 && words[0] == "help" {
		matches := getHelpCommandMatches(prefix)
		if len(matches) == 0 {
			return line, pos
		}

		// Replace the last word with the first match
		words[len(words)-1] = matches[0]
		newLine := strings.Join(words, " ")

		// Use the proper function to clear line and print bottom
		ui.ClearLineAndPrintBottom()
		fmt.Print(newLine)
		return newLine, len(newLine)
	}

	// Special handling for ./ script completion
	if len(words) == 1 && strings.HasPrefix(lastWord, "./") {
		scriptPrefix := lastWord[2:] // Remove "./" from the prefix
		currentDir, err := os.Getwd()
		if err == nil {
			matches, _ := filepath.Glob(filepath.Join(currentDir, scriptPrefix+"*"))
			var scriptMatches []string

			// Script extensions that ExecuteScript can handle
			scriptExtensions := map[string]bool{
				".sh":   true,
				".bash": true,
				".zsh":  true,
				".py":   true,
				".rb":   true,
				".pl":   true,
				".js":   true,
				".php":  true,
				".lua":  true,
				".r":    true,
				".R":    true,
			}

			// Filter for script files and executable files
			for _, match := range matches {
				info, err := os.Stat(match)
				if err == nil && !info.IsDir() {
					filename := filepath.Base(match)
					ext := filepath.Ext(filename)

					// Include if it's executable OR has a supported script extension OR has shebang
					isExecutable := info.Mode()&0111 != 0
					hasScriptExt := scriptExtensions[ext]
					hasShebang := false

					// Check for shebang in non-executable files
					if !isExecutable {
						if file, err := os.Open(match); err == nil {
							scanner := bufio.NewScanner(file)
							if scanner.Scan() {
								firstLine := scanner.Text()
								hasShebang = strings.HasPrefix(firstLine, "#!")
							}
							file.Close()
						}
					}

					if isExecutable || hasScriptExt || hasShebang {
						// Keep the "./" prefix in the matches
						scriptMatches = append(scriptMatches, "./"+filename)
					}
				}
			}

			if len(scriptMatches) > 0 {
				words[len(words)-1] = scriptMatches[0]
				newLine := strings.Join(words, " ")

				ui.ClearLineAndPrintBottom()
				fmt.Print(newLine)

				if len(scriptMatches) > 1 {
					fmt.Println()
					for _, match := range scriptMatches {
						fmt.Print(match + "  ")
					}
					fmt.Println()
					ui.ClearLineAndPrintBottom()
					fmt.Print(newLine)
				}
				return newLine, len(newLine)
			}
		}
		return line, pos
	}

	// Command completion for first word
	if len(words) == 1 {
		// Command completion
		matches := getCommandMatches(prefix)
		if len(matches) == 0 {
			return line, pos
		}

		// Replace the last word with the first match
		words[len(words)-1] = matches[0]
		newLine := strings.Join(words, " ")

		ui.ClearLineAndPrintBottom()
		fmt.Print(newLine)
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

		ui.ClearLineAndPrintBottom()
		fmt.Print(newLine)

		// If there are multiple matches, show them below
		if len(matches) > 1 {
			fmt.Println()
			for _, match := range matches {
				fmt.Print(match + "  ")
			}
			fmt.Println()
			ui.ClearLineAndPrintBottom()
			fmt.Print(newLine)
		}

		return newLine, len(newLine)
	}
}

// getHelpCommandMatches returns a list of help command matches based on the prefix
func getHelpCommandMatches(prefix string) []string {
	var matches []string
	for name := range GlobalCommandMap {
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, name)
		}
	}
	return matches
}

// getCommandMatches returns a list of command matches based on the prefix
func getCommandMatches(prefix string) []string {
	var matches []string
	for name := range GlobalCommandMap {
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, name)
		}
	}
	return matches
}

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"secshell/core"
	"secshell/help"
	"secshell/sanitize"
	"secshell/ui"
	"strings"
)

var AllowedDirs = []string{}
var AllowedCommands = []string{}
var BuiltInCommands = []string{}
var ProgramCommands = []string{}

// Scan AllowedDirs for binaries
func ScanAllowedDirs() {
	ProgramCommands = []string{} // Reset program commands
	uniqueCommands := make(map[string]bool)

	for _, dir := range AllowedDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*"))
		if err != nil {
			continue
		}
		for _, file := range files {
			// Check if file is executable
			info, err := os.Stat(file)
			if err != nil || info.IsDir() {
				continue
			}
			if info.Mode()&0111 == 0 { // Check if executable bit is set
				continue
			}

			// Get base name and sanitize it
			baseName := filepath.Base(file)
			sanitized, err := sanitize.SanitizeFileName(baseName)
			if err != nil {
				continue
			}
			if !uniqueCommands[sanitized] {
				uniqueCommands[sanitized] = true
				ProgramCommands = append(ProgramCommands, sanitized)
			}
		}
	}
}

func Init() {
	ScanAllowedDirs()
}

// Modify the completeCommand function:
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
		ui.ClearLine()
		ui.ClearLineAndPrintBottom()
		fmt.Print(newLine)
		return newLine, len(newLine)
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
		ui.ClearLine()
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
		ui.ClearLine()
		ui.ClearLineAndPrintBottom()
		fmt.Print(newLine)

		// If there are multiple matches, show them below
		if len(matches) > 1 {
			for _, match := range matches {
				fmt.Print(match + "  ")
			}
			ui.ClearLine()
			ui.ClearLineAndPrintBottom() // Clear line and print the bottom prompt
			fmt.Print(newLine)           // Reprint the new input with the first match
		}

		return newLine, len(newLine)
	}
}

// GetHelpCommandMatches returns a list of help command matches based on the prefix
func getHelpCommandMatches(prefix string) []string {
	var matches []string
	for _, cmd := range help.HelpCommands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

// GetCommandMatches returns a list of command matches based on the prefix
func getCommandMatches(prefix string) []string {
	var matches []string
	for _, cmd := range AllowedCommands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}

	// Include built-in commands
	for _, cmd := range BuiltInCommands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}

	// Include program commands
	for _, cmd := range ProgramCommands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

// Print Program Commands
func PrintProgramCommands() {
	fmt.Println("Program Commands:")
	core.More(ProgramCommands)
}

// Print Allowed Directories
func PrintAllowedDirs() {
	fmt.Println("Allowed Directories:")
	for _, dir := range AllowedDirs {
		fmt.Println(" - " + dir)
	}
}

// Print Allowed Commands
func PrintAllowedCommands() {
	fmt.Println("Allowed Commands:")
	for _, cmd := range AllowedCommands {
		fmt.Println(" - " + cmd)
	}
}

// Print Built-in Commands
func PrintBuiltInCommands() {
	fmt.Println("Built-in Commands:")
	for _, cmd := range BuiltInCommands {
		fmt.Println(" - " + cmd)
	}
}

// Print All Commands
func PrintAllCommands() {
	fmt.Println(("Allowed Directories:"))
	ui.NewLine()
	for _, cmd := range AllowedDirs {
		fmt.Println(" - " + cmd)
	}
	fmt.Println("All Commands:")
	ui.NewLine()
	for _, cmd := range AllowedCommands {
		fmt.Println(" - " + cmd)
	}
	fmt.Println("Built-In Commands:")
	ui.NewLine()
	for _, cmd := range BuiltInCommands {
		fmt.Println(" - " + cmd)
	}
}

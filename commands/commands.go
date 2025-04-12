package commands

import (
	"fmt"
	"path/filepath"
	"secshell/help"
	"secshell/ui"
	"strings"
)

var AllowedCommands = []string{}
var BuiltInCommands = []string{}

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
	return matches
}

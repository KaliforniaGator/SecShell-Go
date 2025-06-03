package cmdmap

import (
	"os/exec"
	"secshell/help"
)

// CommandCategory defines the category a command belongs to
type CommandCategory string

const (
	// Command categories
	CategorySystem      CommandCategory = "System"
	CategorySecurity    CommandCategory = "Security"
	CategoryFileSystem  CommandCategory = "FileSystem"
	CategoryProcess     CommandCategory = "Process"
	CategoryNetwork     CommandCategory = "Network"
	CategoryUtility     CommandCategory = "Utility"
	CategoryEnvironment CommandCategory = "Environment"
	CategoryExternal    CommandCategory = "External"
)

// TerminalMode defines how a command interacts with the terminal
type TerminalMode int

const (
	// Terminal modes
	ModeNormal     TerminalMode = iota // Standard terminal mode
	ModeRaw                            // Raw terminal mode (needed for commands like ssh, vim)
	ModePiped                          // For piped commands
	ModeBackground                     // For background processes
)

// CommandHandler is a function that handles execution of a command
type CommandHandler func(args []string) (int, error)

// Command represents a shell command
type Command struct {
	Name        string          // Command name
	Description string          // Brief description
	Usage       string          // Usage information
	Examples    []string        // Example usages
	Category    CommandCategory // Command category
	Handler     CommandHandler  // Function to handle command execution
	TermMode    TerminalMode    // Terminal mode required
	Admin       bool            // Whether admin privileges are required
	AllowArgs   bool            // Whether the command can accept arguments
}

// CommandMap is a map of command names to Command objects
type CommandMap map[string]Command

// SystemCommand represents a system (external) command
type SystemCommand struct {
	Cmd         *exec.Cmd      // Underlying exec.Cmd
	Mode        TerminalMode   // Terminal mode
	Background  bool           // Whether the command runs in the background
	RawTerminal bool           // Whether the command needs raw terminal
	PID         int            // Process ID if running
	Name        string         // Command name
	Args        []string       // Command arguments
	HelpTopic   help.HelpTopic // Associated help topic
}

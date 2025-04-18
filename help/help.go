package help

import (
	"fmt"
	"os"
	"secshell/colors"
	"secshell/drawbox"
	"strings"
)

// HelpTopic structure to store detailed information about commands
type HelpTopic struct {
	Command     string
	Description string
	Usage       string
	Examples    []string
	Category    string
}

var HelpCommands = []string{
	"allowed",
	"help",
	"exit",
	"services",
	"jobs",
	"cd",
	"history",
	"export",
	"env",
	"unset",
	"blacklist",
	"whitelist",
	"edit-blacklist",
	"edit-whitelist",
	"reload-blacklist",
	"reload-whitelist",
	"download",
	"time",
	"date",
	"--version",
	"--update",
	"logs",
	"portscan",
	"hostscan",
	"webscan",
	"payload",
	"session",
}

// HelpTopics contains detailed help information for each command
var HelpTopics = map[string]HelpTopic{
	"allowed": {
		Command:     "allowed",
		Description: "List allowed system commands",
		Usage:       "allowed <dirs|commands|bins|builtins|all>",
		Examples:    []string{"allowed dirs", "allowed commands", "allowed all"},
		Category:    "Security",
	},
	"help": {
		Command:     "help",
		Description: "Show help message or specific command help",
		Usage:       "help [command]",
		Examples:    []string{"help", "help cd", "help services"},
		Category:    "System",
	},
	"exit": {
		Command:     "exit",
		Description: "Exit the shell",
		Usage:       "exit",
		Examples:    []string{"exit"},
		Category:    "System",
	},
	"services": {
		Command:     "services",
		Description: "Manage system services",
		Usage:       "services <start|stop|restart|status|list> <service_name>",
		Examples:    []string{"services list", "services status ssh", "services restart apache2"},
		Category:    "System",
	},
	"jobs": {
		Command:     "jobs",
		Description: "List active background jobs",
		Usage:       "jobs <list|stop|status|start|clear-finished> [PID]",
		Examples:    []string{"jobs", "jobs list", "jobs status 1234", "jobs stop 1234"},
		Category:    "Process",
	},
	"cd": {
		Command:     "cd",
		Description: "Change directory",
		Usage:       "cd (--prev | -p) [directory]",
		Examples:    []string{"cd /tmp", "cd ~", "cd --prev", "cd -p"},
		Category:    "FileSystem",
	},
	"history": {
		Command:     "history",
		Description: "Show command history",
		Usage:       "history [-s <query>] [-i]\n   -s: Search history for a query\n   -i: Interactive history search\n   ![number]: Execute command by number\n   !!: Execute last command\n   clear: Clear history",
		Examples:    []string{"history", "history -s ls", "history -i", "!5", "!!", "history clear"},
		Category:    "System",
	},
	"export": {
		Command:     "export",
		Description: "Set an environment variable",
		Usage:       "export VAR=value",
		Examples:    []string{"export PATH=$PATH:/usr/local/bin", "export DEBUG=true"},
		Category:    "Environment",
	},
	"env": {
		Command:     "env",
		Description: "List all environment variables",
		Usage:       "env",
		Examples:    []string{"env"},
		Category:    "Environment",
	},
	"unset": {
		Command:     "unset",
		Description: "Unset an environment variable",
		Usage:       "unset VAR",
		Examples:    []string{"unset DEBUG", "unset TEMP_VAR"},
		Category:    "Environment",
	},
	"blacklist": {
		Command:     "blacklist",
		Description: "List blacklisted commands",
		Usage:       "blacklist",
		Examples:    []string{"blacklist"},
		Category:    "Security",
	},
	"whitelist": {
		Command:     "whitelist",
		Description: "List whitelisted commands",
		Usage:       "whitelist",
		Examples:    []string{"whitelist"},
		Category:    "Security",
	},
	"edit-blacklist": {
		Command:     "edit-blacklist",
		Description: "Edit the blacklist file (admin only)",
		Usage:       "edit-blacklist",
		Examples:    []string{"edit-blacklist"},
		Category:    "Security",
	},
	"edit-whitelist": {
		Command:     "edit-whitelist",
		Description: "Edit the whitelist file (admin only)",
		Usage:       "edit-whitelist",
		Examples:    []string{"edit-whitelist"},
		Category:    "Security",
	},
	"reload-blacklist": {
		Command:     "reload-blacklist",
		Description: "Reload the blacklisted commands (admin only)",
		Usage:       "reload-blacklist",
		Examples:    []string{"reload-blacklist"},
		Category:    "Security",
	},
	"reload-whitelist": {
		Command:     "reload-whitelist",
		Description: "Reload the whitelisted commands (admin only)",
		Usage:       "reload-whitelist",
		Examples:    []string{"reload-whitelist"},
		Category:    "Security",
	},
	"download": {
		Command:     "download",
		Description: "Download a file from URL",
		Usage:       "download [-o output1,output2,...] <url [url2 ...]>",
		Examples:    []string{"download https://example.com/file.txt", "download -o file1.txt,file2.txt https://example.com/file1 https://example.com/file2"},
		Category:    "Network",
	},
	"time": {
		Command:     "time",
		Description: "Show the current time",
		Usage:       "time",
		Examples:    []string{"time"},
		Category:    "System",
	},
	"date": {
		Command:     "date",
		Description: "Show the current date",
		Usage:       "date",
		Examples:    []string{"date"},
		Category:    "System",
	},
	"--version": {
		Command:     "--version",
		Description: "Show the version of SecShell",
		Usage:       "--version",
		Examples:    []string{"--version"},
		Category:    "System",
	},
	"--update": {
		Command:     "--update",
		Description: "Update SecShell to the latest version",
		Usage:       "--update",
		Examples:    []string{"--update"},
		Category:    "System",
	},
	"logs": {
		Command:     "logs",
		Description: "Manage SecShell logs",
		Usage:       "logs <list|clear>",
		Examples:    []string{"logs list", "logs clear"},
		Category:    "System",
	},
	"toggle-security": {
		Command:     "toggle-security",
		Description: "Toggle security enforcement (admin only)",
		Usage:       "toggle-security",
		Examples:    []string{"toggle-security"},
		Category:    "Security",
	},
	"portscan": {
		Command:     "portscan",
		Description: "Scan ports on a target host",
		Usage:       "portscan <target> [port-range]",
		Examples:    []string{"portscan 192.168.1.1", "portscan example.com 1-1000"},
		Category:    "Pentesting",
	},
	"hostscan": {
		Command:     "hostscan",
		Description: "Discover hosts on a network",
		Usage:       "hostscan <network-range>",
		Examples:    []string{"hostscan 192.168.1.0/24"},
		Category:    "Pentesting",
	},
	"webscan": {
		Command:     "webscan",
		Description: "Perform basic web security scanning",
		Usage:       "webscan <url>",
		Examples:    []string{"webscan https://example.com"},
		Category:    "Pentesting",
	},
	"payload": {
		Command:     "payload",
		Description: "Generate reverse shell payload",
		Usage:       "payload <ip-address> <port>",
		Examples:    []string{"payload 192.168.1.100 4444"},
		Category:    "Pentesting",
	},
	"session": {
		Command:     "session",
		Description: "Manage reverse shell sessions",
		Usage:       "session [-l|-i <id>|-c <port>|-k <id>]\n   -l: List sessions\n   -i: Interact with session\n   -c: Create/listen for new session\n   -k: Kill/terminate session",
		Examples:    []string{"session -l", "session -i 1", "session -c 4444", "session -k 1"},
		Category:    "Pentesting",
	},
}

// DisplayHelp shows the help message or specific command help
func DisplayHelp(args ...string) {
	// If we have arguments, display specific command help
	if len(args) > 0 && args[0] != "" {
		displayCommandHelp(args[0])
		return
	}

	drawbox.RunDrawbox("SecShell Help", "bold_white")

	// Group commands by category
	commandsByCategory := make(map[string][]string)

	// Add all commands to their respective categories
	for cmd, topic := range HelpTopics {
		commandsByCategory[topic.Category] = append(commandsByCategory[topic.Category], cmd)
	}

	// Print commands by category
	fmt.Println("\nAvailable Commands:")

	// Order of categories to display
	categories := []string{"System", "FileSystem", "Process", "Environment", "Security", "Network", "Pentesting"}

	for _, category := range categories {
		commands, exists := commandsByCategory[category]
		if exists && len(commands) > 0 {
			fmt.Printf("\n%s%s Commands:%s\n", colors.Cyan, category, colors.Reset)

			// Print each command in this category
			for _, cmd := range commands {
				if topic, exists := HelpTopics[cmd]; exists {
					fmt.Printf("  %s%-12s%s - %s\n",
						colors.BoldWhite,
						topic.Command,
						colors.Reset,
						topic.Description)
				}
			}
		}
	}

	fmt.Printf("\n%sAllowed System Commands:%s\n", colors.Cyan, colors.Reset)
	fmt.Println("  ls, ps, netstat, tcpdump, cd, clear, ifconfig")

	fmt.Printf("\n%sSecurity Features:%s\n", colors.Cyan, colors.Reset)
	fmt.Println("  - Command whitelisting")
	fmt.Println("  - Input sanitization")
	fmt.Println("  - Process isolation")
	fmt.Println("  - Job tracking")
	fmt.Println("  - Service Management")
	fmt.Println("  - Background job execution")
	fmt.Println("  - Piped command execution")
	fmt.Println("  - Input/output redirection")

	fmt.Printf("\n%sUsage:%s Type '%shelp <command>%s' for more details on a specific command\n",
		colors.Cyan, colors.Reset, colors.BoldWhite, colors.Reset)
}

// displayCommandHelp shows help for a specific command
func displayCommandHelp(command string) {
	command = strings.ToLower(command)
	topic, exists := HelpTopics[command]

	if !exists {
		fmt.Fprintf(os.Stdout, "No help available for command: %s\n", command)
		return
	}

	drawbox.RunDrawbox(fmt.Sprintf("Help: %s", command), "bold_white")

	fmt.Printf("\n%sDescription:%s %s\n\n", colors.BoldWhite, colors.Reset, topic.Description)
	fmt.Printf("%sUsage:%s %s\n\n", colors.BoldWhite, colors.Reset, topic.Usage)

	if len(topic.Examples) > 0 {
		fmt.Printf("%sExamples:%s\n", colors.BoldWhite, colors.Reset)
		for _, example := range topic.Examples {
			fmt.Printf("  > %s\n", example)
		}
	}

	fmt.Printf("\n%sCategory:%s %s\n", colors.BoldWhite, colors.Reset, topic.Category)
	fmt.Println()
}

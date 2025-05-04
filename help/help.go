package help

import (
	"fmt"
	"os"
	"secshell/admin"
	"secshell/colors"
	"secshell/globals"
	"secshell/ui/gui"
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
	"toggle-security",
	"portscan",
	"hostscan",
	"webscan",
	"payload",
	"session",
	"base64",
	"hex",
	"urlencode",
	"url",
	"./",
	"hash",
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
		Description: "Advanced port scanner with service detection and OS fingerprinting",
		Usage: `portscan [options] <target>
Options:
    -p <ports>     Port range (e.g., 80,443 or 1-1000)
    -udp           Scan UDP ports instead of TCP
    -t <1-5>       Timing (1=slow/stealthy, 5=fast/aggressive)
    -v             Enable version detection
    -syn          Use TCP SYN scanning (requires root)
    -os           Attempt OS fingerprinting
    -e            Use enhanced service detection
    -j            Output in JSON format
    -html         Output in HTML format
    -o <file>     Save results to file`,
		Examples: []string{
			"portscan example.com",
			"portscan -p 80,443,8080 192.168.1.1",
			"portscan -syn -os -e example.com",
			"portscan -udp -p 53,161 example.com",
			"portscan -t 4 -v -os example.com",
			"portscan -e -j -o results.json example.com",
			"portscan -syn -html -o scan.html example.com",
		},
		Category: "Pentesting",
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
		Description: "Advanced web application security scanner",
		Usage: `webscan [options] <url>
Options:
    -t, --timeout <seconds>     Set request timeout (default: 10)
    -H, --header <header>       Add custom header (format: "Key: Value")
    -k, --insecure             Skip SSL verification
    -A, --user-agent <agent>   Set custom User-Agent
    --threads <number>         Number of concurrent scans (default: 10)
    -w, --wordlist <file>      Use custom wordlist for directory scanning
    -m, --methods <methods>    Test specific HTTP methods (comma-separated)
    -v, --verbose             Enable verbose output
    --follow-redirects        Follow redirects
    --cookie <cookie>         Set custom cookie
    --auth <token>           Set Authorization header
    -f, --format <format>     Output format (text|json|html)
    -o, --output <file>       Save results to file`,
		Examples: []string{
			"webscan example.com",
			"webscan -k -t 20 https://example.com",
			"webscan -H 'X-Custom: value' --auth 'Bearer token' example.com",
			"webscan -w wordlist.txt -v example.com",
			"webscan -m 'GET,POST,PUT' example.com",
			"webscan -f json -o results.json example.com",
		},
		Category: "Pentesting",
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
	"base64": {
		Command:     "base64",
		Description: "Encode or decode data using Base64",
		Usage:       "base64 [-e|-d] <string> OR base64 [-e|-d] -f <file> [> output_file]",
		Examples: []string{
			"base64 -e \"Hello, world!\"",
			"base64 -e 'Hello, world!'",
			"base64 -d \"SGVsbG8sIHdvcmxkIQ==\"",
			"base64 -e -f input.txt > encoded.txt",
			"base64 -d -f encoded.txt -o decoded.txt",
		},
		Category: "Encoding",
	},
	"hex": {
		Command:     "hex",
		Description: "Encode or decode data using hexadecimal",
		Usage:       "hex [-e|-d] <string> OR hex [-e|-d] -f <file> [> output_file]",
		Examples: []string{
			"hex -e \"Hello\"",
			"hex -e 'Hello'",
			"hex -d \"48656c6c6f\"",
			"hex -e -f binary.dat > encoded.txt",
			"hex -d -f encoded.txt -o original.dat",
		},
		Category: "Encoding",
	},
	"urlencode": {
		Command:     "urlencode",
		Description: "URL-encode or decode a string",
		Usage:       "urlencode [-e|-d] <string> [> output_file]",
		Examples: []string{
			"urlencode -e \"Hello world!\"",
			"urlencode -e 'Hello world!'",
			"urlencode -d \"Hello%20world%21\"",
			"urlencode -e \"user=test&pass=secret\" > encoded.txt",
		},
		Category: "Encoding",
	},
	"url": {
		Command:     "url",
		Description: "Alias for urlencode - URL-encode or decode a string",
		Usage:       "url [-e|-d] <string> [> output_file]",
		Examples: []string{
			"url -e \"Hello world!\"",
			"url -e 'Hello world!'",
			"url -d \"Hello%20world%21\"",
		},
		Category: "Encoding",
	},
	"./": {
		Command:     "./",
		Description: "Execute a script file with automatic interpreter detection",
		Usage:       "./<script_file> [arguments]",
		Examples: []string{
			"./script.sh",
			"./script.py arg1 arg2",
			"./custom_script --verbose",
		},
		Category: "Scripting",
	},
	"hash": {
		Command:     "hash",
		Description: "Calculate cryptographic hashes for strings or files",
		Usage:       "hash -s|-f <String|file> [algo]\n   -s: Hash a string\n   -f: Hash a file\n   [algo]: Optional hash algorithm (md5, sha1, sha256, sha512, all)",
		Examples: []string{
			"hash -s \"Hello, world!\"",
			"hash -f /path/to/file.txt",
			"hash -s \"test string\" md5",
			"hash -f document.pdf sha256",
			"hash -f image.jpg all",
		},
		Category: "Encoding",
	},
}

// DisplayHelp shows the help message or specific command help
func DisplayHelp(args ...string) {
	// If we have arguments, display specific command help
	if len(args) > 0 && args[0] != "" {
		displayCommandHelp(args[0])
		return
	}

	gui.TitleBox("SecShell Help")

	// Group commands by category
	commandsByCategory := make(map[string][]string)

	// Add all commands to their respective categories
	for cmd, topic := range HelpTopics {
		// Only add commands that the user has permission to see
		if admin.IsAdmin() || globals.IsCommandAllowed(cmd) {
			commandsByCategory[topic.Category] = append(commandsByCategory[topic.Category], cmd)
		}
	}

	// Print commands by category
	fmt.Println("\nAvailable Commands:")

	// Order of categories to display
	categories := []string{"System", "FileSystem", "Process", "Environment", "Security", "Network", "Pentesting", "Encoding", "Scripting"}

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
	fmt.Println("  ls, ps, netstat, tcpdump, clear, ifconfig")

	fmt.Printf("\n%sSecurity Features:%s\n", colors.Cyan, colors.Reset)
	fmt.Println("  - Command whitelisting")
	fmt.Println("  - Input sanitization")
	fmt.Println("  - Process isolation")
	fmt.Println("  - Job tracking")
	fmt.Println("  - Service Management")
	fmt.Println("  - Background job execution")
	fmt.Println("  - Piped command execution")
	fmt.Println("  - Input/output redirection")
	fmt.Println("  - Data encoding/decoding utilities")
	fmt.Println("  - Script execution with interpreter detection")

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

	// Check if user has permission to view this command's help
	if !admin.IsAdmin() && !globals.IsCommandAllowed(command) {
		fmt.Fprintf(os.Stdout, "Access denied: This command requires admin privileges\n")
		return
	}

	gui.TitleBox(fmt.Sprintf("Help: %s", command))

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

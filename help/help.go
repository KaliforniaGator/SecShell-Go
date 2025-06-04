package help

import (
	"fmt"
	"os"
	"secshell/admin"
	"secshell/colors"
	"secshell/globals"
	"secshell/terminal"
	"secshell/ui/gui"
	"sort" // Import the sort package
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
	"binary",
	"./",
	"hash",
	"extract-strings",
	"more",
	"edit",
	"features",
	"changelog",
	"colors",
	"edit-prompt",
	"reload-prompt",
	"prompt",
	"files",
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
		Examples:    []string{"help", "help cd", "help services", "help -i [--interactive]"},
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
		Examples:    []string{"jobs", "jobs list", "jobs status 1234", "jobs stop 1234", "jobs -i [--interactive]"},
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
	"binary": {
		Command:     "binary",
		Description: "Encode or decode data using binary (0s and 1s)",
		Usage:       "binary [-e|-d] <string> OR binary [-e|-d] -f <file> [> output_file]",
		Examples: []string{
			"binary -e \"Hello\"",
			"binary -e 'A'",
			"binary -d \"01000001\"",
			"binary -e -f input.txt > binary.txt",
			"binary -d -f binary.txt -o decoded.txt",
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
		Description: "Calculate cryptographic hashes for strings or files and compare hash values",
		Usage:       "hash -s|-f <String|file> [algo] [-c <hash-to-compare>]\n   -s: Hash a string\n   -f: Hash a file\n   [algo]: Optional hash algorithm (md5, sha1, sha256, sha512, all)\n   -c, --compare: Compare the calculated hash with the provided hash value",
		Examples: []string{
			"hash -s \"Hello, world!\"",
			"hash -f /path/to/file.txt",
			"hash -s \"test string\" md5",
			"hash -f document.pdf sha256",
			"hash -f image.jpg all",
			"hash -s \"Hello, world!\" sha256 -c 315f5bdb76d078c43b8ac0064e4a0164612b1fce77c869345bfc94c75894edd3",
			"hash -f /path/to/file.txt md5 -c d41d8cd98f00b204e9800998ecf8427e",
		},
		Category: "Encoding",
	},
	"extract-strings": {
		Command:     "extract-strings",
		Description: "Extract printable strings from binary files and output as a JSON array",
		Usage:       "extract-strings <file> [-n min-len] [-o output.json]\n   You can also use '> output.json' for redirection",
		Examples: []string{
			"extract-strings binary_file",
			"extract-strings executable -n 8",
			"extract-strings firmware.bin -n 10 -o strings.json",
			"extract-strings malware.bin > output.json",
		},
		Category: "Analysis",
	},
	"more": {
		Command:     "more",
		Description: "Display text files or command output with interactive paging and search",
		Usage:       "more <file> or command | more or more < input_file",
		Examples: []string{
			"more /etc/passwd",
			"cat /var/log/syslog | more",
			"ls -la /usr | more",
			"more < document.txt",
		},
		Category: "FileSystem",
	},
	"edit": { // Added help topic for edit
		Command:     "edit",
		Description: "Open a file in the built-in text editor",
		Usage:       "edit <filename>",
		Examples: []string{
			"edit my_document.txt",
			"edit /etc/hosts",
			"edit new_script.sh",
		},
		Category: "FileSystem",
	},
	"features": { // Added help topic for features
		Command:     "features",
		Description: "List all available features",
		Usage:       "features",
		Examples:    []string{"features"},
		Category:    "System",
	},
	"changelog": { // Added help topic for changelog
		Command:     "changelog",
		Description: "Display the application changelog",
		Usage:       "changelog",
		Examples:    []string{"changelog"},
		Category:    "System",
	},
	"colors": { // Added help topic for colors
		Command:     "colors",
		Description: "Display all available colors and styles",
		Usage:       "colors",
		Examples:    []string{"colors"},
		Category:    "System",
	},
	"edit-prompt": { // Added help topic for edit-prompt
		Command:     "edit-prompt",
		Description: "Edit the command prompt",
		Usage:       "edit-prompt",
		Examples:    []string{"edit-prompt"},
		Category:    "System",
	},
	"reload-prompt": { // Added help topic for reload-prompt
		Command:     "reload-prompt",
		Description: "Reload the command prompt configuration",
		Usage:       "reload-prompt",
		Examples:    []string{"reload-prompt"},
		Category:    "System",
	},
	"prompt": { // Added help topic for prompt
		Command:     "prompt",
		Description: "Display the current command prompt configuration and options",
		Usage:       "prompt",
		Examples:    []string{"prompt", "prompt -r [--reset]"},
		Category:    "System",
	},
	"files": { // Added help topic for files
		Command:     "files",
		Description: "Opens interactive file manager",
		Usage:       "files",
		Examples:    []string{"files"},
		Category:    "FileSystem",
	},
}

// clearLine clears the current line before printing
func clearLine() {
	// ANSI escape sequence to clear the entire line
	fmt.Print("\033[2K\r")
}

// DisplayHelp shows the help message or specific command help
func DisplayHelp(args ...string) {
	// If we have arguments, display specific command help
	if len(args) > 0 && args[0] != "" {
		displayCommandHelp(args[0])
		return
	}

	// Group commands by category
	commandsByCategory := make(map[string][]string)

	// Add all commands to their respective categories
	for cmd, topic := range HelpTopics {
		// Only add commands that the user has permission to see
		if admin.IsAdmin() || globals.IsCommandAllowed(cmd) {
			commandsByCategory[topic.Category] = append(commandsByCategory[topic.Category], cmd)
		}
	}

	// Sort commands within each category alphabetically
	for category := range commandsByCategory {
		sort.Strings(commandsByCategory[category])
	}

	// Order of categories to display
	categories := []string{"System", "FileSystem", "Process", "Environment", "Security", "Network", "Pentesting", "Encoding", "Scripting", "Analysis"}

	// Try to use the interactive mode with pagination
	err := terminal.WithInteractiveMode(func() error {
		return displayPaginatedHelp(categories, commandsByCategory)
	})

	if err != nil {
		// Fallback if interactive mode fails
		displaySimpleHelp(categories, commandsByCategory)
	}
}

// displayPaginatedHelp shows commands in a paginated interface
func displayPaginatedHelp(categories []string, commandsByCategory map[string][]string) error {
	// Get terminal dimensions
	width, height, err := terminal.GetTerminalSize()
	if err != nil {
		return err
	}

	// Calculate how many command lines can fit on screen
	// Reserve lines for: title (2), header(1), footer instructions (2)
	maxLinesPerPage := height - 5
	if maxLinesPerPage < 5 {
		maxLinesPerPage = 5 // Minimum reasonable size
	}

	// Prepare content for all pages
	var pages [][]string
	var currentPage []string
	linesOnCurrentPage := 0

	// Calculate column widths for command and description
	cmdWidth := 15                    // Width for command column
	descWidth := width - cmdWidth - 5 // Width for description column, adjusting for formatting chars

	// Add title to first page
	currentPage = append(currentPage, fmt.Sprintf("%sSecShell Help%s", colors.BoldCyan, colors.Reset))
	currentPage = append(currentPage, "")
	currentPage = append(currentPage, fmt.Sprintf("%sAvailable Commands:%s", colors.BoldWhite, colors.Reset))
	linesOnCurrentPage = 3

	for _, category := range categories {
		commands, exists := commandsByCategory[category]
		if exists && len(commands) > 0 {
			// Check if we need to start a new page for this category
			// Category header + commands + extra space before next category
			categorySize := 2 + len(commands)

			if linesOnCurrentPage+categorySize > maxLinesPerPage && linesOnCurrentPage > 3 {
				// Start a new page if current category won't fit
				pages = append(pages, currentPage)
				currentPage = []string{}
				linesOnCurrentPage = 0
			}

			// Add category header
			currentPage = append(currentPage, "")
			currentPage = append(currentPage, fmt.Sprintf("%s%s Commands:%s", colors.Cyan, category, colors.Reset))
			linesOnCurrentPage += 2

			// Add commands
			for _, cmd := range commands {
				if topic, exists := HelpTopics[cmd]; exists {
					// Truncate description if needed
					desc := topic.Description
					if len(desc) > descWidth && descWidth > 3 {
						desc = desc[:descWidth-3] + "..."
					}

					cmdLine := fmt.Sprintf("  %s%-*s%s - %s",
						colors.BoldWhite,
						cmdWidth,
						topic.Command,
						colors.Reset,
						desc)

					currentPage = append(currentPage, cmdLine)
					linesOnCurrentPage++

					// If page is full, start a new one
					if linesOnCurrentPage >= maxLinesPerPage {
						pages = append(pages, currentPage)
						currentPage = []string{}
						linesOnCurrentPage = 0
					}
				}
			}
		}
	}

	// Add the last page if it has content
	if len(currentPage) > 0 {
		pages = append(pages, currentPage)
	}

	// Display pages with navigation
	currentPageIndex := 0
	totalPages := len(pages)

	for {
		// Clear screen
		fmt.Print("\033[H\033[2J")

		// Display current page content
		for _, line := range pages[currentPageIndex] {
			clearLine() // Clear the line before printing
			fmt.Println(line)
		}

		// Fill remaining lines with empty space up to footer position
		remainingLines := maxLinesPerPage - len(pages[currentPageIndex]) + 3
		for i := 0; i < remainingLines; i++ {
			clearLine() // Clear each empty line
			fmt.Println()
		}

		// Display navigation footer
		clearLine()
		fmt.Println()
		clearLine()
		nav := fmt.Sprintf("%sPage %d/%d%s", colors.BoldWhite, currentPageIndex+1, totalPages, colors.Reset)
		controls := fmt.Sprintf("%s(← prev | next → | q quit)%s", colors.Gray, colors.Reset)
		fmt.Printf("%s    %s\n", nav, controls)

		// Read a single key press
		b := make([]byte, 3)
		os.Stdin.Read(b)

		// Process navigation
		if b[0] == 'q' || b[0] == 'Q' {
			break
		} else if b[0] == 27 && b[1] == 91 { // Arrow keys
			switch b[2] {
			case 68: // Left arrow
				if currentPageIndex > 0 {
					currentPageIndex--
				}
			case 67: // Right arrow
				if currentPageIndex < totalPages-1 {
					currentPageIndex++
				}
			}
		}
	}

	return nil
}

// displaySimpleHelp shows help without pagination (fallback)
func displaySimpleHelp(categories []string, commandsByCategory map[string][]string) {
	// Clear the screen first
	fmt.Print("\033[H\033[2J")

	gui.TitleBox("SecShell Help")

	clearLine()
	fmt.Println("\nAvailable Commands:")

	// Calculate column widths for command and description
	cmdWidth := 15  // Width for command column
	descWidth := 50 // Width for description column

	for _, category := range categories {
		commands, exists := commandsByCategory[category]
		if exists && len(commands) > 0 {
			clearLine()
			fmt.Printf("\n%s%s Commands:%s\n", colors.Cyan, category, colors.Reset)

			// Print each command in this category
			for _, cmd := range commands {
				if topic, exists := HelpTopics[cmd]; exists {
					clearLine() // Clear line before printing

					// Truncate description if too long
					desc := topic.Description
					if len(desc) > descWidth {
						desc = desc[:descWidth-3] + "..."
					}

					// Print command with fixed width formatting
					fmt.Printf("  %s%-*s%s - %s\n",
						colors.BoldWhite,
						cmdWidth,
						topic.Command,
						colors.Reset,
						desc)
				}
			}
		}
	}

	clearLine()
	fmt.Printf("\n%sUsage:%s Type '%shelp <command>%s' for more details on a specific command\n",
		colors.Cyan, colors.Reset, colors.BoldWhite, colors.Reset)
}

// displayCommandHelp shows help for a specific command
func displayCommandHelp(command string) {
	command = strings.ToLower(command)
	topic, exists := HelpTopics[command]

	if !exists {
		clearLine()
		fmt.Fprintf(os.Stdout, "No help available for command: %s\n", command)
		return
	}

	// Check if user has permission to view this command's help
	if !admin.IsAdmin() && !globals.IsCommandAllowed(command) {
		clearLine()
		fmt.Fprintf(os.Stdout, "Access denied: This command requires admin privileges\n")
		return
	}

	// Clear the screen first
	fmt.Print("\033[H\033[2J")

	gui.TitleBox(fmt.Sprintf("Help: %s", command))

	clearLine()
	fmt.Printf("\n%sDescription:%s %s\n", colors.BoldWhite, colors.Reset, topic.Description)

	clearLine()
	fmt.Printf("\n%sUsage:%s %s\n", colors.BoldWhite, colors.Reset, topic.Usage)

	if len(topic.Examples) > 0 {
		clearLine()
		fmt.Printf("\n%sExamples:%s\n", colors.BoldWhite, colors.Reset)
		for _, example := range topic.Examples {
			clearLine()
			fmt.Printf("  > %s\n", example)
		}
	}

	clearLine()
	fmt.Printf("\n%sCategory:%s %s\n", colors.BoldWhite, colors.Reset, topic.Category)
	clearLine()
	fmt.Println()
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

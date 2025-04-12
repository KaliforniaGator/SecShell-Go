package help

import (
	"fmt"
	"os"
	"secshell/colors"
	"secshell/drawbox"
	"strings"
)

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
	"--version",
	"--update",
}

// DisplayHelp shows the help message or specific command help
func DisplayHelp(args ...string) {
	// If we have arguments, display specific command help
	if len(args) > 0 && args[0] != "" {
		displayCommandHelp(args[0])
		return
	}

	drawbox.RunDrawbox("SecShell Help", "bold_white")
	fmt.Fprintf(os.Stdout, `
Built-in Commands:
  %sallowed%s   - List allowed system commands
			Usage: allowed <dirs|commands|bins|builtins|all>
  %shelp%s       - Show this help message
  %sexit%s       - Exit the shell
  %sservices%s   - Manage system services
               		Usage: services <start|stop|restart|status|list> <service_name>

  %sjobs%s       - List active background jobs
                        Usage: jobs <list|stop|status|start|clear-finished> [PID]

  %scd%s         - Change directory
               		Usage: cd [directory]

  %shistory%s    - Show command history
  			Usage: history [-s <query>] [-i]
			   -s: Search history for a query
			   -i: Interactive history search
			   ![number]: Execute command by number
			   !!: Execute last command

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

  %sdownload%s    - Download a file from URL
               		Usage: download <url>

  %s--version%s   - Show the version of SecShell
  %s--update%s    - Update SecShell to the latest version

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
  > reload-blacklist
  > whitelist
  > edit-whitelist
  > reload-whitelist
  > exit

%sNote:%s
All commands are subject to security checks and sanitization.
Only executables from trusted directories are permitted.
`,
		colors.BoldWhite, colors.Reset, // allowed
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
		colors.BoldWhite, colors.Reset, // download
		colors.BoldWhite, colors.Reset, // --version
		colors.BoldWhite, colors.Reset, // --update
		colors.Cyan, colors.Reset, // Allowed System Commands
		colors.Cyan, colors.Reset, // Security Features
		colors.Cyan, colors.Reset, // Examples
		colors.Cyan, colors.Reset, // Note
	)
}

// displayCommandHelp shows help for a specific command
func displayCommandHelp(command string) {
	commandHelp := map[string]string{
		"allowed":          "List allowed system commands\nUsage: allowed <dirs|commands|bins|builtins|all>",
		"help":             "Show this help message\nUsage: help [command]",
		"exit":             "Exit the shell\nUsage: exit",
		"services":         "Manage system services\nUsage: services <start|stop|restart|status|list> <service_name>",
		"jobs":             "List active background jobs\nUsage: jobs <list|stop|status|start|clear-finished> [PID]",
		"cd":               "Change directory\nUsage: cd (--prev | -p) [directory]",
		"history":          "Show command history\nUsage: history [-s <query>] [-i]\n   -s: Search history for a query\n   -i: Interactive history search\n   ![number]: Execute command by number\n   !!: Execute last command",
		"export":           "Set an environment variable\nUsage: export VAR=value",
		"env":              "List all environment variables",
		"unset":            "Unset an environment variable\nUsage: unset VAR",
		"blacklist":        "List blacklisted commands\nUsage: blacklist",
		"whitelist":        "List whitelisted commands\nUsage: whitelist",
		"edit-blacklist":   "Edit the blacklist file\nUsage: edit-blacklist",
		"edit-whitelist":   "Edit the whitelist file\nUsage: edit-whitelist",
		"reload-blacklist": "Reload the blacklisted commands\nUsage: reload-blacklist",
		"reload-whitelist": "Reload the whitelisted commands\nUsage: reload-whitelist",
		"download":         "Download a file from URL\nUsage: download <url>",
		"--version":        "Show the version of SecShell\nUsage: --version",
		"--update":         "Update SecShell to the latest version\nUsage: --update",
	}

	help, exists := commandHelp[strings.ToLower(command)]
	if !exists {
		fmt.Fprintf(os.Stdout, "No help available for command: %s\n", command)
		return
	}

	drawbox.RunDrawbox(fmt.Sprintf("Help: %s", command), "bold_white")
	fmt.Fprintf(os.Stdout, "\n%s\n", help)
}

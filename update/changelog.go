package update

import (
	"fmt"
	"secshell/colors"
	"secshell/ui/gui"
)

// ChangelogItem holds the details for a single version's changelog.
type ChangelogItem struct {
	Version  string
	Date     string
	Sections map[string][]string // Key: Section title (e.g., "New Features"), Value: List of changes
}

// changelogData stores all changelog entries, newest first.
var changelogData = []ChangelogItem{
	{
		Version: "1.0.0",
		Date:    "2025-03-09",
		Sections: map[string][]string{
			"🚀 New Features": {
				"Initial release of SecShell.",
				"Core features: command whitelisting, input sanitization, process isolation.",
				"Basic functionality: command execution and error handling.",
			},
			"✨ Improvements": {
				"Optimized startup sequence.",
			},
		},
	},
	{
		Version: "1.1.0",
		Date:    "2025-04-12",
		Sections: map[string][]string{
			"🚀 New Features": {
				"Added Auto-completion for commands and files.",
				"Implemented command history with search functionality.",
				"Introduced a new help command with categorized sections.",
			},
			"✨ Improvements": {
				"Updated UI elements for better readability.",
				"Improved error handling for command execution.",
				"Enhanced input sanitization to prevent command injection.",
			},
			"🐛 Bug Fixes": {
				"Fixed minor bugs",
			},
			"⚠️ Known Issues": {
				"Interactive history and More utility having visual scrollback buffer issues.",
			},
		},
	},
	{
		Version: "1.2.0",
		Date:    "2025-04-15",
		Sections: map[string][]string{
			"🚀 New Features": {
				"Added new Penetration Testing tools: Portscan, Webscan, Hostscan, Payload generator.",
				"Added better history tracking.",
				"Added several UI components throughout the application.",
				"Added new help commands.",
			},
			"✨ Improvements": {
				"Improved command execution. Piping, redirection, and background execution.",
				"Improved error handling and logging.",
			},
			"🐛 Bug Fixes": {
				"Fixed visual bugs with the payload generator.",
			},
			"⚠️ Known Issues": {
				"Interactive history and More utility having visual scrollback buffer issues.",
			},
		},
	},
	{
		Version: "1.2.4",
		Date:    "2025-04-22",
		Sections: map[string][]string{
			"🚀 New Features": {
				"Complete UI overhaul for better user experience.",
				"Less reliance on Drawbox for better performance.",
				"Windows now possible in SecShell.",
			},
			"✨ Improvements": {
				"Improved command execution and error handling.",
				"Improved UI elements for better readability.",
				"Improved utilities like More, Built-In Editor, History, Help, and others.",
			},
			"🐛 Bug Fixes": {
				"Fixed endless loops in certain commands.",
				"Minor bug fixes in the UI.",
			},
		},
	},
	{
		Version: "1.2.9",
		Date:    "2025-05-10",
		Sections: map[string][]string{
			"🚀 New Features": {
				"Search in More utility.",
				"Better error handling in SecShell.",
				"Better job exit handling.",
				"More GUI elements.",
				"Changelog now available",
			},
			"✨ Improvements": {
				"Improved Box printing in SecShell.",
				"Improved control over jobs.",
			},
			"🐛 Bug Fixes": {
				"Fixed endless loops in More utility when piping.",
				"Fixed Auto-completion for script commands.",
				"Fixed Exiting of jobs.",
			},
			"⚙️ Under the Hood": {
				"Completely new GUI library.",
				"New Changelog system.",
				"Re-designed UI's for multiple tools.",
				"New ANSI codes for Underline and Italic.",
			},
			"⚠️ Known Issues": {
				"Interactive history and More utility having visual scrollback buffer issues.",
			},
		},
	},
	{
		Version: "1.3.0",
		Date:    "2025-05-12",
		Sections: map[string][]string{
			"🚀 New Features": {
				"Fully configurable prompt with Colors, Endcaps, Logo, and More.",
				"Fully Integrated NerdFont support.",
				"New commands: edit-prompt, reload-prompt, colors, prompt.",
			},
			"🐛 Bug Fixes": {
				"Fixes issue with built-in editor not accounting for Tab characters.",
			},
			"⚙️ Under the Hood": {
				"New prompt system with full customization.",
				"New NerdFont system for better font rendering.",
			},
			"⚠️ Known Issues": {
				"Interactive history and More utility having visual scrollback buffer issues.",
				"Some GUI elements may not render correctly on certain terminals.",
				"Some ANSI codes may not be supported on all terminals.",
			},
		},
	},
	{
		Version: "1.3.1",
		Date:    "2025-05-13",
		Sections: map[string][]string{
			"🚀 New Features": {
				"Interactive Help App.",
				"Interactive Job Manager.",
			},
			"✨ Improvements": {
				"Improved interactive History for better usability.",
			},
			"🐛 Bug Fixes": {
				"Fixed issue interactive history not running commands correctly.",
				"Fixed More utility and Interactive history scrollback issue.",
				"Fixed small visual bug with the new prompt system.",
			},
		},
	},
	{
		Version: "1.3.2",
		Date:    "2025-05-14",
		Sections: map[string][]string{
			"🚀 New Features": {
				"Interactive File Manager.",
				"Window Library now supports Menu Bars, prompts, and more.",
			},
			"✨ Improvements": {
				"Improved interactive Job Manager for better usability.",
				"Improved SecShell for better job exit code handling.",
				"Added real-time CPU and Memory usage to the Job Manager.",
				"Added Gradient support to SecShell.",
				"Upgraded GUI library for better performance.",
			},
			"🐛 Bug Fixes": {
				"Fixed issue with interactive Job Manager not displaying exit codes correctly.",
				"Removed print after closing interactive apps.",
			},
			"⚠️ Known Issues": {
				"Interactive history and More utility having visual scrollback buffer issues. But this time less often.",
			},
		},
	},
	{
		Version: "1.3.3",
		Date:    "2025-05-15",
		Sections: map[string][]string{
			"🚀 New Features": {
				"Now compatible with macOS.",
			},
			"🐛 Bug Fixes": {
				"Project would not compile on macOS.",
			},
		},
	},
}

// sectionOrder defines the display order for changelog sections.
var sectionOrder = []string{"🚀 New Features", "✨ Improvements", "🐛 Bug Fixes", "⚙️ Under the Hood", "⚠️ Known Issues"}

// DisplayChangelog prints the formatted changelog to the console.
func DisplayChangelog() {
	gui.TitleBox("SecShell Changelog")

	if len(changelogData) == 0 {
		fmt.Printf("\n%sNo changelog entries found.%s\n\n", colors.Yellow, colors.Reset)
		return
	}

	for _, item := range changelogData {
		fmt.Printf("\n%s%sVersion %s%s %s(%s)%s\n",
			colors.BoldYellow, colors.Underline, item.Version, colors.Reset,
			colors.Cyan, item.Date, colors.Reset)

		// Sort section keys for consistent order if not using predefined sectionOrder
		// Or, iterate through predefined sectionOrder
		for _, sectionTitle := range sectionOrder {
			changes, ok := item.Sections[sectionTitle]
			if ok && len(changes) > 0 {
				fmt.Printf("  %s%s:%s\n", colors.BoldWhite, sectionTitle, colors.Reset)
				for _, change := range changes {
					// Determine color based on section or use a default
					changeColor := colors.Green
					if sectionTitle == "🐛 Bug Fixes" {
						changeColor = colors.Red
					} else if sectionTitle == "🚀 New Features" {
						changeColor = colors.BoldGreen
					}
					fmt.Printf("    %s• %s%s%s\n", changeColor, colors.Reset, changeColor, change)
				}
			}
		}
		fmt.Println(colors.Reset) // Reset colors and add a newline for spacing
	}
	fmt.Println()
}

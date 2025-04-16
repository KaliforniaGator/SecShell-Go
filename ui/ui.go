package ui

import (
	"fmt"
	"os"
	"secshell/colors"
	"secshell/drawbox"
)

// Add this after the printError method and before main:
func DisplayWelcomeScreen(version string, needsUpdate bool) {
	// Check for Update
	versionIcon := ""
	if needsUpdate {
		versionIcon = drawbox.PrintIcon("warning")
	} else {
		versionIcon = drawbox.PrintIcon("success")
	}
	// Clear the screen first
	fmt.Print("\033[H\033[2J")

	// ASCII art logo
	logo := `
    ███████╗███████╗ ██████╗███████╗██╗  ██╗███████╗██╗     ██╗     
    ██╔════╝██╔════╝██╔════╝██╔════╝██║  ██║██╔════╝██║     ██║     
    ███████╗█████╗  ██║     ███████╗███████║█████╗  ██║     ██║     
    ╚════██║██╔══╝  ██║     ╚════██║██╔══██║██╔══╝  ██║     ██║     
    ███████║███████╗╚██████╗███████║██║  ██║███████╗███████╗███████╗
    ╚══════╝╚══════╝ ╚═════╝╚══════╝╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝
    `

	fmt.Printf("%s%s%s\n", colors.BoldYellow, logo, colors.Reset)
	// Display welcome message
	drawbox.RunDrawbox("Welcome to SecShell - A Secure Command Shell", "bold_green")
	// Add version display
	fmt.Printf("\n%sVersion: %s %s%s\n", colors.BoldWhite, version, versionIcon, colors.Reset)
	fmt.Printf("\n%sFeatures:%s\n", colors.BoldWhite, colors.Reset)
	features := []string{
		"✓ Command whitelisting and blacklisting",
		"✓ Input sanitization",
		"✓ Process isolation",
		"✓ Service and Job management",
		"✓ Logging and auditing",
		"✓ Interactive Command history",
		"✓ Tab Command / Patch completions",
		"✓ Built-In Penetration testing tools",
	}

	for _, feature := range features {
		fmt.Printf("  %s%s%s\n", colors.BoldGreen, feature, colors.Reset)
	}

	fmt.Printf("\n%sType 'help' for available commands%s\n\n", colors.BoldCyan, colors.Reset)
}

// Change the displayPrompt function:
func DisplayPrompt() {
	user := os.Getenv("USER")
	if user == "" {
		user = "unknown"
	}
	host, _ := os.Hostname()
	cwd, err := os.Getwd()
	if err != nil {
		drawbox.PrintError("Failed to get current working directory")
		return
	}

	// Background color for the bar
	textReset := colors.Reset
	bgReset := colors.BgReset
	frameColor := colors.BoldGreen
	bgColor := colors.BgGray2
	endCapColor := colors.Gray2   // End caps should match the background
	logoColor := colors.BoldGreen // Text should contrast with the background
	userColor := colors.BoldCyan  // User/host should have a different color
	dirColor := colors.BoldYellow // Directory should have a different color

	// Print top bar with seamless end caps and proper alignment
	fmt.Printf("\n%s╭─%s%s%s [SecShell] %s %s@%s %s%s %s %s%s%s\n",
		frameColor, endCapColor, bgColor, logoColor, userColor, user, host, frameColor, dirColor, cwd, bgReset, endCapColor, textReset)

	// Print bottom input line
	fmt.Printf("%s╰─%s$ %s", frameColor, colors.BoldWhite, textReset)
}

// Print only top:
func ClearLine() {
	// Clear the entire current line and return carriage
	fmt.Print("\033[2K\r")

}

// Add this new method:
func ClearLineAndPrintBottom() {
	// Clear the entire current line and return carriage
	fmt.Print("\033[2K\r")
	// Print only the bottom prompt exactly as defined
	fmt.Print(colors.BoldGreen + "╰─" + colors.Reset + "$ ")
}

func NewLine() {
	// Print a new line
	fmt.Print("\n")
}

func ClearScreen() {
	// Clear the screen
	fmt.Print("\033[H\033[2J")
}

func ClearScreenAndBuffer() {
	// Clear the screen and buffer
	fmt.Print("\033[H\033[2J\033[3J")
	// Clear the scrollback buffer
}

package ui

import (
	"fmt"
	"os"
	"secshell/colors"
	"secshell/drawbox"
	"secshell/ui/chars"
	"secshell/ui/gui"
	"strings"
)

// Var for Prompt Options
var PromptOptions = promptOptions{}

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
	gui.SuccessBox("Welcome to SecShell - A Secure Command Shell")
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
	InitPrompt()
}

type promptOptions struct {
	PromptType        string
	PromptEndCaps     string
	PromptEndCapColor string
	PromptBackground  string
	PromptText        string
	PromptDivider     string
	PromptLogoColor   string
	PromptUserColor   string
	PromptHostColor   string
	PromptDirColor    string
}

func NewPrompt(opts promptOptions) {

	var CornerColor = colors.BoldGreen
	var TopLeftCorner = chars.RoundedCornerLeftTop
	var BottomLeftCorner = chars.RoundedCornerLeftBottom
	var CornerSpacer = "─"
	var LeftEndCap = chars.LeftCircleHalfFilled
	var RightEndCap = chars.RightCircleHalfFilled
	var EndCapColor = colors.Gray2
	var Divider = chars.RightArrow
	var DividerColor = colors.BoldGreen
	var Logo = "[SecShell]"
	var LogoColor = colors.BoldGreen
	var UserColor = colors.BoldCyan
	var HostColor = colors.BoldCyan
	var DirColor = colors.BoldYellow
	var PromptBackground = colors.BgGray2

	if opts.PromptBackground != "" {
		PromptBackground = colors.ColorMap[opts.PromptBackground]
	}
	if opts.PromptEndCapColor != "" {
		EndCapColor = colors.ColorMap[opts.PromptEndCapColor]
	}
	if opts.PromptText != "" {
		Logo = opts.PromptText
	}
	if opts.PromptLogoColor != "" {
		LogoColor = colors.ColorMap[opts.PromptLogoColor]
	}
	if opts.PromptUserColor != "" {
		UserColor = colors.ColorMap[opts.PromptUserColor]
	}
	if opts.PromptHostColor != "" {
		HostColor = colors.ColorMap[opts.PromptHostColor]
	}
	if opts.PromptDirColor != "" {
		DirColor = colors.ColorMap[opts.PromptDirColor]
	}
	// Get the current user and host

	user := os.Getenv("USER")
	if user == "" {
		user = "unknown"
	}
	host, _ := os.Hostname()
	cwd, err := os.Getwd()
	if err != nil {
		gui.ErrorBox("Failed to get current working directory")
	}

	switch opts.PromptDivider {
	case "arrow":
		Divider = chars.RightArrow
	case "thin":
		Divider = chars.ThinRightArrow
	case "round":
		Divider = chars.RightCircleHalf
	case "glitch":
		Divider = chars.GlitchDivider
	case "dashed":
		Divider = chars.ThreeDashedVertical
	case "simple":
		Divider = chars.SimpleLine
	default:
		Divider = chars.RightArrow
	}

	switch opts.PromptEndCaps {
	case "rounded":
		LeftEndCap = chars.LeftCircleHalfFilled
		RightEndCap = chars.RightCircleHalfFilled
	case "arrow":
		LeftEndCap = chars.LeftArrowFilled
		RightEndCap = chars.RightArrowFilled
	case "flame":
		LeftEndCap = chars.LeftFlameFilled
		RightEndCap = chars.RightFlameFilled
	case "glitch":
		LeftEndCap = chars.LeftGlitchFilled
		RightEndCap = chars.RightGlitchFilled
	default:
		LeftEndCap = chars.LeftCircleHalfFilled
		RightEndCap = chars.RightCircleHalfFilled
	}

	switch opts.PromptType {
	case "rounded":
		TopLeftCorner = chars.RoundedCornerLeftTop
		BottomLeftCorner = chars.RoundedCornerLeftBottom
	case "square":
		TopLeftCorner = chars.SquareCornerLeftTop
		BottomLeftCorner = chars.SquareCornerLeftBottom
	case "double":
		TopLeftCorner = chars.DoubleCornerLeftTop
		BottomLeftCorner = chars.DoubleCornerLeftBottom
		CornerSpacer = "═"
	default:
		TopLeftCorner = chars.RoundedCornerLeftTop
		BottomLeftCorner = chars.RoundedCornerLeftBottom
	}

	// PromptOrder (
	// CornerColor,
	// TopLeftCorner,
	// EndCapColor,
	// EndCapLeft,
	// Space,
	// BackgroundColor,
	// LogoColor,
	// Logo,
	// Space,
	// DividerColor,
	// Divider,
	// Space,
	// UserColor,
	// User,
	// @,
	// HostColor,
	// Host,
	// Space,
	// DividerColor,
	// Divider,
	// Space,
	// DirColor,
	// Dir,
	// Space,
	// EndCapColor,
	// EndCapRight,)

	fmt.Printf("\n%s%s%s%s%s%s %s%s %s%s %s%s@%s%s %s%s %s%s %s%s%s", CornerColor, TopLeftCorner, CornerSpacer, EndCapColor, LeftEndCap, PromptBackground, LogoColor, Logo, DividerColor, Divider, UserColor, user, HostColor, host, DividerColor, Divider, DirColor, cwd, colors.Reset, EndCapColor, RightEndCap)
	fmt.Printf("\n%s%s%s%s$ %s", CornerColor, BottomLeftCorner, CornerSpacer, colors.BoldWhite, colors.Reset)

}

func ParsePromptOptions(configFile string) promptOptions {
	// Default options
	opts := promptOptions{
		PromptType:        "rounded",
		PromptEndCaps:     "rounded",
		PromptEndCapColor: "gray2",
		PromptBackground:  "bg_gray2",
		PromptText:        "[SecShell]",
		PromptDivider:     "arrow",
	}

	// If no config file is provided, return defaults
	if configFile == "" {
		return opts
	}

	// Read the config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Printf("%sWarning: Could not read prompt config file: %v%s\n", colors.Yellow, err, colors.Reset)
		return opts
	}

	// Parse the config file content
	lines := strings.Split(string(data), "\n")
	inConfigSection := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for config section start/end
		if line == "CONFIG {" {
			inConfigSection = true
			continue
		}
		if line == "}" {
			inConfigSection = false
			continue
		}

		// Skip if not in config section or empty line
		if !inConfigSection || line == "" {
			continue
		}

		// Parse key-value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		// Remove quotes from value if present
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")

		// Apply configuration based on key
		switch key {
		case "PROMPT_TYPE":
			opts.PromptType = value
		case "PROMPT_ENDCAPS":
			opts.PromptEndCaps = value
		case "PROMPT_ENDCAPCOLOR":
			opts.PromptEndCapColor = value
		case "PROMPT_BACKGROUND":
			opts.PromptBackground = value
		case "PROMPT_TEXT":
			opts.PromptText = value
		case "PROMPT_DIVIDER":
			opts.PromptDivider = value
		case "PROMPT_LOGOCOLOR":
			opts.PromptLogoColor = value
		case "PROMPT_USERCOLOR":
			opts.PromptUserColor = value
		case "PROMPT_HOSTCOLOR":
			opts.PromptHostColor = value
		case "PROMPT_DIRCOLOR":
			opts.PromptDirColor = value
		}
	}

	return opts
}

func GetPromptConfigFile() string {
	// Check if the config file exists
	configFile := os.Getenv("HOME") + "/.secshell/secshell_prompt.conf"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// If it doesn't exist, create a default one
		defaultConfig := `CONFIG {
	PROMPT_TYPE = "rounded"
	PROMPT_ENDCAPS = "rounded"
	PROMPT_ENDCAPCOLOR = "gray2"
	PROMPT_BACKGROUND = "bg_gray2"
	PROMPT_TEXT = "[SecShell]"
	PROMPT_DIVIDER = "arrow"
	PROMPT_LOGOCOLOR = "bold_green"
	PROMPT_USERCOLOR = "bold_cyan"
	PROMPT_HOSTCOLOR = "bold_cyan"
	PROMPT_DIRCOLOR = "bold_yellow"
}`
		os.WriteFile(configFile, []byte(defaultConfig), 0644)
		return configFile
	} else {
		// If it exists, return the path
		return configFile
	}
}

func InitPrompt() {
	chars.InitFont()

	// Get the prompt config file
	configFile := GetPromptConfigFile()
	// Parse the prompt options from the config file
	opts := ParsePromptOptions(configFile)
	if (opts == promptOptions{}) {
		if (PromptOptions == promptOptions{}) {
			if (opts == promptOptions{}) {
				// Set the prompt options
				opts = promptOptions{
					PromptType:        "rounded",
					PromptEndCaps:     "rounded",
					PromptEndCapColor: "gray2",
					PromptBackground:  "bg_gray2",
					PromptText:        "[SecShell]",
					PromptDivider:     "arrow",
				}
			}
			PromptOptions = opts
		}
	} else {

		// Set the prompt options
		PromptOptions = opts

	}
}

func DisplayPrompt() {

	NewPrompt(PromptOptions)

}

// DisplayPromptOptions showcases all the Prompt Options and their corresponding values that the user can configure.
func DisplayPromptOptions() {
	fmt.Printf("\n%sCurrent Prompt Configuration:%s\n", colors.BoldWhite, colors.Reset)
	fmt.Printf("  %sPROMPT_TYPE:%s %s%s%s\n", colors.BoldCyan, colors.Reset, colors.Yellow, PromptOptions.PromptType, colors.Reset)
	fmt.Printf("  %sPROMPT_ENDCAPS:%s %s%s%s\n", colors.BoldCyan, colors.Reset, colors.Yellow, PromptOptions.PromptEndCaps, colors.Reset)
	fmt.Printf("  %sPROMPT_ENDCAPCOLOR:%s %s%s%s\n", colors.BoldCyan, colors.Reset, colors.Yellow, PromptOptions.PromptEndCapColor, colors.Reset)
	fmt.Printf("  %sPROMPT_BACKGROUND:%s %s%s%s\n", colors.BoldCyan, colors.Reset, colors.Yellow, PromptOptions.PromptBackground, colors.Reset)
	fmt.Printf("  %sPROMPT_TEXT:%s %s%s%s\n", colors.BoldCyan, colors.Reset, colors.Yellow, PromptOptions.PromptText, colors.Reset)
	fmt.Printf("  %sPROMPT_DIVIDER:%s %s%s%s\n", colors.BoldCyan, colors.Reset, colors.Yellow, PromptOptions.PromptDivider, colors.Reset)
	fmt.Printf("  %sPROMPT_LOGOCOLOR:%s %s%s%s\n", colors.BoldCyan, colors.Reset, colors.Yellow, PromptOptions.PromptLogoColor, colors.Reset)
	fmt.Printf("  %sPROMPT_USERCOLOR:%s %s%s%s\n", colors.BoldCyan, colors.Reset, colors.Yellow, PromptOptions.PromptUserColor, colors.Reset)
	fmt.Printf("  %sPROMPT_HOSTCOLOR:%s %s%s%s\n", colors.BoldCyan, colors.Reset, colors.Yellow, PromptOptions.PromptHostColor, colors.Reset)
	fmt.Printf("  %sPROMPT_DIRCOLOR:%s %s%s%s\n", colors.BoldCyan, colors.Reset, colors.Yellow, PromptOptions.PromptDirColor, colors.Reset)

	fmt.Printf("\n%sAvailable options for PROMPT_DIVIDER:%s %sarrow, thin, round, glitch, dashed, simple%s\n", colors.BoldWhite, colors.Reset, colors.Green, colors.Reset)
	fmt.Printf("%sAvailable options for PROMPT_ENDCAPS:%s %srounded, arrow, flame, glitch%s\n", colors.BoldWhite, colors.Reset, colors.Green, colors.Reset)
	fmt.Printf("%sAvailable options for PROMPT_TYPE:%s %srounded, square, double%s\n", colors.BoldWhite, colors.Reset, colors.Green, colors.Reset)
	fmt.Printf("\n%sYou can configure these options by typing: %sedit-prompt%s\n", colors.Gray, colors.Italic, colors.Reset)
}

func ReloadPrompt(version string, needsUpdate bool) {
	// Display the welcome screen
	DisplayWelcomeScreen(version, needsUpdate)
	// Display the prompt
}

func ClearLine() {
	// Clear the entire current line and return carriage
	fmt.Print("\033[2K\r")

}

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

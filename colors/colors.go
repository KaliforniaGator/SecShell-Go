package colors

import "runtime"

var (

	// Gray Shades
	Gray1 = "\033[38;5;232m" // Very Dark Gray
	Gray2 = "\033[38;5;235m" // Dark Gray
	Gray3 = "\033[38;5;239m" // Medium Gray
	Gray4 = "\033[38;5;243m" // Light Gray
	Gray5 = "\033[38;5;247m" // Very Light Gray

	// Regular Foreground Colors
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	White  = "\033[97m"
	Black  = "\033[30m"

	// Bold Gray Variants
	BoldGray1 = "\033[1;38;5;232m"
	BoldGray2 = "\033[1;38;5;235m"
	BoldGray3 = "\033[1;38;5;239m"
	BoldGray4 = "\033[1;38;5;243m"
	BoldGray5 = "\033[1;38;5;247m"

	// Bold/Bright Foreground Colors
	BoldRed    = "\033[1;31m"
	BoldGreen  = "\033[1;32m"
	BoldYellow = "\033[1;33m"
	BoldBlue   = "\033[1;34m"
	BoldPurple = "\033[1;35m"
	BoldCyan   = "\033[1;36m"
	BoldGray   = "\033[1;37m"
	BoldWhite  = "\033[1;97m"
	BoldBlack  = "\033[1;30m"

	// Gray Backgrounds
	BgGray1 = "\033[48;5;232m" // Very Dark Gray Background
	BgGray2 = "\033[48;5;235m" // Dark Gray Background
	BgGray3 = "\033[48;5;239m" // Medium Gray Background
	BgGray4 = "\033[48;5;243m" // Light Gray Background
	BgGray5 = "\033[48;5;247m" // Very Light Gray Background

	// Regular Background Colors
	BgBlack  = "\033[40m"
	BgRed    = "\033[41m"
	BgGreen  = "\033[42m"
	BgYellow = "\033[43m"
	BgBlue   = "\033[44m"
	BgPurple = "\033[45m"
	BgCyan   = "\033[46m"
	BgGray   = "\033[47m"
	BgWhite  = "\033[107m" // White background

	// Bold Gray Background Variants
	BgBoldGray1 = "\033[1;48;5;232m"
	BgBoldGray2 = "\033[1;48;5;235m"
	BgBoldGray3 = "\033[1;48;5;239m"
	BgBoldGray4 = "\033[1;48;5;243m"
	BgBoldGray5 = "\033[1;48;5;247m"

	// Bold/Bright Background Colors
	BgBrightBlack  = "\033[100m"
	BgBrightRed    = "\033[101m"
	BgBrightGreen  = "\033[102m"
	BgBrightYellow = "\033[103m"
	BgBrightBlue   = "\033[104m"
	BgBrightPurple = "\033[105m"
	BgBrightCyan   = "\033[106m"
	BgBrightWhite  = "\033[107m"

	// Reset Background Color
	BgReset = "\033[49m"
)

// Disable colors on Windows if necessary
func init() {
	if runtime.GOOS == "windows" {
		Reset = ""
		Red = ""
		Green = ""
		Yellow = ""
		Blue = ""
		Purple = ""
		Cyan = ""
		Gray = ""
		White = ""
		BoldRed = ""
		BoldGreen = ""
		BoldYellow = ""
		BoldBlue = ""
		BoldPurple = ""
		BoldCyan = ""
		BoldGray = ""
		BoldWhite = ""
		BgBlack = ""
		BgRed = ""
		BgGreen = ""
		BgYellow = ""
		BgBlue = ""
		BgPurple = ""
		BgCyan = ""
		BgGray = ""
		BgWhite = ""
		BgBrightBlack = ""
		BgBrightRed = ""
		BgBrightGreen = ""
		BgBrightYellow = ""
		BgBrightBlue = ""
		BgBrightPurple = ""
		BgBrightCyan = ""
		BgBrightWhite = ""
		BgReset = ""

		Gray1 = ""
		Gray2 = ""
		Gray3 = ""
		Gray4 = ""
		Gray5 = ""

		BgGray1 = ""
		BgGray2 = ""
		BgGray3 = ""
		BgGray4 = ""
		BgGray5 = ""

		BoldGray1 = ""
		BoldGray2 = ""
		BoldGray3 = ""
		BoldGray4 = ""
		BoldGray5 = ""

		BgBoldGray1 = ""
		BgBoldGray2 = ""
		BgBoldGray3 = ""
		BgBoldGray4 = ""
		BgBoldGray5 = ""
	}
}

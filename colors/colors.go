package colors

import "runtime"

var (
	// Regular colors
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	White  = "\033[97m"

	// Bold/bright colors
	BoldRed    = "\033[1;31m"
	BoldGreen  = "\033[1;32m"
	BoldYellow = "\033[1;33m"
	BoldBlue   = "\033[1;34m"
	BoldPurple = "\033[1;35m"
	BoldCyan   = "\033[1;36m"
	BoldGray   = "\033[1;37m"
	BoldWhite  = "\033[1;97m"
)

// init checks if we're running on Windows and disables colors if necessary
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
	}
}

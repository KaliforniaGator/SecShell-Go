package colors

import (
	"fmt"
	"runtime"
	"sort"
)

var (

	// Gray Shades
	Gray1 = "\033[38;5;232m" // Very Dark Gray
	Gray2 = "\033[38;5;235m" // Dark Gray
	Gray3 = "\033[38;5;239m" // Medium Gray
	Gray4 = "\033[38;5;243m" // Light Gray
	Gray5 = "\033[38;5;247m" // Very Light Gray

	// Regular Foreground Colors
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Purple  = "\033[35m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Gray    = "\033[37m"
	White   = "\033[97m"
	Black   = "\033[30m"

	// Text Styles
	Underline = "\033[4m"
	Italic    = "\033[3m"

	// Bold Gray Variants
	BoldGray1 = "\033[1;38;5;232m"
	BoldGray2 = "\033[1;38;5;235m"
	BoldGray3 = "\033[1;38;5;239m"
	BoldGray4 = "\033[1;38;5;243m"
	BoldGray5 = "\033[1;38;5;247m"

	// Bold/Bright Foreground Colors
	BoldRed     = "\033[1;31m"
	BoldGreen   = "\033[1;32m"
	BoldYellow  = "\033[1;33m"
	BoldBlue    = "\033[1;34m"
	BoldPurple  = "\033[1;35m"
	BoldMagenta = "\033[1;35m"
	BoldCyan    = "\033[1;36m"
	BoldGray    = "\033[1;37m"
	BoldWhite   = "\033[1;97m"
	BoldBlack   = "\033[1;30m"

	// Gray Backgrounds
	BgGray1 = "\033[48;5;232m" // Very Dark Gray Background
	BgGray2 = "\033[48;5;235m" // Dark Gray Background
	BgGray3 = "\033[48;5;239m" // Medium Gray Background
	BgGray4 = "\033[48;5;243m" // Light Gray Background
	BgGray5 = "\033[48;5;247m" // Very Light Gray Background

	// Regular Background Colors
	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgPurple  = "\033[45m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgGray    = "\033[47m"
	BgWhite   = "\033[107m" // White background

	// Bold Gray Background Variants
	BgBoldGray1 = "\033[1;48;5;232m"
	BgBoldGray2 = "\033[1;48;5;235m"
	BgBoldGray3 = "\033[1;48;5;239m"
	BgBoldGray4 = "\033[1;48;5;243m"
	BgBoldGray5 = "\033[1;48;5;247m"

	// Bold/Bright Background Colors
	BgBrightBlack   = "\033[100m"
	BgBrightRed     = "\033[101m"
	BgBrightGreen   = "\033[102m"
	BgBrightYellow  = "\033[103m"
	BgBrightBlue    = "\033[104m"
	BgBrightPurple  = "\033[105m"
	BgBrightMagenta = "\033[105m"
	BgBrightCyan    = "\033[106m"
	BgBrightWhite   = "\033[107m"

	// Reset Background Color
	BgReset = "\033[49m"

	// ColorMap provides a mapping between color names and their ANSI codes
	ColorMap = map[string]string{
		// Regular colors
		"red":    Red,
		"green":  Green,
		"yellow": Yellow,
		"blue":   Blue,
		"purple": Purple,
		"cyan":   Cyan,
		"gray":   Gray,
		"white":  White,
		"black":  Black,

		// Text Styles
		"underline": Underline,
		"italic":    Italic,

		// Bold colors
		"bold_red":    BoldRed,
		"bold_green":  BoldGreen,
		"bold_yellow": BoldYellow,
		"bold_blue":   BoldBlue,
		"bold_purple": BoldPurple,
		"bold_cyan":   BoldCyan,
		"bold_gray":   BoldGray,
		"bold_white":  BoldWhite,
		"bold_black":  BoldBlack,

		// Background colors
		"bg_red":    BgRed,
		"bg_green":  BgGreen,
		"bg_yellow": BgYellow,
		"bg_blue":   BgBlue,
		"bg_purple": BgPurple,
		"bg_cyan":   BgCyan,
		"bg_gray":   BgGray,
		"bg_white":  BgWhite,
		"bg_black":  BgBlack,

		// Gray variants
		"gray1": Gray1,
		"gray2": Gray2,
		"gray3": Gray3,
		"gray4": Gray4,
		"gray5": Gray5,

		// Background gray variants
		"bg_gray1": BgGray1,
		"bg_gray2": BgGray2,
		"bg_gray3": BgGray3,
		"bg_gray4": BgGray4,
		"bg_gray5": BgGray5,
	}
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

		Underline = "" // Disable Underline on Windows
		Italic = ""    // Disable Italic on Windows

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

// DisplayColors showcases all the colors and their corresponding colormap name.
func DisplayColors() {
	fmt.Println("Available colors and styles:")

	categories := map[string][]string{
		"Regular Colors":           {},
		"Text Styles":              {},
		"Bold Colors":              {},
		"Background Colors":        {},
		"Gray Variants":            {},
		"Background Gray Variants": {},
	}

	// Temporary slices to hold keys for sorting
	regularColors := []string{}
	textStyles := []string{}
	boldColors := []string{}
	backgroundColors := []string{}
	grayVariants := []string{}
	bgGrayVariants := []string{}

	for name := range ColorMap {
		switch {
		case name == "underline" || name == "italic":
			textStyles = append(textStyles, name)
		case name == "gray1" || name == "gray2" || name == "gray3" || name == "gray4" || name == "gray5":
			grayVariants = append(grayVariants, name)
		case name == "bg_gray1" || name == "bg_gray2" || name == "bg_gray3" || name == "bg_gray4" || name == "bg_gray5":
			bgGrayVariants = append(bgGrayVariants, name)
		case len(name) > 3 && name[:3] == "bg_":
			backgroundColors = append(backgroundColors, name)
		case len(name) > 5 && name[:5] == "bold_":
			boldColors = append(boldColors, name)
		default:
			regularColors = append(regularColors, name)
		}
	}

	sort.Strings(regularColors)
	sort.Strings(textStyles)
	sort.Strings(boldColors)
	sort.Strings(backgroundColors)
	sort.Strings(grayVariants)
	sort.Strings(bgGrayVariants)

	categories["Regular Colors"] = regularColors
	categories["Text Styles"] = textStyles
	categories["Bold Colors"] = boldColors
	categories["Background Colors"] = backgroundColors
	categories["Gray Variants"] = grayVariants
	categories["Background Gray Variants"] = bgGrayVariants

	categoryOrder := []string{
		"Regular Colors",
		"Text Styles",
		"Bold Colors",
		"Background Colors",
		"Gray Variants",
		"Background Gray Variants",
	}

	for _, categoryName := range categoryOrder {
		names := categories[categoryName]
		if len(names) > 0 {
			fmt.Printf("\n%s%s%s:\n", BoldWhite, categoryName, Reset)
			for _, name := range names {
				code := ColorMap[name]
				fmt.Printf("%s%s%s - %s\n", code, "Sample Text", Reset, name)
			}
		}
	}
}

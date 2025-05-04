package gui

import (
	"fmt"
	"os"
	"secshell/colors"

	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// ANSI Escape Codes
const (
	clearScreen          = "\x1b[2J"
	clearScreenAndBuffer = "\033[H\033[2J\033[3J"
	moveCursorFormat     = "\x1b[%d;%dH" // row, col (1-based) - Renamed format string
	hideCursor           = "\x1b[?25l"
	showCursor           = "\x1b[?25h"
)

// ClearScreen clears the entire terminal screen.
func ClearScreen() string { // Return string instead of printing directly
	return clearScreen
}

// ClearScreenAndBuffer clears the terminal screen and scrollback buffer.
func ClearScreenAndBuffer() string { // Return string instead of printing directly
	return clearScreenAndBuffer
}

// MoveCursor positions the cursor at the specified row and column.
// Note: row and col are 0-based for convenience, but converted to 1-based for ANSI.
func MoveCursor(row, col int) { // Keep this for direct printing if needed elsewhere
	fmt.Printf(moveCursorFormat, row+1, col+1)
}

// MoveCursorCmd returns the ANSI escape code string to move the cursor.
// Note: row and col are 0-based for convenience.
func MoveCursorCmd(row, col int) string {
	return fmt.Sprintf(moveCursorFormat, row+1, col+1)
}

// HideCursor makes the terminal cursor invisible.
func HideCursor() string { // Return string
	return hideCursor
}

// ShowCursor makes the terminal cursor visible.
func ShowCursor() string { // Return string
	return showCursor
}

// ClearLineSuffix returns ANSI sequence to clear from cursor to end of line
func ClearLineSuffix() string {
	return "\x1b[K"
}

// ResetStyle returns ANSI escape sequence to reset text formatting
func ResetStyle() string {
	return "\x1b[0m"
}

// ReverseVideo returns ANSI escape sequence for reverse video (inverted colors)
func ReverseVideo() string {
	return "\x1b[7m"
}

// ResetVideo returns ANSI escape sequence to reset text formatting
func ResetVideo() string {
	return "\x1b[27m"
}

// BoxType defines the structure for different box styles
type BoxType struct {
	TopLeft     string
	TopRight    string
	BottomLeft  string
	BottomRight string
	Horizontal  string
	Vertical    string
}

// TextAlignment defines the structure for text alignment
type TextAlignment struct {
	Horizontal string
	Vertical   string
}

var (
	BoxTypes = map[string]BoxType{
		"single": {
			TopLeft:     "┌",
			TopRight:    "┐",
			BottomLeft:  "└",
			BottomRight: "┘",
			Horizontal:  "─",
			Vertical:    "│",
		},
		"double": {
			TopLeft:     "╔",
			TopRight:    "╗",
			BottomLeft:  "╚",
			BottomRight: "╝",
			Horizontal:  "═",
			Vertical:    "║",
		},
		"round": {
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "╰",
			BottomRight: "╯",
			Horizontal:  "─",
			Vertical:    "│",
		},
		"bold": {
			TopLeft:     "┏",
			TopRight:    "┓",
			BottomLeft:  "┗",
			BottomRight: "┛",
			Horizontal:  "━",
			Vertical:    "┃",
		},
	}
)

func PrintColoredText(text string, color string) {
	// Print colored text
	fmt.Printf("%s%s%s", color, text, colors.Reset)
}
func PrintError(text string) {
	// Print error message
	fmt.Printf("%s%s%s", colors.BoldRed, text, colors.Reset)
}
func PrintSuccess(text string) {
	// Print success message
	fmt.Printf("%s%s%s", colors.BoldGreen, text, colors.Reset)
}
func PrintWarning(text string) {
	// Print warning message
	fmt.Printf("%s%s%s", colors.BoldYellow, text, colors.Reset)
}
func PrintInfo(text string) {
	// Print info message
	fmt.Printf("%s%s%s", colors.BoldCyan, text, colors.Reset)
}
func PrintDebug(text string) {
	// Print debug message
	fmt.Printf("%s%s%s", colors.BoldGray, text, colors.Reset)
}
func PrintAlert(text string) {
	// Print alert message
	fmt.Printf("%s%s%s", colors.BoldWhite, text, colors.Reset)
}

func GetTerminalWidth() int {
	// Get the terminal width
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // Default width if unable to get terminal size
	}
	return width
}

func GetTerminalHeight() int {
	// Get the terminal height
	_, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 24 // Default height if unable to get terminal size
	}
	return height
}

// Estimate the width of a string based on average character width
func EstimateStringWidth(s string) int {
	// Assume an average width of 8 pixels per character
	// You can adjust this value based on your needs
	const averageCharWidth = 8

	// Count the number of runes (characters) in the string
	charCount := utf8.RuneCountInString(s)

	// Calculate the estimated width
	return charCount * averageCharWidth
}

func NormalizeWidth(text string) int {
	maxWidth := GetTerminalWidth() / 2

	width := EstimateStringWidth(text)
	if width > maxWidth {
		width = maxWidth - 4
	}
	return width
}

func TitleBox(text string) {
	width := -1
	height := -1
	if width < 0 {
		width = len(text) + 4
	}
	if height < 0 {
		height = 3
	}
	PrintBanner(text, "double", colors.BoldWhite, "", colors.BoldWhite, width, height, TextAlignment{
		Horizontal: "center",
		Vertical:   "center",
	})
	fmt.Println()
}

func ErrorBox(text string) {
	width := NormalizeWidth(text)
	height := -1
	if width < 0 {
		width = len(text) + 4
	}
	if height < 0 {
		height = 3
	}
	PrintBanner(text, "single", colors.BoldRed, "", colors.BoldRed, width, height, TextAlignment{
		Horizontal: "center",
		Vertical:   "center",
	})
	fmt.Println()
}
func SuccessBox(text string) {
	width := NormalizeWidth(text)
	height := -1
	if width < 0 {
		width = len(text) + 4
	}
	if height < 0 {
		height = 3
	}
	PrintBanner(text, "single", colors.BoldGreen, "", colors.BoldGreen, width, height, TextAlignment{
		Horizontal: "center",
		Vertical:   "center",
	})
	fmt.Println()
}
func WarningBox(text string) {
	width := NormalizeWidth(text)
	height := -1
	if width < 0 {
		width = len(text) + 4
	}
	if height < 0 {
		height = 3
	}
	PrintBanner(text, "single", colors.BoldYellow, "", colors.BoldYellow, width, height, TextAlignment{
		Horizontal: "center",
		Vertical:   "center",
	})
	fmt.Println()
}
func InfoBox(text string) {
	width := NormalizeWidth(text)
	height := -1
	if width < 0 {
		width = len(text) + 4
	}
	if height < 0 {
		height = 3
	}
	PrintBanner(text, "single", colors.BoldCyan, "", colors.BoldCyan, width, height, TextAlignment{
		Horizontal: "center",
		Vertical:   "center",
	})
	fmt.Println()
}
func DebugBox(text string) {
	width := NormalizeWidth(text)
	height := -1
	if width < 0 {
		width = len(text) + 4
	}
	if height < 0 {
		height = 3
	}
	PrintBanner(text, "single", colors.BoldGray, "", colors.BoldGray, width, height, TextAlignment{
		Horizontal: "center",
		Vertical:   "center",
	})
	fmt.Println()
}
func AlertBox(text string) {
	width := NormalizeWidth(text)
	height := -1
	if width < 0 {
		width = len(text) + 4
	}
	if height < 0 {
		height = 3
	}
	PrintBanner(text, "single", colors.BoldYellow, "", colors.BoldYellow, width, height, TextAlignment{
		Horizontal: "center",
		Vertical:   "center",
	})
	fmt.Println()
}

func wrapText(text string, width int) []string {
	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+len(word)+1 <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)
	return lines
}

func PrintBanner(text string, boxStyle string, textColor string, bgColor string, borderColor string, width int, height int, alignment TextAlignment) {
	fmt.Print(colors.Reset)
	box, exists := BoxTypes[boxStyle]
	if !exists {
		box = BoxTypes["single"]
	}

	padding := 2
	effectiveWidth := width - (padding * 2)
	wrappedText := wrapText(text, effectiveWidth)
	textHeight := len(wrappedText)

	if width < padding*2 {
		width = padding * 2
	}

	if height < textHeight+2 {
		height = textHeight + 2
	}

	// Top border with border color
	fmt.Print(bgColor + borderColor)
	fmt.Print(box.TopLeft)
	for i := 0; i < width; i++ {
		fmt.Print(box.Horizontal)
	}
	fmt.Print(box.TopRight + "\n")

	// Calculate vertical position
	var startRow int
	switch alignment.Vertical {
	case "top":
		startRow = padding
	case "bottom":
		startRow = height - padding - textHeight
	default: // center
		startRow = height/2 - textHeight/2 - 1
	}

	// Print empty lines before text
	for i := 1; i < startRow; i++ {
		fmt.Print(borderColor + box.Vertical + colors.Reset + bgColor)
		for j := 0; j < width; j++ {
			fmt.Print(" ")
		}
		fmt.Print(borderColor + box.Vertical + "\n")
	}

	// Print text lines
	for _, line := range wrappedText {
		fmt.Print(borderColor + box.Vertical + colors.Reset + textColor + bgColor)
		lineLength := len(line)
		leftPadding := padding

		switch alignment.Horizontal {
		case "left":
			leftPadding = padding
		case "right":
			leftPadding = width - lineLength - padding
		default: // center
			leftPadding = (width - lineLength) / 2
		}

		for i := 0; i < leftPadding; i++ {
			fmt.Print(" ")
		}
		fmt.Print(textColor + line + colors.Reset + bgColor)
		rightPadding := width - leftPadding - lineLength
		for i := 0; i < rightPadding; i++ {
			fmt.Print(" ")
		}
		fmt.Print(borderColor + box.Vertical + "\n")
	}

	// Print empty lines after text
	for i := startRow + textHeight + 1; i < height-1; i++ {
		fmt.Print(borderColor + box.Vertical + colors.Reset + bgColor)
		for j := 0; j < width; j++ {
			fmt.Print(" ")
		}
		fmt.Print(borderColor + box.Vertical + "\n")
	}

	// Bottom border
	fmt.Print(borderColor)
	fmt.Print(box.BottomLeft)
	for i := 0; i < width; i++ {
		fmt.Print(box.Horizontal)
	}
	fmt.Print(box.BottomRight)
	fmt.Print(colors.Reset)
}

func PrintBannerColors() {
	// Group colors by type
	colorGroups := map[string][]string{
		"Regular Colors":    {"red", "green", "yellow", "blue", "purple", "cyan", "gray", "white", "black"},
		"Bold Colors":       {"bold_red", "bold_green", "bold_yellow", "bold_blue", "bold_purple", "bold_cyan", "bold_gray", "bold_white", "bold_black"},
		"Background Colors": {"bg_red", "bg_green", "bg_yellow", "bg_blue", "bg_purple", "bg_cyan", "bg_gray", "bg_white", "bg_black"},
		"Gray Shades":       {"gray1", "gray2", "gray3", "gray4", "gray5"},
		"Background Grays":  {"bg_gray1", "bg_gray2", "bg_gray3", "bg_gray4", "bg_gray5"},
	}

	fmt.Println("\nAvailable Colors:")
	for group, colorNames := range colorGroups {
		fmt.Printf("\n%s%s%s\n", colors.BoldWhite, group, colors.Reset)
		for _, name := range colorNames {
			if color, exists := colors.ColorMap[name]; exists {
				fmt.Printf("  %s%-15s%s %s(%-12s)%s\n",
					color, "Sample Text", colors.Reset,
					colors.Gray, name, colors.Reset)
			}
		}
	}
	fmt.Println()
}

func ParseAndPrintBanner(args []string) {
	// Show color list if --colors flag is present
	for _, arg := range args {
		if arg == "--colors" {
			PrintBannerColors()
			return
		}
	}

	if len(args) < 2 {
		PrintError("Error: Text is required for banner\n")
		return
	}

	// Join all args and find quoted text
	fullCmd := strings.Join(args[1:], " ")
	var text string

	// Find text between quotes
	if strings.Count(fullCmd, "\"") >= 2 {
		// Extract text between first set of quotes
		start := strings.Index(fullCmd, "\"")
		end := strings.Index(fullCmd[start+1:], "\"") + start + 1
		if start != -1 && end != -1 {
			text = fullCmd[start+1 : end]
			// Remove the quoted text from fullCmd for flag processing
			fullCmd = fullCmd[:start] + fullCmd[end+1:]
		}
	}

	// If no quoted text found, use first non-flag argument
	if text == "" {
		for _, arg := range args[1:] {
			if !strings.HasPrefix(arg, "-") {
				text = arg
				break
			}
		}
	}

	boxStyle := "single"
	textColor := colors.BoldWhite
	bgColor := colors.Reset
	borderColor := colors.Reset
	width := -1
	height := -1
	alignment := TextAlignment{
		Horizontal: "center",
		Vertical:   "center",
	}

	// Split remaining command for flag processing
	flagArgs := strings.Fields(fullCmd)
	for i := 0; i < len(flagArgs); i++ {
		switch flagArgs[i] {
		case "-s", "--style":
			if i+1 < len(flagArgs) {
				if _, exists := BoxTypes[flagArgs[i+1]]; exists {
					boxStyle = flagArgs[i+1]
				} else {
					PrintWarning("Invalid box style, using default\n")
				}
				i++
			}
		case "-c", "--color":
			if i+1 < len(flagArgs) {
				if color, exists := colors.ColorMap["bold_"+flagArgs[i+1]]; exists {
					textColor = color
				} else {
					PrintWarning("Invalid text color, using default\n")
				}
				i++
			}
		case "-b", "--background":
			if i+1 < len(flagArgs) {
				if color, exists := colors.ColorMap["bg_"+flagArgs[i+1]]; exists {
					bgColor = color
				} else {
					PrintWarning("Invalid background color, using default\n")
				}
				i++
			}
		case "-w", "--width":
			if i+1 < len(flagArgs) {
				if w, err := strconv.Atoi(flagArgs[i+1]); err == nil && w > 0 {
					width = w
				}
				i++
			}
		case "-h", "--height":
			if i+1 < len(flagArgs) {
				if h, err := strconv.Atoi(flagArgs[i+1]); err == nil && h > 0 {
					height = h
				}
				i++
			}
		case "-a", "--align":
			if i+1 < len(flagArgs) {
				switch flagArgs[i+1] {
				case "left", "right", "center":
					alignment.Horizontal = flagArgs[i+1]
				case "top", "bottom":
					alignment.Vertical = flagArgs[i+1]
				}
				i++
			}
		case "-bc", "--border-color":
			if i+1 < len(flagArgs) {
				if color, exists := colors.ColorMap["bold_"+flagArgs[i+1]]; exists {
					borderColor = color
				} else {
					PrintWarning("Invalid border color, using default\n")
				}
				i++
			}
		}
	}

	text = strings.TrimSpace(text)
	if width < 0 {
		width = len(text) + 4
	}
	if height < 0 {
		height = 3
	}

	PrintBanner(text, boxStyle, textColor, bgColor, borderColor, width, height, alignment)
}

func PrintWindow(icon string, title string, content string, bgColor string, borderColor string,
	titleColor string, contentColor string, width int, height int) {

	// Print title bar
	titleText := icon + " " + title
	titleAlignment := TextAlignment{
		Horizontal: "left",
		Vertical:   "center",
	}
	PrintBanner(titleText, "single", titleColor, bgColor, borderColor, width, 3, titleAlignment)

	// Print content area
	contentAlignment := TextAlignment{
		Horizontal: "left",
		Vertical:   "top",
	}
	fmt.Print("\n")
	// Remove top border by using special handling in PrintBanner
	PrintBanner(content, "single", contentColor, bgColor, borderColor, width, height, contentAlignment)
}

func ParseAndPrintWindow(args []string) {
	if len(args) < 3 {
		PrintError("Error: Icon and title are required for window\n")
		return
	}

	// Parse icon and title first
	icon := args[1]

	// Join remaining args and find quoted title
	fullCmd := strings.Join(args[2:], " ")
	var title, content string

	// Find title between first set of quotes
	if strings.Count(fullCmd, "\"") >= 2 { // Changed from 4 to 2 since we only need title quotes
		// Extract title
		start := strings.Index(fullCmd, "\"")
		end := strings.Index(fullCmd[start+1:], "\"") + start + 1
		if start != -1 && end != -1 {
			title = fullCmd[start+1 : end]
			// Remove the title from fullCmd
			fullCmd = fullCmd[:start] + fullCmd[end+1:]

			// Try to extract content if it exists
			if strings.Count(fullCmd, "\"") >= 2 {
				start = strings.Index(fullCmd, "\"")
				end = strings.Index(fullCmd[start+1:], "\"") + start + 1
				if start != -1 && end != -1 {
					content = fullCmd[start+1 : end]
					fullCmd = fullCmd[:start] + fullCmd[end+1:]
				}
			}
		}
	}

	// If no title was found in quotes, use first non-flag argument
	if title == "" {
		for _, arg := range args[2:] {
			if !strings.HasPrefix(arg, "-") {
				title = arg
				break
			}
		}
	}

	// Default values
	bgColor := colors.Reset
	borderColor := colors.BoldWhite
	titleColor := colors.BoldWhite
	contentColor := colors.BoldWhite
	width := 50
	height := 10

	// Parse flags
	flagArgs := strings.Fields(fullCmd)
	for i := 0; i < len(flagArgs); i++ {
		switch flagArgs[i] {
		case "-w", "--width":
			if i+1 < len(flagArgs) {
				if w, err := strconv.Atoi(flagArgs[i+1]); err == nil && w > 0 {
					width = w
				}
				i++
			}
		case "-h", "--height":
			if i+1 < len(flagArgs) {
				if h, err := strconv.Atoi(flagArgs[i+1]); err == nil && h > 0 {
					height = h
				}
				i++
			}
		case "-b", "--background":
			if i+1 < len(flagArgs) {
				if color, exists := colors.ColorMap["bg_"+flagArgs[i+1]]; exists {
					bgColor = color
				}
				i++
			}
		case "-bc", "--border-color":
			if i+1 < len(flagArgs) {
				if color, exists := colors.ColorMap["bold_"+flagArgs[i+1]]; exists {
					borderColor = color
				}
				i++
			}
		case "-tc", "--title-color":
			if i+1 < len(flagArgs) {
				if color, exists := colors.ColorMap["bold_"+flagArgs[i+1]]; exists {
					titleColor = color
				}
				i++
			}
		case "-cc", "--content-color":
			if i+1 < len(flagArgs) {
				if color, exists := colors.ColorMap["bold_"+flagArgs[i+1]]; exists {
					contentColor = color
				}
				i++
			}
		}
	}

	// Print the window
	PrintWindow(icon, title, content, bgColor, borderColor, titleColor, contentColor, width, height)
}

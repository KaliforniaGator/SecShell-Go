package help

import (
	"fmt"
	"secshell/admin"
	"secshell/colors"
	"secshell/globals"
	"secshell/ui/gui"
	"sort"
	"strings"
)

var (
	allHelpTopics      []HelpTopic
	currentTopics      []HelpTopic
	currentSortMode    string // "category" or "command"
	win                *gui.Window
	searchBox          *gui.TextBox
	commandList        *gui.Container
	detailsArea        *gui.Container // Changed from *gui.TextArea
	infoLabel          *gui.Label
	searchButton       *gui.Button
	sortCategoryButton *gui.Button
	sortCommandButton  *gui.Button
)

func updateCommandListDisplay() {
	if commandList == nil {
		return
	}

	content := []string{}
	if len(currentTopics) == 0 {
		content = append(content, colors.Gray+"<No matching commands>"+colors.Reset)
	} else {
		for _, topic := range currentTopics {
			displayString := fmt.Sprintf("%s%s%s (%s)", colors.BoldWhite, topic.Command, colors.Reset, topic.Category)
			content = append(content, displayString)
		}
	}
	commandList.SetContent(content)

	if len(currentTopics) > 0 {
		commandList.SelectedIndex = 0 // Select the first item
		updateDetailsView(currentTopics[0])
	} else {
		commandList.SelectedIndex = -1
		if detailsArea != nil {
			detailsArea.SetContent([]string{colors.Gray + "Select a command to see details." + colors.Reset})
		}
	}
}

func updateDetailsView(topic HelpTopic) {
	if detailsArea == nil {
		return
	}

	if !admin.IsAdmin() && !globals.IsCommandAllowed(topic.Command) {
		detailsArea.SetContent([]string{colors.Red + "Access Denied: This command requires admin privileges." + colors.Reset})
		return
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%sCommand:%s %s%s%s\n", colors.BoldCyan, colors.Reset, colors.BoldWhite, topic.Command, colors.Reset))
	builder.WriteString(fmt.Sprintf("%sCategory:%s %s\n\n", colors.BoldCyan, colors.Reset, topic.Category))
	builder.WriteString(fmt.Sprintf("%sDescription:%s\n%s\n\n", colors.BoldCyan, colors.Reset, topic.Description))
	builder.WriteString(fmt.Sprintf("%sUsage:%s\n%s\n\n", colors.BoldCyan, colors.Reset, topic.Usage))

	if len(topic.Examples) > 0 {
		builder.WriteString(fmt.Sprintf("%sExamples:%s\n", colors.BoldCyan, colors.Reset))
		for _, example := range topic.Examples {
			builder.WriteString(fmt.Sprintf("  %s>%s %s\n", colors.Green, colors.Reset, example))
		}
	}
	// Split the string by newlines for the container
	contentLines := strings.Split(strings.TrimRight(builder.String(), "\n"), "\n")
	detailsArea.SetContent(contentLines)
}

func performSearchAndSort() {
	searchText := ""
	if searchBox != nil {
		searchText = strings.ToLower(searchBox.Text)
	}

	filtered := []HelpTopic{}
	for _, topic := range allHelpTopics {
		if searchText == "" ||
			strings.Contains(strings.ToLower(topic.Command), searchText) ||
			strings.Contains(strings.ToLower(topic.Description), searchText) ||
			strings.Contains(strings.ToLower(topic.Category), searchText) {
			filtered = append(filtered, topic)
		}
	}

	if currentSortMode == "category" {
		sort.SliceStable(filtered, func(i, j int) bool {
			if filtered[i].Category == filtered[j].Category {
				return filtered[i].Command < filtered[j].Command
			}
			return filtered[i].Category < filtered[j].Category
		})
	} else { // "command"
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].Command < filtered[j].Command
		})
	}
	currentTopics = filtered
	updateCommandListDisplay()
}

// InteractiveHelpApp launches the interactive help UI.
func InteractiveHelpApp() {
	// Initialize allHelpTopics from the global HelpTopics map
	// Filter based on permissions
	allHelpTopics = []HelpTopic{}
	tempTopics := []string{} // To sort keys for consistent initial order if needed
	for k := range HelpTopics {
		tempTopics = append(tempTopics, k)
	}
	sort.Strings(tempTopics) // Sort command names

	for _, cmdName := range tempTopics {
		topic := HelpTopics[cmdName]
		if admin.IsAdmin() || globals.IsCommandAllowed(topic.Command) {
			allHelpTopics = append(allHelpTopics, topic)
		}
	}

	currentSortMode = "category" // Default sort mode

	fmt.Print(gui.ClearScreenAndBuffer())
	termWidth := gui.GetTerminalWidth()
	termHeight := gui.GetTerminalHeight()

	winWidth := termWidth * 9 / 10
	if winWidth < 100 {
		winWidth = 100
	} // Min width
	winHeight := termHeight * 9 / 10
	if winHeight < 25 {
		winHeight = 25
	} // Min height
	winX := (termWidth - winWidth) / 2
	winY := (termHeight - winHeight) / 2

	win = gui.NewWindow("💡", "Interactive Help ", winX, winY, winWidth, winHeight,
		"double", colors.BoldYellow, colors.BoldGreen, colors.BgBlack, colors.White)

	contentAreaWidth := winWidth - 2
	contentAreaHeight := winHeight - 2
	currentY := 1

	// Info Label
	infoLabel = gui.NewLabel("Search, Sort, and Select commands. Tab/Shift-Tab to navigate. Enter to activate. q/Ctrl+C to Quit.", 1, currentY, colors.Gray)
	win.AddElement(infoLabel)
	currentY += 2

	// Search and Sort Controls
	searchLabel := gui.NewLabel("Search:", 1, currentY, colors.White)
	win.AddElement(searchLabel)
	searchBoxWidth := contentAreaWidth / 3
	searchBox = gui.NewTextBox("", 10, currentY, searchBoxWidth, colors.BgWhite+colors.Black, colors.BgCyan+colors.BoldBlack)
	win.AddElement(searchBox)

	buttonX := 10 + searchBoxWidth + 2
	buttonWidth := 10

	searchButton = gui.NewButton("Search", buttonX, currentY, buttonWidth, colors.BoldGreen, colors.BgGreen+colors.BoldWhite, func() bool {
		performSearchAndSort()
		return false // Don't quit
	})
	win.AddElement(searchButton)
	buttonX += buttonWidth + 1

	sortCategoryButton = gui.NewButton("By Category", buttonX, currentY, buttonWidth+2, colors.BoldBlue, colors.BgBlue+colors.BoldWhite, func() bool {
		currentSortMode = "category"
		performSearchAndSort()
		return false
	})
	win.AddElement(sortCategoryButton)
	buttonX += buttonWidth + 2 + 1

	sortCommandButton = gui.NewButton("By Command", buttonX, currentY, buttonWidth+2, colors.BoldMagenta, colors.BgMagenta+colors.BoldWhite, func() bool {
		currentSortMode = "command"
		performSearchAndSort()
		return false
	})
	win.AddElement(sortCommandButton)
	currentY += 2

	// Main Panes
	leftPaneWidth := contentAreaWidth / 3
	rightPaneX := leftPaneWidth + 2 // +1 for divider, +1 for spacing
	rightPaneWidth := contentAreaWidth - rightPaneX

	if leftPaneWidth < 20 {
		leftPaneWidth = 20
	} // Min width for left pane
	if rightPaneWidth < 20 {
		rightPaneWidth = 20
	} // Min width for right pane
	if rightPaneX >= contentAreaWidth-1 { // Adjust if panes are too squeezed
		rightPaneX = leftPaneWidth + 1
		rightPaneWidth = contentAreaWidth - rightPaneX
	}

	paneHeight := contentAreaHeight - currentY
	if paneHeight < 5 {
		paneHeight = 5
	}

	// Command List Container (Left Pane)
	commandList = gui.NewContainer(1, currentY, leftPaneWidth, paneHeight, []string{})
	commandList.Color = colors.BgBlack // Match window background
	commandList.SelectionColor = colors.BgBlue + colors.BoldWhite
	commandList.OnItemSelected = func(index int) {
		if index >= 0 && index < len(currentTopics) {
			updateDetailsView(currentTopics[index])
		}
	}
	win.AddElement(commandList)

	// Divider
	dividerX := leftPaneWidth + 1
	for i := 0; i < paneHeight; i++ {
		divChar := gui.NewLabel(gui.BoxTypes["single"].Vertical, dividerX, currentY+i, colors.Yellow)
		win.AddElement(divChar)
	}

	// Details Area (Right Pane)
	detailsArea = gui.NewContainer(rightPaneX, currentY, rightPaneWidth, paneHeight, []string{})
	detailsArea.Color = colors.BgBlack + colors.White // Match window background, text white
	// detailsArea.IsActive = false // Not applicable for Container in the same way
	win.AddElement(detailsArea)

	// Initial population
	performSearchAndSort()

	win.WindowActions()

	// Cleanup references
	win = nil
	searchBox = nil
	commandList = nil
	detailsArea = nil
	infoLabel = nil
	searchButton = nil
	sortCategoryButton = nil
	sortCommandButton = nil
	allHelpTopics = nil
	currentTopics = nil

}

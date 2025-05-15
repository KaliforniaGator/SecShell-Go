package tests

import (
	"fmt"
	"secshell/colors"
	"secshell/ui/gui" // Import the gui package
	// Removed unused imports: strconv, strings, time
)

// --- Note Data Structure ---
type Note struct {
	Title   string
	Content string
}

// --- Custom KeyStrokeHandler ---
type NotesAppKeyHandler struct {
	notesListContainer *gui.Container
	notes              *[]Note
	selectedNoteIndex  *int
	loadNoteForEditing func(int)
}

// HandleKeyStroke processes keyboard input for the notes app
func (h *NotesAppKeyHandler) HandleKeyStroke(key []byte, w *gui.Window) (handled bool, needsRender bool, shouldQuit bool) {
	// Check if we have Enter key press when the notes list container is focused
	if len(key) == 1 && (key[0] == '\r' || key[0] == '\n') && h.notesListContainer.IsActive {
		highlightedIdx := h.notesListContainer.GetHighlightedIndex()
		if highlightedIdx >= 0 && highlightedIdx < len(*h.notes) {
			// Update the actual selection
			*h.selectedNoteIndex = highlightedIdx
			h.notesListContainer.SelectedIndex = highlightedIdx

			// Load the selected note for editing
			if h.loadNoteForEditing != nil {
				h.loadNoteForEditing(highlightedIdx)
			}

			return true, true, false
		}
	}

	// For other keys, let the default handler process them
	return false, false, false
}

// --- Main Application Function ---
func TestSegmentsApp() {
	// --- Application State ---
	notes := []Note{
		{"Welcome", "This is a simple notes app.\nSelect a note on the left or create a new one."},
		{"Shopping List", "Milk\nEggs\nBread\nCoffee"},
		{"Ideas", "Build a TUI framework.\nLearn Go concurrency.\nTest terminal capabilities."},
	}
	selectedNoteIndex := -1 // Index of the note currently being edited, -1 for new note

	// --- UI Element References ---
	var notesListContainer *gui.Container
	var titleInput *gui.TextBox
	var contentInput *gui.TextArea // Using TextArea
	var infoLabel *gui.Label       // To display status messages

	// --- Helper Functions ---

	// Updates the notes list container content
	updateNotesListDisplay := func() {
		content := []string{}
		if len(notes) == 0 {
			content = append(content, colors.Gray+"<No notes>"+colors.Reset) // Add color directly to the text
		} else {
			for i, note := range notes {
				// Display index and title
				titleLine := fmt.Sprintf("%d: %s", i, note.Title)
				content = append(content, titleLine)
			}
		}
		if notesListContainer != nil {
			notesListContainer.SetContent(content) // This updates the container and its scrollbar

			// Ensure selection index remains valid after update
			if selectedNoteIndex >= len(notes) {
				selectedNoteIndex = -1 // Reset if index is now invalid
			}

			// Update the SelectedIndex property to match our application's selectedNoteIndex
			notesListContainer.SelectedIndex = selectedNoteIndex
		}
	}

	// Clears the editor fields by setting Text to empty
	clearEditor := func() {
		if titleInput != nil {
			titleInput.Text = ""
			// Cursor position and pristine state are managed internally or by interaction loop
		}
		if contentInput != nil {
			contentInput.SetText("") // Use SetText for TextArea
		}
		selectedNoteIndex = -1 // Indicate no specific note is being edited
		if notesListContainer != nil {
			notesListContainer.SelectedIndex = -1 // Clear selection in list
		}
		if infoLabel != nil {
			infoLabel.Text = "Editor cleared. Ready for new note."
			infoLabel.Color = colors.Gray
		}
	}

	// Loads a note into the editor fields by setting Text
	loadNoteForEditing := func(index int) {
		if index >= 0 && index < len(notes) {
			note := notes[index]
			if titleInput != nil {
				titleInput.Text = note.Title
				// Cursor position and pristine state are managed internally or by interaction loop
			}
			if contentInput != nil {
				contentInput.SetText(note.Content) // Use SetText for TextArea
			}
			selectedNoteIndex = index
			if infoLabel != nil {
				infoLabel.Text = fmt.Sprintf("Editing note %d: %s", index, note.Title)
				infoLabel.Color = colors.Cyan
			}

			// Update both the SelectedIndex and HighlightedIndex for visual consistency
			if notesListContainer != nil {
				notesListContainer.SelectedIndex = index
				notesListContainer.HighlightedIndex = index
			}
		} else {
			if infoLabel != nil {
				infoLabel.Text = fmt.Sprintf("Error: Invalid note index %d.", index)
				infoLabel.Color = colors.Red
			}
			clearEditor() // Clear editor if index is invalid
		}
	}

	// --- UI Setup ---
	fmt.Print(gui.ClearScreenAndBuffer())
	termWidth := gui.GetTerminalWidth()
	termHeight := gui.GetTerminalHeight()

	// Window dimensions
	winWidth := termWidth * 9 / 10
	if winWidth < 80 {
		winWidth = 80
	}
	winHeight := termHeight * 9 / 10
	if winHeight < 20 {
		winHeight = 20
	}
	winX := (termWidth - winWidth) / 2
	winY := (termHeight - winHeight) / 2

	// Create Window
	notesWin := gui.NewWindow("📝", "Segmented Notes App", winX, winY, winWidth, winHeight,
		"rounded", colors.BoldYellow, colors.Yellow, colors.BgBlack, colors.White)

	contentAreaWidth := winWidth - 2
	contentAreaHeight := winHeight - 2
	leftSegmentWidth := contentAreaWidth/3 - 1                   // Subtract 1 for divider
	rightSegmentWidth := contentAreaWidth - leftSegmentWidth - 3 // Subtract 2: one for each segment margin and one for divider
	rightSegmentX := leftSegmentWidth + 3                        // Leave space for divider

	// Ensure widths are not negative if window is very small
	if leftSegmentWidth < 0 {
		leftSegmentWidth = 0
	}
	if rightSegmentWidth < 0 {
		rightSegmentWidth = 0
	}

	currentY := 1 // Relative Y within window content

	// --- Info Label (Top) ---
	infoLabel = gui.NewLabel("Welcome! Select a note or create one.", 1, currentY, colors.Gray)
	notesWin.AddElement(infoLabel)
	currentY += 2

	// --- Left Segment: Notes List ---
	notesLabel := gui.NewLabel("Notes:", 1, currentY, colors.BoldWhite)
	notesWin.AddElement(notesLabel)
	currentY++

	listHeight := contentAreaHeight - currentY - 1 // Use remaining height below the label
	if listHeight < 1 {
		listHeight = 1
	}
	notesListContainer = gui.NewContainer(1, currentY, leftSegmentWidth, listHeight, []string{}) // Use calculated width
	notesListContainer.Color = colors.BgYellow + colors.Black                                    // Yellow background with black text
	notesListContainer.SelectionColor = colors.BgBlue + colors.BoldWhite                         // Keep selection highlight
	notesListContainer.OnItemSelected = func(index int) {
		// This callback is triggered by Enter key when the container is focused
		loadNoteForEditing(index)
	}
	notesWin.AddElement(notesListContainer)

	// --- Draw vertical line divider ---
	dividerX := leftSegmentWidth + 1
	dividerY := 1 // Start at the top of content area
	dividerHeight := contentAreaHeight - 1
	for i := 0; i < dividerHeight; i++ {
		divider := gui.NewLabel("│", dividerX, dividerY+i, colors.Gray)
		notesWin.AddElement(divider)
	}

	// --- Right Segment: Editor ---
	editorY := 3 // Start editor elements slightly lower, aligned with Notes label
	editorInputY := editorY

	// Title
	titleLabel := gui.NewLabel("Title:", rightSegmentX, editorInputY, colors.White)
	notesWin.AddElement(titleLabel)
	editorInputY++
	titleInput = gui.NewTextBox("", rightSegmentX, editorInputY, rightSegmentWidth, colors.BgBlack+colors.White, colors.BgCyan+colors.BoldBlack) // Use calculated width
	notesWin.AddElement(titleInput)
	editorInputY += 2 // Add space

	// Content
	contentLabel := gui.NewLabel("Content:", rightSegmentX, editorInputY, colors.White)
	notesWin.AddElement(contentLabel)
	editorInputY++
	// Calculate height for TextArea, leaving space for buttons and bottom margin
	textAreaHeight := contentAreaHeight - editorInputY - 4 // 4 = 1 for margin + 1 for button row + 2 for button height/padding
	if textAreaHeight < 3 {
		textAreaHeight = 3 // Minimum height for TextArea (1 text line + 1 count line + scrollbar)
	}
	// Use NewTextArea instead of NewTextBox
	contentInput = gui.NewTextArea("", rightSegmentX, editorInputY, rightSegmentWidth, textAreaHeight, 0, // Use calculated width
		colors.BgBlack+colors.White, colors.BgCyan+colors.BoldBlack, true, true) // Show word and char count
	contentInput.IsActive = false     // Start inactive, but allow it to be focused
	notesWin.AddElement(contentInput) // TextArea added to the window

	// Calculate Y position for buttons based on the bottom of the window
	buttonY := contentAreaHeight - 2 // Position buttons near the bottom

	// Buttons
	buttonWidth := 10
	buttonSpacing := 2
	totalButtonsWidth := (buttonWidth * 3) + (buttonSpacing * 2)
	// Ensure buttons fit within the right segment's width
	if totalButtonsWidth > rightSegmentWidth {
		// Adjust button width or handle differently if they don't fit
		// For now, let's assume they fit or will be truncated by rendering.
	}
	buttonStartX := rightSegmentX + (rightSegmentWidth-totalButtonsWidth)/2
	if buttonStartX < rightSegmentX {
		buttonStartX = rightSegmentX
	}

	// New Button
	newButton := gui.NewButton("New", buttonStartX, buttonY, buttonWidth, colors.BoldGreen, colors.BgGreen+colors.BoldWhite, func() bool {
		clearEditor()
		updateNotesListDisplay() // Update list to remove selection highlight
		// Optionally move focus back to title input? Requires focus API extension.
		return false // Don't quit
	})
	notesWin.AddElement(newButton)

	// Save Button
	saveButtonX := buttonStartX + buttonWidth + buttonSpacing
	saveButton := gui.NewButton("Save", saveButtonX, buttonY, buttonWidth, colors.BoldBlue, colors.BgBlue+colors.BoldWhite, func() bool {
		title := titleInput.Text
		content := contentInput.GetText() // Use GetText for TextArea
		if title == "" {
			infoLabel.Text = "Error: Title cannot be empty."
			infoLabel.Color = colors.Red
			return false
		}

		if selectedNoteIndex >= 0 && selectedNoteIndex < len(notes) {
			// Update existing note
			notes[selectedNoteIndex].Title = title
			notes[selectedNoteIndex].Content = content
			infoLabel.Text = fmt.Sprintf("Note %d updated.", selectedNoteIndex)
			infoLabel.Color = colors.Blue
		} else {
			// Add new note
			newNote := Note{Title: title, Content: content}
			notes = append(notes, newNote)
			selectedNoteIndex = len(notes) - 1 // Select the newly added note
			infoLabel.Text = "New note saved."
			infoLabel.Color = colors.Green
		}
		updateNotesListDisplay()
		// Keep the current note loaded in the editor after saving
		loadNoteForEditing(selectedNoteIndex) // Reload to ensure consistency and selection highlight
		return false                          // Don't quit
	})
	notesWin.AddElement(saveButton)

	// Delete Button
	deleteButtonX := saveButtonX + buttonWidth + buttonSpacing
	deleteButton := gui.NewButton("Delete", deleteButtonX, buttonY, buttonWidth, colors.BoldRed, colors.BgRed+colors.BoldWhite, func() bool {
		if selectedNoteIndex >= 0 && selectedNoteIndex < len(notes) {
			indexToDelete := selectedNoteIndex
			noteTitle := notes[indexToDelete].Title
			// Remove note from slice
			notes = append(notes[:indexToDelete], notes[indexToDelete+1:]...)
			infoLabel.Text = fmt.Sprintf("Note '%s' deleted.", noteTitle)
			infoLabel.Color = colors.Red
			clearEditor()            // Clear editor after deleting
			updateNotesListDisplay() // Update list display
		} else {
			infoLabel.Text = "Error: No note selected to delete."
			infoLabel.Color = colors.Red
		}
		return false // Don't quit
	})
	notesWin.AddElement(deleteButton)

	// --- Initial Display & Interaction ---
	updateNotesListDisplay() // Load initial notes into the list
	if len(notes) > 0 {
		loadNoteForEditing(0) // Load the first note initially
	} else {
		clearEditor() // Start with a clear editor if no notes exist
	}

	// Create and set the custom key handler
	keyHandler := &NotesAppKeyHandler{
		notesListContainer: notesListContainer,
		notes:              &notes,
		selectedNoteIndex:  &selectedNoteIndex,
		loadNoteForEditing: loadNoteForEditing,
	}
	notesWin.SetKeyStrokeHandler(keyHandler)

	// Start the interaction loop
	notesWin.WindowActions()
}

// Removed unused helper functions: ColorizeText, getTextWidth, wrapTextSimple
// Assuming ColorizeText is available via gui or colors package if needed by updateNotesListDisplay

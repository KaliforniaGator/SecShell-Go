package filemanager

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"secshell/colors"
	"secshell/ui/gui"
	"sort"
	"strings"
	// syscall may be needed by MoveFolder's more granular error checks,
	// but the current MoveFolder implementation doesn't explicitly require it here.
	// "syscall"
)

// UI element references
var (
	fmWindow          *gui.Window
	pathLabel         *gui.Label
	fileListContainer *gui.Container
	infoLabel         *gui.Label
	menuBar           *gui.MenuBar
	activePrompt      *gui.Prompt // Current active prompt, if any
)

// Application state
var (
	currentPath    string
	currentEntries []fs.DirEntry // Stores the actual entries in the current directory

	// State for copy/move operations
	operationState int    // 0: none, 1: source selected for copy, 2: source selected for move
	clipboardPath  string // Full path of the item to be copied/moved
	clipboardIsDir bool   // True if the clipboard item is a directory
	clipboardName  string // Name of the item on the clipboard
)

const (
	opNone = iota
	opSourceSelectedForCopy
	opSourceSelectedForMove
)

const (
	dirPrefix  = "[D] "
	filePrefix = "    "
)

// FileManagerKeyHandler handles keyboard shortcuts for the file manager
type FileManagerKeyHandler struct{}

// HandleKeyStroke processes keyboard shortcuts for the file manager
func (h *FileManagerKeyHandler) HandleKeyStroke(key []byte, w *gui.Window) (handled bool, needsRender bool, shouldQuit bool) {
	// If a prompt is active, let the window handle it
	if activePrompt != nil && activePrompt.IsActive {
		return false, false, false
	}

	// Check for Alt+F to activate File menu (Alt sends ESC then the key)
	if len(key) == 2 && key[0] == 27 && key[1] == 102 { // ESC followed by 'f'
		menuBar.Activate()
		// Activate File menu (first item)
		if len(menuBar.Menu.Items) > 0 {
			menuBar.Menu.SelectedIdx = 0
			menuBar.Menu.Items[0].IsActive = true
		}
		return true, true, false
	}

	// Check for Alt+E to activate Edit menu
	if len(key) == 2 && key[0] == 27 && key[1] == 101 { // ESC followed by 'e'
		menuBar.Activate()
		// Activate Edit menu (second item)
		if len(menuBar.Menu.Items) > 1 {
			menuBar.Menu.SelectedIdx = 1
			menuBar.Menu.Items[1].IsActive = true
		}
		return true, true, false
	}

	// Check for Alt+N to activate Nav menu
	if len(key) == 2 && key[0] == 27 && key[1] == 110 { // ESC followed by 'n'
		menuBar.Activate()
		// Activate Nav menu (third item)
		if len(menuBar.Menu.Items) > 2 {
			menuBar.Menu.SelectedIdx = 2
			menuBar.Menu.Items[2].IsActive = true
		}
		return true, true, false
	}

	// Check for single key commands
	if len(key) == 1 {
		switch key[0] {
		case 'q', 'Q': // Quit
			return true, false, true
		case 'r', 'R': // Refresh
			listDirectoryContents()
			return true, true, false
		case 'u', 'U': // Up directory
			parentPath := filepath.Dir(currentPath)
			if parentPath != currentPath { // Avoid getting stuck at root "/" whose parent is "/"
				currentPath = parentPath
				listDirectoryContents()
				fileListContainer.SelectedIndex = 0
				fileListContainer.GetScrollbar().SetValue(0)
			} else {
				infoLabel.Text = "Already at root."
				infoLabel.Color = colors.Yellow
			}
			return true, true, false
		case 'c', 'C': // Copy
			handleCopyItem()
			return true, true, false
		case 'm', 'M': // Move
			handleMoveItem()
			return true, true, false
		case 'p', 'P': // Paste
			showPastePrompt()
			return true, true, false
		case 'f', 'F': // Create folder
			showCreateFolderPrompt()
			return true, true, false
		case 't', 'T': // Create file (Touch)
			showCreateFilePrompt()
			return true, true, false
		case 'd', 'D': // Delete
			showDeleteConfirmation()
			return true, true, false
		}
	}

	// Check for special keys (like F1, F2, Delete, etc.)
	if len(key) == 3 && key[0] == 27 && key[1] == 79 { // F1-F4 keys
		switch key[2] {
		case 80: // F1 - Create folder
			showCreateFolderPrompt()
			return true, true, false
		case 81: // F2 - Create file
			showCreateFilePrompt()
			return true, true, false
		}
	} else if len(key) == 4 && key[0] == 27 && key[1] == 91 && key[3] == 126 { // Some extended keys
		switch key[2] {
		case 51: // Delete key
			showDeleteConfirmation()
			return true, true, false
		}
	}

	return false, false, false
}

// listDirectoryContents reads the currentPath, updates currentEntries,
// and refreshes the fileListContainer and pathLabel.
func listDirectoryContents() {
	if pathLabel == nil || fileListContainer == nil || infoLabel == nil {
		return // UI not initialized
	}

	pathLabel.Text = "Path: " + currentPath
	pathLabel.Color = colors.BoldWhite

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		infoLabel.Text = "Error reading directory: " + err.Error()
		infoLabel.Color = colors.Red
		currentEntries = []fs.DirEntry{}
		fileListContainer.SetContent([]string{colors.Gray + "<Error loading>" + colors.Reset})
		return
	}

	currentEntries = entries // Store the raw entries

	// Sort entries: directories first, then files, both alphabetically
	sort.SliceStable(currentEntries, func(i, j int) bool {
		isDirI := currentEntries[i].IsDir()
		isDirJ := currentEntries[j].IsDir()
		if isDirI && !isDirJ {
			return true
		}
		if !isDirI && isDirJ {
			return false
		}
		return strings.ToLower(currentEntries[i].Name()) < strings.ToLower(currentEntries[j].Name())
	})

	displayContent := []string{}
	if len(currentEntries) == 0 {
		displayContent = append(displayContent, colors.Gray+"<Empty directory>"+colors.Reset)
	} else {
		for _, entry := range currentEntries {
			name := entry.Name()
			prefix := filePrefix
			color := colors.White
			if entry.IsDir() {
				prefix = dirPrefix
				color = colors.BoldCyan
			}

			// Highlight if this item is on the clipboard
			fullItemPath := filepath.Join(currentPath, name)
			if (operationState == opSourceSelectedForCopy || operationState == opSourceSelectedForMove) && fullItemPath == clipboardPath {
				if operationState == opSourceSelectedForCopy {
					name = colors.Yellow + name + " (copying)" + color // Mark for copy
				} else {
					name = colors.Orange + name + " (moving)" + color // Mark for move
				}
			}

			// Ensure the selected item in the list container is also visually distinct if needed by GUI package
			// For now, just prepending prefix and color
			displayContent = append(displayContent, fmt.Sprintf("%s%s%s%s", color, prefix, name, colors.Reset))
		}
	}

	fileListContainer.SetContent(displayContent)
	infoLabel.Text = fmt.Sprintf("Listed %d items.", len(currentEntries))
	infoLabel.Color = colors.Gray
}

// handleItemActivation is called when an item in the fileListContainer is "activated" (e.g., Enter pressed).
func handleItemActivation(index int) {
	if index < 0 || index >= len(currentEntries) {
		infoLabel.Text = "Invalid selection."
		infoLabel.Color = colors.Yellow
		return
	}

	selectedEntry := currentEntries[index]
	newPath := filepath.Join(currentPath, selectedEntry.Name())

	if selectedEntry.IsDir() {
		currentPath = filepath.Clean(newPath)
		listDirectoryContents()
		fileListContainer.SelectedIndex = 0          // Reset selection to top
		fileListContainer.GetScrollbar().SetValue(0) // Reset scroll
	} else {
		// For files, just show info for now
		infoLabel.Text = fmt.Sprintf("Selected file: %s", selectedEntry.Name())
		infoLabel.Color = colors.Cyan
	}
}

// showCreateFolderPrompt displays a dialog to create a new folder
func showCreateFolderPrompt() {
	// Set up buttons
	okButton := gui.NewPromptButton("Create", colors.BoldGreen, colors.BgGreen+colors.White, func() bool {
		// Access text input from prompt (this would require modification to Prompt to support text input)
		// For now, we'll use a default/hardcoded name as in the original
		name := "NewFolder" // Simulated input

		newFolderPath := filepath.Join(currentPath, name)
		err := CreateFolder(newFolderPath)
		if err != nil {
			infoLabel.Text = "Error creating folder: " + err.Error()
			infoLabel.Color = colors.Red
		} else {
			infoLabel.Text = "Folder '" + name + "' created."
			infoLabel.Color = colors.Green
			listDirectoryContents()
		}
		fmWindow.RemoveElement(activePrompt)
		activePrompt = nil
		return false // Don't exit app
	})

	cancelButton := gui.NewPromptButton("Cancel", colors.BoldRed, colors.BgRed+colors.White, func() bool {
		infoLabel.Text = "Folder creation cancelled."
		infoLabel.Color = colors.Yellow
		fmWindow.RemoveElement(activePrompt)
		activePrompt = nil
		return false // Don't exit app
	})

	// Create the prompt
	width := 40
	x := (fmWindow.Width - width) / 2
	y := fmWindow.Height / 3

	activePrompt = gui.NewDialogPrompt(
		"Create Folder",
		"Enter name for new folder:",
		x, y, width,
		colors.BgBlack, colors.Cyan, colors.BoldWhite, colors.White,
		[]*gui.PromptButton{okButton, cancelButton},
	)
	// Prompt implements ZIndexer with z-index 1000 automatically
	activePrompt.SetActive(true)
	fmWindow.AddElement(activePrompt)
	fmWindow.Render() // Force immediate render
}

// showCreateFilePrompt displays a dialog to create a new file
func showCreateFilePrompt() {
	// Set up buttons
	okButton := gui.NewPromptButton("Create", colors.BoldGreen, colors.BgGreen+colors.White, func() bool {
		// Simulated input for now
		name := "NewFile.txt"

		newFilePath := filepath.Join(currentPath, name)
		err := CreateEmptyFile(newFilePath)
		if err != nil {
			infoLabel.Text = "Error creating file: " + err.Error()
			infoLabel.Color = colors.Red
		} else {
			infoLabel.Text = "File '" + name + "' created."
			infoLabel.Color = colors.Green
			listDirectoryContents()
		}
		fmWindow.RemoveElement(activePrompt)
		activePrompt = nil
		return false // Don't exit app
	})

	cancelButton := gui.NewPromptButton("Cancel", colors.BoldRed, colors.BgRed+colors.White, func() bool {
		infoLabel.Text = "File creation cancelled."
		infoLabel.Color = colors.Yellow
		fmWindow.RemoveElement(activePrompt)
		activePrompt = nil
		return false // Don't exit app
	})

	// Create the prompt
	width := 40
	x := (fmWindow.Width - width) / 2
	y := fmWindow.Height / 3

	activePrompt = gui.NewDialogPrompt(
		"Create File",
		"Enter name for new file:",
		x, y, width,
		colors.BgBlack, colors.Cyan, colors.BoldWhite, colors.White,
		[]*gui.PromptButton{okButton, cancelButton},
	)
	// Prompt implements ZIndexer with z-index 1000 automatically
	activePrompt.SetActive(true)
	fmWindow.AddElement(activePrompt)
	fmWindow.Render() // Force immediate render
}

// showDeleteConfirmation displays a confirmation dialog for deleting items
func showDeleteConfirmation() {
	selectedIndex := fileListContainer.SelectedIndex
	if selectedIndex < 0 || selectedIndex >= len(currentEntries) {
		infoLabel.Text = "No item selected to delete."
		infoLabel.Color = colors.Yellow
		return
	}

	selectedEntry := currentEntries[selectedIndex]
	itemName := selectedEntry.Name()
	itemPath := filepath.Join(currentPath, itemName)

	// Set up buttons
	yesButton := gui.NewPromptButton("Yes", colors.BoldRed, colors.BgRed+colors.White, func() bool {
		var err error
		if selectedEntry.IsDir() {
			err = DeleteFolder(itemPath)
		} else {
			err = os.Remove(itemPath) // Using os.Remove for single files
		}

		if err != nil {
			infoLabel.Text = "Error deleting '" + itemName + "': " + err.Error()
			infoLabel.Color = colors.Red
		} else {
			infoLabel.Text = "'" + itemName + "' deleted."
			infoLabel.Color = colors.Green
			listDirectoryContents()
			// Adjust selection if possible
			if len(currentEntries) > 0 {
				if selectedIndex >= len(currentEntries) {
					fileListContainer.SelectedIndex = len(currentEntries) - 1
				}
			} else {
				fileListContainer.SelectedIndex = -1
			}
		}
		fmWindow.RemoveElement(activePrompt)
		activePrompt = nil
		return false // Don't exit app
	})

	noButton := gui.NewPromptButton("No", colors.BoldBlue, colors.BgBlue+colors.White, func() bool {
		infoLabel.Text = "Deletion cancelled."
		infoLabel.Color = colors.Yellow
		fmWindow.RemoveElement(activePrompt)
		activePrompt = nil
		return false // Don't exit app
	})

	// Create the prompt
	width := 50
	x := (fmWindow.Width - width) / 2
	y := fmWindow.Height / 3

	promptMessage := fmt.Sprintf("Delete '%s'? This is permanent.", itemName)
	activePrompt = gui.NewDialogPrompt(
		"Confirm Delete",
		promptMessage,
		x, y, width,
		colors.BgBlack, colors.Red, colors.BoldWhite, colors.White,
		[]*gui.PromptButton{yesButton, noButton},
	)
	// Prompt implements ZIndexer with z-index 1000 automatically
	activePrompt.SetActive(true)
	fmWindow.AddElement(activePrompt)
	fmWindow.Render() // Force immediate render
}

// showPastePrompt displays a prompt for pasting items
func showPastePrompt() {
	if operationState == opNone || clipboardPath == "" {
		infoLabel.Text = "Nothing to paste. Use Copy [C] or Move [M] first."
		infoLabel.Color = colors.Yellow
		return
	}

	// Set up buttons
	pasteButton := gui.NewPromptButton("Paste", colors.BoldGreen, colors.BgGreen+colors.White, func() bool {
		// Use original name for simplicity
		destName := clipboardName
		destPath := filepath.Join(currentPath, destName)

		// Prevent copying/moving onto itself
		if destPath == clipboardPath && operationState == opSourceSelectedForMove {
			infoLabel.Text = "Cannot move item onto itself without changing name."
			infoLabel.Color = colors.Red
			return true // Close prompt
		}

		var err error
		action := ""

		if operationState == opSourceSelectedForCopy {
			action = "copying"
			if clipboardIsDir {
				err = CopyFolder(clipboardPath, destPath)
			} else {
				err = copyFile(clipboardPath, destPath)
			}
		} else if operationState == opSourceSelectedForMove {
			action = "moving"
			if clipboardIsDir {
				err = MoveFolder(clipboardPath, destPath)
			} else {
				err = os.Rename(clipboardPath, destPath)
				if err != nil {
					// Fallback for cross-device moves
					if copyErr := copyFile(clipboardPath, destPath); copyErr == nil {
						if delErr := os.Remove(clipboardPath); delErr != nil {
							err = fmt.Errorf("moved by copy, but failed to delete original: %w", delErr)
						} else {
							err = nil // Successful copy + delete
						}
					} else {
						err = fmt.Errorf("failed to move file (rename failed: %s, copy failed: %w)", err, copyErr)
					}
				}
			}
		}

		if err != nil {
			infoLabel.Text = fmt.Sprintf("Error %s '%s' to '%s': %s", action, clipboardName, destName, err.Error())
			infoLabel.Color = colors.Red
		} else {
			infoLabel.Text = fmt.Sprintf("'%s' %s to '%s'.", clipboardName, strings.Replace(action, "ing", "ed", 1), destName)
			infoLabel.Color = colors.Green
			if operationState == opSourceSelectedForMove {
				// Clear clipboard after successful move
				clipboardPath = ""
				clipboardName = ""
				operationState = opNone
			}
		}
		listDirectoryContents()
		fmWindow.RemoveElement(activePrompt)
		activePrompt = nil
		return false // Don't exit app
	})

	cancelButton := gui.NewPromptButton("Cancel", colors.BoldRed, colors.BgRed+colors.White, func() bool {
		infoLabel.Text = "Paste cancelled."
		infoLabel.Color = colors.Yellow
		fmWindow.RemoveElement(activePrompt)
		activePrompt = nil
		return false // Don't exit app
	})

	// Create the prompt
	width := 50
	x := (fmWindow.Width - width) / 2
	y := fmWindow.Height / 3

	actionType := "copy"
	if operationState == opSourceSelectedForMove {
		actionType = "move"
	}

	promptMessage := fmt.Sprintf("Ready to %s '%s' to current directory.", actionType, clipboardName)
	activePrompt = gui.NewDialogPrompt(
		fmt.Sprintf("%s Item", actionType),
		promptMessage,
		x, y, width,
		colors.BgBlack, colors.Cyan, colors.BoldWhite, colors.White,
		[]*gui.PromptButton{pasteButton, cancelButton},
	)
	// Prompt implements ZIndexer with z-index 1000 automatically
	activePrompt.SetActive(true)
	fmWindow.AddElement(activePrompt)
	fmWindow.Render() // Force immediate render
}

func handleCopyItem() {
	selectedIndex := fileListContainer.SelectedIndex
	if selectedIndex < 0 || selectedIndex >= len(currentEntries) {
		infoLabel.Text = "No item selected to copy."
		infoLabel.Color = colors.Yellow
		return
	}
	selectedEntry := currentEntries[selectedIndex]
	clipboardName = selectedEntry.Name()
	clipboardPath = filepath.Join(currentPath, clipboardName)
	clipboardIsDir = selectedEntry.IsDir()
	operationState = opSourceSelectedForCopy
	infoLabel.Text = fmt.Sprintf("'%s' marked for copy. Navigate and use Paste [P].", clipboardName)
	infoLabel.Color = colors.Cyan
	listDirectoryContents() // Refresh to show "(copying)"
}

func handleMoveItem() {
	selectedIndex := fileListContainer.SelectedIndex
	if selectedIndex < 0 || selectedIndex >= len(currentEntries) {
		infoLabel.Text = "No item selected to move."
		infoLabel.Color = colors.Yellow
		return
	}
	selectedEntry := currentEntries[selectedIndex]
	clipboardName = selectedEntry.Name()
	clipboardPath = filepath.Join(currentPath, clipboardName)
	clipboardIsDir = selectedEntry.IsDir()
	operationState = opSourceSelectedForMove
	infoLabel.Text = fmt.Sprintf("'%s' marked for move. Navigate and use Paste [P].", clipboardName)
	infoLabel.Color = colors.Yellow
	listDirectoryContents() // Refresh to show "(moving)"
}

// CreateEmptyFile is assumed to exist or would be part of a files.go
func CreateEmptyFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return f.Close()
}

// setupMenuBar creates the application menu bar
func setupMenuBar(contentWidth int) *gui.MenuBar {
	// Reduce width by 2 to account for window borders
	mb := gui.NewMenuBar(1, 0, contentWidth-2, colors.White, colors.Cyan, colors.BgBlue)

	// File menu
	fileMenu := mb.AddSubMenu("File (Alt+F)", colors.White, colors.BgBlue+colors.White)
	fileMenu.AddItem(gui.NewMenuItem("New Folder (F1/F)", colors.White, colors.BgBlue+colors.White, func() bool {
		showCreateFolderPrompt()
		menuBar.Deactivate() // Manually deactivate menu instead of returning true
		return false         // Don't exit
	}))
	fileMenu.AddItem(gui.NewMenuItem("New File (F2/T)", colors.White, colors.BgBlue+colors.White, func() bool {
		showCreateFilePrompt()
		menuBar.Deactivate()
		return false
	}))
	fileMenu.AddItem(gui.NewMenuItem("Refresh (R)", colors.White, colors.BgBlue+colors.White, func() bool {
		listDirectoryContents()
		menuBar.Deactivate()
		return false
	}))
	fileMenu.AddItem(gui.NewMenuItem("Quit (Q)", colors.White, colors.BgRed+colors.White, func() bool {
		return true // Only Quit action should return true to exit
	}))

	// Edit menu
	editMenu := mb.AddSubMenu("Edit (Alt+E)", colors.White, colors.BgBlue+colors.White)
	editMenu.AddItem(gui.NewMenuItem("Copy (C)", colors.White, colors.BgBlue+colors.White, func() bool {
		handleCopyItem()
		menuBar.Deactivate()
		return false
	}))
	editMenu.AddItem(gui.NewMenuItem("Move (M)", colors.White, colors.BgBlue+colors.White, func() bool {
		handleMoveItem()
		menuBar.Deactivate()
		return false
	}))
	editMenu.AddItem(gui.NewMenuItem("Paste (P)", colors.White, colors.BgBlue+colors.White, func() bool {
		showPastePrompt()
		menuBar.Deactivate()
		return false
	}))
	editMenu.AddItem(gui.NewMenuItem("Delete (D/Del)", colors.White, colors.BgRed+colors.White, func() bool {
		showDeleteConfirmation()
		menuBar.Deactivate()
		return false
	}))

	// Navigation menu
	navMenu := mb.AddSubMenu("Nav (Alt+N)", colors.White, colors.BgBlue+colors.White)
	navMenu.AddItem(gui.NewMenuItem("Up Directory (U)", colors.White, colors.BgBlue+colors.White, func() bool {
		parentPath := filepath.Dir(currentPath)
		if parentPath != currentPath {
			currentPath = parentPath
			listDirectoryContents()
			fileListContainer.SelectedIndex = 0
			fileListContainer.GetScrollbar().SetValue(0)
		} else {
			infoLabel.Text = "Already at root."
			infoLabel.Color = colors.Yellow
		}
		menuBar.Deactivate()
		return false
	}))

	return mb
}

// FileManagerApp is the main function to run the file manager.
func FileManagerApp() {
	fmt.Print(gui.ClearScreenAndBuffer())
	termWidth := gui.GetTerminalWidth()
	termHeight := gui.GetTerminalHeight()

	winWidth := termWidth * 9 / 10
	if winWidth < 60 {
		winWidth = 60
	}
	winHeight := termHeight * 9 / 10
	if winHeight < 20 {
		winHeight = 20
	}
	winX := (termWidth - winWidth) / 2
	winY := (termHeight - winHeight) / 2

	fmWindow = gui.NewWindow(" 🗂️", " File Manager ", winX, winY, winWidth, winHeight,
		"double", colors.BoldYellow, colors.Yellow, colors.BgBlack, colors.White)

	contentAreaWidth := winWidth - 2

	// Create a custom key handler for keyboard shortcuts
	keyHandler := &FileManagerKeyHandler{}
	fmWindow.SetKeyStrokeHandler(keyHandler)

	// Setup menu bar first to get lowest z-index among overlays
	menuBar = setupMenuBar(contentAreaWidth)
	fmWindow.AddElement(menuBar) // MenuBar implements ZIndexer with z-index 100

	// Setup main UI elements (these will appear under overlays)
	currentY := 2 // Start after menu bar

	pathLabel = gui.NewLabel("Path: ", 1, currentY, colors.BoldWhite)
	fmWindow.AddElement(pathLabel)
	currentY += 2

	containerHeight := winHeight - currentY - 5
	if containerHeight < 5 {
		containerHeight = 5
	}

	fileListContainer = gui.NewContainer(1, currentY, contentAreaWidth-1, containerHeight, []string{})
	fileListContainer.Color = colors.BgBlack + colors.White
	fileListContainer.SelectionColor = colors.BgBlue + colors.BoldWhite
	fileListContainer.OnItemSelected = handleItemActivation
	fmWindow.AddElement(fileListContainer)
	currentY += containerHeight + 1

	infoLabel = gui.NewLabel("Welcome to File Manager! Press Alt+F, Alt+E or Alt+N to activate menus, Tab to navigate.", 1, currentY, colors.Gray)
	fmWindow.AddElement(infoLabel)
	currentY += 2 // Space after info label

	// Status bar at the bottom with key shortcuts and menu instructions
	statusBar := gui.NewLabel("Alt+F/E/N: Menus | F1: New Folder | F2: New File | C: Copy | M: Move | P: Paste | D: Delete | U: Up | Q: Quit", 1, winHeight-4, colors.Gray)
	fmWindow.AddElement(statusBar)

	// Initial load
	initialPath, err := os.Getwd()
	if err != nil {
		initialPath = "/"             // Fallback to root
		if os.PathSeparator == '\\' { // Windows
			initialPath = "C:\\" // Basic Windows fallback
			// Try to get user's home directory as a better fallback on Windows
			homeDir, homeErr := os.UserHomeDir()
			if homeErr == nil {
				initialPath = homeDir
			}
		}
	}
	currentPath = filepath.Clean(initialPath)
	listDirectoryContents() // Initial population

	// Start interaction loop
	fmWindow.WindowActions()

}

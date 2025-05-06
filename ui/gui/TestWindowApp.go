package gui

import (
	"fmt"
	"secshell/colors"
	"strconv"
	"time"
)

// --- Task Data Structure ---
type Task struct {
	Name     string
	Done     bool
	Priority string // "Low", "Medium", "High"
}

// --- Main Application Function ---
func TestWindowApp() {
	// --- Application State ---
	tasks := []Task{} // Initialize empty slice
	// Generate 25 sample tasks and prepare initial content for the container
	priorities := []string{"Low", "Medium", "High"}
	initialContent := []string{} // Store formatted strings for NewContainer
	for i := 0; i < 25; i++ {
		taskName := fmt.Sprintf("Generated Task %d", i+1)
		// Add some longer names occasionally
		if i%5 == 0 {
			taskName += " - with some extra details to test line wrapping and scrolling behavior"
		}
		isDone := (i%4 == 0)                      // Make roughly 1/4 tasks done initially
		priority := priorities[i%len(priorities)] // Cycle through priorities
		task := Task{
			Name:     taskName,
			Done:     isDone,
			Priority: priority,
		}
		tasks = append(tasks, task)

		// Format the line for initial container content
		status := "[ ]"
		if task.Done {
			status = "[X]"
		}
		line := fmt.Sprintf("%d: %s %s (%s)", i, status, task.Name, task.Priority)
		initialContent = append(initialContent, line)
	}

	var infoLabel *Label
	var scrollLabel *Label
	var taskListContainer *Container
	var nameInput *TextBox
	var doneCheckbox *CheckBox
	var priorityGroup *RadioGroup
	var indexInput *TextBox
	var completionProgress *ProgressBar

	// --- Helper Functions ---

	// Updates the container content and progress bar based on the tasks slice
	updateTaskListDisplay := func() {
		content := []string{}
		doneCount := 0
		if len(tasks) == 0 {
			content = append(content, colors.Gray+"<No tasks yet>"+colors.Reset)
		} else {
			for i, task := range tasks {
				status := "[ ]"
				if task.Done {
					status = "[X]"
					doneCount++
				}
				// Format: "Index: Status Name (Priority)"
				line := fmt.Sprintf("%d: %s %s (%s)", i, status, task.Name, task.Priority)
				content = append(content, line)
			}
		}
		// Only call SetContent if the container already exists
		if taskListContainer != nil {
			taskListContainer.SetContent(content) // Update container's internal content & scroll state
		}

		// Update progress bar based on scroll position
		// Check if completionProgress and taskListContainer exist before using
		if completionProgress != nil && taskListContainer != nil {
			scrollbar := taskListContainer.GetScrollbar()
			if scrollbar != nil {
				// Use scrollbar's state to set MaxValue and initial Value
				completionProgress.MaxValue = float64(scrollbar.MaxValue)
				completionProgress.SetValue(float64(scrollbar.Value)) // Set initial value

				// Set the OnScroll callback if it hasn't been set yet
				if scrollbar.OnScroll == nil {
					scrollbar.OnScroll = func(newValue int) {
						// This function will be called by scrollbar.SetValue
						if completionProgress != nil {
							completionProgress.SetValue(float64(newValue))
							// No need to call Render here, WindowActions handles it
						}
					}
				}
			} else {
				// No scrollbar, set progress to 0
				completionProgress.MaxValue = 0
				completionProgress.SetValue(0)
			}
		}
	}

	// Clears input fields
	clearInputs := func() {
		nameInput.Text = ""
		nameInput.cursorPos = 0
		nameInput.isPristine = true // Reset pristine state if desired, or leave as edited
		doneCheckbox.Checked = false
		priorityGroup.Select(0) // Default to "Low"
		indexInput.Text = ""
		indexInput.cursorPos = 0
		indexInput.isPristine = true
	}

	// Sets the input fields based on a task index
	loadTaskForEditing := func(index int) {
		if index >= 0 && index < len(tasks) {
			task := tasks[index]
			nameInput.Text = task.Name
			nameInput.cursorPos = len(task.Name)
			nameInput.isPristine = false
			doneCheckbox.Checked = task.Done
			// Select correct radio button
			priorityIndex := 0
			switch task.Priority {
			case "Medium":
				priorityIndex = 1
			case "High":
				priorityIndex = 2
			}
			priorityGroup.Select(priorityIndex)
			indexInput.Text = strconv.Itoa(index)
			indexInput.cursorPos = len(indexInput.Text)
			indexInput.isPristine = false
			infoLabel.Text = fmt.Sprintf("Loaded task %d for editing.", index)
			infoLabel.Color = colors.Cyan
		} else {
			infoLabel.Text = fmt.Sprintf("Error: Invalid index %d.", index)
			infoLabel.Color = colors.Red
		}
	}

	// --- UI Setup ---
	fmt.Print(ClearScreenAndBuffer())
	termWidth := GetTerminalWidth()
	termHeight := GetTerminalHeight()

	winWidth := termWidth * 3 / 4
	if winWidth < 80 {
		winWidth = 80
	} // Min width
	winHeight := termHeight * 3 / 4
	if winHeight < 25 {
		winHeight = 25
	} // Min height
	winX := (termWidth - winWidth) / 2
	winY := (termHeight - winHeight) / 2

	testWin := NewWindow("📝", "Task List CRUD", winX, winY, winWidth, winHeight,
		"double", colors.BoldCyan, colors.BoldYellow, colors.BgBlack, colors.White)

	// --- Elements ---
	contentAreaWidth := winWidth - 2
	currentY := 1

	// Info Label (Top)
	infoLabel = NewLabel("Tab/S-Tab: Cycle | Arrows: Scroll List | Enter: Activate/Select | q/Ctrl+C: Quit", 1, currentY, colors.Green)
	testWin.AddElement(infoLabel)
	currentY += 2 // Allow for wrapping

	//Scroll Label (Top)
	scrollLabel = NewLabel("Scroll: ↑↓ | Up Arrow = Scroll Up | Down Arrow = Scroll Down", 1, currentY, colors.Green)
	testWin.AddElement(scrollLabel)
	currentY += 2 // Allow for wrapping

	// Input Area
	inputStartX := 1
	labelWidth := 25 // Width for labels like "Task Name:"
	inputFieldX := inputStartX + labelWidth + 1
	inputFieldWidth := contentAreaWidth - inputFieldX - 1 // Width for text boxes

	// Task Name Input
	nameLabel := NewLabel("Task Name:", inputStartX, currentY, colors.White)
	testWin.AddElement(nameLabel)
	nameInput = NewTextBox("", inputFieldX, currentY, inputFieldWidth, colors.BgWhite+colors.Black, colors.BgCyan+colors.BoldBlack)
	testWin.AddElement(nameInput)
	currentY++

	// Done Checkbox
	doneCheckbox = NewCheckBox("Mark as Done", inputFieldX, currentY, false, colors.White, colors.BgPurple+colors.BoldWhite)
	testWin.AddElement(doneCheckbox)
	currentY++

	// Priority Radio Buttons
	priorityLabel := NewLabel("Priority:", inputStartX, currentY, colors.White)
	testWin.AddElement(priorityLabel)
	priorityGroup = NewRadioGroup()
	prioBtnY := currentY
	prioBtnX := inputFieldX
	prioBtnSpacing := 12
	prioLow := NewRadioButton("Low", "Low", prioBtnX, prioBtnY, colors.White, colors.BgBlue+colors.BoldWhite, priorityGroup)
	testWin.AddElement(prioLow)
	prioMedium := NewRadioButton("Medium", "Medium", prioBtnX+prioBtnSpacing, prioBtnY, colors.White, colors.BgBlue+colors.BoldWhite, priorityGroup)
	testWin.AddElement(prioMedium)
	prioHigh := NewRadioButton("High", "High", prioBtnX+prioBtnSpacing*2, prioBtnY, colors.White, colors.BgBlue+colors.BoldWhite, priorityGroup)
	testWin.AddElement(prioHigh)
	priorityGroup.Select(0) // Default to Low
	currentY++

	// Spacer

	// Task List Container
	containerX := 1
	containerY := currentY
	containerHeight := winHeight - currentY - 7 // Adjusted height calculation
	if containerHeight < 3 {
		containerHeight = 3
	}
	// Use the full contentAreaWidth for the container.
	// The container's internal rendering logic handles the scrollbar placement.
	containerWidth := contentAreaWidth // Use full width

	// Create the container WITH the initial content generated earlier.
	// This ensures the scrollbar exists when AddElement is called.
	taskListContainer = NewContainer(containerX, containerY, containerWidth, containerHeight, initialContent)
	testWin.AddElement(taskListContainer) // Now AddElement should find the scrollbar
	currentY += containerHeight           // Move Y past the container

	// Spacer
	testWin.AddElement(NewSpacer(1, currentY, 1))
	currentY++

	// Progress Bar
	progressY := currentY
	progressWidth := contentAreaWidth - 2 // Slightly inset
	completionProgress = NewProgressBar(1, progressY, progressWidth, 0, 100, colors.BgGreen+colors.Green, colors.Gray, true)
	testWin.AddElement(completionProgress)
	currentY++ // Move past progress bar row

	// Spacer
	testWin.AddElement(NewSpacer(1, currentY, 1))
	currentY++

	// Index Input (Moved to Bottom)
	indexInputY := currentY
	indexLabelWidth := 25
	indexInputX := inputStartX + indexLabelWidth + 1
	indexLabel := NewLabel("Index (for Update/Delete):", inputStartX, indexInputY, colors.White)
	testWin.AddElement(indexLabel)
	indexInputWidth := 6
	indexInput = NewTextBox("", indexInputX, indexInputY, indexInputWidth, colors.BgWhite+colors.Black, colors.BgCyan+colors.BoldBlack)
	testWin.AddElement(indexInput)
	loadButton := NewButton("Load", indexInputX+indexInputWidth+1, indexInputY, 8, colors.BoldCyan, colors.BgCyan+colors.BoldWhite, func() bool {
		idxStr := indexInput.Text
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			infoLabel.Text = "Error: Invalid index format."
			infoLabel.Color = colors.Red
		} else {
			loadTaskForEditing(idx)
		}
		return false // Don't quit
	})
	testWin.AddElement(loadButton)
	currentY++

	// Spacer before buttons
	testWin.AddElement(NewSpacer(1, currentY, 1))

	// Buttons (Bottom)
	buttonWidth := 10
	buttonSpacing := 2
	totalButtonsWidth := (buttonWidth * 4) + (buttonSpacing * 3)
	buttonStartX := (contentAreaWidth - totalButtonsWidth) / 2
	if buttonStartX < 1 {
		buttonStartX = 1
	}
	actionButtonY := winHeight - 4 // Position near bottom

	// Add Button
	addButton := NewButton("Add", buttonStartX, actionButtonY, buttonWidth, colors.BoldGreen, colors.BgGreen+colors.BoldWhite, func() bool {
		taskName := nameInput.Text
		if nameInput.isPristine || taskName == "" {
			infoLabel.Text = "Error: Task name cannot be empty."
			infoLabel.Color = colors.Red
			return false
		}
		newTask := Task{
			Name:     taskName,
			Done:     doneCheckbox.Checked,
			Priority: priorityGroup.SelectedValue,
		}
		tasks = append(tasks, newTask)
		updateTaskListDisplay()
		clearInputs()
		infoLabel.Text = "Task added successfully."
		infoLabel.Color = colors.Green
		return false // Don't quit
	})
	testWin.AddElement(addButton)

	// Update Button
	updateButtonX := buttonStartX + buttonWidth + buttonSpacing
	updateButton := NewButton("Update", updateButtonX, actionButtonY, buttonWidth, colors.BoldBlue, colors.BgBlue+colors.BoldWhite, func() bool {
		idxStr := indexInput.Text
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 0 || idx >= len(tasks) {
			infoLabel.Text = "Error: Invalid index for Update."
			infoLabel.Color = colors.Red
			return false
		}
		taskName := nameInput.Text
		if nameInput.isPristine || taskName == "" {
			infoLabel.Text = "Error: Task name cannot be empty for Update."
			infoLabel.Color = colors.Red
			return false
		}
		tasks[idx].Name = taskName
		tasks[idx].Done = doneCheckbox.Checked
		tasks[idx].Priority = priorityGroup.SelectedValue
		updateTaskListDisplay()
		clearInputs()
		infoLabel.Text = fmt.Sprintf("Task %d updated successfully.", idx)
		infoLabel.Color = colors.Blue
		return false // Don't quit
	})
	testWin.AddElement(updateButton)

	// Delete Button
	deleteButtonX := updateButtonX + buttonWidth + buttonSpacing
	deleteButton := NewButton("Delete", deleteButtonX, actionButtonY, buttonWidth, colors.BoldRed, colors.BgRed+colors.BoldWhite, func() bool {
		idxStr := indexInput.Text
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 0 || idx >= len(tasks) {
			infoLabel.Text = "Error: Invalid index for Delete."
			infoLabel.Color = colors.Red
			return false
		}
		// Remove task from slice
		tasks = append(tasks[:idx], tasks[idx+1:]...)
		updateTaskListDisplay()
		clearInputs()
		infoLabel.Text = fmt.Sprintf("Task %d deleted successfully.", idx)
		infoLabel.Color = colors.Red
		return false // Don't quit
	})
	testWin.AddElement(deleteButton)

	// Quit Button
	quitButtonX := deleteButtonX + buttonWidth + buttonSpacing
	quitButton := NewButton("Quit", quitButtonX, actionButtonY, buttonWidth, colors.BoldWhite, colors.BgGray+colors.BoldWhite, func() bool {
		infoLabel.Text = "Quitting..."
		infoLabel.Color = colors.BoldRed
		testWin.Render() // Render final message
		time.Sleep(300 * time.Millisecond)
		return true // Quit
	})
	testWin.AddElement(quitButton)

	// --- Initial Display & Interaction ---
	// updateTaskListDisplay() // No longer needed here, container created with content
	// Need to update progress bar initially though
	updateTaskListDisplay() // Call once to set initial progress bar state based on initial tasks
	testWin.WindowActions() // Start the interaction loop

	// --- After Interaction ---
	fmt.Println("Application finished.")
	fmt.Printf("Final task list contained %d tasks.\n", len(tasks))
	// You could print the final tasks here if desired
}

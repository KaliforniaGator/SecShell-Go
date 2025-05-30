package jobs

import (
	"fmt"
	"secshell/colors"
	"secshell/ui/gui"
	"strconv"
	"strings"
	"time"
)

// updateJobListInContainer populates the container with job information.
func updateJobListInContainer(jobsMap map[int]*Job, container *gui.Container, infoLabel *gui.Label) {
	content := []string{}
	if len(jobsMap) == 0 {
		content = append(content, colors.Gray+"<No jobs running or defined>"+colors.Reset)
	} else {
		// Skip adding header line - it's now a separate UI element so it doesn't get duplicated

		for _, job := range jobsMap {
			job.Lock()
			updateJobStats(job) // Ensure stats are fresh
			status := job.Status
			command := job.Command
			// Truncate command if too long for display
			maxCmdLen := 30
			if len(command) > maxCmdLen {
				command = command[:maxCmdLen-3] + "..."
			}

			// Format CPU as percentage for text display
			cpuPercent := job.CPU
			cpuText := fmt.Sprintf("%.1f%%", cpuPercent)

			// Create CPU progress bar representation
			// Calculate color gradient based on CPU usage
			var cpuBar string
			maxBarWidth := 15 // Width of progress bar in characters

			if cpuPercent <= 0.1 {
				// Show minimal activity for near-zero CPU
				cpuBar = colors.Gray + "▏" + strings.Repeat("░", maxBarWidth-1) + colors.Reset
			} else {
				// Calculate filled portion of bar
				filledWidth := int((cpuPercent / 100.0) * float64(maxBarWidth))
				if filledWidth > maxBarWidth {
					filledWidth = maxBarWidth // Cap at maximum
				}
				if filledWidth < 1 {
					filledWidth = 1 // At least show minimal activity
				}

				// Color gradient from green to yellow to red based on usage
				var barColor string
				if cpuPercent < 30 {
					barColor = colors.Green
				} else if cpuPercent < 70 {
					barColor = colors.Yellow
				} else {
					barColor = colors.Red
				}

				// Create the bar
				cpuBar = barColor + strings.Repeat("█", filledWidth) +
					colors.Gray + strings.Repeat("░", maxBarWidth-filledWidth) +
					colors.Reset + " " + cpuText
			}

			mem := fmt.Sprintf("%.1fMB", job.Memory)
			threads := strconv.Itoa(job.ThreadCount)
			job.Unlock()

			statusColor := colors.Red
			if status == "running" {
				statusColor = colors.Green
			} else if status == "stopped" {
				statusColor = colors.Yellow
			}

			// Format: PID | Command | Status | CPU (progress bar) | Mem | Threads
			entry := fmt.Sprintf("%s%-5d%s | %s%-30s%s | %s%-10s%s | %s | %s%-9s%s | %s%s%s",
				colors.Cyan, job.ID, colors.Reset, // PID
				colors.White, command, colors.Reset, // Command
				statusColor, status, colors.Reset, // Status
				cpuBar,                         // CPU with progress bar
				colors.Blue, mem, colors.Reset, // Memory
				colors.Gray, threads, colors.Reset) // Threads
			content = append(content, entry)
		}
	}
	container.SetContent(content)
	if infoLabel != nil {
		infoLabel.Text = "Job list updated. Select a job and choose an action."
		infoLabel.Color = colors.Gray
	}
}

// stripAnsi removes ANSI escape codes from a string.
func stripAnsi(str string) string {
	var result strings.Builder
	inEscapeSeq := false
	for _, r := range str {
		if r == '\x1b' {
			inEscapeSeq = true
		}
		if !inEscapeSeq {
			result.WriteRune(r)
		}
		if inEscapeSeq && r == 'm' {
			inEscapeSeq = false
		}
	}
	return result.String()
}

// getSelectedPIDFromContainer extracts the PID from the selected item in the container.
func getSelectedPIDFromContainer(container *gui.Container) (int, error) {
	selectedIndex := container.GetSelectedIndex()
	if selectedIndex < 0 || selectedIndex >= len(container.Content) {
		// If there's no selected item, try using the highlighted item instead
		selectedIndex = container.GetHighlightedIndex()
		if selectedIndex < 0 || selectedIndex >= len(container.Content) {
			return -1, fmt.Errorf("no job selected or selection is invalid")
		}
	}
	selectedLine := container.Content[selectedIndex]

	// The PID is in the first "column", before the first " | "
	parts := strings.SplitN(selectedLine, "|", 2)
	if len(parts) == 0 {
		return -1, fmt.Errorf("selected line format error, no '|' separator")
	}

	pidPartWithColor := strings.TrimSpace(parts[0])

	// Attempt to extract PID by stripping known color patterns
	var potentialPidStr string
	lastMIndex := strings.LastIndex(pidPartWithColor, "m")
	if lastMIndex != -1 && lastMIndex+1 < len(pidPartWithColor) {
		potentialPidStr = pidPartWithColor[lastMIndex+1:]
		resetIndex := strings.Index(potentialPidStr, colors.Reset)
		if resetIndex != -1 {
			potentialPidStr = potentialPidStr[:resetIndex]
		}
	} else {
		// If 'm' or Reset pattern isn't as expected, fall back to stripping all ANSI
		potentialPidStr = stripAnsi(pidPartWithColor)
	}

	finalPidStr := strings.TrimSpace(potentialPidStr)
	pid, err := strconv.Atoi(finalPidStr)
	if err != nil {
		// Broader fallback: strip all ANSI from the original part and try again
		cleanedPidStrFallback := stripAnsi(pidPartWithColor)
		pidFallback, errFallback := strconv.Atoi(strings.TrimSpace(cleanedPidStrFallback))
		if errFallback != nil {
			return -1, fmt.Errorf("failed to parse PID from '%s' (original: '%s'): error '%v', fallback error '%v'", finalPidStr, pidPartWithColor, err, errFallback)
		}
		return pidFallback, nil
	}
	return pid, nil
}

// InteractiveJobManager provides a GUI for managing jobs.
func InteractiveJobManager(jobsMap map[int]*Job) {
	fmt.Print(gui.ClearScreenAndBuffer())
	termWidth := gui.GetTerminalWidth()
	termHeight := gui.GetTerminalHeight()

	winWidth := termWidth * 9 / 10
	if winWidth < 100 {
		winWidth = 100 // Min width for better layout
	}
	winHeight := termHeight * 8 / 10
	if winHeight < 20 {
		winHeight = 20
	}
	winX := (termWidth - winWidth) / 2
	winY := (termHeight - winHeight) / 2

	jobWin := gui.NewWindow(" 💼", " Job Manager ", winX, winY, winWidth, winHeight,
		"rounded", colors.BoldCyan, colors.Cyan, colors.BgBlack, colors.White)

	contentAreaWidth := winWidth - 2
	currentY := 1

	// Info Label
	infoLabel := gui.NewLabel("Select a job using Arrow Keys, then press Enter or click a button.", 1, currentY, colors.Gray)
	jobWin.AddElement(infoLabel)
	currentY += 2

	// Header for the job list
	headerText := fmt.Sprintf("%s%-5s%s | %s%-30s%s | %s%-10s%s | %s%-20s%s | %s%-9s%s | %s%s%s",
		colors.BoldYellow, "PID", colors.Reset,
		colors.BoldYellow, "COMMAND", colors.Reset,
		colors.BoldYellow, "STATUS", colors.Reset,
		colors.BoldYellow, "CPU", colors.Reset, // Wider column for progress bar
		colors.BoldYellow, "MEMORY", colors.Reset,
		colors.BoldYellow, "THREADS", colors.Reset)
	headerLabel := gui.NewLabel(headerText, 1, currentY, colors.BoldYellow)
	jobWin.AddElement(headerLabel)
	currentY++

	// Job List Container
	containerHeight := winHeight - currentY - 5 // Space for buttons and bottom margin
	if containerHeight < 5 {
		containerHeight = 5
	}
	jobListContainer := gui.NewContainer(1, currentY, contentAreaWidth-1, containerHeight, []string{})
	jobListContainer.Color = colors.BgBlack // Match window background
	jobListContainer.SelectionColor = colors.BgBlue + colors.BoldWhite
	jobListContainer.OnItemSelected = func(selectedIndex int) {
		pid, err := getSelectedPIDFromContainer(jobListContainer)
		if err == nil {
			infoLabel.Text = fmt.Sprintf("Selected Job PID: %d. Choose an action.", pid)
			infoLabel.Color = colors.Cyan
		} else {
			infoLabel.Text = fmt.Sprintf("Select a job. Error: %v", err)
			infoLabel.Color = colors.Red
		}
	}
	jobWin.AddElement(jobListContainer)
	currentY += containerHeight + 1

	// Buttons
	buttonWidth := 18
	buttonSpacing := 2
	totalButtonsWidth := (buttonWidth * 4) + (buttonSpacing * 3)
	buttonStartX := (contentAreaWidth - totalButtonsWidth) / 2
	if buttonStartX < 1 {
		buttonStartX = 1
	}
	actionButtonY := currentY

	// Stop Button
	stopButton := gui.NewButton("Stop Job", buttonStartX, actionButtonY, buttonWidth, colors.BoldRed, colors.BgRed+colors.BoldWhite, func() bool {
		pid, err := getSelectedPIDFromContainer(jobListContainer)
		if err != nil {
			infoLabel.Text = fmt.Sprintf("Stop Error: %v", err)
			infoLabel.Color = colors.Red
			return false
		}
		StopJobClean(jobsMap, pid) // This function already logs and shows gui.AlertBox
		updateJobListInContainer(jobsMap, jobListContainer, infoLabel)
		infoLabel.Text = fmt.Sprintf("Attempted to stop job %d.", pid) // StopJob provides its own feedback
		return false
	})
	jobWin.AddElement(stopButton)

	// Start Button
	startButtonX := buttonStartX + buttonWidth + buttonSpacing
	startButton := gui.NewButton("Start/Resume Job", startButtonX, actionButtonY, buttonWidth, colors.BoldGreen, colors.BgGreen+colors.BoldWhite, func() bool {
		pid, err := getSelectedPIDFromContainer(jobListContainer)
		if err != nil {
			infoLabel.Text = fmt.Sprintf("Start Error: %v", err)
			infoLabel.Color = colors.Red
			return false
		}
		StartJobClean(jobsMap, pid) // This function already logs and shows gui.AlertBox
		updateJobListInContainer(jobsMap, jobListContainer, infoLabel)
		infoLabel.Text = fmt.Sprintf("Attempted to start job %d.", pid)
		return false
	})
	jobWin.AddElement(startButton)

	// Clear Finished Button
	clearButtonX := startButtonX + buttonWidth + buttonSpacing
	clearButton := gui.NewButton("Clear Finished", clearButtonX, actionButtonY, buttonWidth, colors.BoldYellow, colors.BgYellow+colors.BoldWhite, func() bool {
		ClearFinishedJobs(jobsMap) // This function shows its own title box and alerts
		updateJobListInContainer(jobsMap, jobListContainer, infoLabel)
		infoLabel.Text = "Cleared finished jobs."
		infoLabel.Color = colors.Yellow
		return false
	})
	jobWin.AddElement(clearButton)

	// Quit Button
	quitButtonX := clearButtonX + buttonWidth + buttonSpacing
	quitButton := gui.NewButton("Quit Manager", quitButtonX, actionButtonY, buttonWidth, colors.BoldRed, colors.BgRed+colors.BoldWhite, func() bool {
		infoLabel.Text = "Exiting Job Manager..."
		infoLabel.Color = colors.BoldWhite
		jobWin.Render() // Render final message
		time.Sleep(200 * time.Millisecond)
		return true // Quit
	})
	jobWin.AddElement(quitButton)

	// Create a channel to coordinate closing the refresh goroutine
	refreshDone := make(chan struct{})

	// Setup a ticker for auto-refresh with shorter interval for more responsive updates
	refreshTicker := time.NewTicker(500 * time.Millisecond)

	// Start a goroutine to periodically update job stats
	go func() {
		defer refreshTicker.Stop()

		for {
			select {
			case <-refreshTicker.C:
				// Update the container content without re-rendering header
				updateJobListInContainer(jobsMap, jobListContainer, infoLabel)

				// Force re-render of the entire window
				jobWin.Render()

			case <-refreshDone:
				return // Exit goroutine when done signal received
			}
		}
	}()

	// Initial population before entering event loop
	updateJobListInContainer(jobsMap, jobListContainer, infoLabel)

	// Start the window event loop (this will block until window is closed)
	jobWin.WindowActions()

	// Once WindowActions returns, the window is closed, so we can clean up
	close(refreshDone) // Signal the refresh goroutine to exit
}

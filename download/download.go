package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"secshell/drawbox"
	"secshell/logging"
	"secshell/sanitize"
	"secshell/ui"
	"strings"
	"sync"
	"time"
)

func DownloadFile(url, fileName string, linePos int) error {
	// Sanitize URL and filename
	sanitizedURL, err := sanitize.SanitizeURL(url)
	if err != nil {
		logging.LogError(err)
		fmt.Print("\x1b[?25h") // Show cursor on error
		return fmt.Errorf("invalid URL: %v", err)
	}

	sanitizedFileName, err := sanitize.SanitizeFileName(fileName)
	if err != nil {
		logging.LogError(err)
		fmt.Print("\x1b[?25h") // Show cursor on error
		return fmt.Errorf("invalid filename: %v", err)
	}

	startTime := time.Now()

	// Clear line and print initial filename
	fmt.Printf("\x1b[%d;0H\x1b[2K", linePos+1) // Move to line and clear it
	fmt.Printf("Starting download of %s", sanitizedFileName)

	// Create the file
	out, err := os.Create(sanitizedFileName)
	if err != nil {
		logging.LogError(err)
		fmt.Print("\x1b[?25h") // Show cursor on error
		return fmt.Errorf("error creating file: %v", err)
	}
	defer out.Close()

	// Start download
	resp, err := http.Get(sanitizedURL)
	if err != nil {
		logging.LogError(err)
		fmt.Print("\x1b[?25h") // Show cursor on error
		return fmt.Errorf("error downloading: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Print("\x1b[?25h") // Show cursor on error
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	counter := &WriteCounter{
		Total:    resp.ContentLength,
		fileName: sanitizedFileName,
		progress: 0,
		linePos:  linePos,
	}

	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		logging.LogError(err)
		fmt.Print("\x1b[?25h") // Show cursor on error
		return fmt.Errorf("error copying data: %v", err)
	}

	// Clear line before printing completion message
	fmt.Printf("\x1b[%d;0H\x1b[2K", linePos+1)
	fmt.Printf("%s completed in %.2f seconds\n", sanitizedFileName, time.Since(startTime).Seconds())
	return nil
}

func DownloadFiles(args []string) {
	if len(args) < 2 {
		drawbox.PrintError("Usage: download [-o output1,output2,...] <url [url2 ...]>")
		return
	}

	var outputFiles []string
	urls := []string{}

	// Parse arguments
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-o":
			if i+1 < len(args) {
				outputFiles = strings.Split(args[i+1], ",")
				i++
			}
		default:
			urls = append(urls, args[i])
		}
	}

	if len(urls) == 0 {
		drawbox.PrintError("No URLs provided")
		return
	}

	if len(outputFiles) > 0 && len(outputFiles) != len(urls) {
		drawbox.PrintError("Number of output files must match number of URLs")
		return
	}

	// Prepare downloads
	type downloadStatus struct {
		url      string
		fileName string
	}

	downloads := make([]downloadStatus, len(urls))
	for i, url := range urls {
		fileName := ""
		if len(outputFiles) > 0 {
			fileName = outputFiles[i]
		} else {
			fileName = filepath.Base(url)
			if fileName == "" || fileName == "." {
				fileName = fmt.Sprintf("downloaded_file_%d", i+1)
			}
		}
		downloads[i] = downloadStatus{
			url:      url,
			fileName: fileName,
		}
	}

	// Clear screen, position cursor at top, and hide cursor
	ui.ClearScreenAndBuffer()
	fmt.Print("\x1b[H")    // Move cursor to home position
	fmt.Print("\x1b[?25l") // Hide cursor

	// Start downloads
	var wg sync.WaitGroup
	linePos := 0
	for i := range downloads {
		wg.Add(1)
		currentLine := linePos
		linePos++
		go func(idx, line int) {
			defer wg.Done()
			err := DownloadFile(downloads[idx].url, downloads[idx].fileName, line)
			if err != nil {
				logging.LogError(err)
				// Move to correct line and print error
				fmt.Printf("\x1b[%d;0H\nError downloading %s: %v\n", line+1, downloads[idx].fileName, err)
			}
		}(i, currentLine)
	}

	// Move cursor below all download bars
	fmt.Printf("\x1b[%d;0H", linePos+1)
	wg.Wait()
	fmt.Print("\x1b[?25h") // Show cursor
	fmt.Println()          // Add newline before alert
	drawbox.PrintAlert("All downloads completed")
}

type WriteCounter struct {
	Total      int64
	Downloaded int64
	progress   int
	fileName   string
	linePos    int
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Downloaded += int64(n)

	if wc.Total > 0 {
		percentage := float64(wc.Downloaded) / float64(wc.Total) * 100
		newProgress := int(percentage)

		if newProgress != wc.progress {
			wc.progress = newProgress

			// Clear the entire line and reset cursor to start of line
			fmt.Printf("\x1b[%d;0H\x1b[2K", wc.linePos+1)
			fmt.Printf("%s: ", wc.fileName)

			// Try to use drawbox command for progress
			cmd := exec.Command("drawbox", "progress",
				fmt.Sprintf("%d", newProgress),
				"100", "50", "block_full", "block_light", "cyan")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				logging.LogError(err)
				// Fallback to built-in progress bar if drawbox fails
				width := 50
				completed := int(float64(width) * float64(wc.Downloaded) / float64(wc.Total))
				fmt.Print("[")
				for i := 0; i < width; i++ {
					if i < completed {
						fmt.Print("=")
					} else {
						fmt.Print(" ")
					}
				}
				fmt.Printf("] %3d%%", newProgress)
			}
		}
	}
	return n, nil
}

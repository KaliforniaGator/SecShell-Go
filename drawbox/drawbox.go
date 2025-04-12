package drawbox

import (
	"fmt"
	"os"
	"os/exec"
	"secshell/colors"
)

// runDrawbox runs the drawbox command to display a message box
func RunDrawbox(title, color string) {
	fmt.Print("\n") // Add newline before box

	// Use exec.LookPath to find the drawbox executable in the PATH
	drawboxPath, err := exec.LookPath("drawbox")
	if err != nil {
		// If drawbox is not found, fallback to the custom box drawing
		fmt.Fprintf(os.Stdout, "%s╔══%s %s %s══╗%s\n",
			colors.BoldWhite, colors.Reset, title, colors.BoldWhite, colors.Reset)
		return
	}

	// Execute the drawbox command
	cmd := exec.Command(drawboxPath, title, color)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Fallback if drawbox fails
		fmt.Fprintf(os.Stdout, "%s╔══%s %s %s══╗%s\n",
			colors.BoldWhite, colors.Reset, title, colors.BoldWhite, colors.Reset)
	}
}

// RunDrawboxCommand runs the drawbox command with the specified parameters
// and returns the output as a string.
// It also handles the case where drawbox is not found by returning an error message.
func RunDrawboxCommand(command string, message, bg_color string, color string) string {

	// Use exec.LookPath to find the drawbox executable in the PATH
	drawboxPath, err := exec.LookPath("drawbox")
	if err != nil {
		return fmt.Sprintf("%s PLEASE INSTALL drawbox %s\n", colors.BoldRed, colors.Reset)
	}

	cmd := exec.Command(drawboxPath, command, message, bg_color, color)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("%s PLEASE INSTALL drawbox %s\n", colors.BoldRed, colors.Reset)
	}

	return string(output)
}

// Print icon onto screen using drawbox
func PrintIcon(icon string) string {
	result := RunDrawboxCommand("unicode", icon, "", "")
	return result
}

// printAlert prints an alert message
func PrintAlert(message string) {
	RunDrawbox("ALERT: "+message, "bold_yellow")
}

// printError prints an error message
func PrintError(message string) {
	RunDrawbox("ERROR: "+message, "bold_red")
}

func DrawTable(titles string, data []string) {
	cmdArgs := []string{"table", titles}
	cmdArgs = append(cmdArgs, data...)

	cmdTable := exec.Command("drawbox", cmdArgs...)
	cmdTable.Stdout = os.Stdout
	cmdTable.Stderr = os.Stderr
	if err := cmdTable.Run(); err != nil {
		fmt.Println("Error executing drawbox table command:", err)
	}
}

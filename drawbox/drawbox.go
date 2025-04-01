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

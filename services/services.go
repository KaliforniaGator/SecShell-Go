package services

import (
	"fmt"
	"os"
	"os/exec"
	"secshell/drawbox"
	"strconv"
	"strings"
)

func runCommand(command string) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		drawbox.PrintError("Failed to execute service command: " + err.Error())
	} else {
		drawbox.PrintAlert("Service command executed successfully.")
	}
}

func RunServicesCommand(action string, serviceName string) {

	var command string

	switch action {
	case "list":
		GetServices()
	case "status":
		drawbox.RunDrawbox("Service Status", "bold_white")
		command = "systemctl status " + serviceName
	case "start":
		drawbox.RunDrawbox("Starting Service", "bold_white")
		command = "sudo systemctl start " + serviceName
	case "stop":
		drawbox.RunDrawbox("Stopping Service", "bold_white")
		command = "sudo systemctl stop " + serviceName
	case "restart":
		drawbox.RunDrawbox("Restarting Service", "bold_white")
		command = "sudo systemctl restart " + serviceName
	case "--help", "-h", "help":
		ShowHelp()
	default:
		drawbox.PrintError("Invalid action: " + action + " Use status, start, stop, restart, list, or help.")
	}

	runCommand(command)

}

// GetServices retrieves and displays system services
func GetServices() {
	// Execute the drawbox command for the header
	drawbox.RunDrawbox("Services", "bold_white")

	// Get all services
	cmd := exec.Command("systemctl", "list-units", "--type=service", "--all", "--no-pager", "--no-legend")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error executing systemctl command:", err)
		return
	}

	var services []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Split the line by whitespace
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		// Get service name (first field)
		serviceName := fields[0]

		// Skip services without a proper name
		if serviceName == "" || serviceName == "●" {
			continue
		}

		// Skip templated services with long names
		if strings.Contains(serviceName, "@") && len(serviceName) > 50 {
			continue
		}

		// Get display name (service name without .service)
		displayName := strings.TrimSuffix(serviceName, ".service")
		if displayName == "" || displayName == "●" {
			continue // Skip if display name is invalid
		}

		if len(displayName) > 0 {
			displayName = strings.ToUpper(displayName[:1]) + displayName[1:]
		}

		// Determine if service is active
		isActive := false
		for _, field := range fields {
			if field == "active" || field == "running" {
				isActive = true
				break
			}
		}

		// Prepare status text and indicator
		statusText := "\033[31mInactive"
		//statusIndicator := "\033[31m●\033[0m " // Red dot
		if isActive {
			statusText = "\033[32mActive"
			//statusIndicator = "\033[32m●\033[0m " // Green dot  // Using = instead of :=
		}

		//statusDisplay := statusIndicator + " " + statusText
		statusDisplay := statusText

		// Add to services list
		id := len(services) + 1
		entry := strconv.Itoa(id) + "," + displayName + "," + serviceName + "," + statusDisplay
		services = append(services, entry)
	}

	// If no services were found, add a message
	if len(services) == 0 {
		services = append(services, "1,No services found,-,\033[31m●\033[0m Inactive")
	}

	cmdArgs := []string{"table", "ID,NAME,SERVICE,STATUS"}
	cmdArgs = append(cmdArgs, services...)
	cmdArgs = append(cmdArgs, "hollow")

	cmdTable := exec.Command("drawbox", cmdArgs...)
	cmdTable.Stdout = os.Stdout
	cmdTable.Stderr = os.Stderr
	if err := cmdTable.Run(); err != nil {
		fmt.Println("Error executing drawbox table command:", err)
	}
}

func ShowHelp() {
	drawbox.RunDrawbox("Services Help", "bold_white")
	fmt.Println("Usage: services [command] [service_name]")
	fmt.Println("Actions:")
	fmt.Println("  status [service_name]   Show the status of a service")
	fmt.Println("  start [service_name]    Start a service")
	fmt.Println("  stop [service_name]     Stop a service")
	fmt.Println("  restart [service_name]  Restart a service")
	fmt.Println("  --help, -h, help        Show this help message")
}

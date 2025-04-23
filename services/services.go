package services

import (
	"fmt"
	"os"
	"os/exec"
	"secshell/logging"
	"secshell/ui/gui"
	"strconv"
	"strings"
)

func runCommand(command string) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logging.LogError(err)
		gui.ErrorBox("Failed to execute service command: " + err.Error())
	} else {
		logging.LogAlert("Service command executed successfully.")
		gui.AlertBox("Service command executed successfully.")
	}
}

func RunServicesCommand(action string, serviceName string) {

	var command string

	switch action {
	case "list":
		GetServices()
	case "status":
		gui.TitleBox("Service Status")
		command = "systemctl status " + serviceName
	case "start":
		gui.TitleBox("Starting Service")
		command = "sudo systemctl start " + serviceName
	case "stop":
		gui.TitleBox("Stopping Service")
		command = "sudo systemctl stop " + serviceName
	case "restart":
		gui.TitleBox("Restarting Service")
		command = "sudo systemctl restart " + serviceName
	case "--help", "-h", "help":
		ShowHelp()
	default:
		gui.ErrorBox("Invalid action: " + action + " Use status, start, stop, restart, list, or help.")
	}

	logging.LogCommand(command, 0)
	runCommand(command)

}

// GetServices retrieves and displays system services
func GetServices() {
	// Execute the drawbox command for the header
	gui.TitleBox("Services")

	// Get all services
	cmd := exec.Command("systemctl", "list-units", "--type=service", "--all", "--no-pager", "--no-legend")
	output, err := cmd.Output()
	if err != nil {
		logging.LogError(err)
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
		logging.LogError(err)
		fmt.Println("Error executing drawbox table command:", err)
	}
}

func ShowHelp() {
	gui.TitleBox("Services Help")
	fmt.Println("Usage: services [command] [service_name]")
	fmt.Println("Actions:")
	fmt.Println("  status [service_name]   Show the status of a service")
	fmt.Println("  start [service_name]    Start a service")
	fmt.Println("  stop [service_name]     Stop a service")
	fmt.Println("  restart [service_name]  Restart a service")
	fmt.Println("  --help, -h, help        Show this help message")
}

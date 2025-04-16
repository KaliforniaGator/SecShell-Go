package env

import (
	"fmt"
	"os"
	"secshell/drawbox"
	"secshell/logging"
	"secshell/sanitize"
	"strings"
)

// sanitizeInput uses the sanitize package to clean input
func sanitizeInput(input string, allowSpecialChars ...bool) string {
	allow := true
	if len(allowSpecialChars) > 0 {
		allow = allowSpecialChars[0]
	}
	return sanitize.Input(input, allow)
}

// exportVariable sets an environment variable
func ExportVariable(args []string) {
	if len(args) < 2 {
		logging.LogAlert("Usage: export VAR=value")
		drawbox.PrintError("Usage: export VAR=value")
		return
	}

	varValue := sanitizeInput(args[1], false)
	equalsPos := strings.Index(varValue, "=")
	if equalsPos == -1 {
		logging.LogAlert("Invalid export syntax. Use VAR=value")
		drawbox.PrintError("Invalid export syntax. Use VAR=value")
		return
	}

	varName := varValue[:equalsPos]
	value := varValue[equalsPos+1:]

	if err := os.Setenv(varName, value); err != nil {
		logging.LogError(err)
		drawbox.PrintError(fmt.Sprintf("Failed to set environment variable: %s", err))
	} else {
		logging.LogAlert(fmt.Sprintf("Successfully exported %s=%s", varName, value))
		drawbox.PrintAlert(fmt.Sprintf("Successfully exported %s=%s", varName, value))
	}
}

// listEnvVariables lists all environment variables
func ListEnvVariables() {
	drawbox.RunDrawbox("Environment Variables", "bold_white")
	for _, env := range os.Environ() {
		fmt.Println(env)
	}
}

// unsetEnvVariable unsets an environment variable
func UnsetEnvVariable(args []string) {
	if len(args) < 2 {
		logging.LogAlert("Usage: unset VAR")
		drawbox.PrintError("Usage: unset VAR")
		return
	}

	varName := sanitizeInput(args[1], false)
	if err := os.Unsetenv(varName); err != nil {
		logging.LogError(err)
		drawbox.PrintError(fmt.Sprintf("Failed to unset environment variable: %s", err))
	} else {
		logging.LogAlert(fmt.Sprintf("Successfully unset environment variable: %s", varName))
		drawbox.PrintAlert(fmt.Sprintf("Successfully unset environment variable: %s", varName))
	}
}

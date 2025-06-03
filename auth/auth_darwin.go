//go:build darwin

package auth

import (
	"fmt"
	"os/exec"
	"os/user"

	"secshell/logging"
)

// authenticateUser authenticates a user on macOS systems using dscl
func AuthenticateUser(password string) bool {
	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		logging.LogError(err)
		fmt.Println("Error getting current user:", err)
		return false
	}

	// Use the builtin user password for macOS via dscl
	cmd := exec.Command("dscl", ".", "-authonly", currentUser.Username, password)
	err = cmd.Run()

	// The dscl command returns exit code 0 for successful authentication
	// and non-zero for failed authentication
	if err != nil {
		// Don't log the full error as it might contain sensitive info
		logging.LogAlert("Authentication failed for user: " + currentUser.Username)
		return false
	}

	return true // Authentication successful
}

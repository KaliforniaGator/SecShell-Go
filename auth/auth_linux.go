//go:build !darwin
// +build !darwin

package auth

import (
	"fmt"
	"os/user"

	"secshell/logging"

	"github.com/msteinert/pam"
)

// authenticateUser authenticates a user on Linux systems using PAM
func AuthenticateUser(password string) bool {
	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		logging.LogError(err)
		fmt.Println("Error getting current user:", err)
		return false
	}

	// Use PAM for Linux systems
	t, err := pam.StartFunc("passwd", currentUser.Username, func(s pam.Style, msg string) (string, error) {
		switch s {
		case pam.PromptEchoOff:
			return password, nil
		case pam.PromptEchoOn:
			return "", nil
		case pam.ErrorMsg:
			return "", nil
		case pam.TextInfo:
			return "", nil
		}
		return "", fmt.Errorf("unrecognized PAM message style")
	})

	if err != nil {
		logging.LogError(err)
		fmt.Println("PAM transaction start failed:", err)
		return false
	}

	// Attempt authentication
	err = t.Authenticate(0)
	if err != nil {
		logging.LogError(err)
		fmt.Println("Authentication failed:", err)
		return false
	}

	return true // Authentication successful
}

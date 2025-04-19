package admin

import (
	"os/user"
	"secshell/logging"
)

// Check if the current user is in an admin group
func IsAdmin() bool {
	currentUser, err := user.Current()
	if err != nil {
		logging.LogError(err)
		return false // Fail-safe: assume not an admin
	}

	// Root (UID 0) is always an admin
	if currentUser.Uid == "0" {
		return true
	}

	// Get the user's group IDs
	groups, err := currentUser.GroupIds()
	if err != nil {
		logging.LogError(err)
		return false
	}

	// Define admin groups (adjust as needed)
	adminGroups := []string{"sudo", "admin", "wheel", "root"}

	// Check if the user belongs to an admin group
	for _, groupID := range groups {
		group, err := user.LookupGroupId(groupID)
		if err == nil {
			logging.LogError(err)
			for _, adminGroup := range adminGroups {
				if group.Name == adminGroup {
					return true
				}
			}
		}
	}

	return false
}

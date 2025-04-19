package admin

import (
	"os"
	"os/user"
	"secshell/logging"
	"strconv"
)

// Define admin groups (adjust as needed)
var AdminGroups = []string{"sudo", "admin", "wheel", "root"}

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

	// Check if the user belongs to an admin group
	for _, groupID := range groups {
		group, err := user.LookupGroupId(groupID)
		if err != nil {
			logging.LogError(err)
			continue
		}
		for _, adminGroup := range AdminGroups {
			if group.Name == adminGroup {
				return true
			}
		}
	}

	return false
}

// SetFilePermissions sets file permissions to be accessible only by admin users
func SetFilePermissions(filepath string) error {
	// Set owner-only read/write permissions
	if err := os.Chmod(filepath, 0600); err != nil {
		return err
	}

	// Try to set ownership to first available admin group
	for _, adminGroup := range AdminGroups {
		group, err := user.LookupGroup(adminGroup)
		if err != nil {
			continue
		}

		gid, err := strconv.Atoi(group.Gid)
		if err != nil {
			continue
		}

		// Change ownership to root:adminGroup
		if err := os.Chown(filepath, 0, gid); err != nil {
			continue
		}

		return nil
	}

	return nil
}

// SetFolderPermissions sets directory permissions to be accessible only by admin users
func SetFolderPermissions(folder string) error {
	// Set owner-only read/write/execute permissions for directory
	if err := os.Chmod(folder, 0700); err != nil {
		return err
	}

	// Try to set ownership to first available admin group
	for _, adminGroup := range AdminGroups {
		group, err := user.LookupGroup(adminGroup)
		if err != nil {
			continue
		}

		gid, err := strconv.Atoi(group.Gid)
		if err != nil {
			continue
		}

		// Change ownership to root:adminGroup
		if err := os.Chown(folder, 0, gid); err != nil {
			continue
		}

		return nil
	}

	return nil
}

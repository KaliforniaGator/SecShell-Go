package core

import (
	"fmt"
	"os"
	"secshell/drawbox"
)

var previousDir string

// changeDirectory changes the current working directory
func ChangeDirectory(args []string) {
	var dir string
	if len(args) < 2 {
		home := os.Getenv("HOME")
		if home == "" {
			drawbox.PrintError("cd failed: HOME environment variable not set")
			return
		}
		dir = home
	} else if args[1] == "--prev" || args[1] == "-p" {
		if previousDir == "" {
			drawbox.PrintError("No previous directory available")
			return
		}
		dir = previousDir
	} else {
		dir = args[1]
	}

	current, err := os.Getwd()
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("cd failed: %s", err))
		return
	}

	if err := os.Chdir(dir); err != nil {
		drawbox.PrintError(fmt.Sprintf("cd failed: %s", err))
		return
	}
	previousDir = current
}

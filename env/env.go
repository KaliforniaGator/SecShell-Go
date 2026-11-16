package env

import (
	"fmt"
	"os"
	"secshell/ui/gui"
)

// ListEnvVariables lists all environment variables.
func ListEnvVariables() {
	gui.TitleBox("Environment Variables")
	for _, env := range os.Environ() {
		fmt.Println(env)
	}
}

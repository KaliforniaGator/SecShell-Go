//go:build linux && !cgo
// +build linux,!cgo

package auth

import "fmt"

// AuthenticateUser authenticates a user on Linux systems without PAM
// This is a fallback when CGO is not available (e.g., cross-compilation)
func AuthenticateUser(password string) bool {
	fmt.Println("Warning: PAM authentication is not available in this build")
	fmt.Println("Authentication disabled - built without CGO support")
	// Return false as authentication cannot be performed without PAM
	return false
}
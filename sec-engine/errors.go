package secengine

import "errors"

var (
	ErrNotADirectory  = errors.New("sec-engine: not a directory")
	ErrContextNotInit = errors.New("sec-engine: context not initialized")
	ErrScriptNotFound = errors.New("sec-engine: script file not found")
)
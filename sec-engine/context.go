package secengine

import (
	"os"
	"path/filepath"
	"sync"
)

// ScriptContext holds the execution context for a .sec script
type ScriptContext struct {
	mu         sync.Mutex
	envVars    map[string]string
	workDir    string
	portMap    map[int]string
	scanResult []string
}

// NewScriptContext creates a new execution context
func NewScriptContext() (*ScriptContext, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Copy current environment
	envVars := make(map[string]string)
	for _, env := range os.Environ() {
		for i := 0; i < len(env); i++ {
			if env[i] == '=' {
				key := env[:i]
				value := env[i+1:]
				envVars[key] = value
				break
			}
		}
	}

	return &ScriptContext{
		envVars: envVars,
		workDir: workDir,
		portMap: make(map[int]string),
	}, nil
}

// GetEnv returns an environment variable value
func (ctx *ScriptContext) GetEnv(key string) string {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.envVars[key]
}

// SetEnv sets an environment variable
func (ctx *ScriptContext) SetEnv(key, value string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.envVars[key] = value
	os.Setenv(key, value)
}

// UnsetEnv removes an environment variable
func (ctx *ScriptContext) UnsetEnv(key string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	delete(ctx.envVars, key)
	os.Unsetenv(key)
}

// GetWorkDir returns the current working directory
func (ctx *ScriptContext) GetWorkDir() string {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.workDir
}

// SetWorkDir changes the working directory
func (ctx *ScriptContext) SetWorkDir(dir string) error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return ErrNotADirectory
	}

	ctx.workDir = absDir
	return os.Chdir(absDir)
}

// PortMap adds a port to protocol mapping
func (ctx *ScriptContext) PortMap(port int, protocol string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.portMap[port] = protocol
}

// GetPortMap returns the port mapping
func (ctx *ScriptContext) GetPortMap() map[int]string {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	result := make(map[int]string)
	for k, v := range ctx.portMap {
		result[k] = v
	}
	return result
}
package secengine

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/gopher-lua"
)

// Engine holds the Lua VM state and execution context
type Engine struct {
	L       *lua.LState
	Context *ScriptContext
}

// NewEngine creates a new SecEngine instance
func NewEngine() (*Engine, error) {
	ctx, err := NewScriptContext()
	if err != nil {
		return nil, fmt.Errorf("failed to create script context: %v", err)
	}

	L := lua.NewState()

	eng := &Engine{
		L:       L,
		Context: ctx,
	}

	// Register all SecShell builtin functions
	RegisterFunctions(L)

	return eng, nil
}

// Close cleans up the Lua VM
func (eng *Engine) Close() {
	if eng.L != nil {
		eng.L.Close()
	}
}

// ExecuteFile loads and runs a .sec script file
func (eng *Engine) ExecuteFile(filePath string) error {
	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("script file not found: %v", err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a script file", filePath)
	}

	// Read file contents
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read script: %v", err)
	}

	// Strip shebang line if present
	script := string(content)
	if strings.HasPrefix(script, "#!") {
		lines := strings.SplitN(script, "\n", 2)
		if len(lines) > 1 {
			script = lines[1]
		}
	}

	return eng.Execute(script)
}

// Execute runs a .sec script string
func (eng *Engine) Execute(script string) error {
	if err := eng.L.DoString(script); err != nil {
		return fmt.Errorf("script execution error: %v", err)
	}
	return nil
}

// ExecuteScriptFile is the main entry point for running .sec files
func ExecuteScriptFile(filePath string, args []string) error {
	// Check if file exists
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %v", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("script file not found: %v", err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a script file", absPath)
	}

	// Read file contents
	file, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("failed to open script: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Strip shebang line if present
	if len(lines) > 0 && strings.HasPrefix(lines[0], "#!") {
		lines = lines[1:]
	}

	script := strings.Join(lines, "\n")

	// Create engine and execute
	eng, err := NewEngine()
	if err != nil {
		return fmt.Errorf("failed to create engine: %v", err)
	}
	defer eng.Close()

	// Pass script arguments as Lua variables
	eng.L.SetGlobal("args", argsToTable(eng.L, args))
	eng.L.SetGlobal("script_name", lua.LString(absPath))

	if err := eng.Execute(script); err != nil {
		return err
	}

	return nil
}

// IsSecScript checks if a file path is a .sec script
func IsSecScript(filePath string) bool {
	ext := filepath.Ext(filePath)
	if ext != ".sec" {
		return false
	}

	// Also check for shebang
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		firstLine := scanner.Text()
		if strings.Contains(firstLine, "secshell") || strings.Contains(firstLine, "sec-engine") {
			return true
		}
	}

	return true
}

// Helper: convert string slice to Lua table
func argsToTable(L *lua.LState, args []string) lua.LValue {
	table := L.NewTable()
	for _, arg := range args {
		table.Append(lua.LString(arg))
	}
	return table
}
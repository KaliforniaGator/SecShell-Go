package secengine

import (
	"github.com/yuin/gopher-lua"
)

// FunctionRegistry maps function names to their Lua implementations
// This is the central registry - add new functions here to make them available in .sec scripts
var FunctionRegistry = map[string]lua.LGFunction{
	// Core execution
	"run":       builtinRun,       // run(cmd) - execute command, return output string
	"exec":      builtinExec,      // exec(cmd) - execute command, return (exitCode, stdout, stderr)
	"pipe":      builtinPipe,      // pipe(cmd1, cmd2, ...) - pipe commands together

	// Directory and environment
	"cd":        builtinCd,        // cd(dir) - change directory
	"env":       builtinEnv,       // env(key) - get environment variable
	"set":       builtinSet,       // set(key, value) - set environment variable
	"unset":     builtinUnset,     // unset(key) - remove environment variable

	// File operations
	"read":      builtinRead,      // read(file) - read file contents
	"write":     builtinWrite,     // write(file, data) - write to file (data can be string or table)
	"glob":      builtinGlob,      // glob(pattern) - file pattern matching

	// Network
	"fetch":     builtinFetch,     // fetch(url, method) - HTTP request
	"portmap":   builtinPortmap,   // portmap(port, protocol) - map port to protocol name
	"scan":      builtinScan,      // scan() - scan current network for live hosts

	// I/O
	"readinput": builtinReadInput, // readinput() - read keyboard input
	"print":     builtinPrint,     // print(...) - print to stdout (overrides Lua print with enhanced features)
}

// RegisterFunctions registers all SecShell builtin functions in the Lua state
func RegisterFunctions(L *lua.LState) {
	for name, fn := range FunctionRegistry {
		L.SetGlobal(name, L.NewFunction(fn))
	}
}
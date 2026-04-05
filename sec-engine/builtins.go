package secengine

import (
	"bufio"
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
	"strings"
	"sync"
	"time"

	"github.com/yuin/gopher-lua"
)

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
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		L.Push(lua.LBool(false))
		L.Push(lua.LString(""))
		L.Push(lua.LString(err.Error()))
		return 3
	}

	stdoutBytes, _ := io.ReadAll(stdout)
	stderrBytes, _ := io.ReadAll(stderr)
	cmd.Wait()

	success := cmd.ProcessState == nil || cmd.ProcessState.Success()
	L.Push(lua.LBool(success))
	L.Push(lua.LString(string(stdoutBytes)))
	L.Push(lua.LString(string(stderrBytes)))
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

	client := &http.Client{}
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
		L.RaiseError("failed to read script '%s': %v", script, err)
		return 0
	}

	// Execute the script in the same Lua state
	if err := L.DoString(string(content)); err != nil {
		L.RaiseError("error executing script '%s': %v", script, err)
		return 0
	}

	return 0
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
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"purple":  "\033[35m",
		"cyan":    "\033[36m",
		"white":   "\033[37m",
		"bold":    "\033[1m",
		"reset":   "\033[0m",
	}

	code, exists := colorCodes[color]
	if !exists {
		code = colorCodes["white"]
	}

	fmt.Printf("%s%s\033[0m\n", code, text)
	return 0
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

package secengine

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	var results []string
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

				// Ping sweep the subnet
				for i := 1; i <= 254; i++ {
					target := fmt.Sprintf("%s.%d", subnet, i)
					conn, err := net.DialTimeout("tcp", target+":22", 100*time.Millisecond)
					if err == nil {
						results = append(results, target)
						conn.Close()
					}
				}
			}
		}
	}

	table := L.NewTable()
	for _, host := range results {
		table.Append(lua.LString(host))
	}

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
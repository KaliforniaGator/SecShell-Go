# SecEngine - SecShell Lua Scripting Engine

SecEngine embeds a Lua VM (gopher-lua) to power `.sec` scripts with simple, shell-like syntax.

## Available Functions

### Core Execution

| Function | Signature | Description |
|----------|-----------|-------------|
| `run` | `run(cmd) -> output` | Execute command, return output as string |
| `exec` | `exec(cmd) -> success, stdout, stderr` | Execute command, return exit status and output |
| `pipe` | `pipe(cmd1, cmd2, ...) -> output` | Pipe commands together |

### Directory & Environment

| Function | Signature | Description |
|----------|-----------|-------------|
| `cd` | `cd(dir) -> success, error` | Change working directory |
| `env` | `env(key) -> value` | Get environment variable |
| `set` | `set(key, value)` | Set environment variable |
| `unset` | `unset(key)` | Remove environment variable |

### File Operations

| Function | Signature | Description |
|----------|-----------|-------------|
| `read` | `read(file) -> success, content` | Read file contents |
| `write` | `write(file, data) -> success, error` | Write to file (data can be string or table) |
| `glob` | `glob(pattern) -> success, matches` | File pattern matching |
| `exists` | `exists(path) -> bool` | Check if file/directory exists |
| `isDir` | `isDir(path) -> bool` | Check if path is a directory |
| `isFile` | `isFile(path) -> bool` | Check if path is a file |
| `mkdir` | `mkdir(path) -> success, error` | Create directory (with parents) |
| `copy` | `copy(src, dst) -> success, error` | Copy file |
| `move` | `move(src, dst) -> success, error` | Move/rename file |
| `delete` | `delete(path) -> success, error` | Delete file or directory |
| `stat` | `stat(path) -> success, info` | Get file metadata (name, size, isDir, mode, perm, modTime) |

### Network

| Function | Signature | Description |
|----------|-----------|-------------|
| `fetch` | `fetch(url, method) -> success, body, status` | HTTP request |
| `portmap` | `portmap(port, protocol)` | Map port to protocol name |
| `scan` | `scan() -> success, hosts` | Scan current network for live hosts |

### Security & Cryptography

| Function | Signature | Description |
|----------|-----------|-------------|
| `hash` | `hash(data, algorithm) -> success, hash, error` | Generate hash (md5, sha1, sha256, sha512) |
| `encode` | `encode(data, format) -> success, encoded, error` | Encode data (base64, hex, url) |
| `decode` | `decode(data, format) -> success, decoded, error` | Decode data (base64, hex, url) |

### String Processing

| Function | Signature | Description |
|----------|-----------|-------------|
| `split` | `split(str, sep) -> table` | Split string into table |
| `join` | `join(table, sep) -> string` | Join table elements with separator |
| `trim` | `trim(str) -> string` | Trim whitespace from string |
| `upper` | `upper(str) -> string` | Convert string to uppercase |
| `lower` | `lower(str) -> string` | Convert string to lowercase |
| `replace` | `replace(str, old, new) -> string` | Replace all occurrences of substring |
| `match` | `match(pattern, str) -> success, matches` | Regex pattern matching |

### Data Formats

| Function | Signature | Description |
|----------|-----------|-------------|
| `jsonEncode` | `jsonEncode(table) -> success, json, error` | Convert Lua table to JSON string |
| `jsonDecode` | `jsonDecode(str) -> success, table` | Parse JSON string to Lua table |

### Random & Time

| Function | Signature | Description |
|----------|-----------|-------------|
| `random` | `random(min, max) -> number` | Generate random number in range |
| `randomString` | `randomString(length) -> string` | Generate random alphanumeric string |
| `sleep` | `sleep(seconds)` | Pause execution (supports decimals) |
| `time` | `time() -> timestamp` | Get current Unix timestamp |
| `formatTime` | `formatTime(timestamp, fmt) -> string` | Format timestamp (Go format: "2006-01-02 15:04:05") |

### I/O & UX

| Function | Signature | Description |
|----------|-----------|-------------|
| `readinput` | `readinput() -> line` | Read keyboard input |
| `print` | `print(...)` | Print to stdout |
| `prompt` | `prompt(text) -> input` | Show prompt and read user input |
| `colorPrint` | `colorPrint(text, color)` | Print colored text (red, green, yellow, blue, purple, cyan, white, bold) |

### Error Handling

| Function | Signature | Description |
|----------|-----------|-------------|
| `attempt` | `attempt(func, ...) -> success, error` | Safe function execution (wraps pcall) |
| `pcall` | `pcall(func, ...) -> success, ...` | Protected call with error handling |

### Network Reconnaissance

| Function | Signature | Description |
|----------|-----------|-------------|
| `tcpConnect` | `tcpConnect(host, port, timeout) -> success, status, banner` | TCP connection test with banner grab |
| `udpProbe` | `udpProbe(host, port, timeout) -> success, response` | UDP port probe |
| `serviceDetect` | `serviceDetect(host, port, timeout) -> success, info, error` | Identify service on port |
| `osDetect` | `osDetect(host) -> success, info, error` | Basic OS fingerprinting via TTL |

### Payload Generation

| Function | Signature | Description |
|----------|-----------|-------------|
| `genReverseShell` | `genReverseShell(lhost, lport, type) -> success, payload, error` | Generate reverse shell payload |
| `genBindShell` | `genBindShell(port, type) -> success, payload, error` | Generate bind shell payload |
| `encodePayload` | `encodePayload(data, encoder) -> success, encoded, error` | Encode with various methods (base64, hex, url, unicode, xor, rot13) |

### Exploitation Helpers

| Function | Signature | Description |
|----------|-----------|-------------|
| `httpRequest` | `httpRequest(url, method, headers, body) -> success, response` | Full HTTP request with custom headers |
| `fuzz` | `fuzz(template, payloads) -> success, results` | Fuzzing helper with template substitution |
| `bruteForce` | `bruteForce(target, service, wordlist) -> success, credentials` | Basic brute force (http, ftp) |

### Post-Exploitation

| Function | Signature | Description |
|----------|-----------|-------------|
| `privCheck` | `privCheck() -> success, info` | Check current privilege level |
| `enumSystem` | `enumSystem() -> success, info` | Enumerate system information |
| `enumNetwork` | `enumNetwork() -> success, interfaces` | Enumerate network interfaces |
| `persistAdd` | `persistAdd(script, method) -> success, result` | Add persistence mechanism (cron, bashrc, systemd) |
| `persistRemove` | `persistRemove() -> success, results` | Remove persistence mechanisms |

### Script Organization

| Function | Signature | Description |
|----------|-----------|-------------|
| `require` | `require(script) -> success, module_or_error` | Import/run another .sec script (supports relative paths, returns module table on success) |
| `exit` | `exit(code)` | Exit script with status code |

### Script Variables

| Variable | Type | Description |
|----------|------|-------------|
| `args` | table | Command-line arguments passed to script |
| `script_name` | string | Full path to the script file |

## Adding New Functions

To add a new function:

1. Implement it in `builtins.go`:
```go
func builtinMyFunc(L *lua.LState) int {
    // Get arguments from L
    arg1 := L.CheckString(1)
    
    // Do work...
    
    // Push return values
    L.Push(lua.LString(result))
    return 1 // number of return values
}
```

2. Register it in `functions.go`:
```go
var FunctionRegistry = map[string]lua.LGFunction{
    // existing functions...
    "myfunc": builtinMyFunc,
}
```

## Creating Reusable Modules

SecEngine supports creating reusable `.sec` library modules that can be loaded with `require()`.

### Creating a Module

Create a `.sec` file that returns a table of functions:

```lua
-- libs/mymodule.sec
local M = {}

function M.greet(name)
    return "Hello, " .. name .. "!"
end

function M.add(a, b)
    return a + b
end

return M  -- Return the module table
```

### Using a Module

```lua
-- main.sec
local ok, mymod = require("libs/mymodule.sec")
if ok then
    print(mymod.greet("World"))  -- Hello, World!
    print(mymod.add(5, 3))       -- 8
else
    print("Failed to load module: " .. mymod)
end
```

### Module Path Resolution

- Relative paths are resolved from the calling script's directory
- Absolute paths are used as-is
- Example: `require("libs/utils.sec")` looks for `libs/utils.sec` relative to the current script

## Example .sec Scripts

### Basic Script
```lua
#!/usr/bin/secshell

-- Simple port scanner script
print("SecShell Port Scanner")
print("=====================")

local target = "192.168.1.1"
local ports = {22, 80, 443, 8080}

for _, port in ipairs(ports) do
    local result = run("nc -z -w1 " .. target .. " " .. port)
    if result ~= "" then
        print("Port " .. port .. " is open")
    end
end
```

### Hash & Encode Example
```lua
#!/usr/bin/secshell

-- Password hashing and encoding
local password = "mySecretPassword"

-- Generate hashes
local ok, md5hash = hash(password, "md5")
print("MD5: " .. md5hash)

local ok, sha256hash = hash(password, "sha256")
print("SHA256: " .. sha256hash)

-- Encode/decode
local ok, encoded = encode("Hello World", "base64")
print("Base64: " .. encoded)

local ok, decoded = decode(encoded, "base64")
print("Decoded: " .. decoded)
```

### File Operations Example
```lua
#!/usr/bin/secshell

-- File management
local testFile = "/tmp/test.txt"

-- Check and create
if not exists(testFile) then
    local ok, err = write(testFile, "Hello from SecShell")
    if ok then
        print("File created successfully")
    end
end

-- Read and stat
local ok, content = read(testFile)
print("Content: " .. content)

local ok, info = stat(testFile)
print("Size: " .. info.size)
print("Modified: " .. info.modTime)

-- Cleanup
delete(testFile)
```

### JSON & String Processing Example
```lua
#!/usr/bin/secshell

-- JSON encoding/decoding
local data = {
    name = "SecShell",
    version = "1.0",
    ports = {22, 80, 443}
}

local ok, jsonStr = jsonEncode(data)
print("JSON: " .. jsonStr)

local ok, parsed = jsonDecode(jsonStr)
print("Parsed name: " .. parsed.name)

-- String processing
local text = "  Hello, World!  "
print("Trimmed: " .. trim(text))
print("Upper: " .. upper(text))
print("Lower: " .. lower(text))

-- Regex matching
local ok, matches = match("\\d+", "Order 123 and 456")
if ok then
    for i, m in ipairs(matches) do
        print("Match " .. i .. ": " .. m)
    end
end
```

### Error Handling Example
```lua
#!/usr/bin/secshell

-- Safe function execution with attempt()
local function divide(a, b)
    if b == 0 then
        error("Division by zero!")
    end
    return a / b
end

-- Using attempt() for safe execution
local ok, result = attempt(divide, 10, 2)
print("divide(10, 2): ok=" .. tostring(ok) .. ", result=" .. tostring(result))

local ok, err = attempt(divide, 10, 0)
print("divide(10, 0): ok=" .. tostring(ok) .. ", error=" .. tostring(err))

-- Using pcall() for more control
local ok, val = pcall(divide, 20, 4)
if ok then
    print("Result: " .. val)
else
    print("Error: " .. val)
end
```

### Network Reconnaissance Example
```lua
#!/usr/bin/secshell

-- TCP connection test
local ok, status, banner = tcpConnect("google.com", 80, 3)
if ok then
    print("Port 80 is " .. status)
    if #banner > 0 then
        print("Banner: " .. banner)
    end
end

-- Service detection
local ok, info = serviceDetect("google.com", 443, 3)
if ok then
    print("Service: " .. info.service .. " on port " .. info.port)
end

-- OS detection via TTL
local ok, osInfo = osDetect("127.0.0.1")
if ok then
    print("OS: " .. osInfo.os .. " (TTL: " .. osInfo.ttl .. ")")
end
```

### Payload Generation Example
```lua
#!/usr/bin/secshell

-- Generate reverse shell payload
local ok, payload = genReverseShell("10.0.0.1", "4444", "bash")
if ok then
    print("Payload: " .. payload.payload)
    print("Base64: " .. payload.encoded)
end

-- Generate bind shell
local ok, bindPayload = genBindShell("8080", "nc")
if ok then
    print("Bind shell: " .. bindPayload.payload)
end

-- Encode payload
local ok, encoded = encodePayload("test data", "base64")
print("Encoded: " .. encoded)

local ok, rot13 = encodePayload("secret", "rot13")
print("ROT13: " .. rot13)
```

### Post-Exploitation Example
```lua
#!/usr/bin/secshell

-- Check privileges
local ok, priv = privCheck()
if ok then
    print("Level: " .. priv.level)
    print("UID: " .. priv.uid)
end

-- Enumerate system
local ok, sys = enumSystem()
if ok then
    print("Hostname: " .. sys.hostname)
    print("OS: " .. sys.os .. " (" .. sys.arch .. ")")
    print("User: " .. sys.user)
end

-- Enumerate network
local ok, net = enumNetwork()
if ok then
    for i, iface in ipairs(net.interfaces) do
        print("Interface: " .. iface.name)
    end
end
```

### Module Usage Example
```lua
#!/usr/bin/secshell

-- Load a reusable library
local ok, utils = require("libs/utils.sec")
if not ok then
    print("Failed to load utils: " .. utils)
    exit(1)
end

-- Use utility functions
utils.header("My Security Tool")

-- Generate random IP
local target = utils.randomIP()
print("Target: " .. target)

-- Hash passwords
local hashes = utils.hashAll("password123")
print("MD5: " .. hashes.md5)

-- Check port
local open, status = utils.checkPort("google.com", 80)
print("Port 80: " .. status)
```

## Interactive REPL

Type `sec` in SecShell to launch an interactive Lua scripting environment:

```
sec> 2 + 2
4
sec> hash("test", "md5")
true    098f6bcd4621d373cade4e832627b4f6
sec> x = 10
sec> x * 2
20
sec> run("whoami")
rafaelrivera
sec> exit
```

### REPL Commands
- `help` - Show available functions
- `clear` - Clear screen
- `history` - Show command history
- `exit` / `quit` - Exit REPL
- Ctrl+D - Also exits

## Usage

Run `.sec` scripts from SecShell:
```
./secshell
# Then at the prompt:
./myscript.sec arg1 arg2
```

Or make scripts executable:
```bash
chmod +x myscript.sec
./myscript.sec arg1 arg2

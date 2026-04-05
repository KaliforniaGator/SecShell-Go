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

### Script Organization

| Function | Signature | Description |
|----------|-----------|-------------|
| `require` | `require(script)` | Import/run another .sec script (supports relative paths) |
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

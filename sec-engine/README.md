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

### Network

| Function | Signature | Description |
|----------|-----------|-------------|
| `fetch` | `fetch(url, method) -> success, body, status` | HTTP request |
| `portmap` | `portmap(port, protocol)` | Map port to protocol name |
| `scan` | `scan() -> success, hosts` | Scan current network for live hosts |

### I/O

| Function | Signature | Description |
|----------|-----------|-------------|
| `readinput` | `readinput() -> line` | Read keyboard input |
| `print` | `print(...)` | Print to stdout |

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

## Example .sec Script

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

## Usage

Run `.sec` scripts from SecShell:
```
./myscript.sec arg1 arg2
```

Or directly if made executable:
```bash
chmod +x myscript.sec
./myscript.sec
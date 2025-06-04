# Terminal Package

This package provides standardized utilities for terminal manipulation in SecShell, particularly for handling the alternate screen buffer and raw mode.

## Features

- Modern escape sequences for alternate screen buffer (`\033[?1049h` and `\033[?1049l`)
- Thread-safe terminal state management
- Simplified API for interactive terminal applications
- Proper cleanup on exit

## Usage

### Basic Usage

```go
// Enter and exit alternate screen buffer
terminal.EnterAlternateScreen()
// ... do something in alternate screen ...
terminal.ExitAlternateScreen()

// Enter and exit raw mode
err := terminal.EnterRawMode()
if err != nil {
    // handle error
}
// ... do something in raw mode ...
terminal.ExitRawMode()
```

### Interactive Mode (Alternate Screen + Raw Mode)

```go
// Enter interactive mode (both alternate screen and raw mode)
err := terminal.EnterInteractiveMode()
if err != nil {
    // handle error
}
// ... do interactive operations ...
terminal.ExitInteractiveMode()
```

### Using the Convenience Function

```go
err := terminal.WithInteractiveMode(func() error {
    // This code runs in interactive mode
    // (alternate screen buffer + raw mode)
    
    // Return an error if something goes wrong
    return nil
})
if err != nil {
    // handle error
}
```

### Helper Functions

```go
// Get terminal dimensions
width, height, err := terminal.GetTerminalSize()

// Check if a file descriptor is a terminal
isTerm := terminal.IsTerminal(fd)
```

## Thread Safety

All functions in this package are thread-safe and can be called from multiple goroutines. 
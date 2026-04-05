package secengine

import (
	"fmt"
	"os"
	"strings"

	"secshell/colors"

	lua "github.com/yuin/gopher-lua"
	"golang.org/x/term"
)

// replPrint is a helper function that prints text followed by \r\n for proper raw mode output.
// In raw terminal mode, \n alone doesn't return to the start of the line,
// so we need \r\n to properly move to the next line.
func replPrint(text string) {
	fmt.Print(text + "\r\n")
}

// replPrintf is a helper function that formats and prints text followed by \r\n for proper raw mode output.
func replPrintf(format string, args ...interface{}) {
	fmt.Print(fmt.Sprintf(format+"\r\n", args...))
}

// REPL provides an interactive Lua prompt similar to Python's shell
type REPL struct {
	engine       *Engine
	history      []string
	historyIndex int
	inBlock      bool
	blockCode    strings.Builder
}

// NewREPL creates a new interactive REPL instance
func NewREPL() (*REPL, error) {
	eng, err := NewEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %v", err)
	}

	return &REPL{
		engine:       eng,
		history:      make([]string, 0),
		historyIndex: 0,
	}, nil
}

// Close cleans up resources
func (r *REPL) Close() {
	r.engine.Close()
}

// Run starts the interactive REPL loop
func (r *REPL) Run() error {
	fmt.Println("SecEngine:")
	fmt.Println("Type 'help' for available functions, 'exit' or 'quit' to exit")
	fmt.Println()

	// Set up terminal raw mode for arrow key support
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to set terminal to raw mode: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	for {
		// Show appropriate prompt
		if r.inBlock {
			fmt.Print("... ")
		} else {
			fmt.Print("sec >> ")
		}

		line, gotEOF := r.getInput()
		if gotEOF {
			// EOF (Ctrl+D)
			fmt.Println()
			break
		}

		// Handle special commands
		if !r.inBlock {
			switch strings.TrimSpace(strings.ToLower(line)) {
			case "exit", "quit":
				return nil
			case "help":
				r.printHelp()
				continue
			case "clear":
				fmt.Print("\033[H\033[2J")
				continue
			case "history":
				r.printHistory()
				continue
			}
		}

		// Add to history
		r.history = append(r.history, line)

		// Check if we need to continue reading a block
		if r.needsContinuation(line) {
			if !r.inBlock {
				r.inBlock = true
				r.blockCode.Reset()
			}
			r.blockCode.WriteString(line)
			r.blockCode.WriteString("\n")
			continue
		}

		// Execute the code
		var code string
		if r.inBlock {
			r.blockCode.WriteString(line)
			code = r.blockCode.String()
			r.inBlock = false
			r.blockCode.Reset()
		} else {
			code = line
		}

		// Skip empty lines
		if strings.TrimSpace(code) == "" {
			continue
		}

		// Execute and show results
		r.executeAndPrint(code)
	}

	return nil
}

// getInput handles reading input with arrow key support and line editing
// Returns the input line and a boolean indicating if EOF was reached
func (r *REPL) getInput() (string, bool) {
	line := ""
	pos := 0
	buf := make([]byte, 8192)

	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return strings.TrimSpace(line), true
		}

		for i := 0; i < n; i++ {
			// Handle bracketed paste mode
			if i+5 < n &&
				buf[i] == 27 && buf[i+1] == '[' && buf[i+2] == '2' &&
				buf[i+3] == '0' && buf[i+4] == '0' && buf[i+5] == '~' {
				i += 6
				continue
			}

			switch buf[i] {
			case 3: // Ctrl+C
				fmt.Print("\r\n")
				fmt.Printf("%s To exit, type 'exit' or 'quit' %s\n", colors.BoldYellow, colors.Reset)
				fmt.Print("\r\n")
				if r.inBlock {
					fmt.Print("... ")
				} else {
					fmt.Print("sec >> ")
				}
				line = ""
				pos = 0
				continue

			case 4: // Ctrl+D (EOF)
				fmt.Print("\r\n")
				return strings.TrimSpace(line), true

			case 27: // ESC sequence
				if i+2 < n && buf[i+1] == '[' {
					switch buf[i+2] {
					case 'A': // Up arrow - history navigation
						if r.historyIndex > 0 {
							r.historyIndex--
							newLine := r.history[r.historyIndex]
							// Clear current line and print history entry
							fmt.Printf("\x1b[%dD\x1b[K%s", pos, newLine)
							line = newLine
							pos = len(line)
						}
						i += 2

					case 'B': // Down arrow - history navigation
						if r.historyIndex < len(r.history)-1 {
							r.historyIndex++
							newLine := r.history[r.historyIndex]
							fmt.Printf("\x1b[%dD\x1b[K%s", pos, newLine)
							line = newLine
							pos = len(line)
						} else {
							// Clear line and go to empty
							fmt.Printf("\x1b[%dD\x1b[K", pos)
							line = ""
							pos = 0
							r.historyIndex = len(r.history)
						}
						i += 2

					case 'C': // Right arrow - move cursor right
						if pos < len(line) {
							pos++
							fmt.Print("\x1b[C")
						}
						i += 2

					case 'D': // Left arrow - move cursor left
						if pos > 0 {
							pos--
							fmt.Print("\x1b[D")
						}
						i += 2
					}
				}

			case 127, 8: // Backspace and Delete
				if pos > 0 {
					line = line[:pos-1] + line[pos:]
					pos--
					fmt.Print("\x1b[D\x1b[K")
					if pos < len(line) {
						fmt.Print(line[pos:])
						fmt.Printf("\x1b[%dD", len(line)-pos)
					}
				}

			case 9: // Tab - insert 4 spaces
				fmt.Print("    ")
				line = line[:pos] + "    " + line[pos:]
				pos += 4

			case 13, 10: // Enter (CR or LF)
				fmt.Print("\r\n") // Use explicit CR LF for raw mode
				input := strings.TrimSpace(line)
				if input != "" {
					r.history = append(r.history, input)
					r.historyIndex = len(r.history)
				}
				return input, false

			default:
				if buf[i] >= 32 { // Printable characters
					// Insert character at current position
					line = line[:pos] + string(buf[i]) + line[pos:]
					fmt.Print(line[pos:])
					pos++
					if pos < len(line) {
						fmt.Printf("\x1b[%dD", len(line)-pos)
					}
				}
			}
		}
	}
}

// needsContinuation checks if the current line needs more input to be complete
func (r *REPL) needsContinuation(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Empty lines don't need continuation
	if trimmed == "" {
		return false
	}

	// Check for unclosed blocks
	// Count keywords that start blocks
	blockStarts := []string{"function", "if", "then", "else", "elseif", "for", "while", "do", "repeat"}
	blockEnds := []string{"end", "until"}

	// Simple heuristic: check for common block starters without matching end
	// This is a simplified version - a full parser would be more accurate
	if r.inBlock {
		// Count nested blocks using the current block code, not all history
		depth := 1
		currentBlock := r.blockCode.String() + line
		lines := strings.Split(currentBlock, "\n")
		for _, l := range lines {
			lower := strings.ToLower(strings.TrimSpace(l))
			for _, kw := range blockStarts {
				if strings.HasPrefix(lower, kw+" ") || strings.HasPrefix(lower, kw+"(") || lower == kw {
					depth++
				}
			}
			for _, kw := range blockEnds {
				if lower == kw || strings.HasPrefix(lower, kw+" ") || strings.HasSuffix(lower, " "+kw) {
					depth--
				}
			}
		}
		return depth > 0
	}

	// Check if line starts a new block
	lower := strings.ToLower(trimmed)
	for _, kw := range blockStarts {
		if strings.HasPrefix(lower, kw+" ") || strings.HasPrefix(lower, kw+"(") || lower == kw {
			// Check if it also has 'end' on the same line
			if !strings.Contains(lower, " end") && !strings.HasSuffix(lower, " end") && !strings.HasSuffix(lower, "end)") {
				return true
			}
		}
	}

	// Check for unclosed parentheses, brackets, or strings
	depth := 0
	inString := false
	stringChar := byte(0)
	esc := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if esc {
			esc = false
			continue
		}
		if c == '\\' {
			esc = true
			continue
		}
		if inString {
			if c == stringChar {
				inString = false
			}
			continue
		}
		if c == '"' || c == '\'' {
			inString = true
			stringChar = c
			continue
		}
		if c == '(' || c == '[' || c == '{' {
			depth++
		}
		if c == ')' || c == ']' || c == '}' {
			depth--
		}
	}

	return depth > 0 || inString
}

// executeAndPrint runs code and prints the result
func (r *REPL) executeAndPrint(code string) {
	trimmed := strings.TrimSpace(code)

	// Check if it's a print() call - execute without trying to return
	if strings.HasPrefix(trimmed, "print(") || strings.HasPrefix(trimmed, "colorPrint(") {
		err := r.engine.L.DoString(trimmed)
		if err != nil {
			replPrintf("error: %v", err)
		}
		return
	}

	// First try to evaluate as an expression (return value)
	wrapped := "return " + trimmed

	// Save stack depth before
	stackDepth := r.engine.L.GetTop()

	err := r.engine.L.DoString(wrapped)
	if err == nil {
		// Check how many values were returned
		newDepth := r.engine.L.GetTop()
		if newDepth > stackDepth {
			// Print all return values
			for i := stackDepth + 1; i <= newDepth; i++ {
				ret := r.engine.L.Get(i)
				if ret != lua.LNil {
					replPrintf("%v", ret)
				}
			}
			// Pop all returned values
			r.engine.L.SetTop(stackDepth)
			return
		}
		// No return value, restore stack
		r.engine.L.SetTop(stackDepth)
	}

	// If expression evaluation failed, try as a statement using the SAME state
	// This preserves variables between statements
	err = r.engine.L.DoString(trimmed)
	if err != nil {
		// If it's a syntax error, might need continuation
		if strings.Contains(err.Error(), "unexpected symbol") ||
			strings.Contains(err.Error(), "unfinished string") ||
			strings.Contains(err.Error(), "expected") {
			if !r.inBlock {
				r.inBlock = true
				r.blockCode.Reset()
				r.blockCode.WriteString(code)
				r.blockCode.WriteString("\n")
				return
			}
		}
		replPrintf("error: %v", err)
	}
}

// printHelp displays available commands and functions
func (r *REPL) printHelp() {
	replPrint("")
	replPrint("=== SecEngine Help ===")
	replPrint("")
	replPrint("Special Commands:")
	replPrint("  help          - Show this help")
	replPrint("  clear         - Clear the screen")
	replPrint("  history       - Show command history")
	replPrint("  exit, quit    - Exit the REPL")
	replPrint("")
	replPrint("Available Functions:")
	replPrint("")
	replPrint("Core Execution:")
	replPrint("  run(cmd)           - Execute command, return output")
	replPrint("  exec(cmd)          - Execute command, return (success, stdout, stderr)")
	replPrint("  pipe(cmd1, cmd2)   - Pipe commands together")
	replPrint("")
	replPrint("Directory & Environment:")
	replPrint("  cd(dir)            - Change directory")
	replPrint("  env(key)           - Get environment variable")
	replPrint("  set(key, value)    - Set environment variable")
	replPrint("  unset(key)         - Remove environment variable")
	replPrint("")
	replPrint("File Operations:")
	replPrint("  read(file)         - Read file contents")
	replPrint("  write(file, data)  - Write to file")
	replPrint("  glob(pattern)      - File pattern matching")
	replPrint("  exists(path)       - Check if file/directory exists")
	replPrint("  isDir(path)        - Check if path is a directory")
	replPrint("  isFile(path)       - Check if path is a file")
	replPrint("  mkdir(path)        - Create directory")
	replPrint("  copy(src, dst)     - Copy file")
	replPrint("  move(src, dst)     - Move/rename file")
	replPrint("  delete(path)       - Delete file or directory")
	replPrint("  stat(path)         - Get file metadata")
	replPrint("")
	replPrint("Security & Cryptography:")
	replPrint("  hash(data, algo)   - Generate hash (md5, sha1, sha256, sha512)")
	replPrint("  encode(data, fmt)  - Encode data (base64, hex, url)")
	replPrint("  decode(data, fmt)  - Decode data (base64, hex, url)")
	replPrint("")
	replPrint("String Processing:")
	replPrint("  split(str, sep)    - Split string into table")
	replPrint("  join(table, sep)   - Join table elements")
	replPrint("  trim(str)          - Trim whitespace")
	replPrint("  upper(str)         - Convert to uppercase")
	replPrint("  lower(str)         - Convert to lowercase")
	replPrint("  replace(s,o,n)     - Replace substring")
	replPrint("  match(pat, str)    - Regex pattern matching")
	replPrint("")
	replPrint("Data Formats:")
	replPrint("  jsonEncode(table)  - Convert table to JSON")
	replPrint("  jsonDecode(str)    - Parse JSON string")
	replPrint("")
	replPrint("Random & Time:")
	replPrint("  random(min, max)   - Random number")
	replPrint("  randomString(len)  - Random string")
	replPrint("  sleep(seconds)     - Pause execution")
	replPrint("  time()             - Unix timestamp")
	replPrint("  formatTime(ts,fmt) - Format timestamp")
	replPrint("")
	replPrint("I/O & UX:")
	replPrint("  print(...)         - Print to stdout")
	replPrint("  prompt(text)       - Read user input")
	replPrint("  colorPrint(t,c)    - Colored output")
	replPrint("")
	replPrint("Network:")
	replPrint("  fetch(url, method) - HTTP request")
	replPrint("  scan()             - Network scan")
	replPrint("")
}

// printHistory displays the command history
func (r *REPL) printHistory() {
	replPrint("")
	replPrint("=== Command History ===")
	if len(r.history) == 0 {
		replPrint("  (empty)")
	} else {
		for i, cmd := range r.history {
			replPrintf("  %d: %s", i+1, cmd)
		}
	}
	replPrint("")
}

// StartInteractiveREPL is the main entry point for the sec command
func StartInteractiveREPL() error {
	repl, err := NewREPL()
	if err != nil {
		return fmt.Errorf("failed to create REPL: %v", err)
	}
	defer repl.Close()

	return repl.Run()
}
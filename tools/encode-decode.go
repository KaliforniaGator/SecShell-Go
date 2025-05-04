package tools

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"secshell/logging"
	"strings"
)

// EncodingOperation represents the type of encoding operation to perform
type EncodingOperation int

const (
	EncodeOp EncodingOperation = iota
	DecodeOp
)

// EncodingType represents the type of encoding/decoding to perform
type EncodingType int

const (
	Base64Encoding EncodingType = iota
	HexEncoding
	URLEncoding
)

// OutputHandler handles redirecting output to a file if specified
func OutputHandler(output string, args []string) error {
	// Check for redirection using > or -o flag
	for i, arg := range args {
		if arg == ">" && i+1 < len(args) {
			// Write to file specified after >
			err := os.WriteFile(args[i+1], []byte(output), 0644)
			if err != nil {
				logging.LogError(fmt.Errorf("failed to write output to file %s: %v", args[i+1], err))
				return err
			}
			logging.LogCommand(fmt.Sprintf("Wrote output to file: %s", args[i+1]), 0)
			return nil
		} else if arg == "-o" && i+1 < len(args) {
			// Write to file specified after -o
			err := os.WriteFile(args[i+1], []byte(output), 0644)
			if err != nil {
				logging.LogError(fmt.Errorf("failed to write output to file %s: %v", args[i+1], err))
				return err
			}
			logging.LogCommand(fmt.Sprintf("Wrote output to file: %s", args[i+1]), 0)
			return nil
		}
	}

	// If no redirection, print to stdout
	fmt.Println(output)
	return nil
}

// ParseEncoderArgs parses command arguments for encoding/decoding operations
func ParseEncoderArgs(args []string, encodingType EncodingType) (op EncodingOperation, input string, isFile bool, remainingArgs []string, err error) {
	// Default operation is encode
	op = EncodeOp
	isFile = false

	// Skip the command name which is the first argument
	cmdArgs := args[1:]

	if len(cmdArgs) == 0 {
		return op, input, isFile, remainingArgs, fmt.Errorf("missing arguments")
	}

	// Check if we have a single argument that looks like a joined string of flags and content
	// For example: url -e"Hello World" might come in as ["-e\"Hello", "World\""]
	// In this case, rejoin the parts and process them accordingly
	if len(cmdArgs) > 1 {
		// Check for patterns like: -e"string with spaces"
		for i := 0; i < len(cmdArgs)-1; i++ {
			if (cmdArgs[i] == "-e" || cmdArgs[i] == "-d") && i+1 < len(cmdArgs) &&
				((strings.HasPrefix(cmdArgs[i+1], "\"") && strings.HasSuffix(cmdArgs[len(cmdArgs)-1], "\"")) ||
					(strings.HasPrefix(cmdArgs[i+1], "'") && strings.HasSuffix(cmdArgs[len(cmdArgs)-1], "'"))) {
				// We have a quoted string split across multiple args
				quoted := strings.Join(cmdArgs[i+1:], " ")
				cmdArgs = append(cmdArgs[:i+1], quoted)
				break
			}
		}
	}

	// Process flags
	i := 0
	for i < len(cmdArgs) {
		arg := cmdArgs[i]

		if arg == "-d" {
			op = DecodeOp
			i++
		} else if arg == "-e" {
			op = EncodeOp
			i++
		} else if arg == "-f" && i+1 < len(cmdArgs) {
			isFile = true
			input = cmdArgs[i+1]
			// If the file path is quoted, extract the path from quotes
			input = removeQuotes(input)
			i += 2
		} else if arg == "-o" && i+1 < len(cmdArgs) {
			// Skip output file arguments but include them in remainingArgs
			remainingArgs = append(remainingArgs, arg, cmdArgs[i+1])
			i += 2
		} else if arg == ">" && i+1 < len(cmdArgs) {
			// Skip redirect arguments but include them in remainingArgs
			remainingArgs = append(remainingArgs, arg, cmdArgs[i+1])
			i += 2
		} else {
			// If we've found a non-flag argument, it's our input (if we don't already have one from -f)
			if !isFile && input == "" {
				input = arg
				// Extract content from quotes if present
				input = removeQuotes(input)
				i++
			} else {
				// Any other arguments should be passed to the output handler
				remainingArgs = append(remainingArgs, arg)
				i++
			}
		}
	}

	// Check if required input is missing
	if input == "" {
		return op, input, isFile, remainingArgs, fmt.Errorf("missing input string or file")
	}

	return op, input, isFile, remainingArgs, nil
}

// removeQuotes removes surrounding single or double quotes from a string if present
func removeQuotes(input string) string {
	if len(input) >= 2 {
		// Check for double quotes
		if input[0] == '"' && input[len(input)-1] == '"' {
			return input[1 : len(input)-1]
		}
		// Check for single quotes
		if input[0] == '\'' && input[len(input)-1] == '\'' {
			return input[1 : len(input)-1]
		}
	}
	return input
}

// ExecuteEncodingCommand executes the appropriate encoding/decoding operation
func ExecuteEncodingCommand(args []string, encodingType EncodingType) error {
	op, input, isFile, remainingArgs, err := ParseEncoderArgs(args, encodingType)
	if err != nil {
		logging.LogError(fmt.Errorf("parsing arguments: %v", err))
		return err
	}

	switch encodingType {
	case Base64Encoding:
		b64 := Base64Functions{}
		if op == EncodeOp {
			return b64.Base64Encode(input, isFile, remainingArgs)
		} else {
			return b64.Base64Decode(input, isFile, remainingArgs)
		}

	case HexEncoding:
		hex := HexFunctions{}
		if op == EncodeOp {
			return hex.HexEncode(input, isFile, remainingArgs)
		} else {
			return hex.HexDecode(input, isFile, remainingArgs)
		}

	case URLEncoding:
		url := URLFunctions{}
		if op == EncodeOp {
			return url.URLEncode(input, remainingArgs)
		} else {
			return url.URLDecode(input, remainingArgs)
		}

	default:
		return fmt.Errorf("unknown encoding type")
	}
}

// Base64Functions contains methods for base64 encoding and decoding
type Base64Functions struct{}

// Base64Encode encodes a string or file content to base64
func (b Base64Functions) Base64Encode(input string, isFile bool, args []string) error {
	var encodedStr string
	var logMsg string

	if isFile {
		// Read from file
		data, err := os.ReadFile(input)
		if err != nil {
			logErr := fmt.Errorf("base64 encode: error reading file %s: %v", input, err)
			logging.LogError(logErr)
			return fmt.Errorf("error reading file: %v", err)
		}
		encodedStr = base64.StdEncoding.EncodeToString(data)
		logMsg = fmt.Sprintf("Base64 encoded file: %s (size: %d bytes)", input, len(data))
	} else {
		// Encode the input string
		encodedStr = base64.StdEncoding.EncodeToString([]byte(input))
		// Truncate input string in log if too long
		displayInput := input
		if len(input) > 50 {
			displayInput = input[:47] + "..."
		}
		logMsg = fmt.Sprintf("Base64 encoded string: %s", displayInput)
	}

	outputErr := OutputHandler(encodedStr, args)
	if outputErr != nil {
		logging.LogError(outputErr)
		return outputErr
	}

	logging.LogCommand(logMsg, 0)
	return nil
}

// Base64Decode decodes a base64 string or file content
func (b Base64Functions) Base64Decode(input string, isFile bool, args []string) error {
	var encodedInput string
	var logMsg string

	if isFile {
		// Read from file
		data, err := os.ReadFile(input)
		if err != nil {
			logErr := fmt.Errorf("base64 decode: error reading file %s: %v", input, err)
			logging.LogError(logErr)
			return fmt.Errorf("error reading file: %v", err)
		}
		encodedInput = strings.TrimSpace(string(data))
		logMsg = fmt.Sprintf("Base64 decoded file: %s", input)
	} else {
		encodedInput = input
		// Truncate input string in log if too long
		displayInput := input
		if len(input) > 50 {
			displayInput = input[:47] + "..."
		}
		logMsg = fmt.Sprintf("Base64 decoded string: %s", displayInput)
	}

	decoded, err := base64.StdEncoding.DecodeString(encodedInput)
	if err != nil {
		logErr := fmt.Errorf("base64 decode: invalid input: %v", err)
		logging.LogError(logErr)
		return fmt.Errorf("error decoding base64: %v", err)
	}

	outputErr := OutputHandler(string(decoded), args)
	if outputErr != nil {
		logging.LogError(outputErr)
		return outputErr
	}

	logging.LogCommand(logMsg, 0)
	return nil
}

// HexFunctions contains methods for hex encoding and decoding
type HexFunctions struct{}

// HexEncode encodes a string or file content to hex
func (h HexFunctions) HexEncode(input string, isFile bool, args []string) error {
	var encodedStr string
	var logMsg string

	if isFile {
		// Read from file
		data, err := os.ReadFile(input)
		if err != nil {
			logErr := fmt.Errorf("hex encode: error reading file %s: %v", input, err)
			logging.LogError(logErr)
			return fmt.Errorf("error reading file: %v", err)
		}
		encodedStr = hex.EncodeToString(data)
		logMsg = fmt.Sprintf("Hex encoded file: %s (size: %d bytes)", input, len(data))
	} else {
		// Encode the input string
		encodedStr = hex.EncodeToString([]byte(input))
		// Truncate input string in log if too long
		displayInput := input
		if len(input) > 50 {
			displayInput = input[:47] + "..."
		}
		logMsg = fmt.Sprintf("Hex encoded string: %s", displayInput)
	}

	outputErr := OutputHandler(encodedStr, args)
	if outputErr != nil {
		logging.LogError(outputErr)
		return outputErr
	}

	logging.LogCommand(logMsg, 0)
	return nil
}

// HexDecode decodes a hex string or file content
func (h HexFunctions) HexDecode(input string, isFile bool, args []string) error {
	var encodedInput string
	var logMsg string

	if isFile {
		// Read from file
		data, err := os.ReadFile(input)
		if err != nil {
			logErr := fmt.Errorf("hex decode: error reading file %s: %v", input, err)
			logging.LogError(logErr)
			return fmt.Errorf("error reading file: %v", err)
		}
		encodedInput = strings.TrimSpace(string(data))
		logMsg = fmt.Sprintf("Hex decoded file: %s", input)
	} else {
		encodedInput = input
		// Truncate input string in log if too long
		displayInput := input
		if len(input) > 50 {
			displayInput = input[:47] + "..."
		}
		logMsg = fmt.Sprintf("Hex decoded string: %s", displayInput)
	}

	decoded, err := hex.DecodeString(encodedInput)
	if err != nil {
		logErr := fmt.Errorf("hex decode: invalid input: %v", err)
		logging.LogError(logErr)
		return fmt.Errorf("error decoding hex: %v", err)
	}

	outputErr := OutputHandler(string(decoded), args)
	if outputErr != nil {
		logging.LogError(outputErr)
		return outputErr
	}

	logging.LogCommand(logMsg, 0)
	return nil
}

// URLFunctions contains methods for URL encoding and decoding
type URLFunctions struct{}

// URLEncode encodes a string to URL format
func (u URLFunctions) URLEncode(input string, args []string) error {
	// Ensure the entire input string is processed as one
	encoded := url.QueryEscape(input)

	// Truncate input string in log if too long
	displayInput := input
	if len(input) > 50 {
		displayInput = input[:47] + "..."
	}
	logMsg := fmt.Sprintf("URL encoded string: %s", displayInput)

	outputErr := OutputHandler(encoded, args)
	if outputErr != nil {
		logging.LogError(outputErr)
		return outputErr
	}

	logging.LogCommand(logMsg, 0)
	return nil
}

// URLDecode decodes a URL encoded string
func (u URLFunctions) URLDecode(input string, args []string) error {
	decoded, err := url.QueryUnescape(input)
	if err != nil {
		logErr := fmt.Errorf("URL decode: invalid input: %v", err)
		logging.LogError(logErr)
		return fmt.Errorf("error decoding URL: %v", err)
	}

	// Truncate input string in log if too long
	displayInput := input
	if len(input) > 50 {
		displayInput = input[:47] + "..."
	}
	logMsg := fmt.Sprintf("URL decoded string: %s", displayInput)

	outputErr := OutputHandler(decoded, args)
	if outputErr != nil {
		logging.LogError(outputErr)
		return outputErr
	}

	logging.LogCommand(logMsg, 0)
	return nil
}

// ProcessFile processes a file line by line using the provided encoding/decoding function
func ProcessFile(filePath string, processFunc func(string) (string, error)) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		logging.LogError(fmt.Errorf("process file: error opening %s: %v", filePath, err))
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var result strings.Builder

	for scanner.Scan() {
		processed, err := processFunc(scanner.Text())
		if err != nil {
			logging.LogError(fmt.Errorf("process file: error processing line in %s: %v", filePath, err))
			return "", err
		}
		result.WriteString(processed)
		result.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		logging.LogError(fmt.Errorf("process file: error scanning %s: %v", filePath, err))
		return "", err
	}

	filename := filepath.Base(filePath)
	logging.LogCommand(fmt.Sprintf("Processed file line by line: %s", filename), 0)
	return result.String(), nil
}

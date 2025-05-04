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
	"strconv"
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
	BinaryEncoding // Added binary encoding type
)

// OutputHandler handles redirecting output to a file if specified
func OutputHandler(output string, args []string) error {
	// Handle redirection with quoted filenames
	// First, rebuild args ensuring quoted segments are properly handled
	processedArgs := make([]string, 0, len(args))
	i := 0

	for i < len(args) {
		arg := args[i]

		// Check if this argument starts with a quote but doesn't end with one
		if (strings.HasPrefix(arg, "\"") && !strings.HasSuffix(arg, "\"")) ||
			(strings.HasPrefix(arg, "'") && !strings.HasSuffix(arg, "'")) {
			// Find the closing quote
			startQuote := arg[0]
			endQuoteIdx := i
			quoted := []string{arg}

			for j := i + 1; j < len(args); j++ {
				quoted = append(quoted, args[j])
				if strings.HasSuffix(args[j], string(startQuote)) {
					endQuoteIdx = j
					break
				}
			}

			if endQuoteIdx > i {
				// Join the quoted parts
				joinedArg := strings.Join(quoted, " ")
				// Remove surrounding quotes
				if len(joinedArg) >= 2 {
					joinedArg = joinedArg[1 : len(joinedArg)-1]
				}
				processedArgs = append(processedArgs, joinedArg)
				i = endQuoteIdx + 1
			} else {
				// No matching end quote found, treat as regular arg
				processedArgs = append(processedArgs, removeQuotes(arg))
				i++
			}
		} else {
			processedArgs = append(processedArgs, removeQuotes(arg))
			i++
		}
	}

	// Now check for redirection using processed arguments
	for i, arg := range processedArgs {
		if arg == ">" && i+1 < len(processedArgs) {
			// Write to file specified after >
			outFile := processedArgs[i+1]
			err := os.WriteFile(outFile, []byte(output), 0644)
			if err != nil {
				logging.LogError(fmt.Errorf("failed to write output to file %s: %v", outFile, err))
				return err
			}
			logging.LogCommand(fmt.Sprintf("Wrote output to file: %s", outFile), 0)
			return nil
		} else if arg == "-o" && i+1 < len(processedArgs) {
			// Write to file specified after -o
			outFile := processedArgs[i+1]
			err := os.WriteFile(outFile, []byte(output), 0644)
			if err != nil {
				logging.LogError(fmt.Errorf("failed to write output to file %s: %v", outFile, err))
				return err
			}
			logging.LogCommand(fmt.Sprintf("Wrote output to file: %s", outFile), 0)
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

	// Pre-process args to handle quoted strings
	processedArgs := make([]string, 0, len(cmdArgs))
	i := 0

	for i < len(cmdArgs) {
		arg := cmdArgs[i]

		// Check if this argument starts with a quote but doesn't end with one
		if (strings.HasPrefix(arg, "\"") && !strings.HasSuffix(arg, "\"")) ||
			(strings.HasPrefix(arg, "'") && !strings.HasSuffix(arg, "'")) {
			// Find the closing quote
			startQuote := arg[0]
			endQuoteIdx := i
			quoted := []string{arg}

			for j := i + 1; j < len(cmdArgs); j++ {
				quoted = append(quoted, cmdArgs[j])
				if strings.HasSuffix(cmdArgs[j], string(startQuote)) {
					endQuoteIdx = j
					break
				}
			}

			if endQuoteIdx > i {
				// Join the quoted parts
				joinedArg := strings.Join(quoted, " ")
				processedArgs = append(processedArgs, joinedArg)
				i = endQuoteIdx + 1
			} else {
				// No matching end quote found, treat as regular arg
				processedArgs = append(processedArgs, arg)
				i++
			}
		} else {
			processedArgs = append(processedArgs, arg)
			i++
		}
	}

	// Process flags with the processed arguments
	cmdArgs = processedArgs
	i = 0

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
				// For binary decode, special handling for long binary strings
				if op == DecodeOp && encodingType == BinaryEncoding {
					// For binary decoding, treat input as a string by default
					// Don't try to check if it's a file - this avoids the segfault
					input = removeQuotes(arg)
					isFile = false // explicitly mark as not a file
				} else if op == DecodeOp && (
				// Only for other encoding types: check if this might be a file path
				filepath.Ext(removeQuotes(arg)) != "" &&
					fileExists(removeQuotes(arg))) {
					isFile = true
					input = removeQuotes(arg)
				} else {
					input = removeQuotes(arg)
				}
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

// fileExists checks if a file exists and is not a directory
func fileExists(filename string) bool {
	if filename == "" {
		return false // Handle empty filenames
	}

	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && !info.IsDir()
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

	case BinaryEncoding:
		bin := BinaryFunctions{}
		if op == EncodeOp {
			return bin.BinaryEncode(input, isFile, remainingArgs)
		} else {
			return bin.BinaryDecode(input, isFile, remainingArgs)
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

// BinaryFunctions contains methods for binary encoding and decoding
type BinaryFunctions struct{}

// BinaryEncode encodes a string or file content to binary
func (b BinaryFunctions) BinaryEncode(input string, isFile bool, args []string) error {
	var encodedStr string
	var logMsg string

	if isFile {
		// Read from file
		data, err := os.ReadFile(input)
		if err != nil {
			logErr := fmt.Errorf("binary encode: error reading file %s: %v", input, err)
			logging.LogError(logErr)
			return fmt.Errorf("error reading file: %v", err)
		}
		var binary strings.Builder
		for _, b := range data {
			binary.WriteString(fmt.Sprintf("%08b", b))
		}
		encodedStr = binary.String()
		logMsg = fmt.Sprintf("Binary encoded file: %s (size: %d bytes)", input, len(data))
	} else {
		// Encode the input string
		var binary strings.Builder
		for _, c := range input {
			binary.WriteString(fmt.Sprintf("%08b", c))
		}
		encodedStr = binary.String()
		// Truncate input string in log if too long
		displayInput := input
		if len(input) > 50 {
			displayInput = input[:47] + "..."
		}
		logMsg = fmt.Sprintf("Binary encoded string: %s", displayInput)
	}

	outputErr := OutputHandler(encodedStr, args)
	if outputErr != nil {
		logging.LogError(outputErr)
		return outputErr
	}

	logging.LogCommand(logMsg, 0)
	return nil
}

// BinaryDecode decodes a binary string or file content
func (b BinaryFunctions) BinaryDecode(input string, isFile bool, args []string) error {
	var binaryInput string
	var logMsg string

	if isFile {
		// Read from file
		data, err := os.ReadFile(input)
		if err != nil {
			logErr := fmt.Errorf("binary decode: error reading file %s: %v", input, err)
			logging.LogError(logErr)
			return fmt.Errorf("error reading file: %v", err)
		}
		binaryInput = strings.TrimSpace(string(data))
		logMsg = fmt.Sprintf("Binary decoded file: %s", input)
	} else {
		// Input is treated as a direct binary string
		binaryInput = input

		// Truncate input string in log if too long
		displayInput := input
		if len(input) > 50 {
			displayInput = input[:47] + "..."
		}
		logMsg = fmt.Sprintf("Binary decoded string: %s", displayInput)
	}

	// Clean the binary input by removing all whitespace characters
	binaryInput = strings.ReplaceAll(binaryInput, " ", "")
	binaryInput = strings.ReplaceAll(binaryInput, "\n", "")
	binaryInput = strings.ReplaceAll(binaryInput, "\r", "")
	binaryInput = strings.ReplaceAll(binaryInput, "\t", "")

	// Special handling for very long binary input - process in chunks
	if len(binaryInput) > 1_000_000 { // 1 million characters threshold
		logging.LogCommand("Binary input extremely large, processing in chunks", 1)
	}

	// Verify we have content to decode
	if len(binaryInput) == 0 {
		logErr := fmt.Errorf("binary decode: empty input after cleaning whitespace")
		logging.LogError(logErr)
		return logErr
	}

	// Additional validation for binary data
	for _, c := range binaryInput {
		if c != '0' && c != '1' {
			logErr := fmt.Errorf("binary decode: invalid character in binary input: '%c', only 0 and 1 are allowed", c)
			logging.LogError(logErr)
			return logErr
		}
	}

	// Check if input length is a multiple of 8 (each byte is 8 bits)
	if len(binaryInput)%8 != 0 {
		// Try to pad with zeros if we're close to a multiple of 8
		remainder := len(binaryInput) % 8
		if remainder > 0 {
			padding := strings.Repeat("0", 8-remainder)
			logging.LogCommand(fmt.Sprintf("Binary input length (%d) not a multiple of 8, padding with %d zeros", len(binaryInput), 8-remainder), 1)
			binaryInput = padding + binaryInput
		} else {
			logErr := fmt.Errorf("binary decode: invalid input length, must be multiple of 8 bits")
			logging.LogError(logErr)
			return logErr
		}
	}

	var result strings.Builder
	// Reserve capacity for the result string (capacity = binaryInput length / 8)
	result.Grow(len(binaryInput) / 8)

	// Convert each 8 bits to a byte
	for i := 0; i < len(binaryInput); i += 8 {
		if i+8 > len(binaryInput) {
			break
		}

		byteStr := binaryInput[i : i+8]
		val, err := strconv.ParseUint(byteStr, 2, 8)
		if err != nil {
			logErr := fmt.Errorf("binary decode: invalid binary sequence '%s': %v", byteStr, err)
			logging.LogError(logErr)
			return logErr
		}

		result.WriteByte(byte(val))
	}

	outputErr := OutputHandler(result.String(), args)
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

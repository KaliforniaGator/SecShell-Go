package tools

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"secshell/ui/gui"
	"strings"
)

// HashInput represents what needs to be hashed - either a string or a file
type HashInput struct {
	IsFile      bool
	Content     string
	FilePath    string
	Algorithm   string
	CompareHash string // Field to store hash for comparison
	IsCompare   bool   // Flag to indicate if we're doing a comparison
	OutputFile  string // Field to store output redirection
}

// SupportedHashAlgorithms contains all the hash algorithms we support
var SupportedHashAlgorithms = []string{"md5", "sha1", "sha256", "sha512", "all"}

// Standard hash lengths for different algorithms (in bytes)
var hashLengths = map[int]string{
	32:  "md5",
	40:  "sha1",
	64:  "sha256",
	128: "sha512",
}

// HashCommand processes the hash command with arguments
func HashCommand(args []string) (string, error) {
	if len(args) < 1 {
		return "", errors.New("insufficient arguments - usage: hash -s|-f <String|file> [algo] [-c <hash-to-compare>] [-o <output-file>]")
	}

	// Properly handle quoted arguments and join them back together
	processedArgs := preprocessArgs(args)

	var hashInput HashInput
	var err error

	// Parse the arguments
	hashInput, _, err = parseHashArgs(processedArgs)
	if err != nil {
		return "", err
	}

	var result string
	// If this is a comparison operation, handle it differently
	if hashInput.IsCompare {
		result, err = compareHash(hashInput)
		if err != nil {
			return "", err
		}
	} else {
		// Apply the hash
		result, err = applyHash(hashInput)
		if err != nil {
			return "", err
		}
	}

	// Handle output redirection if specified
	if hashInput.OutputFile != "" {
		err = os.WriteFile(hashInput.OutputFile, []byte(result), 0644)
		if err != nil {
			return "", fmt.Errorf("error writing to output file: %w", err)
		}
		return fmt.Sprintf("Hash result written to %s", hashInput.OutputFile), nil
	}

	return result, nil
}

// detectHashAlgorithm automatically determines the hash algorithm based on hash length
func detectHashAlgorithm(hash string) string {
	// Normalize hash by removing spaces, colons, etc.
	normalizedHash := normalizeHash(hash)
	length := len(normalizedHash)

	// Check if the length matches any known hash algorithm
	if algo, exists := hashLengths[length]; exists {
		return algo
	}

	// Default to SHA256 if we can't determine
	return "sha256"
}

// preprocessArgs handles quoted arguments and consolidates them
func preprocessArgs(args []string) []string {
	var result []string
	inQuotes := false

	// First, join all args into a single string to properly handle quotes
	cmdLine := strings.Join(args, " ")

	// Replace common escape sequences
	cmdLine = strings.ReplaceAll(cmdLine, "\\\"", "\uE000") // Temporary replacement for escaped quotes
	cmdLine = strings.ReplaceAll(cmdLine, "\\ ", "\uE001")  // Temporary replacement for escaped spaces

	// Now parse the command line respecting quotes
	var buffer strings.Builder
	for i := 0; i < len(cmdLine); i++ {
		char := cmdLine[i]

		switch char {
		case '"', '\'':
			if inQuotes {
				// End of quoted section
				inQuotes = false
				result = append(result, buffer.String())
				buffer.Reset()
			} else {
				// Start of quoted section
				inQuotes = true
			}
		case ' ':
			if inQuotes {
				buffer.WriteByte(char)
			} else if buffer.Len() > 0 {
				result = append(result, buffer.String())
				buffer.Reset()
			}
		default:
			buffer.WriteByte(char)
		}
	}

	// Add any remaining content
	if buffer.Len() > 0 {
		result = append(result, buffer.String())
	}

	// Restore escaped characters
	for i, arg := range result {
		arg = strings.ReplaceAll(arg, "\uE000", "\"")
		arg = strings.ReplaceAll(arg, "\uE001", " ")
		result[i] = arg
	}

	return result
}

// parseHashArgs parses the arguments for the hash command
func parseHashArgs(args []string) (HashInput, []string, error) {
	var hashInput HashInput
	var remainingArgs []string

	// Default algorithm
	hashInput.Algorithm = "all"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-s":
			if i+1 >= len(args) {
				return hashInput, nil, errors.New("missing string after -s flag")
			}
			hashInput.IsFile = false
			hashInput.Content = args[i+1]
			i++
		case "-f":
			if i+1 >= len(args) {
				return hashInput, nil, errors.New("missing file path after -f flag")
			}
			hashInput.IsFile = true
			// Resolve any relative paths, expand ~ to home directory
			filePath := args[i+1]
			if strings.HasPrefix(filePath, "~") {
				homeDir, err := os.UserHomeDir()
				if err == nil {
					filePath = filepath.Join(homeDir, filePath[1:])
				}
			}
			hashInput.FilePath = filepath.Clean(filePath)
			i++
		case "-c", "--compare":
			if i+1 >= len(args) {
				return hashInput, nil, errors.New("missing hash value after -c/--compare flag")
			}
			hashInput.IsCompare = true
			hashInput.CompareHash = args[i+1]

			// If no algorithm was explicitly specified, detect it from the hash
			if hashInput.Algorithm == "all" {
				hashInput.Algorithm = detectHashAlgorithm(args[i+1])
			}
			i++
		case "-o", "--output":
			if i+1 >= len(args) {
				return hashInput, nil, errors.New("missing file path after -o/--output flag")
			}
			// Resolve any relative paths for output file
			outputPath := args[i+1]
			if strings.HasPrefix(outputPath, "~") {
				homeDir, err := os.UserHomeDir()
				if err == nil {
					outputPath = filepath.Join(homeDir, outputPath[1:])
				}
			}
			hashInput.OutputFile = filepath.Clean(outputPath)
			i++
		case ">": // Handle direct redirection syntax
			if i+1 >= len(args) {
				return hashInput, nil, errors.New("missing file path after > redirection")
			}
			// Resolve any relative paths for output file
			outputPath := args[i+1]
			if strings.HasPrefix(outputPath, "~") {
				homeDir, err := os.UserHomeDir()
				if err == nil {
					outputPath = filepath.Join(homeDir, outputPath[1:])
				}
			}
			hashInput.OutputFile = filepath.Clean(outputPath)
			i++
		default:
			// Check if this might be an algorithm
			if isValidAlgorithm(args[i]) {
				hashInput.Algorithm = strings.ToLower(args[i])
			} else {
				remainingArgs = append(remainingArgs, args[i])
			}
		}
	}

	// Validation
	if hashInput.IsFile && hashInput.FilePath == "" {
		return hashInput, remainingArgs, errors.New("file path is required with -f flag")
	}

	if !hashInput.IsFile && hashInput.Content == "" {
		return hashInput, remainingArgs, errors.New("string content is required with -s flag")
	}

	return hashInput, remainingArgs, nil
}

// compareHash compares the calculated hash with the provided hash
func compareHash(input HashInput) (string, error) {
	var data []byte
	var err error

	if input.IsFile {
		// Check if file exists
		if _, err := os.Stat(input.FilePath); os.IsNotExist(err) {
			return "", fmt.Errorf("error: file does not exist: %s", input.FilePath)
		}

		data, err = os.ReadFile(input.FilePath)
		if err != nil {
			return "", fmt.Errorf("error reading file: %w", err)
		}
	} else {
		data = []byte(input.Content)
	}

	// If user provided a hash but no algorithm, detect the algorithm from the hash
	if input.Algorithm == "all" {
		input.Algorithm = detectHashAlgorithm(input.CompareHash)
	}

	// Calculate the hash using the specified or detected algorithm
	var calculatedHash string
	switch input.Algorithm {
	case "md5":
		calculatedHash = calculateMD5(data)
	case "sha1":
		calculatedHash = calculateSHA1(data)
	case "sha256":
		calculatedHash = calculateSHA256(data)
	case "sha512":
		calculatedHash = calculateSHA512(data)
	}

	// Normalize hashes for comparison - remove spaces, colons and convert to lowercase
	compareHash := normalizeHash(input.CompareHash)
	calculatedHash = normalizeHash(calculatedHash)

	// Create the result message
	var source string
	if input.IsFile {
		source = "file: " + input.FilePath
	} else {
		source = "string: " + input.Content
	}

	// Check if hashes match
	if compareHash == calculatedHash {
		successMsg := fmt.Sprintf("The %s matches the %s hash:\n%s", source, strings.ToUpper(input.Algorithm), calculatedHash)
		gui.SuccessBox(successMsg)
		return fmt.Sprintf("Hash Match: %s %s hash = %s", source, strings.ToUpper(input.Algorithm), calculatedHash), nil
	} else {
		errorMsg := fmt.Sprintf("The %s does NOT match the provided %s hash.\nExpected: %s\nCalculated: %s",
			source, strings.ToUpper(input.Algorithm), compareHash, calculatedHash)
		gui.ErrorBox(errorMsg)
		return fmt.Sprintf("Hash Mismatch: %s\nExpected %s hash: %s\nCalculated: %s",
			source, strings.ToUpper(input.Algorithm), compareHash, calculatedHash), nil
	}
}

// normalizeHash removes spaces, colons and converts to lowercase for consistent comparison
func normalizeHash(hash string) string {
	// Remove spaces and colons (common in hash formats)
	hash = strings.ReplaceAll(hash, " ", "")
	hash = strings.ReplaceAll(hash, ":", "")
	// Convert to lowercase
	return strings.ToLower(hash)
}

// isValidAlgorithm checks if the provided algorithm is supported
func isValidAlgorithm(algo string) bool {
	lowerAlgo := strings.ToLower(algo)
	for _, supported := range SupportedHashAlgorithms {
		if lowerAlgo == supported {
			return true
		}
	}
	return false
}

// applyHash applies the requested hash algorithm(s) to the input
func applyHash(input HashInput) (string, error) {
	var data []byte
	var err error

	if input.IsFile {
		data, err = os.ReadFile(input.FilePath)
		if err != nil {
			return "", fmt.Errorf("error reading file: %w", err)
		}
	} else {
		data = []byte(input.Content)
	}

	result := ""
	source := "string"
	if input.IsFile {
		source = "file: " + input.FilePath
	}

	if input.Algorithm == "all" || input.Algorithm == "md5" {
		hash := calculateMD5(data)
		result += fmt.Sprintf("MD5 (%s) = %s\n", source, hash)
	}

	if input.Algorithm == "all" || input.Algorithm == "sha1" {
		hash := calculateSHA1(data)
		result += fmt.Sprintf("SHA1 (%s) = %s\n", source, hash)
	}

	if input.Algorithm == "all" || input.Algorithm == "sha256" {
		hash := calculateSHA256(data)
		result += fmt.Sprintf("SHA256 (%s) = %s\n", source, hash)
	}

	if input.Algorithm == "all" || input.Algorithm == "sha512" {
		hash := calculateSHA512(data)
		result += fmt.Sprintf("SHA512 (%s) = %s\n", source, hash)
	}

	return strings.TrimSpace(result), nil
}

// calculateMD5 calculates MD5 hash of data
func calculateMD5(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// calculateSHA1 calculates SHA1 hash of data
func calculateSHA1(data []byte) string {
	hash := sha1.Sum(data)
	return hex.EncodeToString(hash[:])
}

// calculateSHA256 calculates SHA256 hash of data
func calculateSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// calculateSHA512 calculates SHA512 hash of data
func calculateSHA512(data []byte) string {
	hash := sha512.Sum512(data)
	return hex.EncodeToString(hash[:])
}

// HashFileStream calculates hash for potentially large files by streaming
func HashFileStream(filePath, algorithm string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var hash string

	switch strings.ToLower(algorithm) {
	case "md5":
		h := md5.New()
		if _, err := io.Copy(h, file); err != nil {
			return "", err
		}
		hash = hex.EncodeToString(h.Sum(nil))
	case "sha1":
		h := sha1.New()
		if _, err := io.Copy(h, file); err != nil {
			return "", err
		}
		hash = hex.EncodeToString(h.Sum(nil))
	case "sha256":
		h := sha256.New()
		if _, err := io.Copy(h, file); err != nil {
			return "", err
		}
		hash = hex.EncodeToString(h.Sum(nil))
	case "sha512":
		h := sha512.New()
		if _, err := io.Copy(h, file); err != nil {
			return "", err
		}
		hash = hex.EncodeToString(h.Sum(nil))
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}

	return hash, nil
}

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
	"strings"
)

// HashInput represents what needs to be hashed - either a string or a file
type HashInput struct {
	IsFile    bool
	Content   string
	FilePath  string
	Algorithm string
}

// SupportedHashAlgorithms contains all the hash algorithms we support
var SupportedHashAlgorithms = []string{"md5", "sha1", "sha256", "sha512", "all"}

// HashCommand processes the hash command with arguments
func HashCommand(args []string) (string, error) {
	if len(args) < 1 {
		return "", errors.New("insufficient arguments - usage: hash -s|-f <String|file> [algo]")
	}

	var hashInput HashInput
	var err error

	// Parse the arguments
	hashInput, args, err = parseHashArgs(args)
	if err != nil {
		return "", err
	}

	// Apply the hash
	return applyHash(hashInput)
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
			hashInput.FilePath = args[i+1]
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

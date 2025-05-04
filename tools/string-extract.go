package tools

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// StringExtractOptions defines configuration options for the string extraction
type StringExtractOptions struct {
	MinLength  int
	OutputFile string
}

// ExtractStrings extracts printable strings from a binary file
func ExtractStrings(reader io.Reader, options StringExtractOptions) ([]string, error) {
	// Regular expression for printable ASCII characters
	re := regexp.MustCompile(`[[:print:]]+`)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanBytes)

	var buffer []byte
	var results []string

	// Read the file byte by byte
	for scanner.Scan() {
		b := scanner.Bytes()[0]
		// Check if the byte is printable ASCII
		if b >= 32 && b <= 126 {
			buffer = append(buffer, b)
		} else {
			// When we hit a non-printable character, check if the buffer contains a valid string
			if len(buffer) >= options.MinLength {
				str := string(buffer)
				// Use the regex to match valid printable strings
				matches := re.FindAllString(str, -1)
				for _, m := range matches {
					if len(m) >= options.MinLength {
						results = append(results, m)
					}
				}
			}
			buffer = nil
		}
	}

	// Check the buffer one last time after reading the file
	if len(buffer) >= options.MinLength {
		str := string(buffer)
		matches := re.FindAllString(str, -1)
		for _, m := range matches {
			if len(m) >= options.MinLength {
				results = append(results, m)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// RunStringExtract runs the string extraction command with the given arguments
func RunStringExtract(args []string) error {
	// Create a new FlagSet to parse command arguments
	cmdFlags := flag.NewFlagSet("extract-strings", flag.ExitOnError)
	minLength := cmdFlags.Int("n", 4, "Minimum string length to extract")
	outputFile := cmdFlags.String("o", "", "Output file for extracted strings (JSON array)")

	// Parse the command arguments
	if err := cmdFlags.Parse(args); err != nil {
		return err
	}

	// Check if a filename was provided
	if cmdFlags.NArg() < 1 {
		return fmt.Errorf("usage: extract-strings <file> [-n min-len] [-o output.json]")
	}

	filePath := cmdFlags.Arg(0)

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Extract the strings
	options := StringExtractOptions{
		MinLength:  *minLength,
		OutputFile: *outputFile,
	}

	stringsFound, err := ExtractStrings(file, options)
	if err != nil {
		return fmt.Errorf("error extracting strings: %v", err)
	}

	// Marshal the results as a JSON array
	jsonData, err := json.MarshalIndent(stringsFound, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	// Output to file or stdout
	if options.OutputFile != "" {
		out, err := os.Create(options.OutputFile)
		if err != nil {
			return fmt.Errorf("error creating output file: %v", err)
		}
		defer out.Close()
		_, err = out.Write(jsonData)
		if err != nil {
			return fmt.Errorf("error writing to output file: %v", err)
		}
	} else {
		fmt.Println(string(jsonData))
	}

	return nil
}

// StringExtractCmd provides the command interface for string extraction
func StringExtractCmd(cmdStr string) error {
	args := strings.Fields(cmdStr)
	return RunStringExtract(args)
}

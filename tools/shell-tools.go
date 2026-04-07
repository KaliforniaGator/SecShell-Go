package tools

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var htmlCommentPattern = regexp.MustCompile(`(?s)<!--.*?-->`)
var blockCommentPattern = regexp.MustCompile(`(?s)/\*.*?\*/`)
var multiSpacePattern = regexp.MustCompile(`\s+`)
var htmlTagSpacePattern = regexp.MustCompile(`>\s+<`)

func SizeCommand(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("usage: size <-b|-kb|-mb|-gb|-tb|-pb> <path>")
	}

	unit := "-b"
	pathArg := ""

	if strings.HasPrefix(args[0], "-") {
		unit = strings.ToLower(args[0])
		if len(args) < 2 {
			return "", fmt.Errorf("usage: size <-b|-kb|-mb|-gb|-tb|-pb> <path>")
		}
		pathArg = args[1]
	} else {
		pathArg = args[0]
	}

	sizeBytes, err := calculatePathSize(pathArg)
	if err != nil {
		return "", err
	}

	label, value, err := convertSize(sizeBytes, unit)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s: %.4f %s", pathArg, value, label), nil
}

func MetaCommand(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("usage: meta [-r] <file>")
	}

	remove := false
	pathArg := ""

	if args[0] == "-r" {
		remove = true
		if len(args) < 2 {
			return "", fmt.Errorf("usage: meta [-r] <file>")
		}
		pathArg = args[1]
	} else {
		pathArg = args[0]
	}

	info, err := os.Stat(pathArg)
	if err != nil {
		return "", err
	}

	if remove {
		if info.IsDir() {
			return "", fmt.Errorf("meta -r supports files only")
		}
		if err := removeFileMetadata(pathArg); err != nil {
			return "", err
		}
		return fmt.Sprintf("Removed extended metadata from %s", pathArg), nil
	}

	absPath, _ := filepath.Abs(pathArg)
	typeName := "file"
	if info.IsDir() {
		typeName = "directory"
	}

	metadata := []string{
		fmt.Sprintf("Path: %s", absPath),
		fmt.Sprintf("Type: %s", typeName),
		fmt.Sprintf("Size: %d bytes", info.Size()),
		fmt.Sprintf("Permissions: %s", info.Mode().Perm().String()),
		fmt.Sprintf("Mode: %s", info.Mode().String()),
		fmt.Sprintf("Modified: %s", info.ModTime().Format(time.RFC3339)),
	}

	xattrs, err := listExtendedAttributes(pathArg)
	if err == nil && xattrs != "" {
		metadata = append(metadata, fmt.Sprintf("Extended Attributes:\n%s", xattrs))
	}

	return strings.Join(metadata, "\n"), nil
}

func ObfuCommand(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("usage: obfu <text>")
	}
	text := strings.Join(args, " ")
	data := []byte(text)
	for i := range data {
		data[i] ^= 0x5A
	}
	return hex.EncodeToString(data), nil
}

func MiniCommand(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("usage: mini <file>")
	}

	pathArg := args[0]
	original, err := os.ReadFile(pathArg)
	if err != nil {
		return "", err
	}

	minified := minifyByExtension(pathArg, string(original))
	if err := os.WriteFile(pathArg, []byte(minified), 0644); err != nil {
		return "", err
	}

	return fmt.Sprintf("Minified %s (%d -> %d bytes)", pathArg, len(original), len(minified)), nil
}

func calculatePathSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	if !info.IsDir() {
		return info.Size(), nil
	}

	var total int64
	err = filepath.Walk(path, func(_ string, fileInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !fileInfo.IsDir() {
			total += fileInfo.Size()
		}
		return nil
	})

	if err != nil {
		return 0, err
	}
	return total, nil
}

func convertSize(sizeBytes int64, unit string) (string, float64, error) {
	value := float64(sizeBytes)
	switch unit {
	case "-b":
		return "bytes", value, nil
	case "-kb":
		return "kilobytes", value / 1024, nil
	case "-mb":
		return "megabytes", value / (1024 * 1024), nil
	case "-gb":
		return "gigabytes", value / (1024 * 1024 * 1024), nil
	case "-tb":
		return "terabytes", value / (1024 * 1024 * 1024 * 1024), nil
	case "-pb":
		return "petabytes", value / (1024 * 1024 * 1024 * 1024 * 1024), nil
	default:
		return "", 0, fmt.Errorf("invalid measure flag: %s", unit)
	}
}

func removeFileMetadata(path string) error {
	if _, err := exec.LookPath("xattr"); err == nil {
		cmd := exec.Command("xattr", "-c", path)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		return fmt.Errorf("failed to remove metadata (xattr unavailable or command failed)")
	}

	return fmt.Errorf("metadata removal not supported on %s", runtime.GOOS)
}

func listExtendedAttributes(path string) (string, error) {
	if _, err := exec.LookPath("xattr"); err != nil {
		return "", err
	}
	cmd := exec.Command("xattr", "-l", path)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func minifyByExtension(path, content string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html", ".htm":
		content = htmlCommentPattern.ReplaceAllString(content, "")
		content = htmlTagSpacePattern.ReplaceAllString(content, "><")
		content = multiSpacePattern.ReplaceAllString(content, " ")
		return strings.TrimSpace(content)
	case ".css":
		content = blockCommentPattern.ReplaceAllString(content, "")
		return compressTokens(content)
	case ".js":
		return minifyJSConservative(content)
	default:
		return strings.TrimSpace(multiSpacePattern.ReplaceAllString(content, " "))
	}
}

func minifyJSConservative(content string) string {
	var out strings.Builder
	inSingle := false
	inDouble := false
	inTemplate := false
	inLineComment := false
	inBlockComment := false
	escaped := false
	pendingSpace := false
	lastWritten := byte(0)

	for i := 0; i < len(content); i++ {
		ch := content[i]
		next := byte(0)
		if i+1 < len(content) {
			next = content[i+1]
		}

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
				pendingSpace = true
			}
			continue
		}

		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				i++
				pendingSpace = true
			}
			continue
		}

		if !inSingle && !inDouble && !inTemplate {
			if ch == '/' && next == '/' {
				inLineComment = true
				i++
				continue
			}
			if ch == '/' && next == '*' {
				inBlockComment = true
				i++
				continue
			}
		}

		if !inSingle && !inDouble && !inTemplate && (ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t') {
			pendingSpace = true
			continue
		}

		if pendingSpace {
			if shouldInsertSpace(lastWritten, ch) {
				out.WriteByte(' ')
				lastWritten = ' '
			}
			pendingSpace = false
		}

		out.WriteByte(ch)
		lastWritten = ch

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && (inSingle || inDouble || inTemplate) {
			escaped = true
			continue
		}

		if ch == '\'' && !inDouble && !inTemplate {
			inSingle = !inSingle
		} else if ch == '"' && !inSingle && !inTemplate {
			inDouble = !inDouble
		} else if ch == '`' && !inSingle && !inDouble {
			inTemplate = !inTemplate
		}
	}

	return strings.TrimSpace(out.String())
}

func shouldInsertSpace(prev, curr byte) bool {
	if prev == 0 || prev == ' ' {
		return false
	}
	return isIdentifierByte(prev) && isIdentifierByte(curr)
}

func isIdentifierByte(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_' || ch == '$'
}

func compressTokens(content string) string {
	collapsed := multiSpacePattern.ReplaceAllString(content, " ")
	replacer := strings.NewReplacer(
		" {", "{", "{ ", "{",
		" }", "}", "} ", "}",
		" ;", ";", "; ", ";",
		" :", ":", ": ", ":",
		" ,", ",", ", ", ",",
		" (", "(", " )", ")",
	)
	return strings.TrimSpace(replacer.Replace(collapsed))
}

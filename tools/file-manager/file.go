package filemanager

import (
	"io"
	"os"
)

// CreateFile creates a new empty file at the specified path.
// If the file already exists, it will be truncated.
func CreateFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	return file.Close()
}

// CopyFile copies a file from srcPath to dstPath.
func CopyFile(srcPath, dstPath string) error {
	sourceFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

// MoveFile moves a file from srcPath to dstPath.
// It first attempts os.Rename. If that fails (e.g., cross-device link),
// it falls back to copying the file and then deleting the source.
func MoveFile(srcPath, dstPath string) error {
	err := os.Rename(srcPath, dstPath)
	if err == nil {
		return nil
	}

	// If os.Rename fails, check if it's a cross-device link error
	// For example, on POSIX systems, this might be syscall.EXDEV
	// os.LinkError is a common way this manifests in Go.
	if _, ok := err.(*os.LinkError); ok {
		// Fallback to copy and delete
		// Note: More specific error checking for cross-device link (e.g., linkErr.Err == syscall.EXDEV on POSIX)
		// might be needed for perfect robustness, but this is a common fallback.
		if errCopy := CopyFile(srcPath, dstPath); errCopy != nil {
			return errCopy // Return copy error
		}
		if errDelete := DeleteFile(srcPath); errDelete != nil {
			// If delete fails, the file was copied but the original still exists.
			// This is a partial success/failure state. The caller might need to handle this.
			// For now, we return the delete error.
			return errDelete
		}
		return nil // Successfully copied and deleted
	}

	return err // Return original os.Rename error if it wasn't a LinkError we can handle by copy-delete
}

// DeleteFile deletes the file at the specified path.
func DeleteFile(path string) error {
	return os.Remove(path)
}

// GetFileInfo retrieves information about the file at the specified path.
func GetFileInfo(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// FileExists checks if a file or directory exists at the given path.
// It returns true if the path exists and is accessible, false otherwise.
// An error is returned for issues other than the path not existing.
func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil // Path exists
	}
	if os.IsNotExist(err) {
		return false, nil // Path does not exist
	}
	return false, err // Other error (e.g., permission denied)
}

// ReadFile reads the entire content of a file into a byte slice.
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm;
// otherwise WriteFile truncates it before writing.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// ChangePermissions changes the mode of the named file to mode.
func ChangePermissions(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}

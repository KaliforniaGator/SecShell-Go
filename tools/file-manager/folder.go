package filemanager

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CreateFolder creates a new folder at the specified path.
// It also creates any necessary parent directories.
func CreateFolder(path string) error {
	// 0755 provides read, write, execute for owner, and read, execute for group and others.
	return os.MkdirAll(path, 0755)
}

// CopyFolder recursively copies a folder from srcPath to destPath.
// Note: This is a simplified version. For a robust solution, consider file attributes,
// symbolic links, and more comprehensive error handling.
func CopyFolder(srcPath, destPath string) error {
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("source %s is not a directory", srcPath)
	}

	if err = os.MkdirAll(destPath, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcItemPath := filepath.Join(srcPath, entry.Name())
		destItemPath := filepath.Join(destPath, entry.Name())

		if entry.IsDir() {
			if err = CopyFolder(srcItemPath, destItemPath); err != nil {
				return err
			}
		} else {
			// For file copying, a separate function like CopyFile would be used.
			// This is a placeholder for file copy logic.
			if err = copyFile(srcItemPath, destItemPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyFile is a helper function to copy a single file.
// This is a basic implementation.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// MoveFolder moves a folder from srcPath to destPath.
// It first attempts os.Rename. If that fails (e.g., due to cross-device link),
// it falls back to copying the folder and then deleting the original.
func MoveFolder(srcPath, destPath string) error {
	// Attempt to rename (move) the folder directly.
	err := os.Rename(srcPath, destPath)
	if err == nil {
		// Rename was successful.
		return nil
	}

	// If os.Rename failed, it might be a cross-device move or other issue.
	// Fall back to copy and then delete.
	// Note: This specific error check for *os.LinkError could be made more granular
	// (e.g., checking for syscall.EXDEV on POSIX systems), but a general fallback
	// on any os.Rename error is often a practical approach for robustness.
	if linkErr, ok := err.(*os.LinkError); ok {
		// It's a LinkError, common for cross-device moves.
		// Proceed with copy-then-delete.
		fmt.Printf("os.Rename failed (%s), attempting copy-then-delete for %s to %s\n", linkErr, srcPath, destPath)
	} else {
		// For other errors, we might still want to try copy-then-delete,
		// or we could return the original error. For maximum robustness in moving,
		// we'll try copy-then-delete.
		fmt.Printf("os.Rename failed (%s), attempting copy-then-delete for %s to %s\n", err, srcPath, destPath)
	}

	if copyErr := CopyFolder(srcPath, destPath); copyErr != nil {
		// If copy fails, return the copy error. The original srcPath is still intact.
		return fmt.Errorf("failed to copy folder during move: %w (original rename error: %s)", copyErr, err)
	}

	// Copy was successful, now delete the original folder.
	if deleteErr := DeleteFolder(srcPath); deleteErr != nil {
		// If delete fails, the folder has been copied, but the original remains.
		// This is a partial success/failure state. The user should be informed.
		return fmt.Errorf("failed to delete original folder after copying: %w (original rename error: %s)", deleteErr, err)
	}

	// Move (copy + delete) was successful.
	return nil
}

// DeleteFolder removes a folder and all its contents.
func DeleteFolder(path string) error {
	return os.RemoveAll(path)
}

// GetFolderInfo retrieves information about a folder.
func GetFolderInfo(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", path)
	}
	return info, nil
}

// FolderExists checks if a folder exists at the given path.
func FolderExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && info.IsDir()
}

// ChangeFolderPermissions changes the permissions of a folder.
func ChangeFolderPermissions(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}

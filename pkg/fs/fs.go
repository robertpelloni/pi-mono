package fs

import (
	"os"
)

// Exists reports whether the given path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir reports whether the given path is a directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ReadFileSafe reads a file and returns its contents. It returns an error if the file does not exist.
func ReadFileSafe(path string) ([]byte, error) {
	return os.ReadFile(path)
}

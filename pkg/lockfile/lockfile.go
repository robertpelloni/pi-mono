package lockfile

import (
	"fmt"
	"os"
)

// FileLock represents a simple file‑based lock using atomic file creation.
type FileLock struct {
	path string
	file *os.File
}

// New creates a lock for the given path. It does not acquire the lock yet.
func New(path string) *FileLock { return &FileLock{path: path} }

// Acquire obtains an exclusive lock by atomically creating the lock file.
func (l *FileLock) Acquire() error {
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	l.file = f
	return nil
}

// Release closes and removes the lock file.
func (l *FileLock) Release() error {
	if l.file == nil {
		return nil
	}
	if err := l.file.Close(); err != nil {
		return err
	}
	if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	l.file = nil
	return nil
}

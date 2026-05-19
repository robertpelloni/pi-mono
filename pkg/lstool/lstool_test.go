package lstool

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type mockLsOps struct {
	existsFn      func(path string) bool
	isDirectoryFn func(path string) (bool, error)
	readDirFn     func(path string) ([]string, error)
	statFn        func(path string) (os.FileInfo, error)
}

func (m *mockLsOps) Exists(path string) bool {
	if m.existsFn != nil {
		return m.existsFn(path)
	}
	return false
}

func (m *mockLsOps) IsDirectory(path string) (bool, error) {
	if m.isDirectoryFn != nil {
		return m.isDirectoryFn(path)
	}
	return false, nil
}

func (m *mockLsOps) ReadDir(path string) ([]string, error) {
	if m.readDirFn != nil {
		return m.readDirFn(path)
	}
	return nil, nil
}

func (m *mockLsOps) Stat(path string) (os.FileInfo, error) {
	if m.statFn != nil {
		return m.statFn(path)
	}
	return nil, os.ErrNotExist
}

func TestLsToolInput_Fields(t *testing.T) {
	input := LsToolInput{Path: "/tmp", Limit: 50}
	if input.Path != "/tmp" || input.Limit != 50 {
		t.Error("Field mismatch")
	}
}

func TestExecute_Directory(t *testing.T) {
	dir, _ := os.MkdirTemp("", "ls_test")
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("b"), 0644)

	ops := &mockLsOps{
		existsFn: func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		},
		isDirectoryFn: func(path string) (bool, error) {
			info, err := os.Stat(path)
			if err != nil {
				return false, err
			}
			return info.IsDir(), nil
		},
		readDirFn: func(path string) ([]string, error) {
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}
			names := make([]string, len(entries))
			for i, e := range entries {
				names[i] = e.Name()
			}
			return names, nil
		},
	}

	result, err := Execute(context.Background(), LsToolInput{
		Path: dir,
	}, dir, ops)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestExecute_NonExistentPath(t *testing.T) {
	ops := &mockLsOps{
		existsFn: func(path string) bool { return false },
	}

	result, err := Execute(context.Background(), LsToolInput{
		Path: "/nonexistent",
	}, ".", ops)

	// Should handle gracefully
	_ = result
	_ = err
}

package projects

import (
	"os"
	"path/filepath"
)

// Project represents a discovered project root with optional config.
type Project struct {
	Root   string
	Config map[string]string
}

// Detect attempts to locate the project root starting from dir.
// It looks for a go.mod or .git directory.
func Detect(dir string) (*Project, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	cur := abs
	for {
		if exists(filepath.Join(cur, "go.mod")) || exists(filepath.Join(cur, ".git")) {
			return &Project{Root: cur, Config: make(map[string]string)}, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return nil, os.ErrNotExist
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

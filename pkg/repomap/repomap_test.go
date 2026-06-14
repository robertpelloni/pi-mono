package repomap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	// Create a dummy repo structure
	tmpDir, err := os.MkdirTemp("", "repomap_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	files := map[string]string{
		"main.go": "package main\n\nfunc main() {\n\thelloutils.PrintHello()\n}",
		"utils/hello.go": "package helloutils\n\nimport \"fmt\"\n\nfunc PrintHello() {\n\tfmt.Println(\"Hello\")\n}",
		"README.md": "# Test Repo",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	opts := Options{
		BaseDir: tmpDir,
	}

	result, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(result.Map, "main.go") {
		t.Errorf("Expected Map to contain main.go")
	}
	if !strings.Contains(result.Map, "PrintHello") {
		t.Errorf("Expected Map to contain PrintHello")
	}
}

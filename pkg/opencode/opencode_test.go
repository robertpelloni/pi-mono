package opencode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyPatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("line 1\nline 2\nline 3\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	patch := `*** Update File: test.txt
line 1
-line 2
+line 2 modified
line 3
`
	hunks, err := ParsePatch(patch)
	if err != nil {
		t.Fatalf("ParsePatch failed: %v", err)
	}

	result := ApplyPatch(hunks, tmpDir)
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file result, got %d", len(result.Files))
	}
	if result.Files[0].Err != nil {
		t.Errorf("ApplyPatch error: %v", result.Files[0].Err)
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "line 1\nline 2 modified\nline 3\n" {
		t.Errorf("unexpected content: %s", string(content))
	}
}

func TestApplyMultiEdit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "multiedit-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.go")
	err = os.WriteFile(testFile, []byte("package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	params := MultiEditParams{
		FilePath: testFile,
		Edits: []MultiEditItem{
			{OldString: "hello", NewString: "world"},
			{OldString: "Println", NewString: "Printf"},
		},
	}

	res, err := ApplyMultiEdit(params)
	if err != nil {
		t.Fatalf("ApplyMultiEdit failed: %v", err)
	}

	if res.Additions == 0 {
		t.Errorf("expected additions > 0")
	}

	content, _ := os.ReadFile(testFile)
	expected := "package main\n\nfunc main() {\n\tfmt.Printf(\"world\")\n}\n"
	if string(content) != expected {
		t.Errorf("unexpected content: %s", string(content))
	}
}

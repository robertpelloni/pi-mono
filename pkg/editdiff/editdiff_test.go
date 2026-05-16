package editdiff

import (
	"strings"
	"testing"
)

func TestDetectLineEnding(t *testing.T) {
	if got := DetectLineEnding("hello\nworld"); got != "\n" {
		t.Errorf("Expected \\n, got %q", got)
	}
	if got := DetectLineEnding("hello\r\nworld"); got != "\r\n" {
		t.Errorf("Expected \\r\\n, got %q", got)
	}
	if got := DetectLineEnding("hello"); got != "\n" {
		t.Errorf("Expected \\n for no newlines, got %q", got)
	}
}

func TestNormalizeToLF(t *testing.T) {
	input := "line1\r\nline2\r\nline3"
	expected := "line1\nline2\nline3"
	if got := NormalizeToLF(input); got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}

	// Mixed endings
	input2 := "line1\r\nline2\rline3\n"
	if got := NormalizeToLF(input2); strings.Contains(got, "\r") {
		t.Errorf("Expected no CR, got %q", got)
	}
}

func TestRestoreLineEndings(t *testing.T) {
	input := "line1\nline2\nline3"
	if got := RestoreLineEndings(input, "\r\n"); !strings.Contains(got, "\r\n") {
		t.Errorf("Expected CRLF, got %q", got)
	}
	if got := RestoreLineEndings(input, "\n"); got != input {
		t.Errorf("Expected unchanged, got %q", got)
	}
}

func TestStripBom(t *testing.T) {
	bom, text := StripBom("\uFEFFhello")
	if bom != "\uFEFF" {
		t.Errorf("Expected BOM, got %q", bom)
	}
	if text != "hello" {
		t.Errorf("Expected 'hello', got %q", text)
	}

	bom2, text2 := StripBom("hello")
	if bom2 != "" {
		t.Errorf("Expected no BOM, got %q", bom2)
	}
	if text2 != "hello" {
		t.Errorf("Expected 'hello', got %q", text2)
	}
}

func TestNormalizeForFuzzyMatch(t *testing.T) {
	// Trailing whitespace stripped
	input := "hello  \nworld \t\n"
	got := NormalizeForFuzzyMatch(input)
	if strings.Contains(got, "  \n") || strings.Contains(got, " \t\n") {
		t.Errorf("Trailing whitespace not stripped: %q", got)
	}

	// Smart quotes
	got2 := NormalizeForFuzzyMatch("\u2018hello\u2019")
	if got2 != "'hello'" {
		t.Errorf("Expected single quotes, got %q", got2)
	}

	// Em dash
	got3 := NormalizeForFuzzyMatch("hello\u2014world")
	if got3 != "hello-world" {
		t.Errorf("Expected dash, got %q", got3)
	}
}

func TestFuzzyFindText(t *testing.T) {
	content := "hello world this is a test"

	// Exact match
	result := FuzzyFindText(content, "world")
	if !result.Found {
		t.Error("Expected to find 'world'")
	}
	if result.UsedFuzzyMatch {
		t.Error("Expected exact match, not fuzzy")
	}
	if result.Index != 6 {
		t.Errorf("Expected index 6, got %d", result.Index)
	}

	// Not found
	result2 := FuzzyFindText(content, "xyz")
	if result2.Found {
		t.Error("Expected not to find 'xyz'")
	}

	// Fuzzy match with trailing whitespace
	content3 := "hello world  \nthis is a test"
	result3 := FuzzyFindText(content3, "world\nthis")
	if !result3.Found {
		t.Error("Expected fuzzy match for 'world\\nthis'")
	}
	if !result3.UsedFuzzyMatch {
		t.Error("Expected fuzzy match to be used")
	}
}

func TestApplyEditsToNormalizedContent(t *testing.T) {
	content := "line1\nline2\nline3\nline4"

	// Single edit
	result, err := ApplyEditsToNormalizedContent(content, []Edit{
		{OldText: "line2", NewText: "modified"},
	}, "test.txt")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(result.NewContent, "modified") {
		t.Errorf("Expected 'modified' in result, got %q", result.NewContent)
	}
	if strings.Contains(result.NewContent, "line2") {
		t.Errorf("Expected 'line2' to be replaced, got %q", result.NewContent)
	}

	// Multiple edits
	result2, err := ApplyEditsToNormalizedContent(content, []Edit{
		{OldText: "line1", NewText: "first"},
		{OldText: "line4", NewText: "last"},
	}, "test.txt")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(result2.NewContent, "first") || !strings.Contains(result2.NewContent, "last") {
		t.Errorf("Expected both edits applied, got %q", result2.NewContent)
	}

	// Empty oldText
	_, err = ApplyEditsToNormalizedContent(content, []Edit{
		{OldText: "", NewText: "x"},
	}, "test.txt")
	if err == nil {
		t.Error("Expected error for empty oldText")
	}

	// Not found
	_, err = ApplyEditsToNormalizedContent(content, []Edit{
		{OldText: "nonexistent", NewText: "x"},
	}, "test.txt")
	if err == nil {
		t.Error("Expected error for not found text")
	}

	// Duplicate occurrences
	_, err = ApplyEditsToNormalizedContent("abc abc abc", []Edit{
		{OldText: "abc", NewText: "x"},
	}, "test.txt")
	if err == nil {
		t.Error("Expected error for duplicate occurrences")
	}
}

func TestGenerateDiffString(t *testing.T) {
	oldContent := "line1\nline2\nline3\nline4\nline5"
	newContent := "line1\nmodified\nline3\nline4\nline5"

	result := GenerateDiffString(oldContent, newContent)
	if result.Diff == "" {
		t.Error("Expected non-empty diff")
	}
	if result.FirstChangedLine == 0 {
		t.Error("Expected first changed line to be set")
	}
	if !strings.Contains(result.Diff, "+") || !strings.Contains(result.Diff, "-") {
		t.Errorf("Expected diff to contain + and - markers, got %q", result.Diff)
	}
}

func TestCountOccurrences(t *testing.T) {
	if got := CountOccurrences("abc abc abc", "abc"); got != 3 {
		t.Errorf("Expected 3, got %d", got)
	}
	if got := CountOccurrences("hello world", "xyz"); got != 0 {
		t.Errorf("Expected 0, got %d", got)
	}
}

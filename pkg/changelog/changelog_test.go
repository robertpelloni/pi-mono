package changelog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseChangelog_NonExistent(t *testing.T) {
	entries := ParseChangelog("/nonexistent/CHANGELOG.md")
	if entries != nil {
		t.Error("Expected nil for non-existent file")
	}
}

func TestParseChangelog_Valid(t *testing.T) {
	content := `# Changelog

## [1.2.0] - 2024-01-15
### Added
- New feature X

## [1.1.0] - 2024-01-10
### Fixed
- Bug fix Y

## [1.0.0] - 2024-01-01
### Added
- Initial release
`
	dir, _ := os.MkdirTemp("", "changelog_test")
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "CHANGELOG.md")
	os.WriteFile(path, []byte(content), 0644)

	entries := ParseChangelog(path)
	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(entries))
	}
	if entries[0].Major != 1 || entries[0].Minor != 2 || entries[0].Patch != 0 {
		t.Errorf("Expected 1.2.0, got %d.%d.%d", entries[0].Major, entries[0].Minor, entries[0].Patch)
	}
	if entries[1].Major != 1 || entries[1].Minor != 1 || entries[1].Patch != 0 {
		t.Errorf("Expected 1.1.0, got %d.%d.%d", entries[1].Major, entries[1].Minor, entries[1].Patch)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1, v2   ChangelogEntry
		expected int
	}{
		{ChangelogEntry{1, 0, 0, ""}, ChangelogEntry{1, 0, 0, ""}, 0},
		{ChangelogEntry{1, 1, 0, ""}, ChangelogEntry{1, 0, 0, ""}, 1},
		{ChangelogEntry{1, 0, 0, ""}, ChangelogEntry{1, 1, 0, ""}, -1},
		{ChangelogEntry{2, 0, 0, ""}, ChangelogEntry{1, 9, 9, ""}, 1},
		{ChangelogEntry{1, 0, 1, ""}, ChangelogEntry{1, 0, 0, ""}, 1},
		{ChangelogEntry{1, 0, 0, ""}, ChangelogEntry{2, 0, 0, ""}, -1},
	}

	for _, tt := range tests {
		result := CompareVersions(tt.v1, tt.v2)
		if result != tt.expected {
			t.Errorf("CompareVersions(%d.%d.%d, %d.%d.%d) = %d, want %d",
				tt.v1.Major, tt.v1.Minor, tt.v1.Patch,
				tt.v2.Major, tt.v2.Minor, tt.v2.Patch,
				result, tt.expected)
		}
	}
}

func TestGetNewEntries(t *testing.T) {
	entries := []ChangelogEntry{
		{1, 2, 0, "v1.2.0 content"},
		{1, 1, 0, "v1.1.0 content"},
		{1, 0, 0, "v1.0.0 content"},
	}

	newEntries := GetNewEntries(entries, "1.0.0")
	if len(newEntries) != 2 {
		t.Fatalf("Expected 2 new entries, got %d", len(newEntries))
	}
	if newEntries[0].Minor != 2 {
		t.Errorf("Expected first entry 1.2.0, got %d.%d.%d", newEntries[0].Major, newEntries[0].Minor, newEntries[0].Patch)
	}
}

func TestGetNewEntries_None(t *testing.T) {
	entries := []ChangelogEntry{
		{1, 0, 0, "v1.0.0 content"},
	}
	newEntries := GetNewEntries(entries, "1.0.0")
	if len(newEntries) != 0 {
		t.Errorf("Expected 0 new entries, got %d", len(newEntries))
	}
}

func TestGetNewEntries_AllNew(t *testing.T) {
	entries := []ChangelogEntry{
		{1, 0, 0, "content"},
	}
	newEntries := GetNewEntries(entries, "0.0.0")
	if len(newEntries) != 1 {
		t.Errorf("Expected 1 new entry, got %d", len(newEntries))
	}
}

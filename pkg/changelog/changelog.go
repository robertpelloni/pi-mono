package changelog

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ChangelogEntry represents a parsed version entry from a changelog.
type ChangelogEntry struct {
	Major   int    `json:"major"`
	Minor   int    `json:"minor"`
	Patch   int    `json:"patch"`
	Content string `json:"content"`
}

var versionHeaderRegex = regexp.MustCompile(`^##\s+\[?(\d+)\.(\d+)\.(\d+)\]?`)

// ParseChangelog reads and parses a CHANGELOG.md file.
func ParseChangelog(changelogPath string) []ChangelogEntry {
	if _, err := os.Stat(changelogPath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(changelogPath)
	if err != nil {
		fmt.Printf("Warning: Could not parse changelog: %v\n", err)
		return nil
	}

	lines := strings.Split(string(data), "\n")
	var entries []ChangelogEntry
	var currentLines []string
	var currentVersion *ChangelogEntry

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// Save previous entry
			if currentVersion != nil && len(currentLines) > 0 {
				currentVersion.Content = strings.TrimSpace(strings.Join(currentLines, "\n"))
				entries = append(entries, *currentVersion)
			}

			matches := versionHeaderRegex.FindStringSubmatch(line)
			if len(matches) >= 4 {
				major, _ := strconv.Atoi(matches[1])
				minor, _ := strconv.Atoi(matches[2])
				patch, _ := strconv.Atoi(matches[3])
				currentVersion = &ChangelogEntry{Major: major, Minor: minor, Patch: patch}
				currentLines = []string{line}
			} else {
				currentVersion = nil
				currentLines = nil
			}
		} else if currentVersion != nil {
			currentLines = append(currentLines, line)
		}
	}

	// Save last entry
	if currentVersion != nil && len(currentLines) > 0 {
		currentVersion.Content = strings.TrimSpace(strings.Join(currentLines, "\n"))
		entries = append(entries, *currentVersion)
	}

	return entries
}

// CompareVersions compares two changelog entries.
// Returns -1 if v1 < v2, 0 if equal, 1 if v1 > v2.
func CompareVersions(v1, v2 ChangelogEntry) int {
	if v1.Major != v2.Major {
		if v1.Major < v2.Major {
			return -1
		}
		return 1
	}
	if v1.Minor != v2.Minor {
		if v1.Minor < v2.Minor {
			return -1
		}
		return 1
	}
	if v1.Patch != v2.Patch {
		if v1.Patch < v2.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// GetNewEntries returns entries newer than lastVersion.
func GetNewEntries(entries []ChangelogEntry, lastVersion string) []ChangelogEntry {
	parts := strings.Split(lastVersion, ".")
	last := ChangelogEntry{}
	if len(parts) > 0 {
		last.Major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		last.Minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) > 2 {
		last.Patch, _ = strconv.Atoi(parts[2])
	}

	var result []ChangelogEntry
	for _, entry := range entries {
		if CompareVersions(entry, last) > 0 {
			result = append(result, entry)
		}
	}
	return result
}

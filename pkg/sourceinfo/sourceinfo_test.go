package sourceinfo

import (
	"testing"
)

func TestCreateSourceInfo(t *testing.T) {
	si := CreateSourceInfo("/tmp/test.txt", "project")
	if si.Path != "/tmp/test.txt" {
		t.Errorf("Expected path /tmp/test.txt, got %s", si.Path)
	}
	if si.Source != "project" {
		t.Errorf("Expected source 'project', got %s", si.Source)
	}
}

func TestSourceInfo_Label(t *testing.T) {
	tests := []struct {
		name     string
		si       SourceInfo
		expected string
	}{
		{
			name:     "no extension",
			si:       SourceInfo{Path: "/tmp/test.txt", Source: "global"},
			expected: "global:/tmp/test.txt",
		},
		{
			name:     "with extension",
			si:       SourceInfo{Path: "/tmp/test.txt", Source: "extension", Extension: "my-ext"},
			expected: "my-ext:/tmp/test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.si.Label(); result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

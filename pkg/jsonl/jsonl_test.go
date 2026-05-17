package jsonl

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestSerializeJsonLine(t *testing.T) {
	data := map[string]string{"key": "value"}
	line := SerializeJsonLine(data)

	// Should end with newline
	if line[len(line)-1] != '\n' {
		t.Error("Expected newline at end")
	}

	// Should be valid JSON (without the newline)
	var parsed map[string]string
	if err := json.Unmarshal([]byte(line[:len(line)-1]), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("Expected value, got %s", parsed["key"])
	}
}

func TestJsonlWriterReader(t *testing.T) {
	var buf bytes.Buffer
	writer := NewJsonlWriter(&buf)

	// Write some records
	records := []map[string]string{
		{"type": "user", "text": "hello"},
		{"type": "assistant", "text": "world"},
		{"type": "tool", "name": "read"},
	}

	for _, r := range records {
		if err := writer.Write(r); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	// Read back
	reader := NewJsonlReader(&buf)
	var results []map[string]string

	for {
		var record map[string]string
		err := reader.Read(&record)
		if err != nil {
			break
		}
		results = append(results, record)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 records, got %d", len(results))
	}

	if results[0]["type"] != "user" {
		t.Errorf("Expected first record type=user, got %s", results[0]["type"])
	}
	if results[2]["name"] != "read" {
		t.Errorf("Expected third record name=read, got %s", results[2]["name"])
	}
}

func TestJsonlReader_Empty(t *testing.T) {
	buf := bytes.Buffer{}
	reader := NewJsonlReader(&buf)

	var record map[string]string
	err := reader.Read(&record)
	if err == nil {
		t.Error("Expected error on empty input")
	}
}

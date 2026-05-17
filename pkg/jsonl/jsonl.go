package jsonl

import (
	"bufio"
	"encoding/json"
	"io"
)

// SerializeJsonLine serializes a value as a single strict JSONL record.
// Framing is LF-only. Clients must split records on \n only.
func SerializeJsonLine(value interface{}) string {
	data, _ := json.Marshal(value)
	return string(data) + "\n"
}

// JsonlReader reads JSONL records from a stream.
type JsonlReader struct {
	scanner *bufio.Scanner
}

// NewJsonlReader creates a new JSONL reader from a stream.
func NewJsonlReader(r io.Reader) *JsonlReader {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max line
	return &JsonlReader{scanner: scanner}
}

// Read reads the next JSONL record and unmarshals it.
// Returns io.EOF when there are no more records.
func (j *JsonlReader) Read(v interface{}) error {
	if !j.scanner.Scan() {
		if err := j.scanner.Err(); err != nil {
			return err
		}
		return io.EOF
	}

	line := j.scanner.Text()
	// Strip trailing \r if present (Windows line endings)
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	return json.Unmarshal([]byte(line), v)
}

// JsonlWriter writes JSONL records to a stream.
type JsonlWriter struct {
	writer io.Writer
}

// NewJsonlWriter creates a new JSONL writer.
func NewJsonlWriter(w io.Writer) *JsonlWriter {
	return &JsonlWriter{writer: w}
}

// Write serializes and writes a value as a JSONL record.
func (j *JsonlWriter) Write(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = j.writer.Write(append(data, '\n'))
	return err
}

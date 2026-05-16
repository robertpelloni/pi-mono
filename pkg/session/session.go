package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/badlogic/pi-mono/pkg/ai"
)

// Session stores a conversation history to disk for persistence and replay.
type Session struct {
	ID      string
	CWD     string
	File    string
	Created time.Time
	Updated time.Time

	mu       sync.RWMutex
	messages []ai.Message
	dirty    bool
}

// SessionInfo is a lightweight summary used for listing sessions.
type SessionInfo struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	CWD       string    `json:"cwd"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
	MessageCount int    `json:"messageCount"`
}

// NewSession creates a new session with a unique ID.
func NewSession(cwd, sessionDir string) *Session {
	id := generateSessionID()
	if sessionDir == "" {
		sessionDir = defaultSessionDir(cwd)
	}
	os.MkdirAll(sessionDir, 0755)

	file := filepath.Join(sessionDir, id+".jsonl")
	now := time.Now()
	return &Session{
		ID:       id,
		CWD:      cwd,
		File:     file,
		Created:  now,
		Updated:  now,
		messages: []ai.Message{},
		dirty:    false,
	}
}

// OpenSession loads an existing session from a JSONL file.
func OpenSession(path, cwd string) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open session: %w", err)
	}
	defer f.Close()

	s := &Session{
		ID:      strings.TrimSuffix(filepath.Base(path), ".jsonl"),
		File:    path,
		CWD:     cwd,
		Created: time.Now(),
		Updated: time.Now(),
	}

	dec := json.NewDecoder(f)
	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			break
		}
		msg, err := unmarshalMessage(raw)
		if err != nil {
			continue
		}
		s.messages = append(s.messages, msg)
	}

	if len(s.messages) > 0 {
		s.Updated = time.Now()
	}

	return s, nil
}

// InMemorySession creates a session that is never persisted to disk.
func InMemorySession() *Session {
	return &Session{
		ID:       generateSessionID(),
		CWD:      "",
		File:     "",
		messages: []ai.Message{},
	}
}

// AppendMessage adds a message to the session and persists it if the session has a file.
func (s *Session) AppendMessage(msg ai.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, msg)
	s.Updated = time.Now()
	s.dirty = true

	if s.File == "" {
		return nil // in-memory session
	}

	// Append to JSONL file
	f, err := os.OpenFile(s.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("append session: %w", err)
	}
	defer f.Close()

	data, err := marshalMessage(msg)
	if err != nil {
		return fmt.Errorf("marshal session message: %w", err)
	}
	data = append(data, '\n')
	_, err = f.Write(data)
	return err
}

// Messages returns a copy of the session's messages.
func (s *Session) Messages() []ai.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := make([]ai.Message, len(s.messages))
	copy(msgs, s.messages)
	return msgs
}

// SetMessages replaces the session's messages.
func (s *Session) SetMessages(msgs []ai.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = msgs
	s.dirty = true
}

// Info returns a SessionInfo summary.
func (s *Session) Info() SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return SessionInfo{
		ID:           s.ID,
		Path:         s.File,
		CWD:          s.CWD,
		Created:      s.Created,
		Updated:      s.Updated,
		MessageCount: len(s.messages),
	}
}

// IsInMemory returns true if this session has no backing file.
func (s *Session) IsInMemory() bool {
	return s.File == ""
}

// ListSessions returns all sessions in the given directory.
func ListSessions(cwd, sessionDir string) ([]SessionInfo, error) {
	if sessionDir == "" {
		sessionDir = defaultSessionDir(cwd)
	}

	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var infos []SessionInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		path := filepath.Join(sessionDir, entry.Name())
		sess, err := OpenSession(path, cwd)
		if err != nil {
			continue
		}
		infos = append(infos, sess.Info())
	}

	// Sort by updated time descending
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Updated.After(infos[j].Updated)
	})

	return infos, nil
}

// ContinueRecent finds the most recent session and returns it.
func ContinueRecent(cwd, sessionDir string) (*Session, error) {
	infos, err := ListSessions(cwd, sessionDir)
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return NewSession(cwd, sessionDir), nil
	}
	return OpenSession(infos[0].Path, cwd)
}

// ForkFrom creates a new session by copying the messages from an existing session.
func ForkFrom(sourcePath, cwd, sessionDir string) (*Session, error) {
	source, err := OpenSession(sourcePath, cwd)
	if err != nil {
		return nil, err
	}
	dest := NewSession(cwd, sessionDir)
	dest.messages = source.Messages()
	dest.dirty = true

	// Write all messages to the new session file
	if dest.File != "" {
		f, err := os.Create(dest.File)
		if err != nil {
			return nil, fmt.Errorf("create forked session: %w", err)
		}
		defer f.Close()
		for _, msg := range dest.messages {
			data, err := marshalMessage(msg)
			if err != nil {
				continue
			}
			data = append(data, '\n')
			f.Write(data)
		}
	}

	return dest, nil
}

// --- Helpers ---

func generateSessionID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b) + "-" + time.Now().Format("20060102-150405")
}

func defaultSessionDir(cwd string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".pi", "sessions", cwdHash(cwd))
}

func cwdHash(cwd string) string {
	if cwd == "" {
		return "default"
	}
	// Simple hash of the cwd path
	b := []byte(cwd)
	sum := 0
	for _, c := range b {
		sum = sum*31 + int(c)
	}
	return fmt.Sprintf("%x", sum)
}

// marshalMessage serializes a Message to JSON, wrapping with a role field for round-tripping.
func marshalMessage(msg ai.Message) ([]byte, error) {
	wrapper := struct {
		Role    string      `json:"role"`
		Message ai.Message  `json:"message"`
	}{
		Role:    string(msg.GetRole()),
		Message: msg,
	}
	return json.Marshal(wrapper)
}

// unmarshalMessage deserializes a Message from JSON.
func unmarshalMessage(raw json.RawMessage) (ai.Message, error) {
	var header struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return nil, err
	}

	switch ai.MessageRole(header.Role) {
	case ai.RoleUser:
		var msg ai.UserMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case ai.RoleAssistant:
		var msg ai.AssistantMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case ai.RoleTool:
		var msg ai.ToolResultMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	default:
		return nil, fmt.Errorf("unknown message role: %s", header.Role)
	}
}

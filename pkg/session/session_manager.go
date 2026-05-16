package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/badlogic/pi-mono/pkg/ai"
)

// CurrentSessionVersion is the session file format version.
const CurrentSessionVersion = 3

// SessionHeader is the first line of a session file.
type SessionHeader struct {
	Type          string `json:"type"`                    // "session"
	Version       int    `json:"version,omitempty"`        // session format version
	ID            string `json:"id"`                       // unique session ID
	Timestamp     string `json:"timestamp"`                // ISO timestamp
	CWD           string `json:"cwd"`                      // working directory at session start
	ParentSession string `json:"parentSession,omitempty"`  // parent session ID if forked
}

// Entry types for the session log
type EntryType string

const (
	EntryMessage          EntryType = "message"
	EntryModelChange      EntryType = "model_change"
	EntryThinkingChange   EntryType = "thinking_level_change"
	EntryCompaction       EntryType = "compaction"
	EntryBranchSummary    EntryType = "branch_summary"
	EntryCustom           EntryType = "custom"
)

// SessionEntryBase is the common fields for all session entries.
type SessionEntryBase struct {
	Type      EntryType `json:"type"`
	ID        string    `json:"id"`
	ParentID  string    `json:"parentId,omitempty"`
	Timestamp string    `json:"timestamp"`
}

// SessionMessageEntry wraps a message with tree metadata.
type SessionMessageEntry struct {
	SessionEntryBase
	Message json.RawMessage `json:"message"` // Raw message to avoid circular deps
}

// CompactionEntry records a context compaction event.
type CompactionEntry struct {
	SessionEntryBase
	Summary         string `json:"summary"`
	FirstKeptEntryID string `json:"firstKeptEntryId"`
	TokensBefore    int    `json:"tokensBefore"`
}

// BranchSummaryEntry records a branch summary event.
type BranchSummaryEntry struct {
	SessionEntryBase
	FromID string `json:"fromId"`
	Summary string `json:"summary"`
}

// ModelChangeEntry records a model switch event.
type ModelChangeEntry struct {
	SessionEntryBase
	Provider string `json:"provider"`
	ModelID  string `json:"modelId"`
}

// SessionManager manages session persistence with tree-based history.
type SessionManager struct {
	sessionsDir string
	header      *SessionHeader
	entries     []any // Mixed entry types
	file        *os.File
	filePath    string
	inMemory    bool
}

// SessionManagerOptions configures session persistence.
type SessionManagerOptions struct {
	SessionsDir   string
	InMemory      bool
	ParentSession string
	CWD           string
}

// NewSessionManager creates a new session manager.
func NewSessionManager(opts SessionManagerOptions) *SessionManager {
	sessionsDir := opts.SessionsDir
	if sessionsDir == "" {
		home, _ := os.UserHomeDir()
		sessionsDir = filepath.Join(home, ".pi", "sessions")
	}

	cwd := opts.CWD
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	return &SessionManager{
		sessionsDir: sessionsDir,
		inMemory:    opts.InMemory,
		header: &SessionHeader{
			Type:          "session",
			Version:       CurrentSessionVersion,
			ID:            smGenerateSessionID(),
			Timestamp:     time.Now().Format(time.RFC3339),
			CWD:           cwd,
			ParentSession: opts.ParentSession,
		},
	}
}

// CreateNew creates a new session file.
func (sm *SessionManager) CreateNew() error {
	if sm.inMemory {
		return nil
	}

	if err := os.MkdirAll(sm.sessionsDir, 0755); err != nil {
		return fmt.Errorf("cannot create sessions directory: %w", err)
	}

	sm.filePath = filepath.Join(sm.sessionsDir, sm.header.ID+".jsonl")
	f, err := os.OpenFile(sm.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("cannot create session file: %w", err)
	}
	sm.file = f

	// Write header
	return sm.writeEntry(sm.header)
}

// ContinueSession loads and continues an existing session.
func (sm *SessionManager) ContinueSession(sessionID string) error {
	if sm.inMemory {
		return nil
	}

	sm.filePath = filepath.Join(sm.sessionsDir, sessionID+".jsonl")
	return sm.loadSession()
}

// OpenLatest finds and opens the most recent session.
func (sm *SessionManager) OpenLatest() error {
	if sm.inMemory {
		return nil
	}

	entries, err := os.ReadDir(sm.sessionsDir)
	if err != nil {
		return fmt.Errorf("cannot read sessions directory: %w", err)
	}

	var latest string
	var latestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latest = strings.TrimSuffix(entry.Name(), ".jsonl")
		}
	}

	if latest == "" {
		return fmt.Errorf("no sessions found")
	}

	return sm.ContinueSession(latest)
}

// AppendMessage appends a message entry to the session.
func (sm *SessionManager) AppendMessage(msg ai.Message) error {
	role := string(msg.GetRole())
	wrapper := map[string]any{
		"type":      EntryMessage,
		"id":        smGenerateEntryID(),
		"parentId":  sm.lastEntryID(),
		"timestamp": time.Now().Format(time.RFC3339),
		"message": map[string]any{
			"role": role,
			"data": msg,
		},
	}
	sm.entries = append(sm.entries, wrapper)
	return sm.writeEntry(wrapper)
}

// AppendCompaction appends a compaction entry.
func (sm *SessionManager) AppendCompaction(summary, firstKeptEntryID string, tokensBefore int) error {
	entry := CompactionEntry{
		SessionEntryBase: SessionEntryBase{
			Type:      EntryCompaction,
			ID:        smGenerateEntryID(),
			ParentID:  sm.lastEntryID(),
			Timestamp: time.Now().Format(time.RFC3339),
		},
		Summary:          summary,
		FirstKeptEntryID: firstKeptEntryID,
		TokensBefore:     tokensBefore,
	}
	sm.entries = append(sm.entries, entry)
	return sm.writeEntry(entry)
}

// AppendModelChange appends a model change entry.
func (sm *SessionManager) AppendModelChange(provider, modelID string) error {
	entry := ModelChangeEntry{
		SessionEntryBase: SessionEntryBase{
			Type:      EntryModelChange,
			ID:        smGenerateEntryID(),
			ParentID:  sm.lastEntryID(),
			Timestamp: time.Now().Format(time.RFC3339),
		},
		Provider: provider,
		ModelID:  modelID,
	}
	sm.entries = append(sm.entries, entry)
	return sm.writeEntry(entry)
}

// GetHeader returns the session header.
func (sm *SessionManager) GetHeader() *SessionHeader {
	return sm.header
}

// GetEntries returns all session entries.
func (sm *SessionManager) GetEntries() []any {
	return sm.entries
}

// GetMessages extracts all messages from entries.
func (sm *SessionManager) GetMessages() []ai.Message {
	var messages []ai.Message
	for _, entry := range sm.entries {
		if msgEntry, ok := entry.(map[string]any); ok {
			if msgEntry["type"] == EntryMessage {
				if msgData, ok := msgEntry["message"].(map[string]any); ok {
					if role, ok := msgData["role"].(string); ok {
						switch role {
						case string(ai.RoleUser):
							if data, err := json.Marshal(msgData["data"]); err == nil {
								var m ai.UserMessage
								if json.Unmarshal(data, &m) == nil {
									messages = append(messages, m)
								}
							}
						case string(ai.RoleAssistant):
							if data, err := json.Marshal(msgData["data"]); err == nil {
								var m ai.AssistantMessage
								if json.Unmarshal(data, &m) == nil {
									messages = append(messages, m)
								}
							}
						case string(ai.RoleTool):
							if data, err := json.Marshal(msgData["data"]); err == nil {
								var m ai.ToolResultMessage
								if json.Unmarshal(data, &m) == nil {
									messages = append(messages, m)
								}
							}
						}
					}
				}
			}
		}
	}
	return messages
}

// GetLatestCompactionEntry returns the most recent compaction entry.
func (sm *SessionManager) GetLatestCompactionEntry() *CompactionEntry {
	for i := len(sm.entries) - 1; i >= 0; i-- {
		if entry, ok := sm.entries[i].(CompactionEntry); ok {
			return &entry
		}
	}
	return nil
}

// SessionFilePath returns the path to the session file.
func (sm *SessionManager) SessionFilePath() string {
	return sm.filePath
}

// Close closes the session file.
func (sm *SessionManager) Close() error {
	if sm.file != nil {
		return sm.file.Close()
	}
	return nil
}

// ListSessions returns a list of available sessions.
func (sm *SessionManager) ListSessions() ([]SessionHeader, error) {
	entries, err := os.ReadDir(sm.sessionsDir)
	if err != nil {
		return nil, err
	}

	var sessions []SessionHeader
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		filePath := filepath.Join(sm.sessionsDir, entry.Name())
		f, err := os.Open(filePath)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		if scanner.Scan() {
			var header SessionHeader
			if json.Unmarshal(scanner.Bytes(), &header) == nil && header.Type == "session" {
				sessions = append(sessions, header)
			}
		}
		f.Close()
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Timestamp > sessions[j].Timestamp
	})

	return sessions, nil
}

// --- Internal methods ---

func (sm *SessionManager) writeEntry(entry any) error {
	if sm.inMemory || sm.file == nil {
		return nil
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = sm.file.Write(append(data, '\n'))
	return err
}

func (sm *SessionManager) loadSession() error {
	f, err := os.Open(sm.filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		if lineNum == 1 {
			// First line is the header
			var header SessionHeader
			if json.Unmarshal(line, &header) == nil {
				sm.header = &header
			}
			continue
		}

		// Parse entry type
		var base SessionEntryBase
		if json.Unmarshal(line, &base) != nil {
			continue
		}

		switch base.Type {
		case EntryMessage:
			var entry map[string]any
			if json.Unmarshal(line, &entry) == nil {
				sm.entries = append(sm.entries, entry)
			}
		case EntryCompaction:
			var entry CompactionEntry
			if json.Unmarshal(line, &entry) == nil {
				sm.entries = append(sm.entries, entry)
			}
		case EntryBranchSummary:
			var entry BranchSummaryEntry
			if json.Unmarshal(line, &entry) == nil {
				sm.entries = append(sm.entries, entry)
			}
		case EntryModelChange:
			var entry ModelChangeEntry
			if json.Unmarshal(line, &entry) == nil {
				sm.entries = append(sm.entries, entry)
			}
		default:
			var entry map[string]any
			if json.Unmarshal(line, &entry) == nil {
				sm.entries = append(sm.entries, entry)
			}
		}
	}

	// Re-open for appending
	sm.file, err = os.OpenFile(sm.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	return err
}

func (sm *SessionManager) lastEntryID() string {
	if len(sm.entries) == 0 {
		return ""
	}
	switch entry := sm.entries[len(sm.entries)-1].(type) {
	case SessionEntryBase:
		return entry.ID
	case map[string]any:
		if id, ok := entry["id"].(string); ok {
			return id
		}
	case CompactionEntry:
		return entry.ID
	case BranchSummaryEntry:
		return entry.ID
	case ModelChangeEntry:
		return entry.ID
	}
	return ""
}

func smGenerateSessionID() string {
	return fmt.Sprintf("%d-%s", time.Now().Unix(), smRandomHex(8))
}

func smGenerateEntryID() string {
	return fmt.Sprintf("e-%s", smRandomHex(6))
}

func smRandomHex(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = "0123456789abcdef"[time.Now().UnixNano()%16]
	}
	return string(b)
}

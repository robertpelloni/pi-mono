package session

import (
	"path/filepath"
	"testing"

	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestNewSession(t *testing.T) {
	tmpDir := t.TempDir()
	sess := NewSession("/test/project", tmpDir)

	if sess.ID == "" {
		t.Error("session should have an ID")
	}
	if sess.CWD != "/test/project" {
		t.Errorf("expected CWD '/test/project', got '%s'", sess.CWD)
	}
	if sess.File == "" {
		t.Error("session should have a file path")
	}
	if sess.IsInMemory() {
		t.Error("new session with dir should not be in-memory")
	}
}

func TestInMemorySession(t *testing.T) {
	sess := InMemorySession()

	if sess.ID == "" {
		t.Error("in-memory session should have an ID")
	}
	if !sess.IsInMemory() {
		t.Error("in-memory session should be flagged as in-memory")
	}
}

func TestAppendAndRetrieveMessages(t *testing.T) {
	tmpDir := t.TempDir()
	sess := NewSession("/test", tmpDir)

	msg := ai.UserMessage{
		Content:   []ai.Content{ai.TextContent{Text: "Hello"}},
		Timestamp: 1000,
	}

	err := sess.AppendMessage(msg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	msgs := sess.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	userMsg, ok := msgs[0].(ai.UserMessage)
	if !ok {
		t.Fatal("expected UserMessage type")
	}

	txt, ok := userMsg.Content[0].(ai.TextContent)
	if !ok || txt.Text != "Hello" {
		t.Errorf("expected 'Hello', got '%v'", userMsg.Content[0])
	}
}

func TestSessionPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	sess := NewSession("/test", tmpDir)

	// Write messages
	sess.AppendMessage(ai.UserMessage{
		Content:   []ai.Content{ai.TextContent{Text: "First message"}},
		Timestamp: 1000,
	})
	sess.AppendMessage(ai.AssistantMessage{
		Content:    []ai.Content{ai.TextContent{Text: "First response"}},
		API:        ai.ApiOpenAIResponses,
		Provider:    ai.ProviderOpenAI,
		Model:      "gpt-4o",
		StopReason: ai.StopReasonStop,
		Timestamp:  2000,
	})

	// Re-open the session
	loaded, err := OpenSession(sess.File, "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loadedMsgs := loaded.Messages()
	if len(loadedMsgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(loadedMsgs))
	}
}

func TestSetMessages(t *testing.T) {
	sess := InMemorySession()

	msgs := []ai.Message{
		ai.UserMessage{Content: []ai.Content{ai.TextContent{Text: "test"}}, Timestamp: 1},
	}
	sess.SetMessages(msgs)

	if len(sess.Messages()) != 1 {
		t.Errorf("expected 1 message, got %d", len(sess.Messages()))
	}
}

func TestSessionInfo(t *testing.T) {
	tmpDir := t.TempDir()
	sess := NewSession("/test", tmpDir)

	info := sess.Info()
	if info.ID != sess.ID {
		t.Error("info ID should match session ID")
	}
	if info.MessageCount != 0 {
		t.Errorf("expected 0 messages, got %d", info.MessageCount)
	}
}

func TestListSessions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a few sessions
	sess1 := NewSession("/test", tmpDir)
	sess1.AppendMessage(ai.UserMessage{Content: []ai.Content{ai.TextContent{Text: "msg1"}}, Timestamp: 1})

	sess2 := NewSession("/test", tmpDir)
	sess2.AppendMessage(ai.UserMessage{Content: []ai.Content{ai.TextContent{Text: "msg2"}}, Timestamp: 2})

	infos, err := ListSessions("/test", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(infos))
	}
}

func TestContinueRecent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a session
	sess := NewSession("/test", tmpDir)
	sess.AppendMessage(ai.UserMessage{Content: []ai.Content{ai.TextContent{Text: "test"}}, Timestamp: 1})

	// Continue should find it
	recent, err := ContinueRecent("/test", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recent.ID != sess.ID {
		t.Errorf("expected ID %s, got %s", sess.ID, recent.ID)
	}
}

func TestContinueRecentEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	recent, err := ContinueRecent("/test", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recent == nil {
		t.Error("should create new session when none exist")
	}
}

func TestForkFrom(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source session
	source := NewSession("/test", tmpDir)
	source.AppendMessage(ai.UserMessage{Content: []ai.Content{ai.TextContent{Text: "original"}}, Timestamp: 1})
	source.AppendMessage(ai.AssistantMessage{
		Content:    []ai.Content{ai.TextContent{Text: "response"}},
		API:        ai.ApiOpenAIResponses,
		Provider:    ai.ProviderOpenAI,
		Model:      "gpt-4o",
		StopReason: ai.StopReasonStop,
		Timestamp:  2,
	})

	// Fork
	forked, err := ForkFrom(source.File, "/test", tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if forked.ID == source.ID {
		t.Error("forked session should have different ID")
	}
	if len(forked.Messages()) != len(source.Messages()) {
		t.Errorf("forked session should have same message count: source=%d forked=%d",
			len(source.Messages()), len(forked.Messages()))
	}
}

func TestCWDHash(t *testing.T) {
	hash1 := cwdHash("/test/project")
	hash2 := cwdHash("/test/project")
	hash3 := cwdHash("/other/project")

	if hash1 != hash2 {
		t.Error("same CWD should produce same hash")
	}
	if hash1 == hash3 {
		t.Error("different CWDs should produce different hashes")
	}
}

func TestDefaultSessionDir(t *testing.T) {
	dir := defaultSessionDir("/test")
	if dir == "" {
		t.Error("should return a non-empty path")
	}
	if !filepath.IsAbs(dir) {
		t.Error("session dir should be absolute")
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()

	if id1 == id2 {
		t.Error("session IDs should be unique")
	}
	if len(id1) < 10 {
		t.Error("session ID should be reasonably long")
	}
}

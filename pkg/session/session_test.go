package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateSession(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "session_test")
	defer os.RemoveAll(tmpDir)

	sm := CreateSession(tmpDir, tmpDir)
	if sm == nil {
		t.Fatal("Expected non-nil SessionManager")
	}
	if sm.GetSessionID() == "" {
		t.Error("Expected non-empty session ID")
	}
	if !sm.IsPersisted() {
		t.Error("Expected persisted session")
	}
}

func TestInMemorySession(t *testing.T) {
	sm := InMemorySession("/tmp/test")
	if sm.IsPersisted() {
		t.Error("Expected non-persisted session")
	}
}

func TestAppendMessage(t *testing.T) {
	sm := InMemorySession("/tmp/test")
	msg := map[string]interface{}{
		"role": "user",
		"content": []map[string]interface{}{
			{"type": "text", "text": "Hello world"},
		},
		"timestamp": 1234567890,
	}
	msgJSON, _ := json.Marshal(msg)
	entryID := sm.AppendMessageByJSON(msgJSON)
	if entryID == "" {
		t.Error("Expected non-empty entry ID")
	}
	entries := sm.GetEntries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}
	if entries[0].Type != EntryMessage {
		t.Errorf("Expected EntryMessage, got %s", entries[0].Type)
	}
}

func TestAppendThinkingLevelChange(t *testing.T) {
	sm := InMemorySession("/tmp/test")
	entryID := sm.AppendThinkingLevelChange("high")
	if entryID == "" {
		t.Error("Expected non-empty entry ID")
	}
	entries := sm.GetEntries()
	if entries[0].ThinkingLevel != "high" {
		t.Errorf("Expected 'high', got %s", entries[0].ThinkingLevel)
	}
}

func TestAppendModelChange(t *testing.T) {
	sm := InMemorySession("/tmp/test")
	sm.AppendModelChange("anthropic", "claude-4-sonnet")
	entries := sm.GetEntries()
	if entries[0].Provider != "anthropic" {
		t.Errorf("Expected 'anthropic', got %s", entries[0].Provider)
	}
}

func TestTreeStructure(t *testing.T) {
	sm := InMemorySession("/tmp/test")
	id1 := sm.AppendThinkingLevelChange("low")
	id2 := sm.AppendThinkingLevelChange("medium")
	id3 := sm.AppendThinkingLevelChange("high")

	entry2 := sm.GetEntry(id2)
	if entry2 == nil || entry2.ParentID == nil || *entry2.ParentID != id1 {
		t.Error("Expected id2 parent to be id1")
	}
	entry3 := sm.GetEntry(id3)
	if entry3 == nil || entry3.ParentID == nil || *entry3.ParentID != id2 {
		t.Error("Expected id3 parent to be id2")
	}

	leafID := sm.GetLeafID()
	if leafID == nil || *leafID != id3 {
		t.Errorf("Expected leaf ID %s, got %v", id3, leafID)
	}
}

func TestBranch(t *testing.T) {
	sm := InMemorySession("/tmp/test")
	id1 := sm.AppendThinkingLevelChange("low")
	sm.AppendThinkingLevelChange("medium")
	sm.AppendThinkingLevelChange("high")

	sm.Branch(id1)
	id4 := sm.AppendThinkingLevelChange("low-again")
	entry4 := sm.GetEntry(id4)
	if entry4 == nil || entry4.ParentID == nil || *entry4.ParentID != id1 {
		t.Error("Expected id4 parent to be id1 after branching")
	}

	children := sm.GetChildren(id1)
	if len(children) != 2 {
		t.Errorf("Expected 2 children for id1, got %d", len(children))
	}
}

func TestBranchWithSummary(t *testing.T) {
	sm := InMemorySession("/tmp/test")
	sm.AppendThinkingLevelChange("low")
	id2 := sm.AppendThinkingLevelChange("medium")
	sm.AppendThinkingLevelChange("high")

	summaryID := sm.BranchWithSummary(&id2, "Switched from high to medium", nil, nil)
	entry := sm.GetEntry(summaryID)
	if entry == nil || entry.Type != EntryBranchSummary {
		t.Error("Expected branch summary entry")
	}
	if entry.Summary != "Switched from high to medium" {
		t.Errorf("Unexpected summary: %s", entry.Summary)
	}
}

func TestGetBranch(t *testing.T) {
	sm := InMemorySession("/tmp/test")
	id1 := sm.AppendThinkingLevelChange("low")
	id2 := sm.AppendThinkingLevelChange("medium")
	sm.AppendThinkingLevelChange("high")

	branch := sm.GetBranch(nil)
	if len(branch) != 3 {
		t.Errorf("Expected 3 entries in branch, got %d", len(branch))
	}
	if branch[0].ID != id1 {
		t.Error("Branch order incorrect")
	}

	branch2 := sm.GetBranch(&id2)
	if len(branch2) != 2 {
		t.Errorf("Expected 2 entries in branch from id2, got %d", len(branch2))
	}
}

func TestSessionInfo(t *testing.T) {
	sm := InMemorySession("/tmp/test")
	sm.AppendSessionInfo("My Session")
	name := sm.GetSessionName()
	if name == nil || *name != "My Session" {
		t.Errorf("Expected 'My Session', got %v", name)
	}
}

func TestLabels(t *testing.T) {
	sm := InMemorySession("/tmp/test")
	id1 := sm.AppendThinkingLevelChange("low")
	labelText := "important"
	sm.AppendLabelChange(id1, &labelText)
	label := sm.GetLabel(id1)
	if label == nil || *label != "important" {
		t.Errorf("Expected 'important', got %v", label)
	}

	sm.AppendLabelChange(id1, nil)
	label = sm.GetLabel(id1)
	if label != nil {
		t.Errorf("Expected nil label after clearing, got %v", label)
	}
}

func TestCompaction(t *testing.T) {
	sm := InMemorySession("/tmp/test")
	id1 := sm.AppendThinkingLevelChange("low")
	sm.AppendThinkingLevelChange("medium")

	fromHook := false
	compID := sm.AppendCompaction("Summary of old context", id1, 10000, nil, &fromHook)
	entry := sm.GetEntry(compID)
	if entry == nil || entry.Type != EntryCompaction {
		t.Error("Expected compaction entry")
	}
	if entry.FirstKeptEntryID != id1 {
		t.Errorf("Expected firstKeptEntryId %s, got %s", id1, entry.FirstKeptEntryID)
	}

	entries := sm.GetEntries()
	latest := GetLatestCompactionEntry(entries)
	if latest == nil || latest.Summary != "Summary of old context" {
		t.Error("GetLatestCompactionEntry failed")
	}
}

func TestOpenSession(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "session_test")
	defer os.RemoveAll(tmpDir)

	header := SessionHeader{
		Type: "session", Version: CurrentSessionVersion,
		ID: "test-session-123", Timestamp: "2024-01-01T00:00:00Z", CWD: tmpDir,
	}
	headerJSON, _ := json.Marshal(header)

	entry := SessionEntry{
		SessionEntryBase: SessionEntryBase{
			Type: EntryThinkingLevel, ID: "abc123", Timestamp: "2024-01-01T00:00:01Z",
		},
		ThinkingLevel: "high",
	}
	entryJSON, _ := json.Marshal(entry)

	filePath := filepath.Join(tmpDir, "test.jsonl")
	os.WriteFile(filePath, []byte(string(headerJSON)+"\n"+string(entryJSON)+"\n"), 0644)

	sm := OpenSession(filePath, tmpDir, nil)
	if sm.GetSessionID() != "test-session-123" {
		t.Errorf("Expected 'test-session-123', got %s", sm.GetSessionID())
	}
	entries := sm.GetEntries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}
}

func TestMigrationV1ToV2(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "session_test")
	defer os.RemoveAll(tmpDir)

	header := map[string]interface{}{
		"type": "session", "id": "v1-session",
		"timestamp": "2024-01-01T00:00:00Z", "cwd": tmpDir,
	}
	headerJSON, _ := json.Marshal(header)

	entry := map[string]interface{}{
		"type": "message", "timestamp": "2024-01-01T00:00:01Z",
		"message": map[string]interface{}{
			"role": "user",
			"content": []map[string]interface{}{{"type": "text", "text": "Hello"}},
		},
	}
	entryJSON, _ := json.Marshal(entry)

	filePath := filepath.Join(tmpDir, "v1.jsonl")
	os.WriteFile(filePath, []byte(string(headerJSON)+"\n"+string(entryJSON)+"\n"), 0644)

	sm := OpenSession(filePath, tmpDir, nil)
	h := sm.GetHeader()
	if h == nil || h.Version != CurrentSessionVersion {
		t.Errorf("Expected version %d after migration, got %d", CurrentSessionVersion, h.Version)
	}
}

func TestListSessions(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "session_test")
	defer os.RemoveAll(tmpDir)

	header := SessionHeader{
		Type: "session", Version: CurrentSessionVersion,
		ID: "list-test-123", Timestamp: "2024-01-01T00:00:00Z", CWD: tmpDir,
	}
	headerJSON, _ := json.Marshal(header)
	filePath := filepath.Join(tmpDir, "test.jsonl")
	os.WriteFile(filePath, []byte(string(headerJSON)+"\n"), 0644)

	infos, err := ListSessions(tmpDir, tmpDir)
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}
	if len(infos) != 1 {
		t.Errorf("Expected 1 session, got %d", len(infos))
	}
}

func TestGetTree(t *testing.T) {
	sm := InMemorySession("/tmp/test")
	id1 := sm.AppendThinkingLevelChange("low")
	sm.AppendThinkingLevelChange("medium")
	sm.AppendThinkingLevelChange("high")

	sm.Branch(id1)
	sm.AppendThinkingLevelChange("low-again")

	tree := sm.GetTree()
	if len(tree) != 1 {
		t.Errorf("Expected 1 root, got %d", len(tree))
	}
	if len(tree[0].Children) != 2 {
		t.Errorf("Expected 2 children for root, got %d", len(tree[0].Children))
	}
}

func TestFindMostRecentSession(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "session_test")
	defer os.RemoveAll(tmpDir)

	result := FindMostRecentSession(tmpDir)
	if result != "" {
		t.Error("Expected empty string for no sessions")
	}

	header := SessionHeader{
		Type: "session", Version: CurrentSessionVersion,
		ID: "recent-test", Timestamp: "2024-01-01T00:00:00Z", CWD: tmpDir,
	}
	headerJSON, _ := json.Marshal(header)
	filePath := filepath.Join(tmpDir, "test.jsonl")
	os.WriteFile(filePath, []byte(string(headerJSON)+"\n"), 0644)

	result = FindMostRecentSession(tmpDir)
	if !strings.HasSuffix(result, "test.jsonl") {
		t.Errorf("Expected test.jsonl, got %s", result)
	}
}

func TestGetDefaultSessionDir(t *testing.T) {
	dir := GetDefaultSessionDir("/home/user/project")
	if !strings.Contains(dir, "sessions") {
		t.Errorf("Expected 'sessions' in dir, got %s", dir)
	}
}

func TestParseSessionEntries(t *testing.T) {
	content := `{"type":"session","id":"test","timestamp":"2024-01-01T00:00:00Z","cwd":"/tmp"}
{"type":"thinking_level_change","id":"abc","parentId":null,"timestamp":"2024-01-01T00:00:01Z","thinkingLevel":"high"}`
	entries := ParseSessionEntries(content)
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
}

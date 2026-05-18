package session

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/messages"
)

// CurrentSessionVersion is the latest session format version.
const CurrentSessionVersion = 3

// ---------------------------------------------------------------------------
// Session Header
// ---------------------------------------------------------------------------

// SessionHeader is the first line of a session JSONL file.
type SessionHeader struct {
	Type          string  `json:"type"`                    // always "session"
	Version       int     `json:"version,omitempty"`       // v1 sessions lack this
	ID            string  `json:"id"`
	Timestamp     string  `json:"timestamp"`
	CWD           string  `json:"cwd"`
	ParentSession *string `json:"parentSession,omitempty"`
}

// ---------------------------------------------------------------------------
// Session Entry types
// ---------------------------------------------------------------------------

// EntryType discriminates the kind of session entry.
type EntryType string

const (
	EntryMessage           EntryType = "message"
	EntryThinkingLevel     EntryType = "thinking_level_change"
	EntryModelChange       EntryType = "model_change"
	EntryCompaction        EntryType = "compaction"
	EntryBranchSummary     EntryType = "branch_summary"
	EntryCustom            EntryType = "custom"
	EntryCustomMessage     EntryType = "custom_message"
	EntryLabel             EntryType = "label"
	EntrySessionInfo       EntryType = "session_info"
)

// SessionEntryBase contains fields common to every session entry.
type SessionEntryBase struct {
	Type      EntryType `json:"type"`
	ID        string    `json:"id"`
	ParentID  *string   `json:"parentId"`
	Timestamp string    `json:"timestamp"`
}

// SessionMessageEntry wraps an AgentMessage in the session tree.
type SessionMessageEntry struct {
	SessionEntryBase
	Message json.RawMessage `json:"message"`
	// Parsed cached message (not serialized)
	parsedMessage ai.Message
}

// ThinkingLevelChangeEntry records a thinking level change.
type ThinkingLevelChangeEntry struct {
	SessionEntryBase
	ThinkingLevel string `json:"thinkingLevel"`
}

// ModelChangeEntry records a model switch.
type ModelChangeEntry struct {
	SessionEntryBase
	Provider string `json:"provider"`
	ModelID  string `json:"modelId"`
}

// CompactionEntry records a context compaction event.
type CompactionEntry struct {
	SessionEntryBase
	Summary          string      `json:"summary"`
	FirstKeptEntryID string      `json:"firstKeptEntryId"`
	TokensBefore     int         `json:"tokensBefore"`
	Details          interface{} `json:"details,omitempty"`
	FromHook         *bool       `json:"fromHook,omitempty"`
}

// BranchSummaryEntry records a branch summary when navigating the tree.
type BranchSummaryEntry struct {
	SessionEntryBase
	FromID   string      `json:"fromId"`
	Summary  string      `json:"summary"`
	Details  interface{} `json:"details,omitempty"`
	FromHook *bool       `json:"fromHook,omitempty"`
}

// CustomEntry stores extension-specific data (not sent to LLM).
type CustomEntry struct {
	SessionEntryBase
	CustomType string      `json:"customType"`
	Data       interface{} `json:"data,omitempty"`
}

// CustomMessageEntry injects extension content into LLM context.
type CustomMessageEntry struct {
	SessionEntryBase
	CustomType string          `json:"customType"`
	Content    json.RawMessage `json:"content"`
	Display    bool            `json:"display"`
	Details    interface{}     `json:"details,omitempty"`
}

// LabelEntry stores a user-defined bookmark on an entry.
type LabelEntry struct {
	SessionEntryBase
	TargetID string  `json:"targetId"`
	Label    *string `json:"label"`
}

// SessionInfoEntry stores session display name.
type SessionInfoEntry struct {
	SessionEntryBase
	Name *string `json:"name,omitempty"`
}

// SessionEntry is the union type for all session entries.
type SessionEntry struct {
	SessionEntryBase

	// Message entry fields
	Message json.RawMessage `json:"message,omitempty"`

	// ThinkingLevelChange fields
	ThinkingLevel string `json:"thinkingLevel,omitempty"`

	// ModelChange fields
	Provider string `json:"provider,omitempty"`
	ModelID  string `json:"modelId,omitempty"`

	// Compaction fields
	Summary          string      `json:"summary,omitempty"`
	FirstKeptEntryID string      `json:"firstKeptEntryId,omitempty"`
	TokensBefore     int         `json:"tokensBefore,omitempty"`
	CompactionDetails interface{} `json:"details,omitempty"`
	FromHook         *bool       `json:"fromHook,omitempty"`

	// BranchSummary fields
	FromID       string      `json:"fromId,omitempty"`
	BranchDetails interface{} `json:"branchDetails,omitempty"`

	// Custom/CustomMessage fields
	CustomType string          `json:"customType,omitempty"`
	Data       interface{}     `json:"data,omitempty"`
	Content    json.RawMessage `json:"content,omitempty"`
	Display    bool            `json:"display,omitempty"`

	// Label fields
	TargetID string  `json:"targetId,omitempty"`
	Label    *string `json:"label,omitempty"`

	// SessionInfo fields
	Name *string `json:"name,omitempty"`
}

// FileEntry is either a SessionHeader or a SessionEntry (for JSONL lines).
type FileEntry struct {
	// We use raw JSON to allow flexible parsing
	Raw json.RawMessage `json:"-"`
}

// ---------------------------------------------------------------------------
// Session context result
// ---------------------------------------------------------------------------

// SessionContext is the resolved message list + settings for the LLM.
type SessionContext struct {
	Messages      []ai.Message
	ThinkingLevel string
	Model         *ModelRef // nil if no model known yet
}

// ModelRef identifies a provider+model pair.
type ModelRef struct {
	Provider string
	ModelID  string
}

// ---------------------------------------------------------------------------
// Session tree node
// ---------------------------------------------------------------------------

// SessionTreeNode is a defensive copy of an entry + its children.
type SessionTreeNode struct {
	Entry         SessionEntry
	Children      []SessionTreeNode
	Label         *string
	LabelTimestamp *string
}

// ---------------------------------------------------------------------------
// SessionInfo (for listing)
// ---------------------------------------------------------------------------

// SessionInfo is a lightweight summary used for listing sessions.
type SessionInfo struct {
	Path              string    `json:"path"`
	ID                string    `json:"id"`
	CWD               string    `json:"cwd"`
	Name              string    `json:"name,omitempty"`
	ParentSessionPath *string   `json:"parentSessionPath,omitempty"`
	Created           time.Time `json:"created"`
	Modified          time.Time `json:"modified"`
	MessageCount      int       `json:"messageCount"`
	FirstMessage      string    `json:"firstMessage"`
	AllMessagesText   string    `json:"allMessagesText"`
}

// ---------------------------------------------------------------------------
// SessionManager - tree-based session manager
// ---------------------------------------------------------------------------

// SessionManager manages conversation sessions as append-only trees in JSONL files.
type SessionManager struct {
	mu sync.Mutex

	sessionID   string
	sessionFile *string
	sessionDir  string
	cwd         string
	persist     bool
	flushed     bool

	fileEntries []json.RawMessage // raw lines (header + entries)
	byID        map[string]*SessionEntry
	labelsByID  map[string]*string
	labelTSByID map[string]string
	leafID      *string
}

// NewSessionOptions configures session creation.
type NewSessionOptions struct {
	ID            *string
	ParentSession *string
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

// CreateSession creates a new persisted session.
func CreateSession(cwd, sessionDir string) *SessionManager {
	if sessionDir == "" {
		sessionDir = GetDefaultSessionDir(cwd)
	}
	os.MkdirAll(sessionDir, 0755)
	return newSessionManager(cwd, sessionDir, nil, true)
}

// OpenSession opens a specific session file.
func OpenSession(path, sessionDir string, cwdOverride *string) *SessionManager {
	// Try to read cwd from session header
	entries := loadEntriesFromFile(path)
	var headerCwd string
	for _, raw := range entries {
		var header SessionHeader
		if err := json.Unmarshal(raw, &header); err == nil && header.Type == "session" {
			headerCwd = header.CWD
			break
		}
	}
	cwd := ""
	if cwdOverride != nil {
		cwd = *cwdOverride
	} else if headerCwd != "" {
		cwd = headerCwd
	} else {
		cwd, _ = os.Getwd()
	}
	dir := sessionDir
	if dir == "" {
		dir = filepath.Dir(path)
	}
	sm := newSessionManager(cwd, dir, &path, true)
	sm.loadFromFile(path)
	return sm
}

// ContinueRecent continues the most recent session or creates a new one.
func ContinueRecent(cwd, sessionDir string) *SessionManager {
	dir := sessionDir
	if dir == "" {
		dir = GetDefaultSessionDir(cwd)
	}
	recent := FindMostRecentSession(dir)
	if recent != "" {
		return OpenSession(recent, dir, nil)
	}
	return CreateSession(cwd, dir)
}

// InMemorySession creates a session that is never persisted to disk.
func InMemorySession(cwd string) *SessionManager {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	return newSessionManager(cwd, "", nil, false)
}

func newSessionManager(cwd, sessionDir string, sessionFile *string, persist bool) *SessionManager {
	sm := &SessionManager{
		cwd:         cwd,
		sessionDir:  sessionDir,
		persist:     persist,
		byID:        make(map[string]*SessionEntry),
		labelsByID:  make(map[string]*string),
		labelTSByID: make(map[string]string),
	}
	if sessionFile != nil {
		sm.setSessionFile(*sessionFile)
	} else {
		sm.NewSession(nil)
	}
	return sm
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

func (sm *SessionManager) GetCWD() string        { return sm.cwd }
func (sm *SessionManager) GetSessionDir() string  { return sm.sessionDir }
func (sm *SessionManager) GetSessionID() string   { return sm.sessionID }
func (sm *SessionManager) GetSessionFile() *string { return sm.sessionFile }
func (sm *SessionManager) IsPersisted() bool       { return sm.persist }

// GetLeafID returns the current leaf entry ID.
func (sm *SessionManager) GetLeafID() *string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.leafID
}

// GetLeafEntry returns the entry at the current leaf.
func (sm *SessionManager) GetLeafEntry() *SessionEntry {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.leafID == nil {
		return nil
	}
	return sm.byID[*sm.leafID]
}

// GetEntry returns an entry by ID.
func (sm *SessionManager) GetEntry(id string) *SessionEntry {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.byID[id]
}

// GetLabel returns the label for an entry, if any.
func (sm *SessionManager) GetLabel(id string) *string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.labelsByID[id]
}

// GetHeader returns the session header.
func (sm *SessionManager) GetHeader() *SessionHeader {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for _, raw := range sm.fileEntries {
		var header SessionHeader
		if err := json.Unmarshal(raw, &header); err == nil && header.Type == "session" {
			return &header
		}
	}
	return nil
}

// GetEntries returns all non-header entries.
func (sm *SessionManager) GetEntries() []SessionEntry {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	result := make([]SessionEntry, 0, len(sm.byID))
	for _, raw := range sm.fileEntries {
		var base SessionEntryBase
		if err := json.Unmarshal(raw, &base); err != nil || base.Type == "session" {
			continue
		}
		var entry SessionEntry
		if err := json.Unmarshal(raw, &entry); err == nil {
			result = append(result, entry)
		}
	}
	return result
}

// GetBranch returns all entries from the given entry to root.
func (sm *SessionManager) GetBranch(fromID *string) []SessionEntry {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var path []SessionEntry
	startID := fromID
	if startID == nil {
		startID = sm.leafID
	}
	current := startID
	for current != nil {
		entry, ok := sm.byID[*current]
		if !ok {
			break
		}
		path = append([]SessionEntry{*entry}, path...)
		current = entry.ParentID
	}
	return path
}

// GetChildren returns all direct children of an entry.
func (sm *SessionManager) GetChildren(parentID string) []SessionEntry {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	var children []SessionEntry
	for _, entry := range sm.byID {
		if entry.ParentID != nil && *entry.ParentID == parentID {
			children = append(children, *entry)
		}
	}
	return children
}

// GetSessionName returns the latest session display name, if any.
func (sm *SessionManager) GetSessionName() *string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	entries := sm.GetEntriesLocked()
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Type == EntrySessionInfo {
			return entries[i].Name
		}
	}
	return nil
}

// GetTree returns the session as a tree structure.
func (sm *SessionManager) GetTree() []SessionTreeNode {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entries := sm.GetEntriesLocked()
	nodeMap := make(map[string]*SessionTreeNode)

	for i := range entries {
		entry := entries[i]
		label := sm.labelsByID[entry.ID]
		labelTS := sm.labelTSByID[entry.ID]
		nodeMap[entry.ID] = &SessionTreeNode{
			Entry:          entry,
			Children:       []SessionTreeNode{},
			Label:          label,
			LabelTimestamp: &labelTS,
		}
	}

	// Build tree - first pass: link children to parents in nodeMap
	for i := range entries {
		entry := entries[i]
		if entry.ParentID == nil || *entry.ParentID == entry.ID {
			// root entry, no parent to link to
			continue
		}
		parent, ok := nodeMap[*entry.ParentID]
		if ok {
			parent.Children = append(parent.Children, *nodeMap[entry.ID])
		}
	}

	// Second pass: collect roots
	var roots []SessionTreeNode
	for i := range entries {
		entry := entries[i]
		if entry.ParentID == nil || *entry.ParentID == entry.ID {
			roots = append(roots, *nodeMap[entry.ID])
		}
	}

	// Sort children by timestamp
	sortTreeChildren(roots)
	return roots
}

func sortTreeChildren(nodes []SessionTreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Entry.Timestamp < nodes[j].Entry.Timestamp
	})
	for i := range nodes {
		sortTreeChildren(nodes[i].Children)
	}
}

// ---------------------------------------------------------------------------
// Append methods
// ---------------------------------------------------------------------------

// AppendMessage appends an AgentMessage as child of current leaf.
func (sm *SessionManager) AppendMessage(msg ai.Message) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	msgJSON, _ := json.Marshal(msg)
	entry := SessionEntry{
		SessionEntryBase: SessionEntryBase{
			Type:      EntryMessage,
			ID:        generateID(sm.byID),
			ParentID:  sm.leafID,
			Timestamp: time.Now().Format(time.RFC3339Nano),
		},
		Message: msgJSON,
	}
	sm.appendEntry(entry)
	return entry.ID
}

// AppendThinkingLevelChange appends a thinking level change.
func (sm *SessionManager) AppendThinkingLevelChange(level string) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entry := SessionEntry{
		SessionEntryBase: SessionEntryBase{
			Type:      EntryThinkingLevel,
			ID:        generateID(sm.byID),
			ParentID:  sm.leafID,
			Timestamp: time.Now().Format(time.RFC3339Nano),
		},
		ThinkingLevel: level,
	}
	sm.appendEntry(entry)
	return entry.ID
}

// AppendModelChange appends a model change.
func (sm *SessionManager) AppendModelChange(provider, modelID string) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entry := SessionEntry{
		SessionEntryBase: SessionEntryBase{
			Type:      EntryModelChange,
			ID:        generateID(sm.byID),
			ParentID:  sm.leafID,
			Timestamp: time.Now().Format(time.RFC3339Nano),
		},
		Provider: provider,
		ModelID:  modelID,
	}
	sm.appendEntry(entry)
	return entry.ID
}

// AppendCompaction appends a compaction entry.
func (sm *SessionManager) AppendCompaction(summary, firstKeptEntryID string, tokensBefore int, details interface{}, fromHook *bool) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entry := SessionEntry{
		SessionEntryBase: SessionEntryBase{
			Type:      EntryCompaction,
			ID:        generateID(sm.byID),
			ParentID:  sm.leafID,
			Timestamp: time.Now().Format(time.RFC3339Nano),
		},
		Summary:          summary,
		FirstKeptEntryID: firstKeptEntryID,
		TokensBefore:     tokensBefore,
		CompactionDetails: details,
		FromHook:         fromHook,
	}
	sm.appendEntry(entry)
	return entry.ID
}

// AppendBranchSummary appends a branch summary.
func (sm *SessionManager) AppendBranchSummary(fromID, summary string, details interface{}, fromHook *bool) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entry := SessionEntry{
		SessionEntryBase: SessionEntryBase{
			Type:      EntryBranchSummary,
			ID:        generateID(sm.byID),
			ParentID:  sm.leafID,
			Timestamp: time.Now().Format(time.RFC3339Nano),
		},
		FromID:        fromID,
		Summary:       summary,
		BranchDetails: details,
		FromHook:      fromHook,
	}
	sm.appendEntry(entry)
	return entry.ID
}

// AppendCustomEntry appends extension-specific data (not sent to LLM).
func (sm *SessionManager) AppendCustomEntry(customType string, data interface{}) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entry := SessionEntry{
		SessionEntryBase: SessionEntryBase{
			Type:      EntryCustom,
			ID:        generateID(sm.byID),
			ParentID:  sm.leafID,
			Timestamp: time.Now().Format(time.RFC3339Nano),
		},
		CustomType: customType,
		Data:       data,
	}
	sm.appendEntry(entry)
	return entry.ID
}

// AppendCustomMessageEntry appends an extension message into LLM context.
func (sm *SessionManager) AppendCustomMessageEntry(customType string, content json.RawMessage, display bool, details interface{}) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entry := SessionEntry{
		SessionEntryBase: SessionEntryBase{
			Type:      EntryCustomMessage,
			ID:        generateID(sm.byID),
			ParentID:  sm.leafID,
			Timestamp: time.Now().Format(time.RFC3339Nano),
		},
		CustomType: customType,
		Content:    content,
		Display:    display,
		CompactionDetails: details,
	}
	sm.appendEntry(entry)
	return entry.ID
}

// AppendLabelChange sets or clears a label on an entry.
func (sm *SessionManager) AppendLabelChange(targetID string, label *string) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.byID[targetID]; !ok {
		panic(fmt.Sprintf("Entry %s not found", targetID))
	}

	entry := SessionEntry{
		SessionEntryBase: SessionEntryBase{
			Type:      EntryLabel,
			ID:        generateID(sm.byID),
			ParentID:  sm.leafID,
			Timestamp: time.Now().Format(time.RFC3339Nano),
		},
		TargetID: targetID,
		Label:    label,
	}
	sm.appendEntry(entry)

	if label != nil && *label != "" {
		sm.labelsByID[targetID] = label
		sm.labelTSByID[targetID] = entry.Timestamp
	} else {
		delete(sm.labelsByID, targetID)
		delete(sm.labelTSByID, targetID)
	}

	return entry.ID
}

// AppendSessionInfo appends a session name entry.
func (sm *SessionManager) AppendSessionInfo(name string) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	trimmed := strings.TrimSpace(name)
	entry := SessionEntry{
		SessionEntryBase: SessionEntryBase{
			Type:      EntrySessionInfo,
			ID:        generateID(sm.byID),
			ParentID:  sm.leafID,
			Timestamp: time.Now().Format(time.RFC3339Nano),
		},
	}
	if trimmed != "" {
		entry.Name = &trimmed
	}
	sm.appendEntry(entry)
	return entry.ID
}

// ---------------------------------------------------------------------------
// Tree operations
// ---------------------------------------------------------------------------

// Branch moves the leaf pointer to the specified entry.
func (sm *SessionManager) Branch(branchFromID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.byID[branchFromID]; !ok {
		panic(fmt.Sprintf("Entry %s not found", branchFromID))
	}
	sm.leafID = &branchFromID
}

// ResetLeaf moves the leaf pointer to null (before any entries).
func (sm *SessionManager) ResetLeaf() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.leafID = nil
}

// BranchWithSummary branches and appends a branch summary.
func (sm *SessionManager) BranchWithSummary(branchFromID *string, summary string, details interface{}, fromHook *bool) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if branchFromID != nil {
		if _, ok := sm.byID[*branchFromID]; !ok {
			panic(fmt.Sprintf("Entry %s not found", *branchFromID))
		}
	}
	sm.leafID = branchFromID

	fromID := "root"
	if branchFromID != nil {
		fromID = *branchFromID
	}

	entry := SessionEntry{
		SessionEntryBase: SessionEntryBase{
			Type:      EntryBranchSummary,
			ID:        generateID(sm.byID),
			ParentID:  sm.leafID,
			Timestamp: time.Now().Format(time.RFC3339Nano),
		},
		FromID:        fromID,
		Summary:       summary,
		BranchDetails: details,
		FromHook:      fromHook,
	}
	sm.appendEntry(entry)
	return entry.ID
}

// CreateBranchedSession creates a new session containing only the path to the specified leaf.
func (sm *SessionManager) CreateBranchedSession(leafID string) *string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	path := sm.getBranchLocked(&leafID)
	if len(path) == 0 {
		panic(fmt.Sprintf("Entry %s not found", leafID))
	}

	// Filter out labels - we'll recreate them from the resolved map
	var pathNoLabels []SessionEntry
	for _, e := range path {
		if e.Type != EntryLabel {
			pathNoLabels = append(pathNoLabels, e)
		}
	}

	newID := generateUUID()
	timestamp := time.Now().Format(time.RFC3339Nano)
	fileTS := strings.ReplaceAll(strings.ReplaceAll(timestamp, ":", "-"), ".", "-")

	var newFile string
	if sm.persist && sm.sessionDir != "" {
		newFile = filepath.Join(sm.sessionDir, fileTS+"_"+newID+".jsonl")
	}

	parentRef := sm.sessionFile
	header := SessionHeader{
		Type:          "session",
		Version:       CurrentSessionVersion,
		ID:            newID,
		Timestamp:     timestamp,
		CWD:           sm.cwd,
		ParentSession: parentRef,
	}

	// Collect labels for entries in the path
	pathEntryIDs := make(map[string]bool)
	for _, e := range pathNoLabels {
		pathEntryIDs[e.ID] = true
	}

	var labelsToWrite []struct {
		TargetID  string
		Label     string
		Timestamp string
	}
	for targetID, label := range sm.labelsByID {
		if pathEntryIDs[targetID] && label != nil {
			labelsToWrite = append(labelsToWrite, struct {
				TargetID  string
				Label     string
				Timestamp string
			}{targetID, *label, sm.labelTSByID[targetID]})
		}
	}

	// Build label entries
	lastEntryID := pathNoLabels[len(pathNoLabels)-1].ID
	parentID := lastEntryID
	var labelEntries []SessionEntry
	for _, lw := range labelsToWrite {
		lEntry := SessionEntry{
			SessionEntryBase: SessionEntryBase{
				Type:      EntryLabel,
				ID:        generateIDWithSet(pathEntryIDs),
				ParentID:  &parentID,
				Timestamp: lw.Timestamp,
			},
			TargetID: lw.TargetID,
			Label:    &lw.Label,
		}
		pathEntryIDs[lEntry.ID] = true
		labelEntries = append(labelEntries, lEntry)
		parentID = lEntry.ID
	}

	// Rebuild file entries
	headerJSON, _ := json.Marshal(header)
	sm.fileEntries = []json.RawMessage{headerJSON}
	for _, e := range pathNoLabels {
		j, _ := json.Marshal(e)
		sm.fileEntries = append(sm.fileEntries, j)
	}
	for _, e := range labelEntries {
		j, _ := json.Marshal(e)
		sm.fileEntries = append(sm.fileEntries, j)
	}

	sm.sessionID = newID
	sm.sessionFile = &newFile
	sm.buildIndex()

	if sm.persist {
		hasAssistant := false
		for _, e := range pathNoLabels {
			if e.Type == EntryMessage {
				var msg struct{ Role string `json:"role"` }
				if err := json.Unmarshal(e.Message, &msg); err == nil && msg.Role == "assistant" {
					hasAssistant = true
					break
				}
			}
		}
		if hasAssistant {
			sm.rewriteFile()
			sm.flushed = true
		} else {
			sm.flushed = false
		}
		return &newFile
	}

	return nil
}

// NewSession creates a new session, optionally with a parent reference.
func (sm *SessionManager) NewSession(opts *NewSessionOptions) *string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := generateUUID()
	if opts != nil && opts.ID != nil {
		id = *opts.ID
	}
	sm.sessionID = id

	timestamp := time.Now().Format(time.RFC3339Nano)
	header := SessionHeader{
		Type:          "session",
		Version:       CurrentSessionVersion,
		ID:            id,
		Timestamp:     timestamp,
		CWD:           sm.cwd,
		ParentSession: nil,
	}
	if opts != nil {
		header.ParentSession = opts.ParentSession
	}

	headerJSON, _ := json.Marshal(header)
	sm.fileEntries = []json.RawMessage{headerJSON}
	sm.byID = make(map[string]*SessionEntry)
	sm.labelsByID = make(map[string]*string)
	sm.labelTSByID = make(map[string]string)
	sm.leafID = nil
	sm.flushed = false

	if sm.persist && sm.sessionDir != "" {
		fileTS := strings.ReplaceAll(strings.ReplaceAll(timestamp, ":", "-"), ".", "-")
		file := filepath.Join(sm.sessionDir, fileTS+"_"+id+".jsonl")
		sm.sessionFile = &file
	}

	return sm.sessionFile
}

// ForkFrom creates a new session by copying history from a source session.
func ForkFrom(sourcePath, cwd, sessionDir string) *SessionManager {
	sourceEntries := loadEntriesFromFile(sourcePath)
	if len(sourceEntries) == 0 {
		panic(fmt.Sprintf("Cannot fork: source session file is empty or invalid: %s", sourcePath))
	}

	dir := sessionDir
	if dir == "" {
		dir = GetDefaultSessionDir(cwd)
	}
	os.MkdirAll(dir, 0755)

	newID := generateUUID()
	timestamp := time.Now().Format(time.RFC3339Nano)
	fileTS := strings.ReplaceAll(strings.ReplaceAll(timestamp, ":", "-"), ".", "-")
	newFile := filepath.Join(dir, fileTS+"_"+newID+".jsonl")

	newHeader := SessionHeader{
		Type:          "session",
		Version:       CurrentSessionVersion,
		ID:            newID,
		Timestamp:     timestamp,
		CWD:           cwd,
		ParentSession: &sourcePath,
	}

	// Write header
	headerJSON, _ := json.Marshal(newHeader)
	f, err := os.Create(newFile)
	if err != nil {
		panic(fmt.Sprintf("Cannot create forked session: %v", err))
	}
	f.Write(headerJSON)
	f.Write([]byte("\n"))

	// Copy non-header entries
	for _, raw := range sourceEntries {
		var base SessionEntryBase
		if err := json.Unmarshal(raw, &base); err == nil && base.Type == "session" {
			continue
		}
		f.Write(raw)
		f.Write([]byte("\n"))
	}
	f.Close()

	return OpenSession(newFile, dir, nil)
}

// ---------------------------------------------------------------------------
// BuildSessionContext
// ---------------------------------------------------------------------------

// BuildSessionContext resolves the message list from root to current leaf.
func (sm *SessionManager) BuildSessionContext() *SessionContext {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entries := sm.GetEntriesLocked()
	return BuildSessionContextFromEntries(entries, sm.leafID, sm.byID)
}

// BuildSessionContextFromEntries builds a SessionContext from a flat entry list.
func BuildSessionContextFromEntries(entries []SessionEntry, leafID *string, byID map[string]*SessionEntry) *SessionContext {
	if byID == nil {
		byID = make(map[string]*SessionEntry)
		for i := range entries {
			byID[entries[i].ID] = &entries[i]
		}
	}

	// Find leaf
	var leaf *SessionEntry
	if leafID != nil {
		leaf = byID[*leafID]
	} else if len(entries) > 0 {
		leaf = &entries[len(entries)-1]
	}
	if leaf == nil {
		return &SessionContext{Messages: []ai.Message{}, ThinkingLevel: "off", Model: nil}
	}

	// Walk from leaf to root
	var path []SessionEntry
	current := leaf
	for current != nil {
		path = append([]SessionEntry{*current}, path...)
		if current.ParentID == nil {
			break
		}
		current = byID[*current.ParentID]
	}

	// Extract settings and find compaction
	thinkingLevel := "off"
	var model *ModelRef
	var compaction *CompactionEntry

	for i := range path {
		entry := path[i]
		switch entry.Type {
		case EntryThinkingLevel:
			thinkingLevel = entry.ThinkingLevel
		case EntryModelChange:
			model = &ModelRef{Provider: entry.Provider, ModelID: entry.ModelID}
		case EntryMessage:
			var msg struct {
				Role     string `json:"role"`
				Provider string `json:"provider"`
				Model    string `json:"model"`
			}
			if err := json.Unmarshal(entry.Message, &msg); err == nil && msg.Role == "assistant" {
				model = &ModelRef{Provider: msg.Provider, ModelID: msg.Model}
			}
		case EntryCompaction:
			ce := CompactionEntry{
				Summary:          entry.Summary,
				FirstKeptEntryID: entry.FirstKeptEntryID,
				TokensBefore:     entry.TokensBefore,
			}
			compaction = &ce
		}
	}

	// Build messages
	var msgs []ai.Message
	appendMsg := func(entry SessionEntry) {
		switch entry.Type {
		case EntryMessage:
			msg := unmarshalAIMessage(entry.Message)
			if msg != nil {
				msgs = append(msgs, msg)
			}
		case EntryCustomMessage:
			cm := messages.CustomMessage{
				Role:       "custom",
				CustomType: entry.CustomType,
				Display:    entry.Display,
				Timestamp:  time.Now().UnixMilli(),
			}
			if err := json.Unmarshal(entry.Content, &cm.Content); err != nil {
				cm.Content = string(entry.Content)
			}
			// Use the custom message as a user message for LLM
			userMsg := ai.UserMessage{
				Content:   []ai.Content{ai.TextContent{Text: cm.Content}},
				Timestamp: cm.Timestamp,
			}
			msgs = append(msgs, userMsg)
		case EntryBranchSummary:
			if entry.Summary != "" {
				msgs = append(msgs, messages.CreateBranchSummaryMessage(entry.Summary, entry.FromID))
			}
		}
	}

	if compaction != nil {
		// Emit compaction summary
		msgs = append(msgs, messages.CreateCompactionSummaryMessage(compaction.Summary, compaction.TokensBefore))

		// Find compaction index
		compactionIdx := -1
		for i, e := range path {
			if e.Type == EntryCompaction && e.Summary == compaction.Summary {
				compactionIdx = i
				break
			}
		}

		// Emit kept messages (from firstKeptEntryId to compaction)
		foundFirst := false
		if compactionIdx >= 0 {
			for i := 0; i < compactionIdx; i++ {
				if path[i].ID == compaction.FirstKeptEntryID {
					foundFirst = true
				}
				if foundFirst {
					appendMsg(path[i])
				}
			}
			// Emit messages after compaction
			for i := compactionIdx + 1; i < len(path); i++ {
				appendMsg(path[i])
			}
		}
	} else {
		for i := range path {
			appendMsg(path[i])
		}
	}

	return &SessionContext{Messages: msgs, ThinkingLevel: thinkingLevel, Model: model}
}

// ---------------------------------------------------------------------------
// List sessions
// ---------------------------------------------------------------------------

// ListSessions returns all sessions for a directory, sorted by modified time.
func ListSessions(cwd, sessionDir string) ([]SessionInfo, error) {
	dir := sessionDir
	if dir == "" {
		dir = GetDefaultSessionDir(cwd)
	}
	return listSessionsFromDir(dir)
}

// ListAllSessions returns all sessions across all project directories.
func ListAllSessions() ([]SessionInfo, error) {
	sessionsDir := GetSessionsDir()
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil, err
	}

	var allInfos []SessionInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(sessionsDir, entry.Name())
		infos, err := listSessionsFromDir(dir)
		if err != nil {
			continue
		}
		allInfos = append(allInfos, infos...)
	}

	sort.Slice(allInfos, func(i, j int) bool {
		return allInfos[i].Modified.After(allInfos[j].Modified)
	})
	return allInfos, nil
}

// ---------------------------------------------------------------------------
// Internal methods
// ---------------------------------------------------------------------------

func (sm *SessionManager) setSessionFile(path string) {
	sm.sessionFile = &path
}

func (sm *SessionManager) loadFromFile(path string) {
	sm.fileEntries = loadEntriesFromFile(path)
	if len(sm.fileEntries) == 0 {
		// Empty or corrupted file - start fresh
		explicitPath := path
		sm.NewSession(nil)
		sm.sessionFile = &explicitPath
		sm.rewriteFile()
		sm.flushed = true
		return
	}

	// Find header
	var header SessionHeader
	for _, raw := range sm.fileEntries {
		if err := json.Unmarshal(raw, &header); err == nil && header.Type == "session" {
			break
		}
	}
	sm.sessionID = header.ID

	// Migrate if needed
	if migrateEntries(sm.fileEntries) {
		sm.rewriteFile()
	}

	sm.buildIndex()
	sm.flushed = true
}

func (sm *SessionManager) buildIndex() {
	sm.byID = make(map[string]*SessionEntry)
	sm.labelsByID = make(map[string]*string)
	sm.labelTSByID = make(map[string]string)
	sm.leafID = nil

	for _, raw := range sm.fileEntries {
		var base SessionEntryBase
		if err := json.Unmarshal(raw, &base); err != nil || base.Type == "session" {
			continue
		}
		var entry SessionEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}
		sm.byID[entry.ID] = &entry
		sm.leafID = &entry.ID

		if entry.Type == EntryLabel {
			if entry.Label != nil && *entry.Label != "" {
				sm.labelsByID[entry.TargetID] = entry.Label
				sm.labelTSByID[entry.TargetID] = entry.Timestamp
			} else {
				delete(sm.labelsByID, entry.TargetID)
				delete(sm.labelTSByID, entry.TargetID)
			}
		}
	}
}

func (sm *SessionManager) appendEntry(entry SessionEntry) {
	raw, _ := json.Marshal(entry)
	sm.fileEntries = append(sm.fileEntries, raw)
	sm.byID[entry.ID] = &entry
	sm.leafID = &entry.ID
	sm.persistEntry(raw)
}

func (sm *SessionManager) persistEntry(raw json.RawMessage) {
	if !sm.persist || sm.sessionFile == nil {
		return
	}

	// Check if there's an assistant message
	hasAssistant := false
	for _, e := range sm.byID {
		if e.Type == EntryMessage {
			var msg struct{ Role string `json:"role"` }
			if err := json.Unmarshal(e.Message, &msg); err == nil && msg.Role == "assistant" {
				hasAssistant = true
				break
			}
		}
	}

	if !hasAssistant {
		sm.flushed = false
		return
	}

	if !sm.flushed {
		// Write all entries
		for _, e := range sm.fileEntries {
			sm.appendToFile(e)
		}
		sm.flushed = true
	} else {
		sm.appendToFile(raw)
	}
}

func (sm *SessionManager) appendToFile(data []byte) {
	if sm.sessionFile == nil {
		return
	}
	f, err := os.OpenFile(*sm.sessionFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(data)
	f.Write([]byte("\n"))
}

func (sm *SessionManager) rewriteFile() {
	if !sm.persist || sm.sessionFile == nil {
		return
	}
	f, err := os.Create(*sm.sessionFile)
	if err != nil {
		return
	}
	defer f.Close()
	for _, raw := range sm.fileEntries {
		f.Write(raw)
		f.Write([]byte("\n"))
	}
}

// GetEntriesLocked returns entries while holding the lock.
func (sm *SessionManager) GetEntriesLocked() []SessionEntry {
	var result []SessionEntry
	for _, raw := range sm.fileEntries {
		var base SessionEntryBase
		if err := json.Unmarshal(raw, &base); err != nil || base.Type == "session" {
			continue
		}
		var entry SessionEntry
		if err := json.Unmarshal(raw, &entry); err == nil {
			result = append(result, entry)
		}
	}
	return result
}

func (sm *SessionManager) getBranchLocked(fromID *string) []SessionEntry {
	var path []SessionEntry
	current := fromID
	if current == nil {
		current = sm.leafID
	}
	for current != nil {
		entry, ok := sm.byID[*current]
		if !ok {
			break
		}
		path = append([]SessionEntry{*entry}, path...)
		current = entry.ParentID
	}
	return path
}

// ---------------------------------------------------------------------------
// File I/O helpers
// ---------------------------------------------------------------------------

func loadEntriesFromFile(path string) []json.RawMessage {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var entries []json.RawMessage
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entries = append(entries, json.RawMessage(line))
	}
	return entries
}

func isValidSessionFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	lines := strings.Split(string(buf[:n]), "\n")
	if len(lines) == 0 {
		return false
	}

	var header struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &header); err != nil {
		return false
	}
	return header.Type == "session" && header.ID != ""
}

// FindMostRecentSession returns the path to the most recent session file.
func FindMostRecentSession(sessionDir string) string {
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return ""
	}

	type fileInfo struct {
		path  string
		mtime time.Time
	}

	var files []fileInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		path := filepath.Join(sessionDir, entry.Name())
		if !isValidSessionFile(path) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{path: path, mtime: info.ModTime()})
	}

	if len(files) == 0 {
		return ""
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].mtime.After(files[j].mtime)
	})
	return files[0].path
}

func listSessionsFromDir(dir string) ([]SessionInfo, error) {
	entries, err := os.ReadDir(dir)
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
		path := filepath.Join(dir, entry.Name())
		info := buildSessionInfo(path)
		if info != nil {
			infos = append(infos, *info)
		}
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Modified.After(infos[j].Modified)
	})
	return infos, nil
}

func buildSessionInfo(filePath string) *SessionInfo {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	var allEntries []json.RawMessage
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		allEntries = append(allEntries, json.RawMessage(line))
	}
	if len(allEntries) == 0 {
		return nil
	}

	var header SessionHeader
	if err := json.Unmarshal(allEntries[0], &header); err != nil || header.Type != "session" {
		return nil
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		return nil
	}

	messageCount := 0
	firstMessage := ""
	var allText []string
	var name *string

	for _, raw := range allEntries[1:] {
		var base SessionEntryBase
		if err := json.Unmarshal(raw, &base); err != nil {
			continue
		}

		// Extract session name
		if base.Type == EntrySessionInfo {
			var si SessionInfoEntry
			if err := json.Unmarshal(raw, &si); err == nil && si.Name != nil {
				trimmed := strings.TrimSpace(*si.Name)
				if trimmed != "" {
					name = &trimmed
				} else {
					name = nil
				}
			}
			continue
		}

		if base.Type != EntryMessage {
			continue
		}

		var msgEntry struct {
			Message json.RawMessage `json:"message"`
		}
		if err := json.Unmarshal(raw, &msgEntry); err != nil {
			continue
		}

		messageCount++

		var msg struct {
			Role    string `json:"role"`
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(msgEntry.Message, &msg); err != nil {
			continue
		}
		if msg.Role != "user" && msg.Role != "assistant" {
			continue
		}

		text := extractTextFromContent(msg.Content)
		if text == "" {
			continue
		}
		allText = append(allText, text)
		if firstMessage == "" && msg.Role == "user" {
			firstMessage = text
		}
	}

	created, _ := time.Parse(time.RFC3339Nano, header.Timestamp)
	if created.IsZero() {
		created = stat.ModTime()
	}

	modified := stat.ModTime()
	// Try to get last activity time from messages
	for i := len(allEntries) - 1; i >= 0; i-- {
		var base SessionEntryBase
		if err := json.Unmarshal(allEntries[i], &base); err != nil || base.Type != EntryMessage {
			continue
		}
		t, err := time.Parse(time.RFC3339Nano, base.Timestamp)
		if err == nil && !t.IsZero() {
			modified = t
			break
		}
	}

	result := &SessionInfo{
		Path:         filePath,
		ID:           header.ID,
		CWD:          header.CWD,
		Created:      created,
		Modified:     modified,
		MessageCount: messageCount,
		FirstMessage: firstMessage,
	}
	if name != nil {
		result.Name = *name
	}
	if header.ParentSession != nil {
		result.ParentSessionPath = header.ParentSession
	}
	if result.FirstMessage == "" {
		result.FirstMessage = "(no messages)"
	}
	if len(allText) > 0 {
		result.AllMessagesText = strings.Join(allText, " ")
	}
	return result
}

func extractTextFromContent(content json.RawMessage) string {
	// Try string content first
	var s string
	if err := json.Unmarshal(content, &s); err == nil {
		return s
	}

	// Try array of content blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(content, &blocks); err == nil {
		var texts []string
		for _, b := range blocks {
			if b.Type == "text" {
				texts = append(texts, b.Text)
			}
		}
		return strings.Join(texts, " ")
	}

	return ""
}

// ---------------------------------------------------------------------------
// Migration
// ---------------------------------------------------------------------------

func migrateEntries(entries []json.RawMessage) bool {
	var header SessionHeader
	headerIdx := -1
	for i, raw := range entries {
		if err := json.Unmarshal(raw, &header); err == nil && header.Type == "session" {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return false
	}

	version := header.Version
	if version >= CurrentSessionVersion {
		return false
	}

	if version < 2 {
		migrateV1ToV2(entries, headerIdx)
	}
	if version < 3 {
		migrateV2ToV3(entries, headerIdx)
	}
	return true
}

func migrateV1ToV2(entries []json.RawMessage, headerIdx int) {
	// Update header version
	var header map[string]interface{}
	json.Unmarshal(entries[headerIdx], &header)
	header["version"] = 2
	entries[headerIdx], _ = json.Marshal(header)

	// Add id/parentId tree structure
	var prevID *string
	idSet := make(map[string]bool)

	for i, raw := range entries {
		if i == headerIdx {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}
		if _, hasID := entry["id"]; !hasID {
			id := generateUUIDShort(idSet)
			entry["id"] = id
			entry["parentId"] = prevID
			prevID = &id
		} else {
			if pid, ok := entry["parentId"]; ok {
				if pid == nil {
					entry["parentId"] = prevID
				}
			} else {
				entry["parentId"] = prevID
			}
			if id, ok := entry["id"].(string); ok {
				prevID = &id
			}
		}
		entries[i], _ = json.Marshal(entry)
	}
}

func migrateV2ToV3(entries []json.RawMessage, headerIdx int) {
	// Update header version
	var header map[string]interface{}
	json.Unmarshal(entries[headerIdx], &header)
	header["version"] = 3
	entries[headerIdx], _ = json.Marshal(header)

	// Rename hookMessage role to custom
	for i, raw := range entries {
		if i == headerIdx {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}
		if entry["type"] == "message" {
			if msg, ok := entry["message"].(map[string]interface{}); ok {
				if role, ok := msg["role"].(string); ok && role == "hookMessage" {
					msg["role"] = "custom"
					entries[i], _ = json.Marshal(entry)
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// ID generation
// ---------------------------------------------------------------------------

func generateUUID() string {
	b := make([]byte, 16)
	io.ReadFull(rand.Reader, b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func generateUUIDShort(existing map[string]bool) string {
	for i := 0; i < 100; i++ {
		id := generateUUID()[:8]
		if !existing[id] {
			existing[id] = true
			return id
		}
	}
	return generateUUID()[:8]
}

func generateID(byID map[string]*SessionEntry) string {
	existing := make(map[string]bool)
	for id := range byID {
		existing[id] = true
	}
	return generateUUIDShort(existing)
}

func generateIDWithSet(existing map[string]bool) string {
	return generateUUIDShort(existing)
}

// ---------------------------------------------------------------------------
// Path helpers
// ---------------------------------------------------------------------------

// GetDefaultSessionDir returns the default session directory for a cwd.
func GetDefaultSessionDir(cwd string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	safePath := "--" + strings.TrimLeft(strings.ReplaceAll(strings.ReplaceAll(cwd, "/", "-"), "\\", "-"), "-") + "--"
	return filepath.Join(homeDir, ".pi", "sessions", safePath)
}

// GetSessionsDir returns the top-level sessions directory.
func GetSessionsDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".pi", "sessions")
}

// GetLatestCompactionEntry finds the most recent compaction entry.
func GetLatestCompactionEntry(entries []SessionEntry) *CompactionEntry {
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Type == EntryCompaction {
			return &CompactionEntry{
				Summary:          entries[i].Summary,
				FirstKeptEntryID: entries[i].FirstKeptEntryID,
				TokensBefore:     entries[i].TokensBefore,
			}
		}
	}
	return nil
}

// ParseSessionEntries parses JSONL content into raw entries.
func ParseSessionEntries(content string) []json.RawMessage {
	var entries []json.RawMessage
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entries = append(entries, json.RawMessage(line))
	}
	return entries
}

// unmarshalAIMessage attempts to unmarshal a raw JSON message into an ai.Message.
func unmarshalAIMessage(raw json.RawMessage) ai.Message {
	var header struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return nil
	}

	switch ai.MessageRole(header.Role) {
	case ai.RoleUser:
		var msg ai.UserMessage
		if err := json.Unmarshal(raw, &msg); err == nil {
			return msg
		}
	case ai.RoleAssistant:
		var msg ai.AssistantMessage
		if err := json.Unmarshal(raw, &msg); err == nil {
			return msg
		}
	case ai.RoleTool:
		var msg ai.ToolResultMessage
		if err := json.Unmarshal(raw, &msg); err == nil {
			return msg
		}
	}
	return nil
}

// Ensure messages package functions exist
var _ = messages.CreateBranchSummaryMessage
var _ = messages.CreateCompactionSummaryMessage

// AppendMessageByJSON appends a raw JSON message as child of current leaf.
// This is useful when you have a pre-serialized message.
func (sm *SessionManager) AppendMessageByJSON(msgJSON json.RawMessage) string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	entry := SessionEntry{
		SessionEntryBase: SessionEntryBase{
			Type:      EntryMessage,
			ID:        generateID(sm.byID),
			ParentID:  sm.leafID,
			Timestamp: time.Now().Format(time.RFC3339Nano),
		},
		Message: msgJSON,
	}
	sm.appendEntry(entry)
	return entry.ID
}

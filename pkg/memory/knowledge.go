package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// KnowledgeScope defines the scope of a knowledge entry.
type KnowledgeScope string

const (
	ScopeGlobal  KnowledgeScope = "global"
	ScopeProject KnowledgeScope = "project"
	ScopeSession KnowledgeScope = "session"
)

// KnowledgeEntry is a single memory/knowledge item.
type KnowledgeEntry struct {
	ID        string                 `json:"id"`
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	Tags      []string               `json:"tags"`
	Scope     KnowledgeScope         `json:"scope"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	AccessCount  int                 `json:"accessCount"`
	LastAccessed time.Time           `json:"lastAccessed,omitempty"`
}

// KnowledgeBase manages stored knowledge entries with disk persistence.
type KnowledgeBase struct {
	entries    []*KnowledgeEntry
	entryIndex map[string]*KnowledgeEntry
	tagIndex   map[string][]*KnowledgeEntry
	scopeIndex map[KnowledgeScope][]*KnowledgeEntry
	storePath  string
	mu         sync.RWMutex
}

// NewKnowledgeBase creates a new knowledge base with the given store path.
func NewKnowledgeBase(storePath string) (*KnowledgeBase, error) {
	if storePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("home dir: %w", err)
		}
		storePath = filepath.Join(home, ".pi", "memory.db")
	}

	kb := &KnowledgeBase{
		entries:    make([]*KnowledgeEntry, 0),
		entryIndex: make(map[string]*KnowledgeEntry),
		tagIndex:   make(map[string][]*KnowledgeEntry),
		scopeIndex: make(map[KnowledgeScope][]*KnowledgeEntry),
		storePath:  storePath,
	}

	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
		return nil, err
	}

	// Load existing entries; non-fatal if not found
	if err := kb.load(); err != nil {
		// Start fresh
	}

	return kb, nil
}

// Store adds or updates a knowledge entry.
func (kb *KnowledgeBase) Store(entry *KnowledgeEntry) error {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	now := time.Now()

	if existing, found := kb.entryIndex[entry.ID]; found {
		existing.Title = entry.Title
		existing.Content = entry.Content
		existing.Tags = entry.Tags
		existing.UpdatedAt = now
		existing.Metadata = entry.Metadata
		return kb.save()
	}

	entry.ID = fmt.Sprintf("kb_%d", now.UnixNano())
	entry.CreatedAt = now
	entry.UpdatedAt = now
	if entry.Scope == "" {
		entry.Scope = ScopeProject
	}

	kb.entries = append(kb.entries, entry)
	kb.entryIndex[entry.ID] = entry
	for _, tag := range entry.Tags {
		kb.tagIndex[tag] = append(kb.tagIndex[tag], entry)
	}
	kb.scopeIndex[entry.Scope] = append(kb.scopeIndex[entry.Scope], entry)

	return kb.save()
}

// Search searches for knowledge by keywords, tags, and scope.
func (kb *KnowledgeBase) Search(keywords string, tags []string, scope KnowledgeScope) []*KnowledgeEntry {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	keyLower := strings.ToLower(strings.TrimSpace(keywords))
	var results []*KnowledgeEntry

	for _, entry := range kb.entries {
		if scope != "" && entry.Scope != scope {
			continue
		}

		if len(tags) > 0 {
			tagMatch := false
			for _, tag := range tags {
				for _, et := range entry.Tags {
					if strings.EqualFold(tag, et) {
						tagMatch = true
						break
					}
				}
				if tagMatch {
					break
				}
			}
			if !tagMatch {
				continue
			}
		}

		score := 0.0
		if keyLower != "" {
			titleLower := strings.ToLower(entry.Title)
			contentLower := strings.ToLower(entry.Content)
			if strings.Contains(titleLower, keyLower) {
				score += 2.0
			}
			if strings.Contains(contentLower, keyLower) {
				score += 1.0
			}
			if score == 0 {
				continue
			}
		} else {
			score = 1.0
		}

		entry.AccessCount++
		entry.LastAccessed = time.Now()
		results = append(results, entry)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].AccessCount > results[j].AccessCount
	})

	return results
}

// Get retrieves an entry by ID.
func (kb *KnowledgeBase) Get(id string) (*KnowledgeEntry, bool) {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	entry, ok := kb.entryIndex[id]
	if ok {
		entry.AccessCount++
		entry.LastAccessed = time.Now()
	}
	return entry, ok
}

// Delete removes an entry by ID.
func (kb *KnowledgeBase) Delete(id string) error {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	entry, ok := kb.entryIndex[id]
	if !ok {
		return fmt.Errorf("entry not found: %s", id)
	}

	for _, tag := range entry.Tags {
		if entries, ok := kb.tagIndex[tag]; ok {
			for i, e := range entries {
				if e == entry {
					kb.tagIndex[tag] = append(entries[:i], entries[i+1:]...)
					break
				}
			}
		}
	}
	if entries, ok := kb.scopeIndex[entry.Scope]; ok {
		for i, e := range entries {
			if e == entry {
				kb.scopeIndex[entry.Scope] = append(entries[:i], entries[i+1:]...)
				break
			}
		}
	}
	for i, e := range kb.entries {
		if e == entry {
			kb.entries = append(kb.entries[:i], kb.entries[i+1:]...)
			break
		}
	}
	delete(kb.entryIndex, id)

	return kb.save()
}

// List returns all entries, optionally filtered by scope.
func (kb *KnowledgeBase) List(scope KnowledgeScope) []*KnowledgeEntry {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	if scope == "" {
		return kb.entries
	}
	return kb.scopeIndex[scope]
}

// BuildContextForAgent builds a context string from relevant memories for agent prompts.
func (kb *KnowledgeBase) BuildContextForAgent(topics []string) string {
	entries := kb.Search(strings.Join(topics, " "), nil, "")
	if len(entries) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("\n## Relevant Context from Memory\n\n")
	for _, entry := range entries {
		buf.WriteString(fmt.Sprintf("### %s (scope: %s)\n\n%s\n\n", entry.Title, entry.Scope, entry.Content))
	}
	return buf.String()
}

// Stats returns memory statistics.
func (kb *KnowledgeBase) Stats() map[string]interface{} {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	stats := map[string]interface{}{
		"totalEntries": len(kb.entries),
		"totalTags":    len(kb.tagIndex),
	}
	for scope, entries := range kb.scopeIndex {
		stats["byScope_"+string(scope)] = len(entries)
	}
	return stats
}

// ExportAll returns a copy of all entries.
func (kb *KnowledgeBase) ExportAll() []*KnowledgeEntry {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	export := make([]*KnowledgeEntry, len(kb.entries))
	for i, entry := range kb.entries {
		cpy := *entry
		export[i] = &cpy
	}
	return export
}

// ImportEntries imports entries into the knowledge base.
func (kb *KnowledgeBase) ImportEntries(entries []*KnowledgeEntry) error {
	kb.mu.Lock()
	defer kb.mu.Unlock()

	for _, entry := range entries {
		if entry.ID == "" {
			entry.ID = fmt.Sprintf("mem_%d", time.Now().UnixNano())
		}
		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = time.Now()
		}
		if entry.UpdatedAt.IsZero() {
			entry.UpdatedAt = time.Now()
		}
		if entry.Scope == "" {
			entry.Scope = ScopeProject
		}

		if existing, exists := kb.entryIndex[entry.ID]; exists {
			existing.Title = entry.Title
			existing.Content = entry.Content
			existing.Tags = entry.Tags
			existing.Scope = entry.Scope
			existing.UpdatedAt = time.Now()
			existing.Metadata = entry.Metadata
		} else {
			kb.entries = append(kb.entries, entry)
			kb.entryIndex[entry.ID] = entry
		}
	}

	// Rebuild indexes
	kb.tagIndex = make(map[string][]*KnowledgeEntry)
	kb.scopeIndex = make(map[KnowledgeScope][]*KnowledgeEntry)
	for _, entry := range kb.entries {
		for _, tag := range entry.Tags {
			kb.tagIndex[tag] = append(kb.tagIndex[tag], entry)
		}
		kb.scopeIndex[entry.Scope] = append(kb.scopeIndex[entry.Scope], entry)
	}

	return kb.save()
}

func (kb *KnowledgeBase) load() error {
	data, err := os.ReadFile(kb.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var entries []*KnowledgeEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	kb.entries = entries
	for _, entry := range entries {
		kb.entryIndex[entry.ID] = entry
		for _, tag := range entry.Tags {
			kb.tagIndex[tag] = append(kb.tagIndex[tag], entry)
		}
		kb.scopeIndex[entry.Scope] = append(kb.scopeIndex[entry.Scope], entry)
	}
	return nil
}

func (kb *KnowledgeBase) save() error {
	data, err := json.MarshalIndent(kb.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(kb.storePath, data, 0o644)
}

// Save explicitly persists entries to disk.
func (kb *KnowledgeBase) Save() error {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	return kb.save()
}

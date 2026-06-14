package workspaces

import (
	"sync"
)

// Workspace holds metadata for a named workspace.
type Workspace struct {
	Name    string
	Root    string
	Setting map[string]string
}

// Manager maintains a thread‑safe set of named workspaces.
type Manager struct {
	mu   sync.RWMutex
	ws   map[string]*Workspace
}

// NewManager creates an empty workspace manager.
func NewManager() *Manager {
	return &Manager{ws: make(map[string]*Workspace)}
}

// Add registers a workspace. If a workspace with the same name exists it is overwritten.
func (m *Manager) Add(ws *Workspace) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ws[ws.Name] = ws
}

// Get returns the workspace by name. The bool reports existence.
func (m *Manager) Get(name string) (*Workspace, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	w, ok := m.ws[name]
	return w, ok
}

// Remove deletes a workspace by name.
func (m *Manager) Remove(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.ws, name)
}

package subagents

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "sync"
)

// Subagent represents a spawned sub-agent process.
type Subagent struct {
    ID      string
    Cmd     *exec.Cmd
    stdin   *os.File
    stdout  *os.File
    // For simplicity we will use stdin/stdout pipes; in a real implementation we would use a structured protocol.
}

// Manager manages multiple subagents.
type Manager struct {
    mu        sync.Mutex
    subagents map[string]*Subagent
    nextID    int
}

// NewManager creates a new subagent manager.
func NewManager() *Manager {
    return &Manager{subagents: make(map[string]*Subagent)}
}

// StartSubagent spawns a new subagent process using the provided command line arguments.
// The binary path should point to the same executable (or another) that implements the subagent protocol.
func (m *Manager) StartSubagent(ctx context.Context, binaryPath string, args ...string) (*Subagent, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    // Prepare command
    cmd := exec.CommandContext(ctx, binaryPath, args...)
    // Create pipes for stdin/stdout
    stdin, err := cmd.StdinPipe()
    if err != nil {
        return nil, fmt.Errorf("subagent stdin pipe: %w", err)
    }
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, fmt.Errorf("subagent stdout pipe: %w", err)
    }
    // For debugging, inherit stderr
    cmd.Stderr = os.Stderr

    // Start process
    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("subagent start: %w", err)
    }

    id := fmt.Sprintf("sub-%d", m.nextID)
    m.nextID++
    sub := &Subagent{ID: id, Cmd: cmd, stdin: stdin.(*os.File), stdout: stdout.(*os.File)}
    m.subagents[id] = sub
    return sub, nil
}

// SendMessage writes a message to the subagent's stdin.
func (s *Subagent) SendMessage(msg []byte) error {
    if s.stdin == nil {
        return fmt.Errorf("subagent stdin not available")
    }
    _, err := s.stdin.Write(append(msg, '\n'))
    return err
}

// ReceiveMessage reads a single line response from the subagent's stdout.
func (s *Subagent) ReceiveMessage() (string, error) {
    if s.stdout == nil {
        return "", fmt.Errorf("subagent stdout not available")
    }
    buf := make([]byte, 4096)
    n, err := s.stdout.Read(buf)
    if err != nil {
        return "", err
    }
    return string(buf[:n]), nil
}

// Stop terminates the subagent process.
func (s *Subagent) Stop() error {
    if s.Cmd == nil {
        return fmt.Errorf("subagent not started")
    }
    if err := s.Cmd.Process.Kill(); err != nil {
        return fmt.Errorf("kill subagent: %w", err)
    }
    return s.Cmd.Wait()
}

// List returns IDs of active subagents.
func (m *Manager) List() []string {
    m.mu.Lock()
    defer m.mu.Unlock()
    ids := make([]string, 0, len(m.subagents))
    for id := range m.subagents {
        ids = append(ids, id)
    }
    return ids
}

// Get returns a subagent by ID.
func (m *Manager) Get(id string) (*Subagent, bool) {
    m.mu.Lock()
    defer m.mu.Unlock()
    sub, ok := m.subagents[id]
    return sub, ok
}

// Remove stops and deletes a subagent from the manager.
func (m *Manager) Remove(id string) error {
    sub, ok := m.Get(id)
    if !ok {
        return fmt.Errorf("subagent %s not found", id)
    }
    if err := sub.Stop(); err != nil {
        return err
    }
    m.mu.Lock()
    delete(m.subagents, id)
    m.mu.Unlock()
    return nil
}

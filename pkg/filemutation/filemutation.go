package filemutation

import (
	"os"
	"path/filepath"
	"sync"
)

// FileMutationQueue serializes file mutation operations targeting the same file.
// Operations for different files still run in parallel.
type FileMutationQueue struct {
	mu     sync.Mutex
	queues map[string]chan struct{}
}

// NewFileMutationQueue creates a new file mutation queue.
func NewFileMutationQueue() *FileMutationQueue {
	return &FileMutationQueue{
		queues: make(map[string]chan struct{}),
	}
}

// getMutationQueueKey returns a unique key for a file path.
func (q *FileMutationQueue) getMutationQueueKey(filePath string) string {
	resolved, err := filepath.Abs(filePath)
	if err != nil {
		resolved = filePath
	}
	// Try to resolve symlinks
	if realPath, err := os.Readlink(resolved); err == nil {
		return realPath
	}
	return resolved
}

// WithFileMutationQueue serializes file mutation operations targeting the same file.
func (q *FileMutationQueue) WithFileMutationQueue(filePath string, fn func() error) error {
	key := q.getMutationQueueKey(filePath)

	q.mu.Lock()
	if _, ok := q.queues[key]; !ok {
		q.queues[key] = make(chan struct{}, 1)
		q.queues[key] <- struct{}{} // Initialize with token
	}
	ch := q.queues[key]
	q.mu.Unlock()

	// Acquire lock
	<-ch

	// Execute
	err := fn()

	// Release lock
	ch <- struct{}{}

	return err
}

// Dispose cleans up all queues.
func (q *FileMutationQueue) Dispose() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queues = make(map[string]chan struct{})
}

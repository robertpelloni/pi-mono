package agent

import (
	"path/filepath"
	"sync"
)

// FileMutationQueue ensures that destructive file operations (like edit/write/patch)
// targeting the same file are executed sequentially to prevent race conditions and data corruption.
// Operations for different files run concurrently.
type FileMutationQueue struct {
	mu     sync.Mutex
	queues map[string]chan struct{}
}

var (
	globalMutationQueue = &FileMutationQueue{
		queues: make(map[string]chan struct{}),
	}
)

// getMutationQueueKey resolves the absolute path to use as a unique key for the file.
func getMutationQueueKey(filePath string) string {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return filePath
	}

	// Try to resolve symlinks like realpathSync in TS
	evalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return absPath
	}
	return evalPath
}

// WithFileMutationQueue executes the provided function fn safely,
// ensuring no other file operations can occur on the same filePath concurrently.
func WithFileMutationQueue[T any](filePath string, fn func() (T, error)) (T, error) {
	key := getMutationQueueKey(filePath)

	globalMutationQueue.mu.Lock()
	queue, exists := globalMutationQueue.queues[key]
	if !exists {
		// Create a buffered channel of size 1 to act as a mutex for this specific file
		queue = make(chan struct{}, 1)
		globalMutationQueue.queues[key] = queue
	}
	globalMutationQueue.mu.Unlock()

	// Acquire the lock for this specific file
	queue <- struct{}{}

	// Ensure we release the lock when we're done
	defer func() {
		<-queue

		// Optional cleanup: we could remove the channel from the map if it's empty,
		// but tracking waiters without race conditions requires more complex locking.
		// Leaving the allocated channel in the map is a minor memory trade-off for simplicity and safety.
	}()

	return fn()
}

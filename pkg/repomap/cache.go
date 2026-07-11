package repomap

import (
	"sync"
	"time"
)

// FileCacheEntry holds the computed data for a single file along with its last modified time.
type FileCacheEntry struct {
	ModTime     time.Time
	Symbols     []Symbol
	Identifiers map[string]int
}

var (
	// globalCache maps absolute file paths to their parsed FileCacheEntry
	globalCache = make(map[string]FileCacheEntry)
	cacheMutex  sync.RWMutex
)

// getCachedFile checks if a file's parsed symbols and identifiers are in the cache
// and returns them if the file's ModTime matches the cached ModTime.
func getCachedFile(path string, currentModTime time.Time) (FileCacheEntry, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	entry, exists := globalCache[path]
	if exists && entry.ModTime.Equal(currentModTime) {
		return entry, true
	}
	return FileCacheEntry{}, false
}

// setCachedFile saves the parsed file data into the global cache.
func setCachedFile(path string, entry FileCacheEntry) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	globalCache[path] = entry
}

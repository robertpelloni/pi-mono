package repomap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCacheHits(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repomap_cache_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "main.go")
	os.WriteFile(file1, []byte("package main\nfunc oldFunction() {}"), 0644)

	opts := Options{BaseDir: tmpDir}

	// Run 1: Should parse and cache.
	result1, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result1.Map, "oldFunction") {
		t.Errorf("expected oldFunction in map")
	}

	// Change file content without updating modified time -> should return old cached symbol!
	// (this verifies that we are actually reading from the cache)
	os.WriteFile(file1, []byte("package main\nfunc newFunction() {}"), 0644)

	// Explicitly reset the mtime back to what it was before the write to trigger a cache hit
	info, _ := os.Stat(file1)
	os.Chtimes(file1, info.ModTime(), info.ModTime().Add(-time.Hour)) // offset back

	// Better test approach: capture the time from run 1


	// Reset the global cache for a clean slate
	cacheMutex.Lock()
	globalCache = make(map[string]FileCacheEntry)
	cacheMutex.Unlock()

	os.WriteFile(file1, []byte("package main\nfunc oldFunction() {}"), 0644)
	Generate(opts) // caches oldFunction

	info2, _ := os.Stat(file1)

	// Now modify the file content
	os.WriteFile(file1, []byte("package main\nfunc newFunction() {}"), 0644)
	// Force the mtime back to exactly info2 to trick the cache
	os.Chtimes(file1, info2.ModTime(), info2.ModTime())

	result2, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result2.Map, "oldFunction") {
		t.Errorf("cache hit failed, it should have returned oldFunction because we tricked the mtime")
	}
	if strings.Contains(result2.Map, "newFunction") {
		t.Errorf("cache hit failed, it parsed the file again instead of using cache")
	}

	// Now bump the mtime to a newer time to simulate a real change
	os.Chtimes(file1, time.Now(), time.Now())

	result3, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(result3.Map, "oldFunction") {
		t.Errorf("cache invalidate failed, it should not have oldFunction anymore")
	}
	if !strings.Contains(result3.Map, "newFunction") {
		t.Errorf("cache invalidate failed, it should have parsed newFunction")
	}
}

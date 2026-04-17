package agent

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestWithFileMutationQueue(t *testing.T) {
	// Create a temporary directory for our test file
	tmpDir, err := ioutil.TempDir("", "mutation_queue_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFilePath := filepath.Join(tmpDir, "test.txt")

	// Write initial content
	err = ioutil.WriteFile(testFilePath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	var wg sync.WaitGroup
	// Run concurrent operations that all target the same file.
	// If the queue works, they won't interleave unexpectedly.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			_, err := WithFileMutationQueue(testFilePath, func() (any, error) {
				// Simulate read-modify-write
				content, err := ioutil.ReadFile(testFilePath)
				if err != nil {
					return nil, err
				}

				// Artificial delay to encourage race conditions if the lock fails
				time.Sleep(10 * time.Millisecond)

				newContent := string(content) + fmt.Sprintf("%d\n", id)

				err = ioutil.WriteFile(testFilePath, []byte(newContent), 0644)
				return nil, err
			})

			if err != nil {
				t.Errorf("Goroutine %d failed: %v", id, err)
			}
		}(i)
	}

	// Also run an operation on a DIFFERENT file to ensure it's not globally locked
	wg.Add(1)
	go func() {
		defer wg.Done()
		otherFile := filepath.Join(tmpDir, "other.txt")
		_, err := WithFileMutationQueue(otherFile, func() (any, error) {
			return nil, ioutil.WriteFile(otherFile, []byte("other file"), 0644)
		})
		if err != nil {
			t.Errorf("Other file goroutine failed: %v", err)
		}
	}()

	wg.Wait()

	// Verify the final file has exactly 5 lines (0, 1, 2, 3, 4 in some order)
	// Because they were executed sequentially via the mutation queue, no writes were lost.
	content, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read final file: %v", err)
	}

	// Just a simple length check, the exact order is non-deterministic but we shouldn't have lost any updates.
	// 5 IDs + 5 newlines = 10 bytes (since digits 0-4 are 1 byte each)
	if len(content) != 10 {
		t.Errorf("Expected 10 bytes, got %d. Content: %q", len(content), string(content))
	}
}

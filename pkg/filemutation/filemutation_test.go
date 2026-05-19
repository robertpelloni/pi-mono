package filemutation

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNewFileMutationQueue(t *testing.T) {
	q := NewFileMutationQueue()
	if q == nil {
		t.Fatal("Expected non-nil queue")
	}
}

func TestWithFileMutationQueue_SingleFile(t *testing.T) {
	q := NewFileMutationQueue()
	var executed int32
	err := q.WithFileMutationQueue("test.txt", func() error {
		atomic.AddInt32(&executed, 1)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&executed) != 1 {
		t.Error("Expected function to be called once")
	}
}

func TestWithFileMutationQueue_Error(t *testing.T) {
	q := NewFileMutationQueue()
	err := q.WithFileMutationQueue("test.txt", func() error {
		return errors.New("test error")
	})
	if err == nil || err.Error() != "test error" {
		t.Errorf("Expected test error, got %v", err)
	}
}

func TestWithFileMutationQueue_ConcurrentSameFile(t *testing.T) {
	q := NewFileMutationQueue()
	var order []int
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			q.WithFileMutationQueue("same.txt", func() error {
				mu.Lock()
				order = append(order, idx)
				mu.Unlock()
				return nil
			})
		}(i)
	}
	wg.Wait()

	if len(order) != 5 {
		t.Errorf("Expected 5 operations, got %d", len(order))
	}
}

func TestWithFileMutationQueue_DifferentFiles(t *testing.T) {
	q := NewFileMutationQueue()
	var count int32
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			q.WithFileMutationQueue(string(rune('a'+idx)), func() error {
				atomic.AddInt32(&count, 1)
				return nil
			})
		}(i)
	}
	wg.Wait()

	if atomic.LoadInt32(&count) != 10 {
		t.Errorf("Expected 10 operations, got %d", count)
	}
}

func TestDispose(t *testing.T) {
	q := NewFileMutationQueue()
	q.WithFileMutationQueue("test.txt", func() error { return nil })
	q.Dispose()
	// Should still work after dispose (creates new queues)
	err := q.WithFileMutationQueue("test2.txt", func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetMutationQueueKey(t *testing.T) {
	q := NewFileMutationQueue()
	key1 := q.getMutationQueueKey("test.txt")
	key2 := q.getMutationQueueKey("test.txt")
	if key1 != key2 {
		t.Error("Same file should produce same key")
	}
}

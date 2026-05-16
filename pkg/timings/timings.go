package timings

import (
	"fmt"
	"sync"
	"time"
)

// Timing tracks the duration of a named operation.
type Timing struct {
	Name      string        `json:"name"`
	Start     time.Time     `json:"start"`
	Duration  time.Duration `json:"duration"`
}

// Tracker tracks multiple named timing operations.
type Tracker struct {
	mu      sync.Mutex
	timings map[string]*Timing
	order   []string
}

// NewTracker creates a new timing tracker.
func NewTracker() *Tracker {
	return &Tracker{
		timings: make(map[string]*Timing),
	}
}

// Start begins tracking a named operation.
func (t *Tracker) Start(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.timings[name] = &Timing{
		Name:  name,
		Start: time.Now(),
	}
	t.order = append(t.order, name)
}

// End finishes tracking a named operation.
func (t *Tracker) End(name string) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	if timing, ok := t.timings[name]; ok {
		timing.Duration = time.Since(timing.Start)
		return timing.Duration
	}
	return 0
}

// Measure runs a function and tracks its duration.
func (t *Tracker) Measure(name string, fn func() error) error {
	t.Start(name)
	err := fn()
	t.End(name)
	return err
}

// Get returns the timing for a named operation.
func (t *Tracker) Get(name string) (time.Duration, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if timing, ok := t.timings[name]; ok {
		return timing.Duration, true
	}
	return 0, false
}

// FormatAll returns all timings as a formatted string.
func (t *Tracker) FormatAll() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	var lines []string
	for _, name := range t.order {
		if timing, ok := t.timings[name]; ok && timing.Duration > 0 {
			lines = append(lines, fmt.Sprintf("  %-20s %s", name, timing.Duration.Round(time.Millisecond)))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "Timings:\n" + joinStrings(lines, "\n")
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

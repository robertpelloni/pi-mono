package timings

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewTracker(t *testing.T) {
	tracker := NewTracker()
	if tracker == nil {
		t.Fatal("Expected non-nil tracker")
	}
}

func TestTracker_StartEnd(t *testing.T) {
	tracker := NewTracker()
	tracker.Start("test_op")
	time.Sleep(10 * time.Millisecond)
	duration := tracker.End("test_op")

	if duration < 10*time.Millisecond {
		t.Errorf("Expected at least 10ms, got %v", duration)
	}
}

func TestTracker_Get(t *testing.T) {
	tracker := NewTracker()
	tracker.Start("op1")
	time.Sleep(2 * time.Millisecond)
	tracker.End("op1")

	_, ok := tracker.Get("op1")
	if !ok {
		t.Error("Expected to find timing for op1")
	}

	_, ok = tracker.Get("nonexistent")
	if ok {
		t.Error("Expected not to find timing for nonexistent operation")
	}
}

func TestTracker_EndNonExistent(t *testing.T) {
	tracker := NewTracker()
	duration := tracker.End("nonexistent")
	if duration != 0 {
		t.Errorf("Expected 0 duration for nonexistent op, got %v", duration)
	}
}

func TestTracker_Measure(t *testing.T) {
	tracker := NewTracker()
	err := tracker.Measure("measured_op", func() error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	duration, ok := tracker.Get("measured_op")
	if !ok {
		t.Error("Expected to find timing for measured_op")
	}
	if duration < 5*time.Millisecond {
		t.Errorf("Expected at least 5ms, got %v", duration)
	}
}

func TestTracker_MeasureError(t *testing.T) {
	tracker := NewTracker()
	err := tracker.Measure("failing_op", func() error {
		return fmt.Errorf("test error")
	})
	if err == nil {
		t.Error("Expected error from Measure")
	}
}

func TestTracker_FormatAll(t *testing.T) {
	tracker := NewTracker()
	tracker.Start("op1")
	time.Sleep(2 * time.Millisecond)
	tracker.End("op1")
	tracker.Start("op2")
	time.Sleep(2 * time.Millisecond)
	tracker.End("op2")

	output := tracker.FormatAll()
	if !strings.Contains(output, "Timings") {
		t.Error("Expected 'Timings' header in output")
	}
	if !strings.Contains(output, "op1") || !strings.Contains(output, "op2") {
		t.Error("Expected operation names in output")
	}
}

func TestTracker_FormatAll_Empty(t *testing.T) {
	tracker := NewTracker()
	output := tracker.FormatAll()
	if output != "" {
		t.Errorf("Expected empty string for no timings, got %q", output)
	}
}

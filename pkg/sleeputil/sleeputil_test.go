package sleeputil

import (
	"testing"
	"time"
)

func TestSleep(t *testing.T) {
	start := time.Now()
	Sleep(10 * time.Millisecond)
	elapsed := time.Since(start)
	if elapsed < 8*time.Millisecond {
		t.Error("Sleep returned too quickly")
	}
}

func TestSleepMs(t *testing.T) {
	start := time.Now()
	SleepMs(10)
	elapsed := time.Since(start)
	if elapsed < 8*time.Millisecond {
		t.Error("SleepMs returned too quickly")
	}
}

func TestSleepSec(t *testing.T) {
	start := time.Now()
	SleepSec(0) // 0 seconds should return immediately
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Error("SleepSec(0) took too long")
	}
}

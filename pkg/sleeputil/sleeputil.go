package sleeputil

import (
	"time"
)

// Sleep pauses the current goroutine for the given duration.
// Can be interrupted by context cancellation if needed.
func Sleep(d time.Duration) {
	time.Sleep(d)
}

// SleepMs pauses for the given number of milliseconds.
func SleepMs(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// SleepSec pauses for the given number of seconds.
func SleepSec(sec int) {
	time.Sleep(time.Duration(sec) * time.Second)
}

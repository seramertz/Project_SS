package watchdog

import (
	"time"
)

// Watchdog is a timer that resets itself upon receiving a reset signal.
// If it stops receiving signals for the specified duration, it can indicate a failure.

func Watchdog(seconds int, ch_reset chan bool, ch_signal chan bool) {
	// Initialize the watchdog timer
	watchdogTimer := time.NewTimer(time.Duration(seconds) * time.Second)

	for {
		select {
		case <-ch_reset:
			watchdogTimer.Reset(time.Duration(seconds) * time.Second)

		case <-ch_signal:
			ch_reset <- true
			watchdogTimer.Reset(time.Duration(seconds) * time.Second)
		}
	}
}

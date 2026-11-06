// Package clock supplies the wall clock and timers to the application layer.
package clock

import (
	"time"

	"webvigil/internal/application"
)

// System reads the real clock.
type System struct{}

// Now returns the current time.
func (System) Now() time.Time { return time.Now() }

// AfterFunc schedules f on the real clock.
func AfterFunc(d time.Duration, f func()) application.Timer { return time.AfterFunc(d, f) }

package domain

import "time"

// MaxBackoff caps how long repeated failures or a Retry-After header may push a check away.
const MaxBackoff = 24 * time.Hour

// NextDue returns when a site is next due, measured from the check just made. Repeated failures
// double the wait per failure, starting from the interval, up to MaxBackoff (or the interval
// when that is longer). A Retry-After longer than the computed wait is honoured, up to the cap.
// The result is always measured from the latest check, so a long sleep never builds a backlog.
func NextDue(last time.Time, interval time.Duration, failures int, retryAfter time.Duration) time.Time {
	ceiling := max(MaxBackoff, interval)
	wait := interval
	for i := 1; i < failures && wait < ceiling; i++ {
		wait *= 2
	}
	wait = min(wait, ceiling)
	if retryAfter > wait {
		wait = min(retryAfter, ceiling)
	}
	return last.Add(wait)
}

package moreatomic

import (
	"sync/atomic"
	"time"
)

type Time struct {
	unixnano int64
}

func Now() *Time {
	return &Time{
		unixnano: time.Now().UnixNano(),
	}
}

func (t *Time) Get() time.Time {
	nano := atomic.LoadInt64(&t.unixnano)
	return time.Unix(0, nano)
}

func (t *Time) Set(time time.Time) {
	atomic.StoreInt64(&t.unixnano, time.UnixNano())
}

// HasBeen checks if it has been this long since the last time. If yes, it will
// set the time.
func (t *Time) HasBeen(dura time.Duration) bool {
	now := time.Now()
	nano := atomic.LoadInt64(&t.unixnano)

	// We have to be careful of zero values.
	if nano != 0 {
		// Subtract the duration to now. If subtracted now is before the stored
		// time, that means it hasn't been that long yet. We also have to be careful
		// of an unitialized time.
		if now.Add(-dura).Before(time.Unix(0, nano)) {
			return false
		}
	}

	// It has been that long, so store the variable.
	t.Set(now)
	return true
}

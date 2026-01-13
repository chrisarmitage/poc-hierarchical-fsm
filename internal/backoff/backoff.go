package backoff

import "time"

// backoff
type Backoff interface {
	Next() time.Duration
	Reset()
}

type ExponentialBackoff struct {
	base time.Duration
	max  time.Duration
	curr time.Duration
}

func NewExponentialBackoff(base, max time.Duration) *ExponentialBackoff {
	return &ExponentialBackoff{base: base, max: max}
}

func (b *ExponentialBackoff) Next() time.Duration {
	if b.curr == 0 {
		b.curr = b.base
	} else {
		b.curr *= 2
		if b.curr > b.max {
			b.curr = b.max
		}
	}
	return b.curr
}

func (b *ExponentialBackoff) Reset() {
	b.curr = 0
}

package tunnel

import (
	"math/rand"
	"time"
)

type Backoff struct {
	base time.Duration
	max  time.Duration
	rnd  *rand.Rand
}

func NewBackoff(base, max time.Duration, seed int64) Backoff {
	return Backoff{
		base: base,
		max:  max,
		rnd:  rand.New(rand.NewSource(seed)),
	}
}

func (b Backoff) Duration(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	d := b.base << attempt
	if d < b.base {
		d = b.max
	}
	if d > b.max {
		d = b.max
	}
	jitterFraction := (b.rnd.Float64() * 0.6) - 0.3
	jitter := float64(d) * jitterFraction
	result := time.Duration(float64(d) + jitter)
	if result < b.base {
		return b.base
	}
	if result > b.max {
		return b.max
	}
	return result
}

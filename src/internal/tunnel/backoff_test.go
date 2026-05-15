package tunnel

import (
	"testing"
	"time"
)

func TestBackoffDurationIsBounded(t *testing.T) {
	b := NewBackoff(1*time.Second, 60*time.Second, 42)

	for attempt := 0; attempt < 20; attempt++ {
		d := b.Duration(attempt)
		if d < 1*time.Second {
			t.Fatalf("duration below base: %v", d)
		}
		if d > 60*time.Second {
			t.Fatalf("duration above max: %v", d)
		}
	}
}

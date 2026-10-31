package telemetry

import "time"

type sample struct {
	t time.Time
	v float64
}

// Rate tracks a cumulative counter and answers "per hour" over sliding windows.
// Memory is bounded: samples are thinned to at most one per MinGap.
type Rate struct {
	start   time.Time
	started bool
	total   float64
	ring    *Ring[sample]
	minGap  time.Duration
}

func NewRate(capacity int, minGap time.Duration) *Rate {
	return &Rate{ring: NewRing[sample](capacity), minGap: minGap}
}

// Add adds delta to the counter at time t.
func (r *Rate) Add(t time.Time, delta float64) {
	if !r.started {
		r.started = true
		r.start = t
		r.ring.Push(sample{t, 0})
	}
	r.total += delta
	if n := r.ring.Newest(); n != nil && t.Sub(n.t) < r.minGap {
		n.v = r.total // refresh the newest sample instead of growing
		return
	}
	r.ring.Push(sample{t, r.total})
}

// Touch records time passing with no change so idle periods lower the rate.
func (r *Rate) Touch(t time.Time) { r.Add(t, 0) }

func (r *Rate) Total() float64 { return r.total }

// PerHour is the rate over the trailing window ending at now. If less than a
// full window has elapsed, it uses the elapsed time and reports partial=true.
func (r *Rate) PerHour(now time.Time, window time.Duration) (perHour float64, partial bool) {
	if !r.started {
		return 0, true
	}
	from := now.Add(-window)
	if from.Before(r.start) {
		from = r.start
		partial = true
	}
	span := now.Sub(from)
	if span <= 0 {
		return 0, true
	}
	// counter value at `from`: newest sample at or before it
	base := 0.0
	for i := r.ring.Len() - 1; i >= 0; i-- {
		s := r.ring.At(i)
		if !s.t.After(from) {
			base = s.v
			break
		}
	}
	if r.ring.Len() > 0 && r.ring.At(0).t.After(from) && !partial {
		partial = true // ring has forgotten the window start
		from = r.ring.At(0).t
		span = now.Sub(from)
		if span <= 0 {
			return 0, true
		}
		base = r.ring.At(0).v
	}
	return (r.total - base) / span.Hours(), partial
}

// SessionPerHour is total divided by elapsed time since the first sample.
func (r *Rate) SessionPerHour(now time.Time) float64 {
	if !r.started || now.Sub(r.start) <= 0 {
		return 0
	}
	return r.total / now.Sub(r.start).Hours()
}

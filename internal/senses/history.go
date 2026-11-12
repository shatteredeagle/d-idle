package senses

import "time"

// Sample retention: one sample about every 2 s for an hour (1800 samples, a few hundred KB at most).
const (
	SampleEvery = 2 * time.Second
	SampleCap   = 1800
)

// Sample is the bot's state at one moment, for the minimap's timeline: life and mana percent, the activity code, the
// whyline code, the level and the experience. Gold is not here: the runner's status does not carry it (U).
type Sample struct {
	At    time.Time
	Life  int
	Mana  int
	Area  int
	Act   uint8
	Why   string
	Level int
	XP    int64
}

// Samples is the sample ring. Not safe for concurrent use; Publisher holds its lock.
type Samples struct{ s []Sample }

// Add stores v unless the newest sample is younger than SampleEvery; it reports whether it stored.
func (r *Samples) Add(v Sample) bool {
	if n := len(r.s); n > 0 && v.At.Sub(r.s[n-1].At) < SampleEvery {
		return false
	}
	r.s = append(r.s, v)
	if over := len(r.s) - SampleCap; over > 0 {
		r.s = append(r.s[:0], r.s[over:]...)
	}
	return true
}

// AddSample records the bot's state now (called about 2 Hz; the ring keeps one per SampleEvery).
func (p *Publisher) AddSample(v Sample) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v.At = p.clock()
	v.Area = p.areaNo
	p.samples.Add(v)
}

// SamplesOf returns the samples, oldest first, and the time they were read.
func (p *Publisher) SamplesOf() ([]Sample, time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]Sample(nil), p.samples.s...), p.clock()
}

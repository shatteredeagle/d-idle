package d2dsrc

import (
	"sort"
	"strings"
	"sync"
)

// Report says, for one snapshot, which fields came from d2d and which are unknown. A field that is
// unknown is also left at its zero value in the d2.Snapshot (the d2 types have no "absent"), so the Report
// is the truth and the value is not: never read a zero from a field listed in Unknown.
type Report struct {
	Present []string          // fields filled from d2d (or derived, see Derived)
	Derived map[string]string // field -> how it was derived (uses an unverified assumption)
	Unknown map[string]string // field -> why d2d cannot provide it
	Source  map[string]string // section -> "dump", "verbs" (stock d2d fallback) or "none"
}

// Has reports whether a field was filled.
func (r *Report) Has(field string) bool {
	for _, f := range r.Present {
		if f == field {
			return true
		}
	}
	return false
}

// UnknownFields returns the unknown field names sorted.
func (r *Report) UnknownFields() []string {
	out := make([]string, 0, len(r.Unknown))
	for k := range r.Unknown {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// tracker keeps the process-wide unknown counters and logs each field once.
type tracker struct {
	mu     sync.Mutex
	counts map[string]uint64
	logf   func(string, ...any)
	fresh  []string // fields first seen since the last flush
}

func (t *tracker) hit(field, why string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.counts == nil {
		t.counts = map[string]uint64{}
	}
	t.counts[field]++
	if t.counts[field] == 1 {
		t.fresh = append(t.fresh, field)
	}
}

// flush logs the fields first seen since the last call as ONE line (reasons are in Report.Unknown).
func (t *tracker) flush() {
	t.mu.Lock()
	f := t.fresh
	t.fresh = nil
	t.mu.Unlock()
	if len(f) > 0 && t.logf != nil {
		sort.Strings(f)
		t.logf("d2dsrc: %d new unknown fields: %s", len(f), strings.Join(f, ", "))
	}
}

func (t *tracker) snapshot() map[string]uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make(map[string]uint64, len(t.counts))
	for k, v := range t.counts {
		out[k] = v
	}
	return out
}

// rep is the per-snapshot builder of a Report.
type rep struct {
	Report
	t *tracker
}

func newRep(t *tracker) *rep {
	return &rep{Report: Report{Derived: map[string]string{}, Unknown: map[string]string{}, Source: map[string]string{}}, t: t}
}

func (r *rep) have(fields ...string) { r.Present = append(r.Present, fields...) }

func (r *rep) derived(field, how string) {
	r.Present = append(r.Present, field)
	r.Derived[field] = how
}

// unknown records a field d2d did not provide; counted every time, logged once per process.
func (r *rep) unknown(field, why string) {
	if _, dup := r.Unknown[field]; dup {
		return
	}
	r.Unknown[field] = why
	r.t.hit(field, why)
}

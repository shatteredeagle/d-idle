package telemetry

// Ring is a fixed-capacity FIFO that overwrites the oldest entry.
type Ring[T any] struct {
	buf  []T
	next int
	n    int
}

func NewRing[T any](capacity int) *Ring[T] { return &Ring[T]{buf: make([]T, capacity)} }

func (r *Ring[T]) Push(v T) {
	r.buf[r.next] = v
	r.next = (r.next + 1) % len(r.buf)
	if r.n < len(r.buf) {
		r.n++
	}
}

func (r *Ring[T]) Len() int { return r.n }

// At returns the i-th oldest element (0 = oldest).
func (r *Ring[T]) At(i int) T {
	start := (r.next - r.n + len(r.buf)) % len(r.buf)
	return r.buf[(start+i)%len(r.buf)]
}

// Slice copies the contents, oldest first.
func (r *Ring[T]) Slice() []T {
	out := make([]T, r.n)
	for i := range out {
		out[i] = r.At(i)
	}
	return out
}

// Last returns up to n newest elements, oldest first.
func (r *Ring[T]) Last(n int) []T {
	if n > r.n {
		n = r.n
	}
	out := make([]T, n)
	for i := range out {
		out[i] = r.At(r.n - n + i)
	}
	return out
}

// Last returns a pointer to the newest element, or nil.
func (r *Ring[T]) Newest() *T {
	if r.n == 0 {
		return nil
	}
	return &r.buf[(r.next-1+len(r.buf))%len(r.buf)]
}

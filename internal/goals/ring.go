package goals

// ring is a fixed-capacity FIFO of the most recent items.
type ring[T any] struct {
	buf  []T
	next int
	n    int
}

func newRing[T any](capacity int) *ring[T] { return &ring[T]{buf: make([]T, capacity)} }

func (r *ring[T]) push(v T) {
	r.buf[r.next] = v
	r.next = (r.next + 1) % len(r.buf)
	if r.n < len(r.buf) {
		r.n++
	}
}

// each visits items oldest to newest; stop by returning false.
func (r *ring[T]) each(f func(T) bool) {
	start := (r.next - r.n + len(r.buf)) % len(r.buf)
	for i := 0; i < r.n; i++ {
		if !f(r.buf[(start+i)%len(r.buf)]) {
			return
		}
	}
}

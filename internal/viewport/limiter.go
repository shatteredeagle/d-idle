package viewport

import "context"

// Limiter is a counting semaphore shared by all streams of a process (global capture concurrency cap).
type Limiter struct{ ch chan struct{} }

func NewLimiter(n int) *Limiter {
	if n < 1 {
		n = 1
	}
	return &Limiter{ch: make(chan struct{}, n)}
}

// Acquire takes a slot or returns the context error. A nil Limiter never blocks.
func (l *Limiter) Acquire(ctx context.Context) error {
	if l == nil {
		return nil
	}
	select {
	case l.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *Limiter) Release() {
	if l != nil {
		<-l.ch
	}
}

// InUse is the number of slots taken now.
func (l *Limiter) InUse() int {
	if l == nil {
		return 0
	}
	return len(l.ch)
}

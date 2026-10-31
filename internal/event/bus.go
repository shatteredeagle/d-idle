package event

import (
	"sync"
	"sync/atomic"
	"time"
)

// Bus fans events out to subscribers. Publish never blocks: each subscriber has a bounded buffer and, when it is
// full, the oldest queued event is discarded and counted in Sub.Dropped.
type Bus struct {
	instance string
	clock    func() time.Time

	mu   sync.Mutex // serialises Publish, so each Sub has a single sender
	seq  uint64
	subs map[*Sub]struct{}
}

// NewBus returns a bus whose events are stamped with instance. clock may be nil (time.Now).
func NewBus(instance string, clock func() time.Time) *Bus {
	if clock == nil {
		clock = time.Now
	}
	return &Bus{instance: instance, clock: clock, subs: map[*Sub]struct{}{}}
}

// Sub is one subscription. Read events from C; call Close when done (C is then closed).
type Sub struct {
	Name string
	c    chan Event
	bus  *Bus
	drop atomic.Uint64
}

func (s *Sub) C() <-chan Event { return s.c }

// Dropped is the number of events discarded because this subscriber was too slow.
func (s *Sub) Dropped() uint64 { return s.drop.Load() }

// Close unsubscribes and closes C. Safe to call more than once.
func (s *Sub) Close() {
	b := s.bus
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.subs[s]; ok {
		delete(b.subs, s)
		close(s.c)
	}
}

// Subscribe registers a subscriber with a buffer of buf events (minimum 1).
func (b *Bus) Subscribe(name string, buf int) *Sub {
	if buf < 1 {
		buf = 1
	}
	s := &Sub{Name: name, c: make(chan Event, buf), bus: b}
	b.mu.Lock()
	b.subs[s] = struct{}{}
	b.mu.Unlock()
	return s
}

// Publish stamps V, Seq, Time and Instance (when unset), delivers to every subscriber and returns the stamped
// event. It never blocks on a subscriber.
func (b *Bus) Publish(e Event) Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	e.Seq = b.seq
	if e.V == 0 {
		e.V = Version
	}
	if e.Time.IsZero() {
		e.Time = b.clock()
	}
	if e.Instance == "" {
		e.Instance = b.instance
	}
	for s := range b.subs {
		for {
			select {
			case s.c <- e:
			default:
				select {
				case <-s.c:
					s.drop.Add(1)
				default:
				}
				continue
			}
			break
		}
	}
	return e
}

// Seq is the last assigned sequence number.
func (b *Bus) Seq() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.seq
}

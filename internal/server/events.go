package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"d2lod-bot/internal/event"
	"d2lod-bot/internal/goals"
)

const (
	ringSize  = 1000 // history kept per instance for ?since= and Last-Event-ID
	clientBuf = 256  // per SSE client (docs: event/MIGRATION.md Rules)
)

// ring keeps the last ringSize events of one bus. A goroutine moves them off the bus subscription as they
// are published, so the goal can be stamped close to the moment of the event; readers first wait (briefly)
// until everything published before the read has arrived.
type ring struct {
	mu    sync.Mutex
	sub   *event.Sub
	bus   *event.Bus
	hist  []event.Event
	last  uint64 // highest seq appended
	stamp func(*event.Event)
}

func newRing(b *event.Bus, stamp func(*event.Event)) *ring {
	r := &ring{sub: b.Subscribe("api-ring", ringSize), bus: b, stamp: stamp}
	go func() {
		for e := range r.sub.C() {
			r.stamp(&e)
			r.mu.Lock()
			r.hist = append(r.hist, e)
			if n := len(r.hist) - ringSize; n > 0 {
				r.hist = append(r.hist[:0], r.hist[n:]...)
			}
			r.last = e.Seq
			r.mu.Unlock()
		}
	}()
	return r
}

// settle waits for the events published so far (an event the subscription dropped never arrives: the wait is bounded).
func (r *ring) settle() {
	want := r.bus.Seq()
	for i := 0; i < 100; i++ {
		r.mu.Lock()
		ok := r.last >= want
		r.mu.Unlock()
		if ok {
			return
		}
		time.Sleep(time.Millisecond)
	}
}

// since returns events with Seq > since, oldest first. With since == 0 and limit > 0 it returns the newest
// limit events; otherwise at most limit (0: all) from the oldest newer one.
func (r *ring) since(since uint64, limit int) []event.Event {
	r.settle()
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []event.Event
	for _, e := range r.hist {
		if e.Seq > since {
			out = append(out, e)
		}
	}
	if limit > 0 && len(out) > limit {
		if since == 0 {
			out = out[len(out)-limit:]
		} else {
			out = out[:limit]
		}
	}
	return out
}

func (r *ring) close() { r.sub.Close() }

func (s *Server) ringFor(i *Instance) *ring {
	if i.Bus == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.rings[i.ID]
	if r == nil {
		r = newRing(i.Bus, func(e *event.Event) { stampGoal(i, e) })
		s.rings[i.ID] = r
	}
	return r
}

// feed is where an instance's events come from: its Bus, or (compat) the manager's flat log.
type feed struct {
	inst *Instance
	ring *ring
}

// goalKinds are the events the goal manager attributes to the active goal when it receives them from the runner
// (goals.Manager.Handle). The same events reach the bus straight from the runner with no goal, so the server
// stamps the active goal on them the same way, at the moment it first sees the event.
var goalKinds = map[event.Kind]bool{event.KindKill: true, event.KindDrop: true, event.KindDeath: true,
	event.KindLevelUp: true, event.KindAreaChange: true, event.KindPotion: true, event.KindError: true}

func stampGoal(i *Instance, e *event.Event) {
	if e.Goal == "" && goalKinds[e.Kind] {
		e.Goal = i.Manager.Snapshot().ActiveGoalID
	}
}

func (s *Server) feedFor(i *Instance) feed { return feed{i, s.ringFor(i)} }

func (f feed) history(since uint64, limit int) []event.Event {
	if f.ring != nil {
		return f.ring.since(since, limit)
	}
	var out []event.Event
	for _, g := range f.inst.Manager.Events(since, limit) {
		out = append(out, f.wrap(g))
	}
	return out
}

func (f feed) wrap(g goals.Event) event.Event {
	e := event.FromGoals(g, event.SrcManager)
	e.Instance = f.inst.ID
	return e
}

// subscribe returns live events and a cancel func. Subscribe before reading history, then drop Seq <= last.
func (f feed) subscribe() (<-chan event.Event, func()) {
	if f.inst.Bus != nil {
		sub := f.inst.Bus.Subscribe("api-sse", clientBuf)
		out := make(chan event.Event, clientBuf)
		go func() {
			defer close(out)
			for e := range sub.C() {
				stampGoal(f.inst, &e)
				select {
				case out <- e:
				default:
				}
			}
		}()
		return out, sub.Close
	}
	gch, cancel := f.inst.Manager.Subscribe()
	out := make(chan event.Event, clientBuf)
	done := make(chan struct{})
	go func() {
		defer close(out)
		for {
			select {
			case g, ok := <-gch:
				if !ok {
					return
				}
				select {
				case out <- f.wrap(g):
				default:
				}
			case <-done:
				return
			}
		}
	}()
	var once sync.Once
	return out, func() { once.Do(func() { close(done); cancel() }) }
}

func atoiDefault(s string, d int) int {
	if n, err := strconv.Atoi(s); err == nil && n > 0 {
		return n
	}
	return d
}

func (s *Server) events(w http.ResponseWriter, r *http.Request, i *Instance) {
	since, _ := strconv.ParseUint(r.URL.Query().Get("since"), 10, 64)
	limit := min(atoiDefault(r.URL.Query().Get("limit"), 200), 1000)
	evs := s.feedFor(i).history(since, limit)
	if evs == nil {
		evs = []event.Event{}
	}
	writeJSON(w, 200, evs)
}

type sseWriter struct {
	w  http.ResponseWriter
	fl http.Flusher
}

func newSSE(w http.ResponseWriter) (*sseWriter, bool) {
	fl, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, 500, errBody{"streaming unsupported", "internal"})
		return nil, false
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(200)
	return &sseWriter{w, fl}, true
}

func (x *sseWriter) send(id uint64, name string, v any) bool {
	b, err := json.Marshal(v)
	if err != nil {
		return true
	}
	if id != 0 {
		fmt.Fprintf(x.w, "id: %d\n", id)
	}
	_, err = fmt.Fprintf(x.w, "event: %s\ndata: %s\n\n", name, b)
	return err == nil
}

func (x *sseWriter) beat() bool {
	_, err := fmt.Fprint(x.w, ": keepalive\n\n")
	x.fl.Flush()
	return err == nil
}

// stream is events/stream of one instance: "log" (event envelope, id = seq; resume with Last-Event-ID or
// ?since=), "status" every second, "goals" when the manager revision changed.
func (s *Server) stream(w http.ResponseWriter, r *http.Request, i *Instance) {
	x, ok := newSSE(w)
	if !ok {
		return
	}
	f := s.feedFor(i)
	ch, cancel := f.subscribe() // before the replay, so nothing falls in the gap
	defer cancel()

	since, _ := strconv.ParseUint(r.Header.Get("Last-Event-ID"), 10, 64)
	if q := r.URL.Query().Get("since"); q != "" {
		since, _ = strconv.ParseUint(q, 10, 64)
	}
	limit := 0
	if since == 0 {
		limit = 100
	}
	last := since
	catchUp := func(lim int) bool {
		for _, e := range f.history(last, lim) {
			if e.Seq > last {
				if !x.send(e.Seq, "log", e) {
					return false
				}
				last = e.Seq
			}
		}
		return true
	}
	if !catchUp(limit) {
		return
	}
	a := s.apiFor(i)
	lastRev := ^uint64(0)
	push := func() bool {
		st := i.Manager.Snapshot()
		if !x.send(0, "status", a.Status()) {
			return false
		}
		if st.Rev != lastRev {
			lastRev = st.Rev
			if !x.send(0, "goals", i.Manager.Goals()) {
				return false
			}
		}
		x.fl.Flush()
		return true
	}
	if !push() {
		return
	}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	beat := time.NewTicker(15 * time.Second)
	defer beat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case e, ok := <-ch:
			if !ok {
				return
			}
			if e.Seq <= last {
				continue
			}
			last = e.Seq
			if !x.send(e.Seq, "log", e) {
				return
			}
			x.fl.Flush()
		case <-tick.C:
			if !catchUp(500) || !push() { // catch-up refills what a slow client lost from its buffer
				return
			}
		case <-beat.C:
			if !x.beat() {
				return
			}
		}
	}
}

// fleetStream merges every instance's live events ("log", envelope with "instance", no id: seq is per
// instance) and sends the fleet body as "fleet" every second. It does not replay; read events/ for history.
func (s *Server) fleetStream(w http.ResponseWriter, r *http.Request) {
	x, ok := newSSE(w)
	if !ok {
		return
	}
	var wg sync.WaitGroup
	defer wg.Wait() // runs after stop
	ctx, stop := context.WithCancel(r.Context())
	defer stop()
	merged := make(chan event.Event, clientBuf)
	for _, i := range s.cfg.Instances.List() {
		ch, cancel := s.feedFor(i).subscribe()
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel()
			for {
				select {
				case e, ok := <-ch:
					if !ok {
						return
					}
					select {
					case merged <- e:
					default: // slow client: drop, the fleet body still carries the state
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	if !x.send(0, "fleet", s.fleet(ctx)) {
		return
	}
	x.fl.Flush()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	beat := time.NewTicker(15 * time.Second)
	defer beat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-merged:
			if !x.send(0, "log", e) {
				return
			}
			x.fl.Flush()
		case <-tick.C:
			if !x.send(0, "fleet", s.fleet(ctx)) {
				return
			}
			x.fl.Flush()
		case <-beat.C:
			if !x.beat() {
				return
			}
		}
	}
}

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// stream is the Server-Sent Events endpoint. Event names:
//
//	log     one goals.Event; id = Seq (use Last-Event-ID or ?since= to resume)
//	status  the status body, every second
//	goals   the full goal list, when anything changed
func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, 500, errBody{"streaming unsupported", "internal"})
		return
	}
	m := s.cfg.Manager
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(200)

	send := func(id uint64, name string, v any) bool {
		b, err := json.Marshal(v)
		if err != nil {
			return true
		}
		if id != 0 {
			fmt.Fprintf(w, "id: %d\n", id)
		}
		_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, b)
		return err == nil
	}

	// Subscribe before replaying so no event falls in the gap; duplicates are dropped by seq.
	ch, cancel := m.Subscribe()
	defer cancel()

	since, _ := strconv.ParseUint(r.Header.Get("Last-Event-ID"), 10, 64)
	if q := r.URL.Query().Get("since"); q != "" {
		since, _ = strconv.ParseUint(q, 10, 64)
	}
	limit := 0
	if since == 0 {
		limit = 100 // a fresh client gets recent history
	}
	last := since
	for _, e := range m.Events(since, limit) {
		if !send(e.Seq, "log", e) {
			return
		}
		last = e.Seq
	}
	lastRev := ^uint64(0)
	push := func() bool {
		st := s.status()
		if !send(0, "status", st) {
			return false
		}
		if st.Rev != lastRev {
			lastRev = st.Rev
			if !send(0, "goals", m.Goals()) {
				return false
			}
		}
		fl.Flush()
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
			if !send(e.Seq, "log", e) {
				return
			}
			fl.Flush()
		case <-tick.C:
			// refill from the ring if the subscriber channel overflowed
			for _, e := range m.Events(last, 500) {
				if e.Seq > last {
					if !send(e.Seq, "log", e) {
						return
					}
					last = e.Seq
				}
			}
			if !push() {
				return
			}
		case <-beat.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			fl.Flush()
		}
	}
}

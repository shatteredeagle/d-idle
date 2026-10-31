// Package status is the live-display hub: a registry of panels, a snapshot
// endpoint and a Server-Sent Events stream with snapshot-then-delta semantics.
//
// Mount it from any binary with one line:
//
//	hub.Mount(mux, "/live/")
package status

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/web/live"
)

// Panel is one box on the display. The browser side is web/live/panels/<ID>.js.
type Panel struct {
	ID    string
	Title string
	Order int
	// Snapshot returns the panel's full state (JSON-serializable). Polled every
	// Every (default 1s) by Hub.Run; unchanged output is not re-sent.
	Snapshot func() any
	Every    time.Duration
	// Routes, if set, is served under api/panel/<ID>/ (prefix stripped), for
	// bulky data that should not ride the event stream (e.g. map terrain).
	Routes http.Handler

	lastPoll time.Time
}

type panelDTO struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Order int    `json:"order"`
	Data  any    `json:"data"`
}

type message struct{ event, data string }

type sub struct{ ch chan message }

// Hub aggregates panels.
type Hub struct {
	mu     sync.Mutex
	panels map[string]*Panel
	last   map[string][]byte
	subs   map[*sub]struct{}
	seq    uint64
}

func NewHub() *Hub {
	return &Hub{panels: map[string]*Panel{}, last: map[string][]byte{}, subs: map[*sub]struct{}{}}
}

// Register adds (or replaces) a panel.
func (h *Hub) Register(p Panel) {
	if p.Every <= 0 {
		p.Every = time.Second
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.panels[p.ID] = &p
	delete(h.last, p.ID)
}

func (h *Hub) sorted() []*Panel {
	ps := make([]*Panel, 0, len(h.panels))
	for _, p := range h.panels {
		ps = append(ps, p)
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].Order != ps[j].Order {
			return ps[i].Order < ps[j].Order
		}
		return ps[i].ID < ps[j].ID
	})
	return ps
}

func (h *Hub) broadcast(event string, v any) {
	b, _ := json.Marshal(v)
	m := message{event, string(b)}
	for s := range h.subs {
		select {
		case s.ch <- m:
		default: // slow client: drop it; EventSource reconnects and gets a fresh snapshot
			close(s.ch)
			delete(h.subs, s)
		}
	}
}

// Push stores new data for a panel and broadcasts it if it changed.
// Use it for event-driven feeders; pass nil Snapshot when registering such a panel
// only if you always Push before clients connect (otherwise supply Snapshot too).
func (h *Hub) Push(id string, data any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pushLocked(id, data)
}

func (h *Hub) pushLocked(id string, data any) {
	b, err := json.Marshal(data)
	if err != nil || bytes.Equal(b, h.last[id]) {
		return
	}
	h.last[id] = b
	h.seq++
	h.broadcast("update", map[string]any{"seq": h.seq, "id": id, "data": json.RawMessage(b)})
}

// Append broadcasts one new item for an append-style panel (a log). The panel's
// Snapshot should also contain recent items; the browser de-duplicates by item.seq.
func (h *Hub) Append(id string, item any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.seq++
	h.broadcast("append", map[string]any{"seq": h.seq, "id": id, "item": item})
}

// Poll calls Snapshot on every due panel and pushes changes.
func (h *Hub) Poll() {
	h.mu.Lock()
	ps := h.sorted()
	h.mu.Unlock()
	now := time.Now()
	for _, p := range ps {
		if p.Snapshot == nil || now.Sub(p.lastPoll) < p.Every {
			continue
		}
		p.lastPoll = now
		d := p.Snapshot() // outside the lock: snapshots may be slow
		h.mu.Lock()
		h.pushLocked(p.ID, d)
		h.mu.Unlock()
	}
}

// Run polls until ctx ends.
func (h *Hub) Run(ctx context.Context) {
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.Poll()
		}
	}
}

func (h *Hub) snapshot() map[string]any {
	h.mu.Lock()
	ps := h.sorted()
	seq := h.seq
	h.mu.Unlock()
	out := make([]panelDTO, 0, len(ps))
	for _, p := range ps {
		var d any
		if p.Snapshot != nil {
			d = p.Snapshot()
		} else {
			h.mu.Lock()
			if b, ok := h.last[p.ID]; ok {
				d = json.RawMessage(b)
			}
			h.mu.Unlock()
		}
		out = append(out, panelDTO{p.ID, p.Title, p.Order, d})
	}
	return map[string]any{"seq": seq, "now": time.Now().UnixMilli(), "panels": out}
}

// Mount serves the display under prefix (for example "/live/") on mux.
func (h *Hub) Mount(mux *http.ServeMux, prefix string) {
	prefix = "/" + strings.Trim(prefix, "/")
	mux.Handle(prefix+"/", http.StripPrefix(prefix, h.Handler()))
	mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, prefix+"/", http.StatusMovedPermanently)
	})
}

// Handler serves everything with the mount prefix already stripped.
// All URLs in the page are relative, so any prefix works.
func (h *Hub) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/snapshot", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(h.snapshot())
	})
	mux.HandleFunc("/api/events", h.serveSSE)
	mux.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		files, _ := fs.Glob(live.FS, "panels/*.js")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(map[string]any{"modules": files})
	})
	mux.HandleFunc("/api/panel/", func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/api/panel/")
		id, sub, _ := strings.Cut(rest, "/")
		h.mu.Lock()
		p := h.panels[id]
		h.mu.Unlock()
		if p == nil || p.Routes == nil {
			http.NotFound(w, r)
			return
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/" + sub
		p.Routes.ServeHTTP(w, r2)
	})
	files := http.FileServer(http.FS(live.FS))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	}))
	return mux
}

func (h *Hub) serveSSE(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	s := &sub{ch: make(chan message, 256)}
	// Subscribe before taking the snapshot so no update is lost in between;
	// clients ignore updates whose seq is not newer than the snapshot's.
	h.mu.Lock()
	h.subs[s] = struct{}{}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		if _, ok := h.subs[s]; ok {
			delete(h.subs, s)
			close(s.ch)
		}
		h.mu.Unlock()
	}()
	b, _ := json.Marshal(h.snapshot())
	fmt.Fprintf(w, "retry: 2000\nevent: snapshot\ndata: %s\n\n", b)
	fl.Flush()
	hb := time.NewTicker(15 * time.Second)
	defer hb.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case m, ok := <-s.ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", m.event, m.data)
			fl.Flush()
		case <-hb.C:
			fmt.Fprint(w, ": ping\n\n")
			fl.Flush()
		}
	}
}

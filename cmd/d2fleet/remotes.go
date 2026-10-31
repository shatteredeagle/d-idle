package main

import (
	"net/http"
	"sync"

	"d2lod-bot/internal/fleet"
	"d2lod-bot/internal/server"
	"d2lod-bot/internal/viewport"
)

// Limits of the per-slot game view: one shared capture per slot, at most viewerMax clients per slot, and
// captureMax captures in flight over all slots (xwd and the JPEG encode cost a little CPU each).
const (
	viewerMax  = 8
	captureMax = 4
)

// viewHub keeps one viewport stream per slot display; the capture only runs while somebody watches.
type viewHub struct {
	capt viewport.Capturer
	lim  *viewport.Limiter

	mu sync.Mutex
	m  map[string]*viewEntry // by slot id
}

type viewEntry struct {
	display string
	st      *viewport.Stream
	h       http.Handler
}

func newViewHub(c viewport.Capturer) *viewHub {
	return &viewHub{capt: c, lim: viewport.NewLimiter(captureMax), m: map[string]*viewEntry{}}
}

// handler returns the viewport handler of slot id on its private X display (the slot's Alloc.Display).
func (v *viewHub) handler(id, display string) http.Handler {
	v.mu.Lock()
	defer v.mu.Unlock()
	if e := v.m[id]; e != nil && e.display == display {
		return e.h
	}
	if e := v.m[id]; e != nil {
		e.st.Close()
	}
	st := viewport.New(viewport.Config{Capturer: v.capt, Limiter: v.lim, FPS: 4, MaxW: 800, Quality: 65,
		Target: func() (viewport.Spec, error) { return viewport.Spec{Display: display}, nil }})
	e := &viewEntry{display: display, st: st, h: viewport.Handler(viewport.HandlerConfig{Stream: st, MaxViewers: viewerMax})}
	v.m[id] = e
	return e.h
}

// prune closes the streams of slots that are gone.
func (v *viewHub) prune(keep map[string]bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for id, e := range v.m {
		if !keep[id] {
			e.st.Close()
			delete(v.m, id)
		}
	}
}

func (v *viewHub) Close() { v.prune(nil) }

// remotesOf shows every slot as a server instance: the slot's d2bot API at its CURRENT address (d2bot listens on
// an ephemeral port and gets a new one after every restart) and its game display as the "viewport" mount.
func remotesOf(f *fleet.Fleet, vh *viewHub) []server.Remote {
	var out []server.Remote
	keep := map[string]bool{}
	for _, v := range f.List() {
		r := server.Remote{ID: v.ID, Title: v.Spec.Character, State: string(v.State), Error: v.LastError}
		if b := v.Bot; b != nil {
			if b.State == fleet.BotRunning {
				r.Addr = b.Addr
			}
			if r.Error == "" && b.State != fleet.BotRunning && b.LastError != "" {
				r.Error = "d2bot " + b.State + ": " + b.LastError
			}
		}
		if vh != nil && v.Alloc.Display != "" {
			r.Mounts = map[string]http.Handler{"viewport": vh.handler(v.ID, v.Alloc.Display)}
			keep[v.ID] = true
		}
		out = append(out, r)
	}
	if vh != nil {
		vh.prune(keep)
	}
	return out
}

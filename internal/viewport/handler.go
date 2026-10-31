package viewport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/input"
)

// HandlerConfig configures Handler.
type HandlerConfig struct {
	Stream *Stream
	// Scene returns the current overlay scene; nil or false: an empty overlay.
	Scene func() (Scene, bool)
	// Transform projects world points; zero value: input.Calibrated.
	Transform input.Transform
	// MaxFrameAge for /frame.jpg; default max(250 ms, 2 frame periods).
	MaxFrameAge time.Duration
	// MaxViewers caps concurrent /stream clients of this handler (503 beyond it); 0 = no cap.
	MaxViewers int
}

// Handler serves the viewport routes of one instance, relative to the mount (the server strips
// /api/v1/instances/{id}/viewport):
//
//	GET /frame.jpg     one JPEG (503 with the capture error text when there is none)
//	GET /stream.mjpg   multipart/x-mixed-replace MJPEG, new frame whenever one is produced (alias /stream)
//	GET /overlay       Overlay JSON (screen-space shapes for the newest frame)
//	GET /status        Status JSON
//
// GET and HEAD only. No input path exists.
func Handler(c HandlerConfig) http.Handler {
	if c.Transform == (input.Transform{}) {
		c.Transform = input.Calibrated
	}
	h := &handler{c: c}
	if h.c.MaxFrameAge <= 0 {
		h.c.MaxFrameAge = max(250*time.Millisecond, 2*c.Stream.period)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/frame.jpg", h.frame)
	mux.HandleFunc("/stream.mjpg", h.stream)
	mux.HandleFunc("/stream", h.stream)
	mux.HandleFunc("/overlay", h.overlay)
	mux.HandleFunc("/status", h.status)
	return readOnly(mux)
}

func readOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "view only", http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type handler struct {
	c       HandlerConfig
	viewers atomic.Int64
}

func (h *handler) frame(w http.ResponseWriter, r *http.Request) {
	ServeFrame(h.c.Stream, h.c.MaxFrameAge, w, r)
}
func (h *handler) stream(w http.ResponseWriter, r *http.Request) {
	if m := int64(h.c.MaxViewers); m > 0 {
		if h.viewers.Add(1) > m {
			h.viewers.Add(-1)
			w.Header().Set("Retry-After", "5")
			http.Error(w, "too many viewers", http.StatusServiceUnavailable)
			return
		}
		defer h.viewers.Add(-1)
	}
	ServeStream(h.c.Stream, w, r)
}

// ServeFrame writes one JPEG; usable directly by hosts with their own routing (d2watch).
func ServeFrame(s *Stream, maxAge time.Duration, w http.ResponseWriter, r *http.Request) {
	if maxAge <= 0 {
		maxAge = max(250*time.Millisecond, 2*s.period)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	f, err := s.Frame(ctx, maxAge)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", f.Mime)
	w.Header().Set("Content-Length", fmt.Sprint(len(f.Data)))
	w.Write(f.Data)
}

// StreamWriteTimeout bounds one frame write: a viewer that cannot take a frame within it (a backgrounded tab
// whose TCP window is full) is dropped instead of pinning a goroutine and a socket.
const StreamWriteTimeout = 3 * time.Second

// ServeStream writes multipart MJPEG until the client goes away. Latest frame wins: a slow client skips frames.
func ServeStream(s *Stream, w http.ResponseWriter, r *http.Request) {
	done := s.Watch()
	defer done()
	rc := http.NewResponseController(w)
	var last uint64
	started := false
	for {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		f, err := s.Next(ctx, last)
		cancel()
		if r.Context().Err() != nil {
			return
		}
		if err != nil {
			if s.closedNow() {
				return
			}
			if _, ok := s.Latest(); !started && !ok && r.Context().Err() == nil && !errors.Is(err, context.DeadlineExceeded) {
				http.Error(w, err.Error(), http.StatusServiceUnavailable) // nothing captured yet: a status, not a hung stream
				return
			}
			continue
		}
		if !started {
			started = true
			w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
			w.Header().Set("Cache-Control", "no-store")
		}
		last = f.Seq
		rc.SetWriteDeadline(time.Now().Add(StreamWriteTimeout))
		if _, err := fmt.Fprintf(w, "--frame\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n", f.Mime, len(f.Data)); err != nil {
			return
		}
		if _, err := w.Write(f.Data); err != nil {
			return
		}
		if _, err := w.Write([]byte("\r\n")); err != nil {
			return
		}
		if rc.Flush() != nil {
			return
		}
	}
}

func (s *Stream) closedNow() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func (h *handler) overlay(w http.ResponseWriter, r *http.Request) {
	var sc Scene
	if h.c.Scene != nil {
		sc, _ = h.c.Scene()
	}
	o := Project(h.c.Transform, sc)
	if st := h.c.Stream.Status(); st.Seq > 0 {
		o.Frame = st.Seq
	}
	writeJSON(w, o)
}

func (h *handler) status(w http.ResponseWriter, r *http.Request) {
	st := h.c.Stream.Status()
	age := -1
	if !st.FrameAt.IsZero() {
		age = int(time.Since(st.FrameAt) / time.Millisecond)
	}
	writeJSON(w, map[string]any{"running": st.Running, "viewers": st.Viewers, "seq": st.Seq, "frameAgeMs": age, "error": st.Err})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

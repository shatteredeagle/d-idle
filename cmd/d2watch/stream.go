package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/viewport"
)

// streamWriteTimeout bounds one frame write: a viewer that cannot take a frame within it (a backgrounded or
// throttled tab whose TCP window is full) is dropped instead of pinning a goroutine and socket.
const streamWriteTimeout = 3 * time.Second

// viewerLimit caps concurrent MJPEG viewers over all instances. The capture itself is shared (one loop per
// instance), so a viewer only costs a goroutine and a socket, but a leaked page that reconnects in a loop
// should not be able to grow that without bound.
type viewerLimit struct {
	max int64
	cur atomic.Int64
}

func (l *viewerLimit) acquire() bool {
	if l.cur.Add(1) > l.max {
		l.cur.Add(-1)
		return false
	}
	return true
}
func (l *viewerLimit) release() { l.cur.Add(-1) }

// serveStream is a per-client MJPEG writer over the shared Stream. Latest frame wins: Next returns the newest
// frame, so a slow client skips frames instead of queueing them. It logs one stats line per client at close.
func serveStream(s *viewport.Stream, lim *viewerLimit, id string, w http.ResponseWriter, r *http.Request) {
	if !lim.acquire() {
		log.Printf("d2watch: stream %s from %s refused: %d viewers (max)", id, r.RemoteAddr, lim.max)
		w.Header().Set("Retry-After", "5")
		http.Error(w, "too many viewers", http.StatusServiceUnavailable)
		return
	}
	defer lim.release()
	done := s.Watch()
	defer done()

	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-store")
	start := time.Now()
	var last, sent, skipped, bytes uint64
	reason := "client gone"
	defer func() {
		log.Printf("d2watch: stream %s %s closed after %s (%s): %d frames, %d skipped, %d KiB",
			id, r.RemoteAddr, time.Since(start).Round(time.Second), reason, sent, skipped, bytes/1024)
	}()
	for {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		f, err := s.Next(ctx, last)
		cancel()
		if r.Context().Err() != nil {
			return
		}
		if err != nil {
			if s.Closed() { // instance gone: Next would fail instantly forever (a hot loop)
				reason = "instance gone"
				return
			}
			if !errors.Is(err, context.DeadlineExceeded) { // not a mere idle timeout: capture error
				if _, ok := s.Latest(); !ok {
					reason = "no capture: " + err.Error()
					return
				}
			}
			continue
		}
		if last != 0 && f.Seq > last+1 {
			skipped += f.Seq - last - 1
		}
		last = f.Seq
		rc.SetWriteDeadline(time.Now().Add(streamWriteTimeout))
		if _, err := fmt.Fprintf(w, "--frame\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n", f.Mime, len(f.Data)); err != nil {
			reason = "write: " + err.Error()
			return
		}
		if _, err := w.Write(f.Data); err != nil {
			reason = "write: " + err.Error()
			return
		}
		if _, err := w.Write([]byte("\r\n")); err != nil {
			reason = "write: " + err.Error()
			return
		}
		if err := rc.Flush(); err != nil {
			reason = "flush: " + err.Error()
			return
		}
		sent++
		bytes += uint64(len(f.Data))
	}
}

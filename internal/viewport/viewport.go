// Package viewport captures a window on an X display and streams it to many viewers: one shared capture per
// source, lazy (no capture while nobody watches), frame-rate and size limited, one capture in flight, and a
// global concurrency cap. Handler is the http.Handler for the server's Instance.Mounts["viewport"] contract
// (docs/api.md). Docs: docs/viewport.md.
package viewport

import (
	"context"
	"errors"
	"time"
)

// Format names the wire format of a frame. Only MJPEG is implemented; Frame.Mime and Spec.Format are the
// extension point for an H.264 capturer (fragmented MP4 or WebSocket) without changing Stream.
const FormatMJPEG = "mjpeg"

// Frame is one encoded picture.
type Frame struct {
	Data []byte
	Mime string // "image/jpeg" for MJPEG
	Seq  uint64 // assigned by Stream, starts at 1
	At   time.Time
}

// Spec says what to capture and how. Zero limits mean "native" / the capturer default.
type Spec struct {
	Display string   // X display, e.g. ":101"
	Env     []string // extra environment for X tools (XAUTHORITY ...); nil: process environment plus DISPLAY
	Window  string   // X window id (decimal or 0x...); never the root window
	Format  string   // FormatMJPEG (default)
	FPS     float64  // target rate for persistent capturers
	MaxW    int      // downscale to fit, never upscale
	MaxH    int
	Quality int // JPEG quality 1..100, default 65
}

// Capturer opens capture sessions. Implementations: ImportCapturer (one `import -window` per frame),
// FFmpegCapturer (a persistent ffmpeg x11grab process).
type Capturer interface {
	Open(ctx context.Context, s Spec) (Session, error)
	// Persistent reports that an open session keeps a process running (it then holds one concurrency slot
	// for its whole life) instead of costing one process per Next.
	Persistent() bool
}

// Session delivers frames. Next blocks until the next frame (a pull capturer captures now, a persistent one
// waits for its process). Close releases the process. Not safe for concurrent use.
type Session interface {
	Next(ctx context.Context) (Frame, error)
	Close() error
}

// ErrNoTarget is returned by a Target func when there is nothing to capture yet (no window).
var ErrNoTarget = errors.New("viewport: no capture target")

func (s Spec) quality() int {
	if s.Quality < 1 || s.Quality > 100 {
		return 65
	}
	return s.Quality
}

func (s Spec) fps() float64 {
	if s.FPS <= 0 {
		return 4
	}
	return s.FPS
}

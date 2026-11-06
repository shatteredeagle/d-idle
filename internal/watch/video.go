package watch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Game video (docs/research/watch-screen-design.md, "Game video"): one shared capture of the instance's private display,
// started when the first viewer connects and stopped Linger after the last one leaves, served as MJPEG on the viewer
// listener. Off unless the bot is started with -watch-video.

// CaptureFunc starts a capture and returns its JPEG frames. The channel closes when the capture ends or ctx is
// cancelled; a capture that cannot start returns an error.
type CaptureFunc func(ctx context.Context) (<-chan []byte, error)

// VideoOptions configure a Video. Zero values take the defaults.
type VideoOptions struct {
	Capture    CaptureFunc
	MaxViewers int           // default 3
	MaxFrame   int           // bytes; larger frames are dropped; default 512 KiB
	Linger     time.Duration // how long the capture outlives the last viewer; default 5 s
}

// Video is the shared frame source behind /stream and /shot.jpg.
type Video struct {
	o VideoOptions

	mu     sync.Mutex
	subs   map[chan []byte]struct{}
	cancel context.CancelFunc
	last   []byte
	timer  *time.Timer
	starts int
}

var errVideoFull = errors.New("video: viewer limit reached")

// NewVideo returns a Video; it starts nothing until a viewer joins.
func NewVideo(o VideoOptions) (*Video, error) {
	if o.Capture == nil {
		return nil, errors.New("video: no capture")
	}
	if o.MaxViewers <= 0 {
		o.MaxViewers = 3
	}
	if o.MaxFrame <= 0 {
		o.MaxFrame = 512 << 10
	}
	if o.Linger <= 0 {
		o.Linger = 5 * time.Second
	}
	return &Video{o: o, subs: map[chan []byte]struct{}{}}, nil
}

// Running reports whether a capture is active; Viewers the number of joined viewers; Starts how often a capture started.
func (v *Video) Running() bool { v.mu.Lock(); defer v.mu.Unlock(); return v.cancel != nil }
func (v *Video) Viewers() int  { v.mu.Lock(); defer v.mu.Unlock(); return len(v.subs) }
func (v *Video) Starts() int   { v.mu.Lock(); defer v.mu.Unlock(); return v.starts }

// join adds a viewer, starting the capture when it is the first. leave must be called once.
func (v *Video) join() (ch chan []byte, first []byte, leave func(), err error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if len(v.subs) >= v.o.MaxViewers {
		return nil, nil, nil, errVideoFull
	}
	if v.timer != nil {
		v.timer.Stop()
		v.timer = nil
	}
	if v.cancel == nil {
		ctx, cancel := context.WithCancel(context.Background())
		frames, err := v.o.Capture(ctx)
		if err != nil {
			cancel()
			return nil, nil, nil, err
		}
		v.cancel = cancel
		v.starts++
		go v.pump(frames, v.starts)
	}
	ch = make(chan []byte, 1)
	v.subs[ch] = struct{}{}
	return ch, v.last, func() { v.leave(ch) }, nil
}

func (v *Video) leave(ch chan []byte) {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.subs, ch)
	if len(v.subs) == 0 && v.cancel != nil && v.timer == nil {
		v.timer = time.AfterFunc(v.o.Linger, v.idle)
	}
}

// idle stops the capture when nobody came back during the linger time.
func (v *Video) idle() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.timer = nil
	if len(v.subs) == 0 && v.cancel != nil {
		v.cancel()
		v.cancel = nil
		v.last = nil
	}
}

// pump fans frames out to the viewers; a slow viewer only ever misses frames. When the capture ends the viewers are
// closed (their pages reconnect) and the next viewer starts a new one.
func (v *Video) pump(frames <-chan []byte, gen int) {
	for f := range frames {
		if len(f) == 0 || len(f) > v.o.MaxFrame {
			continue
		}
		v.mu.Lock()
		v.last = f
		for ch := range v.subs {
			select {
			case ch <- f:
			default:
				select {
				case <-ch:
				default:
				}
				select {
				case ch <- f:
				default:
				}
			}
		}
		v.mu.Unlock()
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	// Only the capture that ended is cleared; after idle and a rejoin a newer one (higher gen) is running.
	if gen == v.starts && v.cancel != nil {
		v.cancel()
		v.cancel = nil
		v.last = nil
		for ch := range v.subs {
			close(ch)
			delete(v.subs, ch)
		}
	}
}

const boundary = "d2frame"

// handleStream serves multipart/x-mixed-replace until the client leaves or the capture ends.
func (v *Video) handleStream(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "no streaming", http.StatusInternalServerError)
		return
	}
	ch, first, leave, err := v.join()
	if err != nil {
		videoErr(w, err)
		return
	}
	defer leave()
	h := w.Header()
	h.Set("Content-Type", "multipart/x-mixed-replace; boundary="+boundary)
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	fl.Flush() // headers now: the page sees the stream open before the first frame exists
	write := func(f []byte) bool {
		var b bytes.Buffer
		fmt.Fprintf(&b, "--%s\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", boundary, len(f))
		b.Write(f)
		b.WriteString("\r\n")
		if _, err := w.Write(b.Bytes()); err != nil {
			return false
		}
		fl.Flush()
		return true
	}
	if first != nil && !write(first) {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case f, ok := <-ch:
			if !ok || !write(f) {
				return
			}
		}
	}
}

// handleShot serves one JPEG: the newest frame, waiting a few seconds for the first one of a fresh capture.
func (v *Video) handleShot(w http.ResponseWriter, r *http.Request) {
	ch, first, leave, err := v.join()
	if err != nil {
		videoErr(w, err)
		return
	}
	defer leave()
	f := first
	if f == nil {
		select {
		case f = <-ch:
		case <-time.After(5 * time.Second):
		case <-r.Context().Done():
			return
		}
	}
	if len(f) == 0 {
		http.Error(w, "no frame yet", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(f)
}

func videoErr(w http.ResponseWriter, err error) {
	code := http.StatusServiceUnavailable
	if errors.Is(err, errVideoFull) {
		code = http.StatusTooManyRequests
	}
	http.Error(w, err.Error(), code)
}

// SetVideo turns the game video on: the routes serve it and the page learns the stream URL from /api/config.
func (h *Hub) SetVideo(v *Video) {
	h.video = v
	if v != nil {
		h.cfg.FrameURL = "/stream"
	}
}
